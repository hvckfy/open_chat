package client

import (
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"

	"openchat/pkg/crypto"
)

// Wallet is the user's local identity plus the address book entries
// (recipient X25519 public keys) needed to encrypt outgoing messages. In
// a real deployment X25519 pubkeys would be looked up from a directory
// service or exchanged out-of-band/QR code; that lookup is intentionally
// outside this package's scope.
type Wallet struct {
	Keys *crypto.KeyPair

	// nonceCounter implements the client's half of replay protection: a
	// strictly-increasing per-sender counter the node checks against its
	// durable high-water mark. Seeding it from UnixNano() at startup
	// (instead of persisting the last used value to disk) means a fresh
	// process's first nonce is, for all practical purposes, always
	// greater than the highest nonce any previous run of this wallet
	// could have submitted.
	nonceCounter uint64
}

// NewWallet derives a wallet from an existing mnemonic (login) or, if
// mnemonic is empty, generates a brand new one (signup) and returns it so
// the caller can display it to the user exactly once for backup.
func NewWallet(mnemonic, passphrase string) (*Wallet, error) {
	if mnemonic == "" {
		kp, err := crypto.GenerateKeyPair(passphrase)
		if err != nil {
			return nil, err
		}
		return &Wallet{Keys: kp, nonceCounter: uint64(time.Now().UnixNano())}, nil
	}
	kp, err := crypto.DeriveKeyPair(mnemonic, passphrase)
	if err != nil {
		return nil, err
	}
	return &Wallet{Keys: kp, nonceCounter: uint64(time.Now().UnixNano())}, nil
}

func (w *Wallet) Address() string { return w.Keys.Address() }

// NextNonce returns the next strictly-increasing transaction nonce for
// this wallet.
func (w *Wallet) NextNonce() uint64 {
	return atomic.AddUint64(&w.nonceCounter, 1)
}

// DecodeRecipientX25519 parses a hex-encoded X25519 public key (as shared
// out-of-band by the recipient).
func DecodeRecipientX25519(hexPub string) ([32]byte, error) {
	var out [32]byte
	b, err := hex.DecodeString(hexPub)
	if err != nil || len(b) != 32 {
		return out, fmt.Errorf("client: invalid X25519 pubkey %q", hexPub)
	}
	copy(out[:], b)
	return out, nil
}
