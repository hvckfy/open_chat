// Package mempool holds unconfirmed transactions and enforces the
// network's spam/replay defenses before a transaction is ever gossiped or
// proposed into a block:
//
//   - rate limiting: max 5 tx/sec per sender public key (token bucket)
//   - replay protection: strictly-incrementing per-sender nonce, checked
//     both against committed chain state and other pending transactions
//   - freshness: reject transactions whose timestamp has drifted too far
//     from the node's clock (defends against stockpiled old signed txs)
package mempool

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"openchat/internal/blockchain"
)

const (
	// MaxTxPerSecondPerSender enforces "не более 5 транзакций в секунду
	// от одного публичного ключа".
	MaxTxPerSecondPerSender = 5
	rateBucketBurst         = MaxTxPerSecondPerSender

	// MaxClockSkew bounds how far a tx timestamp may drift from "now",
	// in either direction, before it's rejected as stale/replayed or
	// from-the-future.
	MaxClockSkew = 2 * time.Minute

	// MaxPoolSize caps total pending transactions to bound memory.
	MaxPoolSize = 50_000
)

// NonceSource lets the mempool check a sender's last committed nonce
// without depending on the concrete storage engine.
type NonceSource interface {
	LastNonce(address string) (uint64, error)
}

type Mempool struct {
	nonces NonceSource

	mu       sync.Mutex
	byHash   map[string]*blockchain.Transaction
	pending  map[string]uint64 // address -> highest nonce currently pending
	limiters map[string]*rate.Limiter
}

func New(nonces NonceSource) *Mempool {
	return &Mempool{
		nonces:   nonces,
		byHash:   make(map[string]*blockchain.Transaction),
		pending:  make(map[string]uint64),
		limiters: make(map[string]*rate.Limiter),
	}
}

// Add validates and admits a transaction. Called both for transactions
// arriving from local gRPC clients and ones gossiped in from peers.
func (m *Mempool) Add(tx *blockchain.Transaction) error {
	if err := tx.Verify(); err != nil {
		return fmt.Errorf("mempool: %w", err)
	}

	now := time.Now().UnixMilli()
	skew := now - tx.Timestamp
	if skew < 0 {
		skew = -skew
	}
	if time.Duration(skew)*time.Millisecond > MaxClockSkew {
		return fmt.Errorf("mempool: tx timestamp outside freshness window (skew=%dms)", skew)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.byHash) >= MaxPoolSize {
		return fmt.Errorf("mempool: full")
	}

	limiter, ok := m.limiters[tx.From]
	if !ok {
		limiter = rate.NewLimiter(rate.Limit(MaxTxPerSecondPerSender), rateBucketBurst)
		m.limiters[tx.From] = limiter
	}
	if !limiter.Allow() {
		return fmt.Errorf("mempool: rate limit exceeded for sender %s (max %d tx/s)", tx.From, MaxTxPerSecondPerSender)
	}

	lastCommitted, err := m.nonces.LastNonce(tx.From)
	if err != nil {
		return fmt.Errorf("mempool: nonce lookup: %w", err)
	}
	highestPending := m.pending[tx.From]
	floor := lastCommitted
	if highestPending > floor {
		floor = highestPending
	}
	if tx.Nonce <= floor {
		return fmt.Errorf("mempool: replayed/stale nonce for %s: have floor %d, tx has %d", tx.From, floor, tx.Nonce)
	}

	hash := tx.Hash()
	if _, exists := m.byHash[hash]; exists {
		return nil // already known, idempotent
	}

	m.byHash[hash] = tx
	m.pending[tx.From] = tx.Nonce
	return nil
}

// Size returns the current pool size (exported for Prometheus metrics).
func (m *Mempool) Size() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.byHash)
}

// Propose returns up to `limit` pending transactions, ordered by
// (From, Nonce) so that per-sender ordering is preserved when a proposer
// builds the next block.
func (m *Mempool) Propose(limit int) []*blockchain.Transaction {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]*blockchain.Transaction, 0, len(m.byHash))
	for _, tx := range m.byHash {
		out = append(out, tx)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].Nonce < out[j].Nonce
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// Remove evicts confirmed (or permanently invalid) transactions, e.g.
// after a block containing them is committed.
func (m *Mempool) Remove(txs []*blockchain.Transaction) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, tx := range txs {
		delete(m.byHash, tx.Hash())
	}
}
