package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"openchat/internal/grpcserver/pb"
	"openchat/pkg/authproof"
	"openchat/pkg/crypto"
)

// perCallTimeout bounds a single unary RPC attempt (GetBlocks, SendSMS),
// independent of whatever timeout the caller's own ctx carries.
//
// Without this, a single call whose connection has gone quietly stale —
// TCP still "up" but the peer no longer answering, rather than a clean
// connection-refused/closed error — blocks until the *caller's* full
// remaining deadline (e.g. FetchHistory's 60s history-sync budget)
// finally expires before gRPC reports DeadlineExceeded and this package
// notices it's a transport error worth failing over from. By then
// there's essentially no time left in the outer context for Failover's
// own dials, which derive their per-candidate timeout from that same
// shrinking budget (`context.WithTimeout(ctx, dialTimeout)` in
// discovery.go) — so even a perfectly healthy alternate gateway can
// fail to connect in time, not because it's actually unreachable but
// because it was only ever given a fraction of a second instead of
// `dialTimeout`'s real 5s. Bounding each call to `perCallTimeout`
// instead means a stale connection gets detected (and failed over from)
// quickly, leaving the rest of the outer budget genuinely available for
// failover to succeed against a working candidate.
const perCallTimeout = 15 * time.Second

// Client is the top-level object a CLI or mobile app embeds: it owns a
// Discovery (connection + failover) and exposes the four gateway RPCs as
// plain Go calls, transparently retrying once via Failover on transport
// errors.
type Client struct {
	Discovery *Discovery
	OnEvent   func(format string, args ...any)
}

// New builds a Client with the given bootstrap gateway list (nil/empty
// uses DefaultBootstrapGateways) and TLS credentials. It does not connect
// yet — call Connect (or let SendMessage/Listen do it lazily via
// EnsureConnected) before use.
func New(bootstrap []string, tlsCreds credentials.TransportCredentials) *Client {
	d := NewDiscovery(bootstrap, tlsCreds)
	c := &Client{Discovery: d}
	d.OnEvent = func(format string, args ...any) { c.event(format, args...) }
	return c
}

// Connect performs the initial bootstrap-list dial with exponential
// backoff, as described in the architecture doc's "Алгоритм живучести".
func (c *Client) Connect(ctx context.Context) error {
	return c.Discovery.Connect(ctx)
}

// EnsureConnected connects if this is the first use.
func (c *Client) EnsureConnected(ctx context.Context) error {
	if c.Discovery.Conn() != nil {
		return nil
	}
	return c.Connect(ctx)
}

// IncomingMessage is a decrypted message delivered to the application
// layer by Listen.
type IncomingMessage struct {
	From string
	// FromX25519PubHex is the hex-encoded X25519 public key the sender
	// actually used to encrypt this message (the wire's "ephemeral_pubkey"
	// field — see api/protobuf/sms.proto). Every SMSRequest carries it, so
	// the recipient learns everything needed to reply — address plus
	// encryption key — straight from a stranger's first message, without
	// a separate manual contact-code exchange. It's already
	// cryptographically tied to From: decryptIncoming only succeeds if
	// this key and the sender's real private key actually match.
	FromX25519PubHex string
	Plaintext        []byte
	TxHash           string
	BlockHeight      uint64
}

func (c *Client) gw() pb.NodeGatewayClient {
	return pb.NewNodeGatewayClient(c.Discovery.Conn())
}

// GetAddress queries the currently connected gateway's own identity.
func (c *Client) GetAddress(ctx context.Context) (*pb.AddressResponse, error) {
	if err := c.EnsureConnected(ctx); err != nil {
		return nil, err
	}
	return c.gw().GetAddress(ctx, &pb.Empty{})
}

// GetNodesDiscovery exposes the raw RPC for callers that want to inspect
// the live gateway list directly (Discovery.refreshCache uses this
// internally too).
func (c *Client) GetNodesDiscovery(ctx context.Context, max uint32) (*pb.DiscoveryResponse, error) {
	if err := c.EnsureConnected(ctx); err != nil {
		return nil, err
	}
	return c.gw().GetNodesDiscovery(ctx, &pb.DiscoveryRequest{MaxResults: max})
}

