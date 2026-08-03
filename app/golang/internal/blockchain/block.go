package blockchain

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

// ValidatorSig is one validator's commit-phase (PRECOMMIT) signature,
// collected during PBFT consensus (see internal/consensus). Round is
// carried alongside the signature because the PRECOMMIT vote signs
// type+height+round+blockHash (see consensus.Vote.SigningBytes), not the
// bare block hash — verification below must reconstruct the exact same
// bytes or every signature will be (incorrectly) rejected.
type ValidatorSig struct {
	Validator string // hex ed25519 pubkey of the validator
	Round     uint32
	Signature []byte
}

// votePrecommitType mirrors consensus.VotePrecommit's byte value (iota+1,
// so VotePrevote=1, VotePrecommit=2). Duplicated here instead of imported
// to avoid a blockchain<->consensus import cycle.
const votePrecommitType = 2

// precommitSigningBytes reconstructs exactly what a validator signed when
// casting its PRECOMMIT vote for (height, round, hash) — must stay in sync
// with consensus.Vote.SigningBytes().
func precommitSigningBytes(height uint64, round uint32, hash string) []byte {
	buf := make([]byte, 0, 64)
	buf = append(buf, byte(votePrecommitType))
	var h [8]byte
	binary.BigEndian.PutUint64(h[:], height)
	buf = append(buf, h[:]...)
	var r [4]byte
	binary.BigEndian.PutUint32(r[:], round)
	buf = append(buf, r[:]...)
	buf = append(buf, []byte(hash)...)
	return buf
}

// Block bundles a batch of transactions committed by consensus.
type Block struct {
	Height       uint64
	PrevHash     string
	Timestamp    int64
	Proposer     string // hex ed25519 pubkey of the proposing validator
	Transactions []*Transaction
	MerkleRoot   string

	// CommitSigs accumulates >= 2f+1 validator signatures over Hash()
	// proving BFT quorum agreement on this exact block.
	CommitSigs []ValidatorSig
}

// HeaderBytes is the canonical encoding of everything except CommitSigs —
// this is what proposer signs and what PREPARE/COMMIT votes reference.
func (b *Block) HeaderBytes() []byte {
	buf := make([]byte, 0, 256)

	var h [8]byte
	binary.BigEndian.PutUint64(h[:], b.Height)
	buf = append(buf, h[:]...)

	buf = append(buf, []byte(b.PrevHash)...)

	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], uint64(b.Timestamp))
	buf = append(buf, ts[:]...)

	buf = append(buf, []byte(b.Proposer)...)
	buf = append(buf, []byte(b.MerkleRoot)...)
	return buf
}

// Hash returns the sha256 block hash (hex) — this becomes the next block's
// PrevHash, chaining blocks together.
func (b *Block) Hash() string {
	h := sha256.New()
	h.Write(b.HeaderBytes())
	return hex.EncodeToString(h.Sum(nil))
}

// ComputeMerkleRoot derives and sets MerkleRoot from Transactions. A simple
// pairwise sha256 Merkle tree (duplicate last leaf on odd counts).
func (b *Block) ComputeMerkleRoot() {
	if len(b.Transactions) == 0 {
		b.MerkleRoot = hex.EncodeToString(sha256.New().Sum(nil))
		return
	}
	layer := make([][]byte, len(b.Transactions))
	for i, tx := range b.Transactions {
		sum := sha256.Sum256([]byte(tx.Hash()))
		layer[i] = sum[:]
	}
	for len(layer) > 1 {
		if len(layer)%2 == 1 {
			layer = append(layer, layer[len(layer)-1])
		}
		next := make([][]byte, 0, len(layer)/2)
		for i := 0; i < len(layer); i += 2 {
			h := sha256.New()
			h.Write(layer[i])
			h.Write(layer[i+1])
			next = append(next, h.Sum(nil))
		}
		layer = next
	}
	b.MerkleRoot = hex.EncodeToString(layer[0])
}

// VerifyCommitQuorum checks that at least quorum distinct, valid signatures
// over Hash() are present, from addresses that are actually members of
// validators (2f+1 out of 3f+1 for BFT safety).
//
// The validators check matters as much as the signature check: without
// it, anyone able to reach this node's libp2p gossip mesh (the rendezvous
// tag is a public constant, not a secret) could generate a throwaway
// keypair, sign a PRECOMMIT-shaped vote for a legitimately-proposed
// block, and have it counted toward quorum — never having been added to
// VALIDATOR_SET anywhere. Requiring membership means quorum can only ever
// be reached by addresses the caller's own local config already trusts.
func (b *Block) VerifyCommitQuorum(quorum int, validators map[string]bool) bool {
	if len(b.CommitSigs) < quorum {
		return false
	}
	hash := b.Hash()
	seen := make(map[string]bool, len(b.CommitSigs))
	valid := 0
	for _, cs := range b.CommitSigs {
		if seen[cs.Validator] {
			continue // no double counting the same validator
		}
		if !validators[cs.Validator] {
			continue // signature may be internally valid, but this address isn't a known validator
		}
		pub, err := decodeAddress(cs.Validator)
		if err != nil {
			continue
		}
		if ed25519.Verify(pub, precommitSigningBytes(b.Height, cs.Round, hash), cs.Signature) {
			seen[cs.Validator] = true
			valid++
		}
	}
	return valid >= quorum
}

// GenesisBlock returns the deterministic block at height 0.
func GenesisBlock() *Block {
	b := &Block{
		Height:    0,
		PrevHash:  "",
		Timestamp: 0,
		Proposer:  "genesis",
	}
	b.ComputeMerkleRoot()
	return b
}
