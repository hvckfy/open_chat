package client

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

// contactCodePrefix makes pasted codes recognizable/greppable and lets us
// reject obviously-wrong input (e.g. someone pasting their mnemonic here
// by mistake) with a clear error instead of a cryptic decode failure.
const contactCodePrefix = "oc1:"

// MyContactCode returns a single shareable string encoding both keys a
// contact needs to message this wallet: the Ed25519 address (for the
// transaction's `To` field) and the X25519 public key (for ECDH). Sharing
// one code instead of two hex strings is friendlier for a messenger UI —
// think of it as OpenChat's equivalent of a phone number + public key
// bundle, shared once when adding a contact (in person, via QR, etc).
//
// Lives here (not in a UI-specific package) because every frontend needs
// it identically — the CLI, the Fyne app (../fyne), and the mobile
// bridge (../mobile) all call this same function rather than each
// reimplementing the encoding.
func MyContactCode(w *Wallet) string {
	buf := make([]byte, 0, 64)
	buf = append(buf, w.Keys.SigningPublic...)
	buf = append(buf, w.Keys.EncryptionPublic[:]...)
	return contactCodePrefix + base64.RawURLEncoding.EncodeToString(buf)
}

// ParsedContact is what ParseContactCode extracts from a pasted code.
type ParsedContact struct {
	Address      string   // hex ed25519 pubkey
	X25519Pub    [32]byte // raw X25519 pubkey
	X25519PubHex string
}

// ParseContactCode reverses MyContactCode. It also accepts the raw
// "<64-byte-hex-address>:<64-byte-hex-x25519>" form as a fallback, in case
// a user pastes keys copied individually from `openchat-client keygen`.
func ParseContactCode(code string) (*ParsedContact, error) {
	code = strings.TrimSpace(code)

	if strings.Contains(code, ":") && !strings.HasPrefix(code, contactCodePrefix) {
		return parseColonForm(code)
	}

	code = strings.TrimPrefix(code, contactCodePrefix)
	raw, err := base64.RawURLEncoding.DecodeString(code)
	if err != nil {
		return nil, fmt.Errorf("client: not a valid OpenChat contact code: %w", err)
	}
	if len(raw) != 64 {
		return nil, fmt.Errorf("client: contact code has wrong length (got %d bytes, want 64)", len(raw))
	}

	pc := &ParsedContact{Address: hex.EncodeToString(raw[:32])}
	copy(pc.X25519Pub[:], raw[32:])
	pc.X25519PubHex = hex.EncodeToString(pc.X25519Pub[:])
	return pc, nil
}

func parseColonForm(code string) (*ParsedContact, error) {
	parts := strings.SplitN(code, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("client: expected <address>:<x25519pubkey>")
	}
	addrBytes, err := hex.DecodeString(strings.TrimSpace(parts[0]))
	if err != nil || len(addrBytes) != 32 {
		return nil, fmt.Errorf("client: invalid address hex")
	}
	x, err := DecodeRecipientX25519(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, err
	}
	return &ParsedContact{
		Address:      hex.EncodeToString(addrBytes),
		X25519Pub:    x,
		X25519PubHex: hex.EncodeToString(x[:]),
	}, nil
}