// SendMessage builds, encrypts (E2EE) and signs a message transaction and
// submits it via SendSMS, transparently failing over to a different
// gateway once if the currently active one is unreachable.
func (c *Client) SendMessage(ctx context.Context, w *Wallet, toAddress string, toX25519Pub [32]byte, plaintext []byte) (string, error) {
	if err := c.EnsureConnected(ctx); err != nil {
		return "", err
	}

	sealed, err := crypto.Encrypt(w.Keys.EncryptionPrivate, w.Keys.EncryptionPublic, toX25519Pub, plaintext)
	if err != nil {
		return "", fmt.Errorf("client: encrypt: %w", err)
	}

	tx := buildTx(w, toAddress, sealed)

	req := &pb.SMSRequest{
		FromAddress:     tx.From,
		ToAddress:       tx.To,
		Ciphertext:      tx.Ciphertext,
		NonceAEAD:       tx.NonceAEAD[:],
		EphemeralPubkey: tx.EphemeralPubkey[:],
		Nonce:           tx.Nonce,
		Timestamp:       tx.Timestamp,
		Signature:       tx.Signature,
	}

	resp, err := c.callSendSMS(ctx, req)
	if isTransportError(err) {
		if ferr := c.Discovery.Failover(ctx); ferr != nil {
			return "", fmt.Errorf("client: send failed and failover exhausted: %w", ferr)
		}
		resp, err = c.callSendSMS(ctx, req)
	}
	if err != nil {
		return "", fmt.Errorf("client: SendSMS rpc: %w", err)
	}
	if !resp.Accepted {
		return "", fmt.Errorf("client: node rejected message: %s", resp.Error)
	}
	return resp.TxHash, nil
}

// callSendSMS bounds a single SendSMS attempt to perCallTimeout — see
// that constant's doc comment.
func (c *Client) callSendSMS(ctx context.Context, req *pb.SMSRequest) (*pb.SMSResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, perCallTimeout)
	defer cancel()
	return c.gw().SendSMS(callCtx, req)
}

// Listen opens (and, on failure, transparently reopens against a
// different gateway) a StreamIncomingSMS subscription for w, decrypting
// each arriving message and invoking onMessage. Blocks until ctx is
// canceled or onMessage requests it stop by returning false — whichever
// happens first.
func (c *Client) Listen(ctx context.Context, w *Wallet, onMessage func(IncomingMessage) bool) error {
	if err := c.EnsureConnected(ctx); err != nil {
		return err
	}
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		now := time.Now().UnixMilli()
		authSig := ed25519Sign(w, authproof.Bytes(w.Address(), now))

		stream, err := c.gw().StreamIncomingSMS(ctx, &pb.StreamRequest{
			Address:   w.Address(),
			Timestamp: now,
			Signature: authSig,
		})
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.event("client: stream open failed (%v), failing over", err)
			if ferr := c.Discovery.Failover(ctx); ferr != nil {
				return ferr
			}
			continue
		}

		streamErr := c.consume(stream, w, onMessage)
		if streamErr == nil {
			return nil // onMessage asked to stop
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		c.event("client: stream broken (%v), failing over transparently", streamErr)
		if ferr := c.Discovery.Failover(ctx); ferr != nil {
			return ferr
		}
		// loop: reopen the stream against the newly failed-over gateway
	}
}

func (c *Client) consume(stream pb.NodeGateway_StreamIncomingSMSClient, w *Wallet, onMessage func(IncomingMessage) bool) error {
	for {
		resp, err := stream.Recv()
		if err != nil {
			return err
		}
		if resp.Message == nil {
			continue
		}
		plain, err := decryptIncoming(w, resp.Message)
		if err != nil {
			c.event("client: dropping undecryptable message %s: %v", resp.TxHash, err)
			continue
		}
		keepGoing := onMessage(IncomingMessage{
			From:             resp.Message.FromAddress,
			FromX25519PubHex: hex.EncodeToString(resp.Message.EphemeralPubkey),
			Plaintext:        plain,
			TxHash:           resp.TxHash,
			BlockHeight:      resp.BlockHeight,
		})
		if !keepGoing {
			return nil
		}
	}
}

// HistoryEvent is one transaction recovered from the chain by
// FetchHistory — either a message this wallet received or one it sent,
// exactly as StreamIncomingSMS/SendMessage would have delivered/recorded
// it live, just replayed from committed chain state instead of gossip.
type HistoryEvent struct {
	Height      uint64
	TxHash      string
	Timestamp   int64
	Incoming    bool   // true: this wallet was the recipient; false: this wallet sent it
	PeerAddress string // sender's address if Incoming, recipient's otherwise
	// PeerX25519PubHex is the peer's encryption key: for an Incoming
	// event it's learned straight from the transaction; for an outgoing
	// one it's simply the key the caller already had to supply (via
	// knownX25519Pub) to decrypt it in the first place.
	PeerX25519PubHex string
	Plaintext        []byte
}

