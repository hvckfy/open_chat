package consensus

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
)

func hexDecodePubkey(addr string) (ed25519.PublicKey, error) {
	b, err := hex.DecodeString(addr)
	if err != nil || len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("consensus: invalid address %q", addr)
	}
	return ed25519.PublicKey(b), nil
}

// quorum returns the BFT quorum size (2f+1) for a validator set of size n,
// where f = floor((n-1)/3) is the max tolerated byzantine/faulty count.
func quorum(n int) int {
	f := (n - 1) / 3
	return 2*f + 1
}

// Quorum is the exported form of quorum, so that code outside this package
// (e.g. a relay node in cmd/node, which never constructs an *Engine but
// still needs to independently verify Block.CommitSigs against the public
// validator set) can compute the exact same 2f+1 threshold without
// duplicating the formula.
func Quorum(n int) int { return quorum(n) }

func signWith(priv ed25519.PrivateKey, msg []byte) []byte {
	return ed25519.Sign(priv, msg)
}

func verifyWith(pub ed25519.PublicKey, msg, sig []byte) bool {
	return ed25519.Verify(pub, msg, sig)
}
