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

// Start connects and begins listening for incoming messages in the
// background. Safe to call once per Service; cancel ctx to stop.
func (s *Service) Start(ctx context.Context) {
	go func() {
		err := s.Net.Listen(ctx, s.Wallet, func(m client.IncomingMessage) bool {
			stored := &StoredMessage{
				Direction: Incoming,
				Text:      string(m.Plaintext),
				TxHash:    m.TxHash,
				Timestamp: NowMillis(),
			}
			if err := s.Store.AppendMessage(m.From, stored); err != nil && s.OnStatus != nil {
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
	})
}

// MyAddress is a small convenience passthrough for the UI.
func (s *Service) MyAddress() string { return s.Wallet.Address() }

// MyContactCode is the string this device's owner shares with others so
// they can add this wallet as a contact.
func (s *Service) MyContactCode() string { return MyContactCode(s.Wallet) }
