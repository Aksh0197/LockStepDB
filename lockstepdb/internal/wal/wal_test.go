package wal

import (
	"path/filepath"
	"testing"
)

func TestReplayRestoresLatestValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lockstep.wal")

	log, err := Open(path)
	if err != nil {
		t.Fatalf("open wal: %v", err)
	}
	if err := log.AppendCommit(1, map[string]int64{"x": 1, "y": 2}); err != nil {
		t.Fatalf("append 1: %v", err)
	}
	if err := log.AppendCommit(2, map[string]int64{"x": 3}); err != nil {
		t.Fatalf("append 2: %v", err)
	}
	if err := log.Close(); err != nil {
		t.Fatalf("close wal: %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen wal: %v", err)
	}
	defer reopened.Close()

	state, err := reopened.Replay()
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if state["x"] != 3 {
		t.Fatalf("expected x=3, got %d", state["x"])
	}
	if state["y"] != 2 {
		t.Fatalf("expected y=2, got %d", state["y"])
	}
}
