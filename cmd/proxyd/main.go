package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cldfn/reverse.d/internal/control"
	"github.com/cldfn/reverse.d/internal/server"
	"github.com/cldfn/reverse.d/internal/store"
	"github.com/cldfn/reverse.d/internal/tls"
)

func main() {
	// Paths are relative to the binary's working directory

	storagePath := "./storage"

	dbPath := storagePath + "/config.db"
	socketPath := storagePath + "/proxyd.sock"
	certsDir := storagePath + "/certs"

	portArgValue := flag.Int("port", 8080, "listening port")
	tlsArgValue := flag.Bool("tls", false, "enable TLS")
	tlsPortArgValue := flag.Int("tls-port", 443, "TLS listening port")

	readTimeoutArg := flag.Int("read-timeout", 30, "server read timeout in seconds")
	writeTimeoutArg := flag.Int("write-timeout", 30, "server write timeout in seconds")

	flag.Parse()

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

	var proxySrv *http.Server = &http.Server{
		Addr:         fmt.Sprintf(":%d", *portArgValue),
		Handler:      proxyHandler,
		ReadTimeout:  time.Duration(*readTimeoutArg) * time.Second,
		WriteTimeout: time.Duration(*writeTimeoutArg) * time.Second,
	}

	var proxySrvSecure *http.Server

	if *tlsArgValue {

		tlsCfg, err := tls.TLSConfig(certsDir)
		if err != nil {
			log.Fatal(err)
		}

		proxySrvSecure = &http.Server{
			Addr:         fmt.Sprintf(":%d", *tlsPortArgValue),
			Handler:      proxyHandler,
			TLSConfig:    tlsCfg,
			ReadTimeout:  time.Duration(*readTimeoutArg) * time.Second,
			WriteTimeout: time.Duration(*writeTimeoutArg) * time.Second,
		}
	}

	// Run control server on unix socket
	go func() {
		log.Printf("control socket listening on %s", socketPath)
		{
			if err := controlSrv.Serve(l); err != nil && err != http.ErrServerClosed {
				log.Fatalf("control server error: %v", err)
			}
		}
	}()

	// Run proxy server
	go func() {

		log.Printf("proxy listening on %s", proxySrv.Addr)
		if err := proxySrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("proxy server error: %v", err)
		}
	}()

	if *tlsArgValue {
		go func() {
			log.Printf("secure proxy listening on %s", proxySrvSecure.Addr)
			if err := proxySrvSecure.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Fatalf("proxy server error TLS: %s", err.Error())
			}

		}()
	}

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("shutting down...")
	controlSrv.Close()
	proxySrv.Close()
}
