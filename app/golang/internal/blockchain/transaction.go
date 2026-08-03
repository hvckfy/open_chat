// Package blockchain contains the domain model: transactions (encrypted
// SMS messages) and blocks that bundle them. It has zero dependency on
// gRPC, libp2p or any storage engine — those are wired in from outside
// (Clean Architecture: this is the innermost "entities" layer).
package blockchain

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
)

// Transaction is one encrypted message travelling through the network.
// The payload (Ciphertext) is opaque AES-256-GCM output; the chain and
// every validator only ever see ciphertext, never plaintext.
type Transaction struct {
	From            string // hex ed25519 pubkey of sender
	To              string // hex ed25519 pubkey of recipient
	Ciphertext      []byte
	NonceAEAD       [12]byte // AES-GCM nonce
	EphemeralPubkey [32]byte // sender's X25519 public key for this message
	Nonce           uint64   // strictly increasing per-`From` counter (replay protection)
	Timestamp       int64    // unix millis
	Signature       []byte   // ed25519 signature over SigningBytes()
}

var (
	ErrInvalidAddress   = errors.New("blockchain: invalid hex address")
	ErrInvalidSignature = errors.New("blockchain: signature verification failed")
)

// SigningBytes returns the canonical, deterministic byte encoding of the
// transaction (everything except the signature itself). Both client (when
// signing) and node (when verifying) must build this identically.
func (t *Transaction) SigningBytes() []byte {
	buf := make([]byte, 0, 32+32+len(t.Ciphertext)+12+32+8+8)
	buf = append(buf, []byte(t.From)...)
	buf = append(buf, []byte(t.To)...)
	buf = append(buf, t.Ciphertext...)
	buf = append(buf, t.NonceAEAD[:]...)
	buf = append(buf, t.EphemeralPubkey[:]...)

	var n [8]byte
	binary.BigEndian.PutUint64(n[:], t.Nonce)
	buf = append(buf, n[:]...)

	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], uint64(t.Timestamp))
	buf = append(buf, ts[:]...)

	return buf
}

// Sign fills in t.Signature using the given Ed25519 private key. The caller
// (client) is responsible for ensuring `priv` corresponds to t.From.
func (t *Transaction) Sign(priv ed25519.PrivateKey) {
	t.Signature = ed25519.Sign(priv, t.SigningBytes())
}

// Verify checks the Ed25519 signature against t.From.
func (t *Transaction) Verify() error {
	pub, err := decodeAddress(t.From)
	if err != nil {
		return err
	}
	if !ed25519.Verify(pub, t.SigningBytes(), t.Signature) {
		return ErrInvalidSignature
	}
	return nil
}

// Hash returns the sha256 transaction hash (hex), including the signature,
// used as the transaction's unique ID (tx_hash).
func (t *Transaction) Hash() string {
	h := sha256.New()
	h.Write(t.SigningBytes())
	h.Write(t.Signature)
	return hex.EncodeToString(h.Sum(nil))
}

func decodeAddress(addr string) (ed25519.PublicKey, error) {
	b, err := hex.DecodeString(addr)
	if err != nil || len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: %s", ErrInvalidAddress, addr)
	}
	return ed25519.PublicKey(b), nil
}
