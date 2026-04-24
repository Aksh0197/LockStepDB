# LockstepDB

LockstepDB is a transactional key-value database engine written in Go. It exposes a lightweight TCP/RPC interface, supports concurrent clients, uses strict two-phase locking with safe read-to-write promotion, and persists committed updates through a write-ahead log (WAL) for crash recovery.

## Overview

This project began as an Ethos transactional RPC assignment and was redesigned into a standard Go codebase using the Go standard library. The system models the core pieces of a database transaction manager:

- transaction lifecycle with `begin`, `commit`, and `rollback`
- `get` and `put` operations over a shared key-value store
- per-key read and write locking
- read-to-write lock promotion when safe
- private transaction write buffering until commit
- deterministic lock ordering to avoid deadlock-prone acquisition patterns
- durable commit logging and state recovery through WAL replay

## Features

- Concurrent TCP server for multiple clients
- Transaction-local read-your-own-write semantics
- Strict lock retention until commit or rollback
- Safe upgrade path from shared read lock to exclusive write lock
- WAL-backed durability for committed writes
- Crash recovery by replaying committed log entries
- Small concurrent demo client for behavior verification
- Unit tests covering engine semantics and recovery

## Project Layout

- `cmd/server`: starts the LockstepDB server
- `cmd/demo`: runs a multi-client demo against the server
- `internal/engine`: transaction state, lock table, commit logic, and recovery integration
- `internal/wal`: append-only WAL implementation
- `internal/server`: TCP listener and request dispatcher
- `internal/protocol`: JSON request and response types

## Concurrency Model

LockstepDB uses strict two-phase locking:

- Read operations acquire shared locks.
- Write operations acquire exclusive locks.
- Locks are held until the transaction ends.
- Transactions may upgrade a read lock to a write lock only when they are the sole reader.
- First-time lock acquisition follows lexicographic key order, which avoids the lock-order inversions that commonly lead to deadlocks.

## Durability Model

On commit, the engine:

1. Appends the transaction write-set to the WAL
2. Forces the WAL to disk with `fsync`
3. Applies the writes to the in-memory state
4. Releases transaction locks

On restart, the server reconstructs committed state by replaying WAL records.

## Quick Start

Start the server:

```bash
go run ./cmd/server
```

In another terminal, run the concurrent demo:

```bash
go run ./cmd/demo
```

By default, the server listens on `127.0.0.1:9000` and writes commit records to `./data/lockstep.wal`.

## Testing

```bash
go test ./...
```

## Why This Project Matters

LockstepDB is a focused systems project that demonstrates:

- transaction processing fundamentals
- concurrency control in Go
- lock management and upgrade semantics
- durability via write-ahead logging
- crash recovery behavior
- custom networked request handling without external frameworks
