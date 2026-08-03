package consensus

import (
	"context"
	"crypto/ed25519"
)

// PubSub is the networking port this package depends on. internal/p2p
// implements it on top of go-libp2p-pubsub; consensus itself knows
// nothing about libp2p (Clean Architecture / dependency inversion).
type PubSub interface {
	Publish(ctx context.Context, topic string, data []byte) error
	Subscribe(topic string) (Subscription, error)
}

// Subscription yields raw gossip messages for one topic.
type Subscription interface {
	Next(ctx context.Context) ([]byte, error)
	Cancel()
}

// Identity is this node's validator signing identity. The private key is
// supplied by cmd/node from an env var / Docker secret and lives only in
// process memory (zero-trust: never written to disk by this package).
type Identity struct {
	Address string
	Priv    ed25519.PrivateKey
}

const (
	TopicPropose   = "openchat/consensus/propose/v1"
	TopicPrevote   = "openchat/consensus/prevote/v1"
	TopicPrecommit = "openchat/consensus/precommit/v1"
	TopicTxGossip  = "openchat/mempool/tx/v1"

	// TopicBlockCommit carries every block a validator has just committed
	// (header + transactions + the >=2f+1 CommitSigs that made it final),
	// so that non-voting relay nodes — which never run the propose/vote
	// loop above — can still keep a live, verifiable copy of the chain in
	// near-real-time by independently re-checking VerifyCommitQuorum and
	// calling Chain.CommitBlock themselves. Validators also subscribe to
	// this out of convenience (redundant with their own local commit, but
	// harmless: Chain.CommitBlock rejects the resulting out-of-order/
	// already-applied block cheaply).
	TopicBlockCommit = "openchat/consensus/commit/v1"

	// TopicNodeAnnounce carries GatewayInfo records (validator or relay,
	// including live health/version status) so that every node's local
	// discovery Registry converges on the same picture of "who's on the
	// network right now" — this is what ultimately powers the client-
	// facing GetNodesDiscovery RPC.
	TopicNodeAnnounce = "openchat/discovery/announce/v1"
)

// VoteType distinguishes PREVOTE and PRECOMMIT messages.
type VoteType uint8

const (
	VotePrevote VoteType = iota + 1
	VotePrecommit
)

// Vote is a validator's signed opinion about a candidate block at a given
// height/round.
type Vote struct {
	Type      VoteType
	Height    uint64
	Round     uint32
	BlockHash string
	Voter     string // hex ed25519 pubkey
	Signature []byte
}

// SigningBytes is what the validator actually signs.
func (v *Vote) SigningBytes() []byte {
	buf := make([]byte, 0, 64)
	buf = append(buf, byte(v.Type))
	buf = appendU64(buf, v.Height)
	buf = appendU32(buf, v.Round)
	buf = append(buf, []byte(v.BlockHash)...)
	return buf
}

func (v *Vote) Sign(priv ed25519.PrivateKey) {
	v.Signature = ed25519.Sign(priv, v.SigningBytes())
}

func (v *Vote) Verify() bool {
	pub, err := decodeAddr(v.Voter)
	if err != nil {
		return false
	}
	return ed25519.Verify(pub, v.SigningBytes(), v.Signature)
}

func appendU64(b []byte, v uint64) []byte {
	var t [8]byte
	for i := 7; i >= 0; i-- {
		t[i] = byte(v)
		v >>= 8
	}
	return append(b, t[:]...)
}

func appendU32(b []byte, v uint32) []byte {
	var t [4]byte
	for i := 3; i >= 0; i-- {
		t[i] = byte(v)
		v >>= 8
	}
	return append(b, t[:]...)
}

func decodeAddr(addr string) (ed25519.PublicKey, error) {
	return hexDecodePubkey(addr)
}
