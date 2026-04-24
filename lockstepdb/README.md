# LockstepDB

LockstepDB is a transactional key-value database engine written in Go. It exposes a custom TCP/RPC interface, supports concurrent clients, uses strict two-phase locking with safe read-to-write promotion, and persists committed updates through a write-ahead log (WAL) for crash recovery.

## What This Project Does

The original Ethos assignment code models a small transactional database with:

- transaction begin and commit
- `GET` and `PUT` operations
- per-key read and write locks
- lock promotion from read to write
- buffered writes that become visible only on commit
- deadlock avoidance through deterministic key-order locking

This repository converts that idea into a standard Go implementation using the standard library so it can run anywhere Go runs.

## Architecture

- `cmd/server`: starts the TCP database server
- `cmd/demo`: runs a concurrent client demo against the server
- `internal/engine`: transaction manager, lock table, commit logic, recovery
- `internal/wal`: append-only WAL with fsync-backed durability
- `internal/server`: TCP listener and request dispatcher
- `internal/protocol`: request/response message schema

## Guarantees

- Uncommitted writes stay private to the transaction.
- Reads within a transaction see their own buffered writes.
- Locks are held until commit or rollback.
- Only one read-to-write upgrader is allowed per key at a time to avoid upgrade deadlocks.
- The server appends and syncs a commit record to the WAL before applying writes in memory.
- On restart, the server rebuilds state by replaying the WAL.

## Run

Start the server:

```bash
go run ./cmd/server
```

In another terminal, run the demo:

```bash
go run ./cmd/demo
```

The server listens on `127.0.0.1:9000` and stores its log in `./data/lockstep.wal`.
