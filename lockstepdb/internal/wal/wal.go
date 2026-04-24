package wal

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Entry struct {
	TxnID  uint64           `json:"txn_id"`
	Writes map[string]int64 `json:"writes"`
}

type Log struct {
	path string
	file *os.File
}

func Open(path string) (*Log, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	return &Log{path: path, file: file}, nil
}

func (l *Log) AppendCommit(txnID uint64, writes map[string]int64) error {
	if l == nil || l.file == nil {
		return errors.New("wal not initialized")
	}

	record := Entry{
		TxnID:  txnID,
		Writes: cloneWrites(writes),
	}

	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if _, err := l.file.Write(append(payload, '\n')); err != nil {
		return err
	}
	return l.file.Sync()
}

func (l *Log) Replay() (map[string]int64, error) {
	state := make(map[string]int64)
	if l == nil || l.file == nil {
		return state, nil
	}

	if _, err := l.file.Seek(0, 0); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(l.file)
	for scanner.Scan() {
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		for key, value := range entry.Writes {
			state[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	_, err := l.file.Seek(0, 2)
	return state, err
}

func (l *Log) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func cloneWrites(writes map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(writes))
	for key, value := range writes {
		out[key] = value
	}
	return out
}
