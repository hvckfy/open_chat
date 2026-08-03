package grpcserver

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
		return nil, fmt.Errorf("grpcserver: load TLS keypair: %w", err)
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
	cfg := &tls.Config{MinVersion: tls.VersionTLS13, InsecureSkipVerify: insecureSkipVerify}
	if caFile != "" {
		pem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("grpcserver: read CA file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("grpcserver: failed to parse CA file %s", caFile)
		}
		cfg.RootCAs = pool
	}
	return credentials.NewTLS(cfg), nil
}
