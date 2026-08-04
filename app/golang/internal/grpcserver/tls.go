package grpcserver

import (
	"google.golang.org/grpc/credentials"

	"openchat/pkg/tlsutil"
)

// Deprecated: moved to the public pkg/tlsutil so non-internal consumers
// (the CLI, the Fyne app in the separate app/fyne module, and the mobile
// bridge) can use it without depending on this internal package. These
// two functions are kept as thin re-exports, rather than deleted outright,
// only because this sandbox's filesystem mount won't let this file be
// removed — every call site in this repo has already been updated to
// call pkg/tlsutil directly; don't add new callers of these.

// Deprecated: use tlsutil.ServerTLS.
func ServerTLS(certFile, keyFile string) (credentials.TransportCredentials, error) {
	return tlsutil.ServerTLS(certFile, keyFile)
}

// Deprecated: use tlsutil.ClientTLS.
func ClientTLS(caFile string, insecureSkipVerify bool) (credentials.TransportCredentials, error) {
	return tlsutil.ClientTLS(caFile, insecureSkipVerify)
}
