package server

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"

	"lockstepdb/internal/engine"
	"lockstepdb/internal/protocol"
)

type Server struct {
	address string
	engine  *engine.Engine
	logger  *log.Logger
}

func New(address string, eng *engine.Engine, logger *log.Logger) *Server {
	return &Server{
		address: address,
		engine:  eng,
		logger:  logger,
	}
}

func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	defer listener.Close()

	s.logger.Printf("LockstepDB listening on %s", s.address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var req protocol.Request
		if err := decoder.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			s.logger.Printf("decode request: %v", err)
			_ = encoder.Encode(protocol.Response{OK: false, Error: err.Error()})
			return
		}

		resp := s.dispatch(req)
		if err := encoder.Encode(resp); err != nil {
			s.logger.Printf("encode response: %v", err)
			return
		}
	}
}

func (s *Server) dispatch(req protocol.Request) protocol.Response {
	switch req.Action {
	case "begin":
		txnID := s.engine.Begin()
		s.logger.Printf("begin txn=%d", txnID)
		return protocol.Response{OK: true, TxnID: txnID}
	case "get":
		value, err := s.engine.Get(req.TxnID, req.Key)
		if err != nil {
			return protocol.Response{OK: false, Error: err.Error()}
		}
		return protocol.Response{OK: true, Value: value}
	case "put":
		if err := s.engine.Put(req.TxnID, req.Key, req.Value); err != nil {
			return protocol.Response{OK: false, Error: err.Error()}
		}
		return protocol.Response{OK: true}
	case "commit":
		if err := s.engine.Commit(req.TxnID); err != nil {
			return protocol.Response{OK: false, Error: err.Error()}
		}
		s.logger.Printf("commit txn=%d", req.TxnID)
		return protocol.Response{OK: true}
	case "rollback":
		if err := s.engine.Rollback(req.TxnID); err != nil {
			return protocol.Response{OK: false, Error: err.Error()}
		}
		s.logger.Printf("rollback txn=%d", req.TxnID)
		return protocol.Response{OK: true}
	case "snapshot":
		snapshot := s.engine.Snapshot()
		keys := engine.SortedKeys(snapshot)
		var value int64
		if len(keys) > 0 {
			value = int64(len(keys))
		}
		return protocol.Response{OK: true, Value: value}
	default:
		return protocol.Response{OK: false, Error: "unknown action"}
	}
}
