package mobile

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"openchat/pkg/client"
)

const historySyncTimeout = 60 * time.Second

// historyEvent is the JSON shape of one entry in HistoryResult.EventsJSON
// — deliberately plain, Codable-friendly field names/types, since this
// crosses the gomobile boundary as a string rather than a bound struct
// (see the package doc comment for why: gomobile doesn't support slices
// of struct in an exported signature, and a history batch is naturally a
// list).
type historyEvent struct {
	Height        int64  `json:"height"`
	TxHash        string `json:"txHash"`
	TimestampMs   int64  `json:"timestampMs"`
	Incoming      bool   `json:"incoming"`
	PeerAddress   string `json:"peerAddress"`
	PeerX25519Hex string `json:"peerX25519Hex"`
	Text          string `json:"text"`
}

// HistoryResult is FetchHistory's return value.
type HistoryResult struct {
	// EventsJSON is a JSON array of the recovered events, newest last,
	// each shaped like historyEvent above:
	//   [{"height":123,"txHash":"...","timestampMs":...,"incoming":true,
	//     "peerAddress":"...","peerX25519Hex":"...","text":"..."}, ...]
	EventsJSON string
	// NextHeight is the height to pass as fromHeight on the next call —
	// persist it (e.g. in the native shell's local chat store) so a
	// later sync only scans the gap instead of the whole chain again.
	NextHeight int64
}

// FetchHistory recovers this session's message history from the chain,
// from fromHeight (0 for a full resync) up through whatever's currently
// committed — see pkg/client.Client.FetchHistory's doc comment for the
// full explanation, including its one real limitation: this wallet's own
// past OUTGOING messages are only recoverable for peers whose X25519 key
// is already known, either from knownKeysJSON or from an incoming
// message found earlier in the same scan.
//
// knownKeysJSON is a JSON object of address -> hex X25519 pubkey, e.g.
// `{"<address>":"<hex>", ...}` — seed it from the native shell's saved
// contacts. Malformed entries are skipped rather than failing the whole
// call.
func (s *Session) FetchHistory(fromHeight int64, knownKeysJSON string) (*HistoryResult, error) {
	known := make(map[string]string)
	if knownKeysJSON != "" {
		if err := json.Unmarshal([]byte(knownKeysJSON), &known); err != nil {
			return nil, fmt.Errorf("mobile: parse knownKeysJSON: %w", err)
		}
	}
	knownRaw := make(map[string][32]byte, len(known))
	for addr, hexKey := range known {
		pub, err := client.DecodeRecipientX25519(hexKey)
		if err != nil {
			continue // malformed saved key; skip rather than fail the whole sync
		}
		knownRaw[addr] = pub
	}

	ctx, cancel := context.WithTimeout(context.Background(), historySyncTimeout)
	defer cancel()

	var events []historyEvent
	next, err := s.client.FetchHistory(ctx, s.wallet, uint64(fromHeight), knownRaw, func(ev client.HistoryEvent) {
		events = append(events, historyEvent{
			Height:        int64(ev.Height),
			TxHash:        ev.TxHash,
			TimestampMs:   ev.Timestamp,
			Incoming:      ev.Incoming,
			PeerAddress:   ev.PeerAddress,
			PeerX25519Hex: ev.PeerX25519PubHex,
			Text:          string(ev.Plaintext),
		})
	})
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(events)
	if err != nil {
		return nil, fmt.Errorf("mobile: encode history events: %w", err)
	}
	return &HistoryResult{EventsJSON: string(raw), NextHeight: int64(next)}, nil
}
