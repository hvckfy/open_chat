package app

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"golang.org/x/crypto/scrypt"
)

const walletFileName = "wallet.v1.json"

// walletFile is the on-disk (PIN-encrypted) format. The mnemonic — the
// single secret that recovers the whole identity — never touches storage
// in plaintext; only this struct does.
type walletFile struct {
	Salt       string `json:"salt"`       // base64, scrypt salt
	Nonce      string `json:"nonce"`      // base64, AES-GCM nonce
	Ciphertext string `json:"ciphertext"` // base64, AES-256-GCM(mnemonic)
}

// scrypt cost parameters: N=2^15 is a reasonable interactive-login cost
// (~100-300ms on a phone) without being annoying on every unlock.
const (
	scryptN = 1 << 15
	scryptR = 8
	scryptP = 1
)

// WalletStore persists exactly one wallet's mnemonic, PIN-encrypted, using
// Fyne's storage API — the portable way to get an app-private, writable
// directory on macOS, Windows, Linux, iOS *and* Android from one code
// path (raw os.UserConfigDir() is not reliably usable in mobile sandboxes).
type WalletStore struct{}

func NewWalletStore() *WalletStore { return &WalletStore{} }

func (s *WalletStore) walletURI() (fyne.URI, error) {
	root := fyne.CurrentApp().Storage().RootURI()
	return storage.Child(root, walletFileName)
}

// Exists reports whether a wallet has already been onboarded on this
// device.
func (s *WalletStore) Exists() (bool, error) {
	uri, err := s.walletURI()
	if err != nil {
		return false, err
	}
	return storage.Exists(uri)
}

// Delete permanently erases the on-disk wallet file, if one exists. This
// is the core of "Log out": once this returns, no trace of the previous
// identity's encrypted mnemonic remains on this device (the in-memory
// mnemonic/keys the caller was holding must be dropped separately by
// discarding the *client.Wallet/Service and starting a fresh onboarding
// flow — this method only handles the persisted half).
func (s *WalletStore) Delete() error {
	uri, err := s.walletURI()
	if err != nil {
		return err
	}
	exists, err := storage.Exists(uri)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	return storage.Delete(uri)
}

// Save encrypts mnemonic with a key derived from pin (+ a fresh random
// salt) and writes it to app-private storage.
func (s *WalletStore) Save(mnemonic, pin string) error {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("app: read salt: %w", err)
	}
	key, err := scrypt.Key([]byte(pin), salt, scryptN, scryptR, scryptP, 32)
	if err != nil {
		return fmt.Errorf("app: derive key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("app: aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("app: gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("app: read nonce: %w", err)
	}
	ct := gcm.Seal(nil, nonce, []byte(mnemonic), nil)

	wf := walletFile{
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ct),
	}
	data, err := json.Marshal(wf)
	if err != nil {
		return err
	}

	uri, err := s.walletURI()
	if err != nil {
		return err
	}
	w, err := storage.Writer(uri)
	if err != nil {
		return fmt.Errorf("app: open wallet file for writing: %w", err)
	}
	defer w.Close()
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("app: write wallet file: %w", err)
	}
	return nil
}

// ErrWrongPIN is returned by Load when decryption fails, which (barring
// disk corruption) means the PIN was wrong.
var ErrWrongPIN = fmt.Errorf("app: wrong PIN or corrupted wallet file")

// Load decrypts and returns the stored mnemonic.
func (s *WalletStore) Load(pin string) (string, error) {
	uri, err := s.walletURI()
	if err != nil {
		return "", err
	}
	r, err := storage.Reader(uri)
	if err != nil {
		return "", fmt.Errorf("app: open wallet file: %w", err)
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("app: read wallet file: %w", err)
	}

	var wf walletFile
	if err := json.Unmarshal(data, &wf); err != nil {
		return "", fmt.Errorf("app: parse wallet file: %w", err)
	}

	salt, err := base64.StdEncoding.DecodeString(wf.Salt)
	if err != nil {
		return "", fmt.Errorf("app: decode salt: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(wf.Nonce)
	if err != nil {
		return "", fmt.Errorf("app: decode nonce: %w", err)
	}
	ct, err := base64.StdEncoding.DecodeString(wf.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("app: decode ciphertext: %w", err)
	}

	key, err := scrypt.Key([]byte(pin), salt, scryptN, scryptR, scryptP, 32)
	if err != nil {
		return "", fmt.Errorf("app: derive key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("app: aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("app: gcm: %w", err)
	}
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", ErrWrongPIN
	}
	return string(pt), nil
}
