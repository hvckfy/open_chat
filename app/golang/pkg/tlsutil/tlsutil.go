// Package tlsutil builds gRPC transport credentials for OpenChat's node
// and clients. It's a public package (not internal/grpcserver) on purpose:
// every consumer of the node's gRPC contract needs it — the validator
// node itself (ServerTLS), the CLI client and Fyne app (ClientTLS), and
// the mobile bridge (ClientTLSFromPEM, since a phone app would rather
// hold CA material in memory/Keychain than write it to a file path) — and
// several of those consumers live in entirely separate Go modules
// (app/fyne, and mobile bindings consumed from Swift/Kotlin) that have no
// business reaching into app/golang's internal/ tree.
package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
)

// ServerTLS loads a certificate/key pair (paths typically point at
// Kubernetes/Docker-mounted secret files, never at values baked into the
// image) and returns transport credentials enforcing TLS 1.3, matching
// the "gRPC-сервер с поддержкой TLS" requirement.
func ServerTLS(certFile, keyFile string) (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("tlsutil: load TLS keypair: %w", err)
	}
	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}
	return credentials.NewTLS(cfg), nil
}

// ClientTLS builds client-side transport credentials. If caFile is empty
// the host's system trust store is used (fine for publicly-issued gateway
// certs); pass caFile to pin a private/self-signed network CA instead.
func ClientTLS(caFile string, insecureSkipVerify bool) (credentials.TransportCredentials, error) {
	if caFile == "" {
		return credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS13, InsecureSkipVerify: insecureSkipVerify}), nil
	}
	pem, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("tlsutil: read CA file: %w", err)
	}
	return ClientTLSFromPEM(pem, insecureSkipVerify)
}

// ClientTLSFromPEM is ClientTLS's file-free counterpart: caPEM is the CA
// certificate's PEM-encoded bytes directly, for callers (mobile apps in
// particular) that hold network-settings material in memory or platform
// secure storage rather than an on-disk file path. An empty/nil caPEM
// behaves exactly like ClientTLS's empty caFile: the system trust store.
func ClientTLSFromPEM(caPEM []byte, insecureSkipVerify bool) (credentials.TransportCredentials, error) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS13, InsecureSkipVerify: insecureSkipVerify}
	if len(caPEM) > 0 {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("tlsutil: failed to parse CA PEM data")
		}
		cfg.RootCAs = pool
	}
	return credentials.NewTLS(cfg), nil
}
