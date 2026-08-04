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

// quorum returns the BFT quorum size for a validator set of size n, where
// f = floor((n-1)/3) is the max tolerated byzantine/faulty count.
//
// This is n-f, not the more commonly quoted "2f+1" — the two coincide only
// when n is exactly 3f+1 (e.g. 4, 7, 10...). For any other n (notably a
// small/dev deployment like this repo's 2-validator docker-compose demo,
// where f=0), 2f+1 undercounts: with n=2 it evaluates to 1, meaning a
// single validator's own vote already satisfies "quorum" with zero input
// from anyone else. Since consensus/pbft.go's gossip layer redelivers a
// validator's own Publish back to its own Subscribe (see p2p/pubsub.go),
// a quorum of 1 lets each of the two validators commit blocks entirely on
// its own, silently, with no real agreement ever happening between them —
// exactly the divergent per-node chain state (different block heights,
// different commit history) this was found from. n-f instead requires a
// real majority-with-margin (n=2,f=0 -> quorum=2, i.e. both must agree),
// and still equals 2f+1 whenever n really is 3f+1.
func quorum(n int) int {
	f := (n - 1) / 3
	return n - f
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
