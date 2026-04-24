package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"lockstepdb/internal/protocol"
)

type client struct {
	conn net.Conn
	dec  *json.Decoder
	enc  *json.Encoder
}

func main() {
	logger := log.New(os.Stdout, "[demo] ", log.LstdFlags|log.Lmicroseconds)

	seedClient(logger)

	var wg sync.WaitGroup
	wg.Add(2)

	go runClient(&wg, logger, "client-A", func(c *client) {
		txnID := mustBegin(c)
		resp := mustRequest(c, protocol.Request{Action: "get", TxnID: txnID, Key: "account:alice"})
		logger.Printf("client-A read: %+v", resp)
		time.Sleep(250 * time.Millisecond)
		_ = mustRequest(c, protocol.Request{Action: "rollback", TxnID: txnID})
	})

	go runClient(&wg, logger, "client-B", func(c *client) {
		time.Sleep(50 * time.Millisecond)
		txnID := mustBegin(c)
		resp := mustRequest(c, protocol.Request{Action: "get", TxnID: txnID, Key: "account:alice"})
		logger.Printf("client-B read: %+v", resp)
		resp = mustRequest(c, protocol.Request{Action: "put", TxnID: txnID, Key: "account:alice", Value: 150})
		logger.Printf("client-B promotion result: %+v", resp)
		_ = mustRequest(c, protocol.Request{Action: "rollback", TxnID: txnID})
	})

	wg.Wait()

	var verifyWG sync.WaitGroup
	verifyWG.Add(1)
	go runClient(&verifyWG, logger, "client-C", func(c *client) {
		txnID := mustBegin(c)
		resp := mustRequest(c, protocol.Request{Action: "get", TxnID: txnID, Key: "account:alice"})
		logger.Printf("client-C final read: %+v", resp)
		mustCommit(c, txnID)
	})
	verifyWG.Wait()
}

func runClient(wg *sync.WaitGroup, logger *log.Logger, name string, fn func(*client)) {
	defer wg.Done()

	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		logger.Printf("%s dial error: %v", name, err)
		return
	}
	defer conn.Close()

	fn(&client{
		conn: conn,
		dec:  json.NewDecoder(conn),
		enc:  json.NewEncoder(conn),
	})
}

func mustBegin(c *client) uint64 {
	resp := mustRequest(c, protocol.Request{Action: "begin"})
	if !resp.OK {
		panic(resp.Error)
	}
	return resp.TxnID
}

func seedClient(logger *log.Logger) {
	var wg sync.WaitGroup
	wg.Add(1)

	go runClient(&wg, logger, "seed-client", func(c *client) {
		txnID := mustBegin(c)
		mustPut(c, txnID, "account:alice", 100)
		mustCommit(c, txnID)
	})

	wg.Wait()
}

func mustPut(c *client, txnID uint64, key string, value int64) {
	resp := mustRequest(c, protocol.Request{Action: "put", TxnID: txnID, Key: key, Value: value})
	if !resp.OK {
		panic(resp.Error)
	}
}

func mustCommit(c *client, txnID uint64) {
	resp := mustRequest(c, protocol.Request{Action: "commit", TxnID: txnID})
	if !resp.OK {
		panic(resp.Error)
	}
}

func mustRequest(c *client, req protocol.Request) protocol.Response {
	if err := c.enc.Encode(req); err != nil {
		panic(err)
	}
	var resp protocol.Response
	if err := c.dec.Decode(&resp); err != nil {
		panic(err)
	}
	return resp
}

func (c *client) String() string {
	return fmt.Sprintf("client(%s)", c.conn.RemoteAddr())
}
