package store

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// NewSQLiteStore opens (or creates) a sqlite DB at path and ensures schema exists.
func NewSQLiteStore(path string) (*sql.DB, error) {
	// ensure directory exists
	if fi, err := os.Stat(path); err == nil && fi.Mode().IsRegular() {
		// file exists
	} else {
		// create empty file by opening
		f, err := os.OpenFile(path, os.O_CREATE, 0600)
		if err != nil {
			return nil, fmt.Errorf("create db file: %w", err)
		}
		f.Close()
	}
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=1")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

const schema = `
CREATE TABLE IF NOT EXISTS routes (
    domain TEXT PRIMARY KEY,
    target TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`
