# Protocol Guide

LockstepDB uses newline-delimited JSON messages over a TCP connection.

## Request Shape

```json
{
  "action": "begin",
  "txn_id": 1,
  "key": "account:alice",
  "value": 100
}
```

Not every field is required for every action.

## Response Shape

```json
{
  "ok": true,
  "txn_id": 1,
  "value": 100,
  "error": ""
}
```

## Supported Actions

### `begin`

Starts a new transaction.

Request:

```json
{"action":"begin"}
```

Response:

```json
{"ok":true,"txn_id":1}
```

### `get`

Reads a value within a transaction.

Request:

```json
{"action":"get","txn_id":1,"key":"account:alice"}
```

Response:

```json
{"ok":true,"value":100}
```

### `put`

Buffers a write inside a transaction. The value becomes visible to others only after commit.

Request:

```json
{"action":"put","txn_id":1,"key":"account:alice","value":150}
```

Response:

```json
{"ok":true}
```

### `commit`

Durably commits a transaction.

Request:

```json
{"action":"commit","txn_id":1}
```

Response:

```json
{"ok":true}
```

### `rollback`

Aborts a transaction and releases its locks.

Request:

```json
{"action":"rollback","txn_id":1}
```

Response:

```json
{"ok":true}
```

## Error Cases

Common errors include:

- `unknown transaction`
- `unknown key`
- `lock conflict`
- `lock acquisition order violation`

These are returned in the `error` field when `ok` is `false`.
