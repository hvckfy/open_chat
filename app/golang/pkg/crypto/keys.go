// Package crypto implements the client-side key derivation and end-to-end
// encryption scheme described in the architecture doc:
//
//  1. 256 bits of system entropy -> BIP-39 mnemonic (24 words).
//  2. Mnemonic (+ optional passphrase) -> 64-byte BIP-39 seed (PBKDF2).
//  3. Seed -> deterministic Ed25519 signing keypair (transaction signing,
//     also the user's network "address"/phone number in hex) and a
//     deterministic X25519 keypair (ECDH for E2EE), derived independently
//     via HKDF-SHA512 so compromising one does not trivially reveal the
//     other.
//
// None of this ever touches the node/backend: everything in this file runs
// entirely client-side.
package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

const (
	// EntropyBits is 256 bits as required (-> 24 word BIP-39 mnemonic).
	EntropyBits = 256

	hkdfInfoEd25519 = "openchat/ed25519-signing-key/v1"
	hkdfInfoX25519  = "openchat/x25519-encryption-key/v1"
)

// KeyPair holds a client's full derived identity. PrivateKey material stays
// in process memory only (never serialized to disk by this package).
type KeyPair struct {
	Mnemonic string

	SigningPublic  ed25519.PublicKey
	SigningPrivate ed25519.PrivateKey

	EncryptionPublic  [32]byte // X25519 public key
	EncryptionPrivate [32]byte // X25519 private (scalar) key
}

// Address is the user's network address: hex-encoded Ed25519 public key.
func (k *KeyPair) Address() string {
	return hex.EncodeToString(k.SigningPublic)
}

// EncryptionPublicHex hex-encodes the X25519 public key for out-of-band
// sharing (e.g. published alongside the address in a contacts directory).
func (k *KeyPair) EncryptionPublicHex() string {
	return hex.EncodeToString(k.EncryptionPublic[:])
}

// GenerateMnemonic produces a fresh 24-word BIP-39 mnemonic from 256 bits
// of crypto/rand entropy.
func GenerateMnemonic() (string, error) {
	entropy := make([]byte, EntropyBits/8)
	if _, err := rand.Read(entropy); err != nil {
		return "", fmt.Errorf("crypto: read entropy: %w", err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("crypto: bip39 mnemonic: %w", err)
	}
	return mnemonic, nil
}

// DeriveKeyPair walks the full chain: mnemonic (+ passphrase) -> seed ->
// Ed25519 + X25519 keypairs. Deterministic: the same mnemonic+passphrase
// always yields the same address, on any device (this is how a user
// "logs in" by re-entering their seed phrase).
func DeriveKeyPair(mnemonic, passphrase string) (*KeyPair, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("crypto: invalid bip39 mnemonic")
	}
	seed := bip39.NewSeed(mnemonic, passphrase) // 64 bytes

	edPriv, edPub, err := deriveEd25519(seed)
	if err != nil {
		return nil, err
	}
	xPriv, xPub, err := deriveX25519(seed)
	if err != nil {
		return nil, err
	}

	return &KeyPair{
		Mnemonic:          mnemonic,
		SigningPublic:     edPub,
		SigningPrivate:    edPriv,
		EncryptionPublic:  xPub,
		EncryptionPrivate: xPriv,
	}, nil
}

// GenerateKeyPair is a convenience wrapper: generate a new mnemonic and
// derive keys from it in one step. Returns the mnemonic so the caller can
// display it once to the user for backup.
func GenerateKeyPair(passphrase string) (*KeyPair, error) {
	mnemonic, err := GenerateMnemonic()
	if err != nil {
		return nil, err
	}
	return DeriveKeyPair(mnemonic, passphrase)
}

func deriveEd25519(seed []byte) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	sub, err := hkdfExpand(seed, hkdfInfoEd25519, ed25519.SeedSize)
	if err != nil {
		return nil, nil, err
	}
	priv := ed25519.NewKeyFromSeed(sub)
	return priv, priv.Public().(ed25519.PublicKey), nil
}

func deriveX25519(seed []byte) (priv [32]byte, pub [32]byte, err error) {
	sub, err := hkdfExpand(seed, hkdfInfoX25519, 32)
	if err != nil {
		return priv, pub, err
	}
	copy(priv[:], sub)
	// clamp per RFC 7748
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	pubBytes, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return priv, pub, fmt.Errorf("crypto: derive x25519 pubkey: %w", err)
	}
	copy(pub[:], pubBytes)
	return priv, pub, nil
}

func hkdfExpand(seed []byte, info string, size int) ([]byte, error) {
	r := hkdf.New(sha512.New, seed, nil, []byte(info))
	out := make([]byte, size)
	if _, err := io.ReadFull(r, out); err != nil {
		return nil, fmt.Errorf("crypto: hkdf expand (%s): %w", info, err)
	}
	return out, nil
}
