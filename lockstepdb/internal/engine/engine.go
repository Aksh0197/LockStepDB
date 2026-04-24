package engine

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

	"lockstepdb/internal/wal"
)

var (
	ErrUnknownTransaction = errors.New("unknown transaction")
	ErrUnknownKey         = errors.New("unknown key")
	ErrLockOrder          = errors.New("lock acquisition order violation")
	ErrLockConflict       = errors.New("lock conflict")
)

type Engine struct {
	mu      sync.Mutex
	nextTxn atomic.Uint64
	data    map[string]int64
	txns    map[uint64]*txn
	locks   map[string]*keyLock
	wal     *wal.Log
}

type txn struct {
	id        uint64
	active    bool
	held      map[string]lockMode
	writeSet  map[string]int64
	lockOrder []string
}

type lockMode byte

const (
	readLock  lockMode = 'R'
	writeLock lockMode = 'W'
)

type keyLock struct {
	writer          uint64
	readers         map[uint64]bool
	pendingUpgrader uint64
}

func New(log *wal.Log) (*Engine, error) {
	state, err := log.Replay()
	if err != nil {
		return nil, fmt.Errorf("replay wal: %w", err)
	}

	engine := &Engine{
		data:  state,
		txns:  make(map[uint64]*txn),
		locks: make(map[string]*keyLock),
		wal:   log,
	}
	return engine, nil
}

func (e *Engine) Begin() uint64 {
	txnID := e.nextTxn.Add(1)

	e.mu.Lock()
	defer e.mu.Unlock()

	e.txns[txnID] = &txn{
		id:       txnID,
		active:   true,
		held:     make(map[string]lockMode),
		writeSet: make(map[string]int64),
	}
	return txnID
}

func (e *Engine) Get(txnID uint64, key string) (int64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	txn, err := e.mustTxn(txnID)
	if err != nil {
		return 0, err
	}
	if err := e.acquireRead(txn, key); err != nil {
		return 0, err
	}
	if value, ok := txn.writeSet[key]; ok {
		return value, nil
	}
	value, ok := e.data[key]
	if !ok {
		return 0, ErrUnknownKey
	}
	return value, nil
}

func (e *Engine) Put(txnID uint64, key string, value int64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	txn, err := e.mustTxn(txnID)
	if err != nil {
		return err
	}
	if err := e.acquireWrite(txn, key); err != nil {
		return err
	}
	txn.writeSet[key] = value
	return nil
}

func (e *Engine) Commit(txnID uint64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	txn, err := e.mustTxn(txnID)
	if err != nil {
		return err
	}

	if err := e.wal.AppendCommit(txnID, txn.writeSet); err != nil {
		return fmt.Errorf("append wal: %w", err)
	}
	for key, value := range txn.writeSet {
		e.data[key] = value
	}
	e.release(txn)
	delete(e.txns, txnID)
	return nil
}

func (e *Engine) Rollback(txnID uint64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	txn, err := e.mustTxn(txnID)
	if err != nil {
		return err
	}
	e.release(txn)
	delete(e.txns, txnID)
	return nil
}

func (e *Engine) Snapshot() map[string]int64 {
	e.mu.Lock()
	defer e.mu.Unlock()

	out := make(map[string]int64, len(e.data))
	for key, value := range e.data {
		out[key] = value
	}
	return out
}

func (e *Engine) mustTxn(txnID uint64) (*txn, error) {
	txn, ok := e.txns[txnID]
	if !ok || !txn.active {
		return nil, ErrUnknownTransaction
	}
	return txn, nil
}

func (e *Engine) acquireRead(txn *txn, key string) error {
	if mode, ok := txn.held[key]; ok && (mode == readLock || mode == writeLock) {
		return nil
	}
	if err := ensureLockOrder(txn, key); err != nil {
		return err
	}

	lock := e.lockFor(key)
	if lock.writer != 0 && lock.writer != txn.id {
		return ErrLockConflict
	}
	if lock.pendingUpgrader != 0 && lock.pendingUpgrader != txn.id {
		return ErrLockConflict
	}

	lock.readers[txn.id] = true
	txn.held[key] = readLock
	return nil
}

func (e *Engine) acquireWrite(txn *txn, key string) error {
	if mode, ok := txn.held[key]; ok && mode == writeLock {
		return nil
	}
	if _, ok := txn.held[key]; !ok {
		if err := ensureLockOrder(txn, key); err != nil {
			return err
		}
	}

	lock := e.lockFor(key)
	if lock.writer != 0 && lock.writer != txn.id {
		return ErrLockConflict
	}

	if txn.held[key] == readLock {
		if lock.pendingUpgrader != 0 && lock.pendingUpgrader != txn.id {
			return ErrLockConflict
		}
		lock.pendingUpgrader = txn.id
		if len(lock.readers) != 1 || !lock.readers[txn.id] {
			return ErrLockConflict
		}
		delete(lock.readers, txn.id)
		lock.writer = txn.id
		lock.pendingUpgrader = 0
		txn.held[key] = writeLock
		return nil
	}

	if len(lock.readers) > 0 {
		return ErrLockConflict
	}
	lock.writer = txn.id
	txn.held[key] = writeLock
	return nil
}

func (e *Engine) release(txn *txn) {
	for key, mode := range txn.held {
		lock := e.lockFor(key)
		if mode == writeLock && lock.writer == txn.id {
			lock.writer = 0
		}
		delete(lock.readers, txn.id)
		if lock.pendingUpgrader == txn.id {
			lock.pendingUpgrader = 0
		}
		if lock.writer == 0 && len(lock.readers) == 0 && lock.pendingUpgrader == 0 {
			delete(e.locks, key)
		}
	}
	txn.active = false
}

func (e *Engine) lockFor(key string) *keyLock {
	lock, ok := e.locks[key]
	if ok {
		return lock
	}
	lock = &keyLock{readers: make(map[uint64]bool)}
	e.locks[key] = lock
	return lock
}

func ensureLockOrder(txn *txn, key string) error {
	if len(txn.lockOrder) == 0 {
		txn.lockOrder = append(txn.lockOrder, key)
		return nil
	}
	last := txn.lockOrder[len(txn.lockOrder)-1]
	if key < last {
		return fmt.Errorf("%w: tried %q after %q", ErrLockOrder, key, last)
	}
	if key > last {
		txn.lockOrder = append(txn.lockOrder, key)
	}
	return nil
}

func SortedKeys(m map[string]int64) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
