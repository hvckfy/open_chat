package mobile

import (
	"fmt"

	"openchat/pkg/client"
)

// Wallet wraps a derived OpenChat identity for use from Swift/Kotlin.
// Holds private key material in process memory only, exactly like
// pkg/client.Wallet — see the package doc comment for why persisting it
// (Keychain, on a real device) is the native shell's job, not this one's.
type Wallet struct {
	inner *client.Wallet
}

// GenerateMnemonic produces a fresh 24-word BIP-39 recovery phrase. Show
// it to the user for backup (see design-code.md A2) before deriving a
// wallet from it with ImportWallet — CreateWallet does both steps at
// once if you don't need the intermediate mnemonic for anything else.
func GenerateMnemonic() (string, error) {
	w, err := client.NewWallet("", "")
	if err != nil {
		return "", err
	}
	return w.Keys.Mnemonic, nil
}

// CreateWallet generates a brand new recovery phrase and derives a
// wallet from it in one step (onboarding's "Create a new identity",
// design-code.md A1→A2). Call Mnemonic() on the result to get the phrase
// to display for backup.
func CreateWallet() (*Wallet, error) {
	w, err := client.NewWallet("", "")
	if err != nil {
		return nil, err
	}
	return &Wallet{inner: w}, nil
}

// ImportWallet derives a wallet from an existing recovery phrase
// (onboarding's "I already have a recovery phrase", design-code.md A3).
// Returns an error for a malformed phrase (wrong word count, bad
// checksum word) — surface it to the user rather than a generic failure.
func ImportWallet(mnemonic string) (*Wallet, error) {
	if mnemonic == "" {
		return nil, fmt.Errorf("mobile: mnemonic is empty")
	}
	w, err := client.NewWallet(mnemonic, "")
	if err != nil {
		return nil, err
	}
	return &Wallet{inner: w}, nil
}

// Address is this wallet's network address (hex Ed25519 public key) —
// the "phone number" half of its identity.
func (w *Wallet) Address() string { return w.inner.Address() }

// EncryptionPublicHex is this wallet's hex-encoded X25519 public key —
// the other half of its identity, needed by anyone who wants to encrypt
// a message to it.
func (w *Wallet) EncryptionPublicHex() string { return w.inner.Keys.EncryptionPublicHex() }

// Mnemonic returns the 24-word recovery phrase this wallet was derived
// from. Only meaningful to show the user once, right after CreateWallet,
// for backup (design-code.md A2) — never persist what this returns; that
// is precisely the secret the native shell's Keychain entry protects.
func (w *Wallet) Mnemonic() string { return w.inner.Keys.Mnemonic }

// ContactCode is the single shareable string (address + encryption key)
// this wallet's owner gives out so others can add them as a contact
// (design-code.md B5 "My contact code").
func (w *Wallet) ContactCode() string { return client.MyContactCode(w.inner) }

// ParsedContact is a contact code's decoded contents — the two hex
// strings a native UI needs to save a new contact (design-code.md B3
// "Add contact"). A distinct, string-only type from pkg/client.
// ParsedContact (which uses a [32]byte, not bindable by gomobile).
type ParsedContact struct {
	Address      string
	X25519PubHex string
}

// ParseContactCode decodes a pasted contact code (or the raw
// "<address>:<x25519pubkey>" fallback form) into its two components.
// Returns an error for malformed input — surface it inline under the
// field, per design-spec.md B3.
func ParseContactCode(code string) (*ParsedContact, error) {
	pc, err := client.ParseContactCode(code)
	if err != nil {
		return nil, err
	}
	return &ParsedContact{Address: pc.Address, X25519PubHex: pc.X25519PubHex}, nil
}
