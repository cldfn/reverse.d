package control

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// NewHandler returns an http.Handler for control API using the provided DB handle.
func NewHandler(db *sql.DB) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/list", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT domain, target FROM routes ORDER BY domain")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		var out []map[string]string
		for rows.Next() {
			var d, t string
			rows.Scan(&d, &t)
			out = append(out, map[string]string{"domain": d, "target": t})
		}
		json.NewEncoder(w).Encode(out)
	})

	mux.HandleFunc("/api/add", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Domain string `json:"domain"`
			Target string `json:"target"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if req.Domain == "" || req.Target == "" {
			http.Error(w, "domain and target required", 400)
			return
		}
		_, err := db.Exec("INSERT OR REPLACE INTO routes(domain,target) VALUES(?,?)", req.Domain, req.Target)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("/api/remove", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Domain string `json:"domain"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if req.Domain == "" {
			http.Error(w, "domain required", 400)
			return
		}
		_, err := db.Exec("DELETE FROM routes WHERE domain = ?", req.Domain)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("ok"))
	})

	return mux
}
