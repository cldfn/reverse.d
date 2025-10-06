package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
)

func httpClientForSocket(sock string) *http.Client {
	tr := &http.Transport{
		DialContext: func(_ctx context.Context, _network, _addr string) (net.Conn, error) {
			return net.Dial("unix", sock)
		},
	}
	return &http.Client{Transport: tr}
}

func main() {
	sock := flag.String("sock", "./proxyd.sock", "path to proxyd unix socket (relative to binary)")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Println("usage: proxyctl [--sock path] <cmd> [args]")
		fmt.Println("commands: list, add <domain> <target>, remove <domain>")
		return
	}
	cmd := flag.Arg(0)
	client := httpClientForSocket(*sock)

	switch cmd {
	case "list":
		req, _ := http.NewRequest("GET", "http://unix/api/list", nil)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		io.Copy(os.Stdout, resp.Body)
	case "add":
		if flag.NArg() != 3 {
			fmt.Println("usage: proxyctl add <domain> <target>")
			return
		}
		domain := flag.Arg(1)
		target := flag.Arg(2)
		body, _ := json.Marshal(map[string]string{"domain": domain, "target": target})
		req, _ := http.NewRequest("POST", "http://unix/api/add", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		io.Copy(os.Stdout, resp.Body)
	case "remove":
		if flag.NArg() != 2 {
			fmt.Println("usage: proxyctl remove <domain>")
			return
		}
		domain := flag.Arg(1)
		body, _ := json.Marshal(map[string]string{"domain": domain})
		req, _ := http.NewRequest("POST", "http://unix/api/remove", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		io.Copy(os.Stdout, resp.Body)
	default:
		fmt.Println("unknown command:", cmd)
	}
}
