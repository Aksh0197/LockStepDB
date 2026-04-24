package protocol

type Request struct {
	Action string `json:"action"`
	TxnID  uint64 `json:"txn_id,omitempty"`
	Key    string `json:"key,omitempty"`
	Value  int64  `json:"value,omitempty"`
}

type Response struct {
	OK    bool   `json:"ok"`
	TxnID uint64 `json:"txn_id,omitempty"`
	Value int64  `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
}
