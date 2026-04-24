package engine

import (
	"path/filepath"
	"testing"

	"lockstepdb/internal/wal"
)

func newTestEngine(t *testing.T) *Engine {
	t.Helper()

	log, err := wal.Open(filepath.Join(t.TempDir(), "test.wal"))
	if err != nil {
		t.Fatalf("open wal: %v", err)
	}
	t.Cleanup(func() {
		_ = log.Close()
	})

	engine, err := New(log)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	return engine
}

func TestCommitPersistsAndReadsOwnWrites(t *testing.T) {
	engine := newTestEngine(t)

	txnID := engine.Begin()
	if err := engine.Put(txnID, "alpha", 42); err != nil {
		t.Fatalf("put: %v", err)
	}
	value, err := engine.Get(txnID, "alpha")
	if err != nil {
		t.Fatalf("get own write: %v", err)
	}
	if value != 42 {
		t.Fatalf("expected 42, got %d", value)
	}
	if err := engine.Commit(txnID); err != nil {
		t.Fatalf("commit: %v", err)
	}

	afterTxn := engine.Begin()
	defer engine.Rollback(afterTxn)
	value, err = engine.Get(afterTxn, "alpha")
	if err != nil {
		t.Fatalf("get committed: %v", err)
	}
	if value != 42 {
		t.Fatalf("expected committed 42, got %d", value)
	}
}

func TestRejectsOutOfOrderLocking(t *testing.T) {
	engine := newTestEngine(t)

	txnID := engine.Begin()
	if err := engine.Put(txnID, "beta", 1); err != nil {
		t.Fatalf("put beta: %v", err)
	}
	if err := engine.Put(txnID, "alpha", 2); err == nil {
		t.Fatal("expected lock order error")
	}
}

func TestConflictingPromotionFails(t *testing.T) {
	engine := newTestEngine(t)

	seedTxn := engine.Begin()
	if err := engine.Put(seedTxn, "alpha", 10); err != nil {
		t.Fatalf("seed put: %v", err)
	}
	if err := engine.Commit(seedTxn); err != nil {
		t.Fatalf("seed commit: %v", err)
	}

	txnA := engine.Begin()
	txnB := engine.Begin()

	if _, err := engine.Get(txnA, "alpha"); err != nil {
		t.Fatalf("txnA get: %v", err)
	}
	if _, err := engine.Get(txnB, "alpha"); err != nil {
		t.Fatalf("txnB get: %v", err)
	}
	if err := engine.Put(txnA, "alpha", 11); err == nil {
		t.Fatal("expected promotion conflict")
	}
}
