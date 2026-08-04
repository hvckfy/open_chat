// Package relayproof defines the canonical byte encoding signed by a
// community-run relay node when it registers itself with a validator
// ("trusted center"), and verified by the validator on receipt. This is
// the cryptographic half of "verify it's running what we want": it proves
// the caller controls the private key for RelayAddress without ever
// transmitting that key, and binds the signature to the exact
// version/commit/endpoint being claimed so none of those fields can be
// tampered with in transit.
//
// Mirrors pkg/authproof's shape/spirit (address||timestamp) but includes
// the extra self-reported fields relay registration needs.
package relayproof

import "encoding/binary"

// Bytes returns the exact bytes a relay signs (and a validator verifies)
// for RegisterRelay:
//
//	relayAddress || relayGRPCAddress || nodeVersion || nodeCommit || big-endian(timestampMillis)
//
// All string fields are length-implicit (concatenated directly) because
// every field here is itself a fixed-charset, non-attacker-controlled-
// delimiter value (hex address, host:port, semver-ish version, git hex
// commit) — there is no ambiguity a length-prefix would be needed to
// resolve, consistent with how blockchain.Transaction.SigningBytes and
// authproof.Bytes are built elsewhere in this codebase.
func Bytes(relayAddress, relayGRPCAddress, nodeVersion, nodeCommit string, timestampMillis int64) []byte {
	buf := make([]byte, 0, len(relayAddress)+len(relayGRPCAddress)+len(nodeVersion)+len(nodeCommit)+8)
	buf = append(buf, []byte(relayAddress)...)
	buf = append(buf, []byte(relayGRPCAddress)...)
	buf = append(buf, []byte(nodeVersion)...)
	buf = append(buf, []byte(nodeCommit)...)
	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], uint64(timestampMillis))
	return append(buf, ts[:]...)
}
