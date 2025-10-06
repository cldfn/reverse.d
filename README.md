# proxyd - simple reverse proxy daemon (prototype)

This project is a minimal prototype of a reverse proxy daemon that:

- Listens on `:8080` for incoming HTTP requests and routes them to backends based on the `Host` header.
- Exposes a control API over a UNIX socket (`./proxyd.sock`) for adding/removing/listing routes.
- Stores configuration in a local SQLite database (`./config.db`).
- Certificates are **not** stored in the DB; keep TLS certs in `./certs/`.


Build:
```bash
# requires github.com/mattn/go-sqlite3 CGo driver
go build ./cmd/proxyd
go build ./cmd/proxyctl
```

Run (in foreground):
```bash
./proxyd
# separate terminal:
./proxyctl list
./proxyctl add example.com localhost:3000
./proxyctl list
```

Notes:
- Paths (db, socket, certs) are relative to the binary's working directory.