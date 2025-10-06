// go run proxyd.go --domain example.com --target http://localhost:3000

package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"golang.org/x/crypto/acme/autocert"
)

func main() {
	domain := flag.String("domain", "", "Domain to obtain Let's Encrypt certificate for")
	targetStr := flag.String("target", "", "Target backend (e.g. http://localhost:3000)")
	cacheDir := flag.String("cache", "certs", "Directory to store certs")
	flag.Parse()

	if *domain == "" || *targetStr == "" {
		log.Fatal("Usage: proxyd --domain example.com --target http://localhost:3000")
	}

	target, err := url.Parse(*targetStr)
	if err != nil {
		log.Fatalf("Invalid target: %v", err)
	}

	proxy := newReverseProxy(target)

	mgr := &autocert.Manager{
		Cache:      autocert.DirCache(*cacheDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(*domain),
	}

	server := &http.Server{
		Addr: ":443",
		TLSConfig: &tls.Config{
			GetCertificate: mgr.GetCertificate,
		},
		Handler: proxy,
	}

	// HTTP-01 challenge server on port 80
	go func() {
		log.Println("Starting HTTP-01 challenge server on :80")
		http.ListenAndServe(":80", mgr.HTTPHandler(nil))
	}()

	log.Printf("Reverse proxy listening on https://%s → %s", *domain, *targetStr)
	log.Fatal(server.ListenAndServeTLS("", "")) // Let autocert manage certs
}

// newReverseProxy creates a reverse proxy with WebSocket support.
func newReverseProxy(target *url.URL) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Fix for WebSocket upgrades
	proxy.ModifyResponse = func(resp *http.Response) error {
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	// Fix X-Forwarded headers
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Forwarded-For", strings.Split(req.RemoteAddr, ":")[0])
	}

	return proxy
}
