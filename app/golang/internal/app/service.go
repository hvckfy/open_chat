package app

import (
	"context"
	"fmt"

	"openchat/pkg/client"
)

// Service is the small glue object cmd/app's UI talks to: it combines the
// unlocked Wallet, a connected pkg/client.Client, and the local
// ChatStore, and runs the background StreamIncomingSMS loop.
//
// All callbacks fire from background goroutines (gRPC stream reads,
// network retries) — UI code must hop back onto the Fyne main goroutine
// itself (fyne.Do / fyne.DoAndWait) before touching widgets from them.
type Service struct {
	Wallet *client.Wallet
	Net    *client.Client
	Store  *ChatStore

	OnMessage func(fromAddress string) // fired after a new inbound message is persisted
	OnStatus  func(text string)        // fired on connect/disconnect/failover events
}

func NewService(wallet *client.Wallet, net *client.Client, store *ChatStore) *Service {
	s := &Service{Wallet: wallet, Net: net, Store: store}
	net.OnEvent = func(format string, args ...any) {
		if s.OnStatus != nil {
			s.OnStatus(fmt.Sprintf(format, args...))
		}
	}
	return s
}

// Start connects, recovers any message history missed since the last
// run (SyncHistory), and then begins listening for new incoming messages
// in the background. Safe to call once per Service; cancel ctx to stop.
//
// SyncHistory runs first and Listen only takes over once it returns, so
// there's a small window between "history sync says it's caught up to
// height H" and "the live subscription actually opens" where a message
// committed in between could theoretically be missed — an accepted
// trade-off given the alternative (coordinating a live stream's buffered
// backlog against an in-flight historical scan) for a gap that in
// practice is at most the time to commit one more block.
func (s *Service) Start(ctx context.Context) {
	go func() {
		if err := s.SyncHistory(ctx); err != nil && ctx.Err() == nil && s.OnStatus != nil {
			s.OnStatus("history sync failed (will still receive new messages live): " + err.Error())
		}

		err := s.Net.Listen(ctx, s.Wallet, func(m client.IncomingMessage) bool {
			stored := &StoredMessage{
				Direction: Incoming,
				Text:      string(m.Plaintext),
				TxHash:    m.TxHash,
				Timestamp: NowMillis(),
			}
			if err := s.Store.AppendMessage(m.From, stored, m.FromX25519PubHex); err != nil && s.OnStatus != nil {
				s.OnStatus("failed to save incoming message: " + err.Error())
			}
			if s.OnMessage != nil {
				s.OnMessage(m.From)
			}
			return true // keep listening
		})
		if err != nil && ctx.Err() == nil && s.OnStatus != nil {
			s.OnStatus("listen stopped: " + err.Error())
		}
	}()
}

// SyncHistory recovers this wallet's message history from the chain,
// covering everything committed since the last successful sync — the
// whole chain, the first time (ChatStore.LastSyncedHeight is 0 on a
// fresh install, and again after Log out's Wipe erases it along with
// everything else). Run automatically by Start before the live listener
// takes over, so it also covers ordinary gaps like "the app was closed
// for a while" — previously any message committed while not actively
// connected was gone for good, since StreamIncomingSMS only ever pushes
// forward from when it opens.
//
// Known limitation, inherent to the E2EE scheme (see
// client.FetchHistory's doc comment): this wallet's own past OUTGOING
// messages can only be recovered for peers whose X25519 key is already
// known, either from a saved contact or from one of their incoming
// messages found earlier in the same scan. A message sent to someone
// who never messaged back and isn't a saved contact anymore can't be
// decrypted again — the ciphertext is still on-chain forever, but the
// key needed to open it was only ever known locally, and that's gone.
func (s *Service) SyncHistory(ctx context.Context) error {
	from := s.Store.LastSyncedHeight()

	known := make(map[string][32]byte)
	for _, c := range s.Store.Contacts() {
		if c.X25519PubHex == "" {
			continue
		}
		pub, err := client.DecodeRecipientX25519(c.X25519PubHex)
		if err != nil {
			continue // malformed saved key; skip rather than fail the whole sync
		}
		known[c.Address] = pub
	}

	if s.OnStatus != nil {
		s.OnStatus("syncing message history…")
	}

	var appendErr error
	next, err := s.Net.FetchHistory(ctx, s.Wallet, from, known, func(ev client.HistoryEvent) {
		direction := Incoming
		if !ev.Incoming {
			direction = Outgoing
		}
		added, aerr := s.Store.AppendHistoryMessage(ev.PeerAddress, &StoredMessage{
			Direction: direction,
			Text:      string(ev.Plaintext),
			TxHash:    ev.TxHash,
			Timestamp: ev.Timestamp,
		}, ev.PeerX25519PubHex)
		if aerr != nil {
			appendErr = aerr
			return
		}
		if added && s.OnMessage != nil {
			s.OnMessage(ev.PeerAddress)
		}
	})
	if err != nil {
		return fmt.Errorf("app: history sync: %w", err)
	}
	if appendErr != nil {
		return fmt.Errorf("app: history sync: saving recovered message: %w", appendErr)
	}
	if err := s.Store.SetLastSyncedHeight(next); err != nil {
		return fmt.Errorf("app: history sync: saving checkpoint: %w", err)
	}

	if s.OnStatus != nil {
		s.OnStatus("message history up to date")
	}
	return nil
}

// SendText encrypts, signs and submits a text message to a saved contact,
// then records it in local history.
func (s *Service) SendText(ctx context.Context, to *Contact, text string) error {
	if to.X25519PubHex == "" {
		return fmt.Errorf("app: contact %s has no known encryption key yet", to.DisplayName)
	}
	recipientX25519, err := client.DecodeRecipientX25519(to.X25519PubHex)
	if err != nil {
		return err
	}

	txHash, err := s.Net.SendMessage(ctx, s.Wallet, to.Address, recipientX25519, []byte(text))
	if err != nil {
		return err
	}

	return s.Store.AppendMessage(to.Address, &StoredMessage{
		Direction: Outgoing,
		Text:      text,
		TxHash:    txHash,
		Timestamp: NowMillis(),
	}, "")
}

// MyAddress is a small convenience passthrough for the UI.
func (s *Service) MyAddress() string { return s.Wallet.Address() }

// MyContactCode is the string this device's owner shares with others so
// they can add this wallet as a contact.
func (s *Service) MyContactCode() string { return MyContactCode(s.Wallet) }
