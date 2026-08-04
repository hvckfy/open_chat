package mobile

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"openchat/pkg/client"
	"openchat/pkg/tlsutil"
)

const defaultCallTimeout = 20 * time.Second

// MessageListener receives live incoming messages from Session.Listen.
// Implement this as a native (Swift/Kotlin) type and pass it to Listen —
// gomobile turns a Go interface exported from a bound package into a
// native protocol/interface the host language implements, and generates
// a Go-side proxy that calls back into it. OnMessage is called
// synchronously on whatever goroutine Listen is running on (Listen
// itself blocks, so call it from a background thread/Task, never the
// main/UI thread) for every decrypted message as it arrives; return true
// to keep listening, false to make Listen return.
type MessageListener interface {
	OnMessage(fromAddress string, fromX25519Hex string, plaintext []byte, txHash string, blockHeight int64) bool
}

// LogListener receives human-readable status/diagnostic lines from the
// underlying network layer (gateway connect/failover/reconnect events —
// the same strings pkg/client.Client.OnEvent already formats). Optional;
// wire it up if you want to show connection status in the UI (design-
// code.md B1's status strip / design-spec.md's "connecting…" states).
type LogListener interface {
	OnLog(message string)
}

// Session is one wallet's live connection to the OpenChat network: the
// resilient gateway discovery/failover client from pkg/client, bound to
// one Wallet. Create one per unlocked wallet (mirrors how app/fyne's
// internal/app.Service is built per-session); Close it on logout.
type Session struct {
	wallet *client.Wallet
	client *client.Client

	mu     sync.Mutex
	cancel context.CancelFunc // set while Listen is running; nil otherwise
}

// NewSession builds a session for w, ready to Connect.
//
//   - bootstrapCSV: comma-separated "host:port" gateway list, or "" to
//     use the built-in defaults (client.DefaultBootstrapGateways) —
//     this is design-code.md B6's "Network settings" bootstrap field.
//   - caPEM: PEM-encoded CA certificate bytes to trust a private/
//     self-hosted network instead of the public trust store, or nil for
//     the system trust store (the normal case). Bytes rather than a file
//     path so the native app can hold this in memory/Keychain instead of
//     writing a cert to disk.
//   - insecureSkipVerify: skip TLS verification entirely — local/dev
//     networks only, never for a real deployment (see design-spec.md
//     B6's "danger"-tinted warning treatment for this toggle).
func NewSession(w *Wallet, bootstrapCSV string, caPEM []byte, insecureSkipVerify bool) (*Session, error) {
	if w == nil {
		return nil, fmt.Errorf("mobile: wallet is nil")
	}
	tlsCreds, err := tlsutil.ClientTLSFromPEM(caPEM, insecureSkipVerify)
	if err != nil {
		return nil, err
	}
	var bootstrap []string
	if bootstrapCSV = strings.TrimSpace(bootstrapCSV); bootstrapCSV != "" {
		for _, s := range strings.Split(bootstrapCSV, ",") {
			if s = strings.TrimSpace(s); s != "" {
				bootstrap = append(bootstrap, s)
			}
		}
	}
	c := client.New(bootstrap, tlsCreds)
	return &Session{wallet: w.inner, client: c}, nil
}

// SetLogListener wires (or, with nil, unwires) status/diagnostic logging
// — see LogListener.
func (s *Session) SetLogListener(l LogListener) {
	if l == nil {
		s.client.OnEvent = nil
		return
	}
	s.client.OnEvent = func(format string, args ...any) { l.OnLog(fmt.Sprintf(format, args...)) }
}

// Connect performs the initial bootstrap-list dial with exponential
// backoff/failover (see pkg/client/discovery.go), up to timeoutSeconds
// (a non-positive value uses a 20s default). Call this once after
// creating the Session and before SendText/Listen/FetchHistory — those
// also lazily connect on first use, but calling Connect explicitly first
// lets the UI show a "connecting…" state and surface a real error
// instead of blocking the first send/listen call.
func (s *Session) Connect(timeoutSeconds int64) error {
	if timeoutSeconds <= 0 {
		timeoutSeconds = int64(defaultCallTimeout.Seconds())
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	return s.client.Connect(ctx)
}

// CurrentGateway is the "host:port" of the gateway this session is
// currently connected to ("" if never connected) — for status display.
func (s *Session) CurrentGateway() string {
	if s.client.Discovery == nil {
		return ""
	}
	return s.client.Discovery.CurrentAddr()
}

// SendText encrypts, signs and submits a text message to toAddress
// (encrypting for toX25519Hex, its hex-encoded X25519 public key — both
// values come from the recipient's contact code, see ParseContactCode).
// Returns the committed transaction's hash on success.
func (s *Session) SendText(toAddress string, toX25519Hex string, text string) (string, error) {
	toPub, err := client.DecodeRecipientX25519(toX25519Hex)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultCallTimeout)
	defer cancel()
	return s.client.SendMessage(ctx, s.wallet, toAddress, toPub, []byte(text))
}

// Listen opens a live subscription for incoming messages and blocks,
// invoking listener.OnMessage for each one, until Stop is called, the
// listener returns false, or an unrecoverable connection error occurs.
// Call this from a background thread/Task — it does not return until one
// of those happens. Only one Listen call may be active at a time per
// Session; call Stop before starting a new one (e.g. across a
// reconnect-with-different-settings flow).
func (s *Session) Listen(listener MessageListener) error {
	if listener == nil {
		return fmt.Errorf("mobile: listener is nil")
	}
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return fmt.Errorf("mobile: already listening — call Stop first")
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.cancel = nil
		s.mu.Unlock()
	}()

	return s.client.Listen(ctx, s.wallet, func(m client.IncomingMessage) bool {
		return listener.OnMessage(m.From, m.FromX25519PubHex, m.Plaintext, m.TxHash, int64(m.BlockHeight))
	})
}

// Stop ends a running Listen call (a no-op if none is running). Listen
// then returns nil once its background read loop notices the
// cancellation.
func (s *Session) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

// Close stops any running Listen and releases the underlying connection.
// Call this on logout, alongside wiping whatever the native shell itself
// persisted (Keychain entry, local chat history) — Session/Wallet never
// held a copy of either beyond process memory, so there's nothing else
// here to wipe.
func (s *Session) Close() error {
	s.Stop()
	return s.client.Discovery.Close()
}
