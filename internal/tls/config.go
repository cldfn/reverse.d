package tls

import (
	"crypto/tls"
	"fmt"
	"path/filepath"
)

func TLSConfig(certsDir string) (*tls.Config, error) {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
		NextProtos: []string{"h2", "http/1.1"},
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {

			domain := hello.ServerName
			if domain == "" {
				return nil, fmt.Errorf("missing SNI")
			}

			certPath := filepath.Join(certsDir, domain+".crt")
			keyPath := filepath.Join(certsDir, domain+".key")
			cert, err := tls.LoadX509KeyPair(certPath, keyPath)
			if err != nil {
				return nil, fmt.Errorf("cannot load cert for %s: %v", domain, err)
			}

			return &cert, nil
		},
	}, nil
}
