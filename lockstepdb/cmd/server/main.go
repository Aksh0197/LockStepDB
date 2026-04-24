package main

import (
	"log"
	"os"

	"lockstepdb/internal/engine"
	"lockstepdb/internal/server"
	"lockstepdb/internal/wal"
)

func main() {
	logger := log.New(os.Stdout, "[server] ", log.LstdFlags|log.Lmicroseconds)

	logFile, err := wal.Open("data/lockstep.wal")
	if err != nil {
		logger.Fatalf("open wal: %v", err)
	}
	defer logFile.Close()

	db, err := engine.New(logFile)
	if err != nil {
		logger.Fatalf("create engine: %v", err)
	}

	srv := server.New("127.0.0.1:9000", db, logger)
	if err := srv.ListenAndServe(); err != nil {
		logger.Fatalf("server stopped: %v", err)
	}
}
