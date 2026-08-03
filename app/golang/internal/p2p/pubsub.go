package p2p

import (
	"context"
	"fmt"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"openchat/internal/consensus"
)

// This file adapts go-libp2p-pubsub's gossipsub to the small
// consensus.PubSub / consensus.Subscription interfaces, so
// internal/consensus and internal/mempool broadcast logic never import
// libp2p directly (Clean Architecture boundary).

var topicMu sync.Mutex

func (n *Node) topic(name string) (*pubsub.Topic, error) {
	topicMu.Lock()
	defer topicMu.Unlock()
	if t, ok := n.topics[name]; ok {
		return t, nil
	}
	t, err := n.pubsub.Join(name)
	if err != nil {
		return nil, fmt.Errorf("p2p: join topic %s: %w", name, err)
	}
	n.topics[name] = t
	return t, nil
}

// Publish broadcasts data on the given gossipsub topic.
func (n *Node) Publish(ctx context.Context, topicName string, data []byte) error {
	t, err := n.topic(topicName)
	if err != nil {
		return err
	}
	return t.Publish(ctx, data)
}

// Subscription wraps a *pubsub.Subscription to yield only the raw payload.
//
// Deliberately NOT filtering out self-originated messages here: gossipsub
// delivers a node's own Publish back to its own local Subscription (no
// network round-trip needed for that), and internal/consensus relies on
// exactly this to count its own PREVOTE/PRECOMMIT toward its own quorum
// tally — handleVote/handleProposal are idempotent (deduped by voter
// address / proposal key), so redelivery is always safe.
type Subscription struct {
	sub *pubsub.Subscription
}

func (s *Subscription) Next(ctx context.Context) ([]byte, error) {
	msg, err := s.sub.Next(ctx)
	if err != nil {
		return nil, err
	}
	return msg.Data, nil
}

func (s *Subscription) Cancel() { s.sub.Cancel() }

// Subscribe joins topicName (if not already joined) and returns a live
// subscription satisfying consensus.Subscription (and, by the same shape,
// whatever the mempool gossip layer needs).
func (n *Node) Subscribe(topicName string) (consensus.Subscription, error) {
	t, err := n.topic(topicName)
	if err != nil {
		return nil, err
	}
	sub, err := t.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("p2p: subscribe %s: %w", topicName, err)
	}
	return &Subscription{sub: sub}, nil
}

var _ consensus.PubSub = (*Node)(nil)
