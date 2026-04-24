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

	address := envOrDefault("LOCKSTEPDB_ADDR", "127.0.0.1:9000")
	walPath := envOrDefault("LOCKSTEPDB_WAL", "data/lockstep.wal")

	logFile, err := wal.Open(walPath)
	if err != nil {
		logger.Fatalf("open wal: %v", err)
	}
	defer logFile.Close()

	db, err := engine.New(logFile)
	if err != nil {
		logger.Fatalf("create engine: %v", err)
	}

	srv := server.New(address, db, logger)
	if err := srv.ListenAndServe(); err != nil {
		logger.Fatalf("server stopped: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