// FetchHistory walks committed blocks from fromHeight (0 for a full
// resync — first login on a new device, or after ChatStore.Wipe, when
// the local cache has nothing but the chain still does) up through
// whatever's currently committed, decrypting every transaction this
// wallet can find a key for and reporting it via onEvent as it's found.
// Returns the height to resume from next time (one past the last block
// actually seen), so a caller can persist it and only scan the gap on
// the next call instead of the whole chain every time.
//
// Every incoming transaction (To == this wallet) is always decryptable:
// the chain stores the sender's X25519 pubkey right on the transaction,
// which combined with this wallet's own private key is everything ECDH
// needs — no prior contact required. That's not true for this wallet's
// own past OUTGOING transactions: the chain only ever stores the
// *sender's* key, since that's all a recipient needs, so decrypting our
// own old sent ciphertext instead needs the recipient's key, which has
// to come from somewhere else. knownX25519Pub supplies that (seed it
// with whatever ChatStore already has saved); FetchHistory also grows
// it in place as incoming transactions reveal new peers' keys, so an
// outgoing transaction to someone who only messaged us back later in
// the same scan still resolves. An outgoing transaction to an address
// whose key was never learned is skipped, not treated as an error —
// that message is genuinely unrecoverable without it.
func (c *Client) FetchHistory(ctx context.Context, w *Wallet, fromHeight uint64, knownX25519Pub map[string][32]byte, onEvent func(HistoryEvent)) (uint64, error) {
	if err := c.EnsureConnected(ctx); err != nil {
		return fromHeight, err
	}

	myAddr := w.Address()
	height := fromHeight

	for {
		if ctx.Err() != nil {
			return height, ctx.Err()
		}

		req := &pb.GetBlocksRequest{FromHeight: height, MaxBlocks: 500}
		resp, err := c.callGetBlocks(ctx, req)
		if isTransportError(err) {
			if ferr := c.Discovery.Failover(ctx); ferr != nil {
				return height, fmt.Errorf("client: history sync failed and failover exhausted: %w", ferr)
			}
			resp, err = c.callGetBlocks(ctx, req)
		}
		if err != nil {
			return height, fmt.Errorf("client: GetBlocks: %w", err)
		}
		if len(resp.Blocks) == 0 {
			return height, nil // fully caught up
		}

		for _, b := range resp.Blocks {
			for _, tx := range b.Transactions {
				switch {
				case tx.ToAddress == myAddr:
					plain, derr := decryptIncoming(w, tx)
					if derr != nil {
						c.event("client: history: dropping undecryptable incoming tx at height %d: %v", b.Height, derr)
						continue
					}
					var senderPub [32]byte
					copy(senderPub[:], tx.EphemeralPubkey)
					if knownX25519Pub != nil {
						knownX25519Pub[tx.FromAddress] = senderPub
					}
					onEvent(HistoryEvent{
						Height:           b.Height,
						TxHash:           txHashFromPB(tx),
						Timestamp:        tx.Timestamp,
						Incoming:         true,
						PeerAddress:      tx.FromAddress,
						PeerX25519PubHex: hex.EncodeToString(senderPub[:]),
						Plaintext:        plain,
					})
				case tx.FromAddress == myAddr:
					pub, ok := knownX25519Pub[tx.ToAddress]
					if !ok {
						continue
					}
					plain, derr := decryptOutgoing(w, tx, pub)
					if derr != nil {
						c.event("client: history: dropping undecryptable outgoing tx at height %d: %v", b.Height, derr)
						continue
					}
					onEvent(HistoryEvent{
						Height:           b.Height,
						TxHash:           txHashFromPB(tx),
						Timestamp:        tx.Timestamp,
						Incoming:         false,
						PeerAddress:      tx.ToAddress,
						PeerX25519PubHex: hex.EncodeToString(pub[:]),
						Plaintext:        plain,
					})
				}
			}
			height = b.Height + 1
		}
	}
}

// callGetBlocks bounds a single GetBlocks attempt to perCallTimeout —
// see that constant's doc comment. Each iteration of FetchHistory's loop
// calls this fresh, so a connection that goes stale partway through a
// long scan is detected (and failed over from) within perCallTimeout of
// it happening, not only once the entire scan's outer deadline expires.
func (c *Client) callGetBlocks(ctx context.Context, req *pb.GetBlocksRequest) (*pb.GetBlocksResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, perCallTimeout)
	defer cancel()
	return c.gw().GetBlocks(callCtx, req)
}

func (c *Client) event(format string, args ...any) {
	if c.OnEvent != nil {
		c.OnEvent(format, args...)
	}
}

func isTransportError(err error) bool {
	if err == nil {
		return false
	}
	s, ok := status.FromError(err)
	if !ok {
		return true
	}
	switch s.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Aborted, codes.Internal:
		return true
	default:
		return false
	}
}
