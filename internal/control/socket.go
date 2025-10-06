package control

import (
	"fmt"
	"net"
	"os"
	"log"
)

// PrepareSocket ensures the unix socket is available for listening.
// If the socket file exists and a server is active, it returns an error.
// If the socket file exists but is stale (no active server), it removes it.
func PrepareSocket(path string) (net.Listener, error) {
	if _, err := os.Stat(path); err == nil {
		// try to connect to see if someone listens
		conn, err := net.Dial("unix", path)
		if err == nil {
			conn.Close()
			return nil, fmt.Errorf("another proxyd instance is running (%s)", path)
		}
		log.Printf("removing stale socket: %s", path)
		if err := os.Remove(path); err != nil {
			return nil, fmt.Errorf("failed to remove stale socket: %w", err)
		}
	}

	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", path, err)
	}
	// restrict permissions to owner
	os.Chmod(path, 0600)
	return l, nil
}
