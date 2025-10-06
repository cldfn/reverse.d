package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cldfn/reverse.d/internal/control"
	"github.com/cldfn/reverse.d/internal/server"
	"github.com/cldfn/reverse.d/internal/store"
)

func main() {
	// Paths are relative to the binary's working directory
	dbPath := "./config.db"
	socketPath := "./proxyd.sock"
	certsDir := "./certs"

	// Ensure certs dir exists
	if err := os.MkdirAll(certsDir, 0700); err != nil {
		log.Fatalf("failed to create certs dir: %v", err)
	}

	// Open store (sqlite)
	db, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Start control server handler
	controlHandler := control.NewHandler(db)

	// Prepare unix socket listener
	l, err := control.PrepareSocket(socketPath)
	if err != nil {
		log.Fatalf("failed to prepare socket: %v", err)
	}
	defer os.Remove(socketPath)

	controlSrv := &http.Server{
		Handler:      controlHandler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	// Reverse proxy server
	proxyHandler := server.NewReverseProxy(db)
	proxySrv := &http.Server{
		Addr:         ":8080",
		Handler:      proxyHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Run control server on unix socket
	go func() {
		log.Printf("control socket listening on %s", socketPath)
		if err := controlSrv.Serve(l); err != nil && err != http.ErrServerClosed {
			log.Fatalf("control server error: %v", err)
		}
	}()

	// Run proxy server
	go func() {
		log.Printf("proxy listening on %s", proxySrv.Addr)
		if err := proxySrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("proxy server error: %v", err)
		}
	}()

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("shutting down...")
	controlSrv.Close()
	proxySrv.Close()
}
