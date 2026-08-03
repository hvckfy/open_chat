package blockchain

import (
	"fmt"
	"sync"
)

// BlockStore is the persistence port this layer depends on (Dependency
// Inversion: the domain defines the interface, internal/storage provides
// a BadgerDB-backed implementation, cmd/node wires the concrete type in).
type BlockStore interface {
	PutBlock(b *Block) error
	GetBlock(height uint64) (*Block, bool, error)
	Head() (uint64, error)
	LastNonce(address string) (uint64, error)
	SetLastNonce(address string, nonce uint64) error
}

// Chain is the append-only ledger. It validates and persists blocks
// produced by consensus, and fans committed transactions out to
// subscribers (the gRPC gateway's StreamIncomingSMS handlers).
type Chain struct {
	store BlockStore

	mu  sync.RWMutex
	tip uint64

	subMu sync.Mutex
	subs  map[int]chan *CommitEvent
	nextID int
}

// CommitEvent is published once per committed block, carrying every
// transaction in it, so subscribers can filter by recipient.
type CommitEvent struct {
	Height uint64
	Txs    []*Transaction
}

func NewChain(store BlockStore) (*Chain, error) {
	c := &Chain{
		store: store,
		subs:  make(map[int]chan *CommitEvent),
	}
	head, err := store.Head()
	if err != nil {
		return nil, fmt.Errorf("blockchain: load head: %w", err)
	}
	if head == 0 {
		if _, found, _ := store.GetBlock(0); !found {
			if err := store.PutBlock(GenesisBlock()); err != nil {
				return nil, fmt.Errorf("blockchain: write genesis: %w", err)
			}
		}
	}
	c.tip = head
	return c, nil
}

// Height returns the current chain tip height.
func (c *Chain) Height() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tip
}

// Head returns the current tip block.
func (c *Chain) Head() (*Block, error) {
	c.mu.RLock()
	h := c.tip
	c.mu.RUnlock()
	blk, found, err := c.store.GetBlock(h)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("blockchain: missing head block %d", h)
	}
	return blk, nil
}

// GetBlock fetches a block by height.
func (c *Chain) GetBlock(height uint64) (*Block, bool, error) {
	return c.store.GetBlock(height)
}

// CommitBlock appends a consensus-finalized block: validates linkage +
// quorum signatures, persists it, advances the tip, updates per-sender
// nonce watermarks (replay protection) and notifies subscribers.
//
// quorum is 2f+1 for the current validator set size, computed by the
// caller (internal/consensus), since only consensus knows the live set.
func (c *Chain) CommitBlock(b *Block, quorum int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	current, found, err := c.store.GetBlock(c.tip)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("blockchain: local tip %d missing", c.tip)
	}
	if b.Height != c.tip+1 {
		return fmt.Errorf("blockchain: out-of-order block: have tip %d, got height %d", c.tip, b.Height)
	}
	if b.PrevHash != current.Hash() {
		return fmt.Errorf("blockchain: prev hash mismatch at height %d", b.Height)
	}
	if !b.VerifyCommitQuorum(quorum) {
		return fmt.Errorf("blockchain: insufficient/invalid commit quorum on block %d", b.Height)
	}

	for _, tx := range b.Transactions {
		if err := tx.Verify(); err != nil {
			return fmt.Errorf("blockchain: block %d contains invalid tx %s: %w", b.Height, tx.Hash(), err)
		}
		last, err := c.store.LastNonce(tx.From)
		if err != nil {
			return err
		}
		if tx.Nonce <= last {
			return fmt.Errorf("blockchain: replay/out-of-order nonce for %s: have %d, tx has %d", tx.From, last, tx.Nonce)
		}
		if err := c.store.SetLastNonce(tx.From, tx.Nonce); err != nil {
			return err
		}
	}

	if err := c.store.PutBlock(b); err != nil {
		return err
	}
	c.tip = b.Height

	c.publish(&CommitEvent{Height: b.Height, Txs: b.Transactions})
	return nil
}

// Subscribe registers a new listener for committed blocks. Callers must
// call the returned cancel function when done (e.g. client disconnects).
func (c *Chain) Subscribe() (<-chan *CommitEvent, func()) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	id := c.nextID
	c.nextID++
	ch := make(chan *CommitEvent, 64)
	c.subs[id] = ch
	cancel := func() {
		c.subMu.Lock()
		defer c.subMu.Unlock()
		if existing, ok := c.subs[id]; ok {
			delete(c.subs, id)
			close(existing)
		}
	}
	return ch, cancel
}

func (c *Chain) publish(ev *CommitEvent) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	for _, ch := range c.subs {
		select {
		case ch <- ev:
		default:
			// slow subscriber: drop rather than block block-production
		}
	}
}
