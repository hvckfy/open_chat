// Package authproof defines the tiny canonical message both the client
// (when opening a StreamIncomingSMS subscription) and the node (when
// verifying it) sign/verify, proving the caller owns the private key for
// the address it claims without ever transmitting that key.
package authproof

import "encoding/binary"

// Bytes returns address || big-endian(timestampMillis), the exact bytes
// that get Ed25519-signed by the client and Ed25519-verified by the node.
func Bytes(address string, timestampMillis int64) []byte {
	buf := make([]byte, 0, len(address)+8)
	buf = append(buf, []byte(address)...)
	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], uint64(timestampMillis))
	return append(buf, ts[:]...)
}
