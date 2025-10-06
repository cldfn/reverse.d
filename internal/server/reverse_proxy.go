package server

import (
	"database/sql"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// NewReverseProxy returns an http.Handler that routes based on Host header.
// It looks up the target for the request's Host from sqlite via db.
func NewReverseProxy(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		host := r.Host

		// strip optional port
		if i := strings.Index(host, ":"); i != -1 {
			host = host[:i]
		}
		var target string
		err := db.QueryRow("SELECT target FROM routes WHERE domain = ?", host).Scan(&target)
		if err == sql.ErrNoRows {
			http.Error(w, "no route", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("db lookup error: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if !strings.HasPrefix(target, "http") {
			target = "http://" + target
		}
		u, err := url.Parse(target)
		if err != nil {
			http.Error(w, "invalid target", http.StatusBadGateway)
			return
		}

		// log.Printf(" --> proxy target  : %s", target)

		proxy := httputil.NewSingleHostReverseProxy(u)
		// adjust headers
		orig := proxy.Director
		proxy.Director = func(req *http.Request) {
			orig(req)
			req.Header.Set("X-Forwarded-Host", req.Host)
			req.Header.Set("X-Forwarded-For", r.RemoteAddr)
			// preserve original Host for backend
			req.Host = u.Host
		}
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy error: %v", err)
			http.Error(w, "bad gateway", http.StatusBadGateway)
		}
		proxy.ServeHTTP(w, r)
	})
}
