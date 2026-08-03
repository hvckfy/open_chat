package app

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

const chatsFileName = "chats.v1.json"

// Contact is one saved conversation partner.
type Contact struct {
	Address      string `json:"address"`
	X25519PubHex string `json:"x25519_pub_hex"`
	DisplayName  string `json:"display_name"`
}

// Direction distinguishes messages this wallet sent from ones it received.
type Direction string

const (
	Outgoing Direction = "out"
	Incoming Direction = "in"
)

// StoredMessage is one already-decrypted message kept in local history.
// Only plaintext the device has already seen is stored here — this file
// never holds ciphertext or key material.
type StoredMessage struct {
	Direction Direction `json:"direction"`
	Text      string    `json:"text"`
	TxHash    string    `json:"tx_hash"`
	Timestamp int64     `json:"timestamp_unix_ms"`
}

type chatsFile struct {
	Contacts map[string]*Contact         `json:"contacts"` // keyed by address
	Messages map[string][]*StoredMessage `json:"messages"` // keyed by contact address
}

// ChatStore is the local (device-only) address book + message history.
// Safe for concurrent use; every mutating call persists immediately so a
// killed app never loses a message.
type ChatStore struct {
	mu   sync.Mutex
	data chatsFile
}

func NewChatStore() (*ChatStore, error) {
	s := &ChatStore{data: chatsFile{
		Contacts: make(map[string]*Contact),
		Messages: make(map[string][]*StoredMessage),
	}}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *ChatStore) uri() (fyne.URI, error) {
	root := fyne.CurrentApp().Storage().RootURI()
	return storage.Child(root, chatsFileName)
}

func (s *ChatStore) load() error {
	uri, err := s.uri()
	if err != nil {
		return err
	}
	exists, err := storage.Exists(uri)
	if err != nil {
		return err
	}
	if !exists {
		return nil // fresh install, nothing to load
	}
	r, err := storage.Reader(uri)
	if err != nil {
		return fmt.Errorf("app: open chats file: %w", err)
	}
	defer r.Close()
	raw, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("app: read chats file: %w", err)
	}
	if len(raw) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := json.Unmarshal(raw, &s.data); err != nil {
		return fmt.Errorf("app: parse chats file: %w", err)
	}
	if s.data.Contacts == nil {
		s.data.Contacts = make(map[string]*Contact)
	}
	if s.data.Messages == nil {
		s.data.Messages = make(map[string][]*StoredMessage)
	}
	return nil
}

// persist must be called with s.mu held.
func (s *ChatStore) persist() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	uri, err := s.uri()
	if err != nil {
		return err
	}
	w, err := storage.Writer(uri)
	if err != nil {
		return fmt.Errorf("app: open chats file for writing: %w", err)
	}
	defer w.Close()
	_, err = w.Write(raw)
	return err
}

// UpsertContact adds or updates a saved contact.
func (s *ChatStore) UpsertContact(c *Contact) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Contacts[c.Address] = c
	if _, ok := s.data.Messages[c.Address]; !ok {
		s.data.Messages[c.Address] = nil
	}
	return s.persist()
}

// Contact looks up a saved contact by address.
func (s *ChatStore) Contact(address string) (*Contact, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.data.Contacts[address]
	return c, ok
}

// Contacts returns all saved contacts, sorted by display name.
func (s *ChatStore) Contacts() []*Contact {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Contact, 0, len(s.data.Contacts))
	for _, c := range s.data.Contacts {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DisplayName < out[j].DisplayName })
	return out
}

// AppendMessage records a new message (sent or received) for `address`,
// auto-creating a placeholder contact entry if one doesn't exist yet
// (e.g. a first message arriving from someone not yet explicitly added).
//
// senderX25519PubHex is the encryption key the *other side* used for this
// message — pass "" for outgoing messages (we already know our own
// contact's key, if any). For an incoming message it's the sender's real
// X25519 pubkey (see client.IncomingMessage.FromX25519PubHex): the first
// time we hear from an address, this fills in the one piece a reply
// needs that a bare address doesn't carry, so a stranger's first message
// is immediately answerable — no separate contact-code exchange required.
// An already-known key (either learned before, or set explicitly by the
// user adding/editing a contact) is never overwritten by this.
func (s *ChatStore) AppendMessage(address string, msg *StoredMessage, senderX25519PubHex string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.data.Contacts[address]
	if !ok {
		c = &Contact{Address: address, DisplayName: shortAddr(address)}
		s.data.Contacts[address] = c
	}
	if senderX25519PubHex != "" && c.X25519PubHex == "" {
		c.X25519PubHex = senderX25519PubHex
	}
	s.data.Messages[address] = append(s.data.Messages[address], msg)
	return s.persist()
}

// Messages returns the full local history for one contact, oldest first.
func (s *ChatStore) Messages(address string) []*StoredMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*StoredMessage{}, s.data.Messages[address]...)
}

// Wipe erases every saved contact and message, both in memory and on
// disk — the other half of "Log out" (see WalletStore.Delete): together
// they make sure nothing about the previous identity's conversations
// survives on this device.
func (s *ChatStore) Wipe() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = chatsFile{
		Contacts: make(map[string]*Contact),
		Messages: make(map[string][]*StoredMessage),
	}

	uri, err := s.uri()
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

func shortAddr(addr string) string {
	if len(addr) <= 12 {
		return addr
	}
	return addr[:6] + "…" + addr[len(addr)-6:]
}

// NowMillis is a small helper so UI code doesn't need to import "time"
// just to timestamp an outgoing message.
func NowMillis() int64 { return time.Now().UnixMilli() }
