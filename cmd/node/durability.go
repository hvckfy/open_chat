package main

import (
	"context"
	"time"

	"go.uber.org/zap"

	"openchat/internal/blockchain"
	"openchat/internal/mempool"
)

// mempoolRegossipInterval controls how often every still-pending local
// transaction gets re-broadcast. This is the generic (role-independent)
// half of "если нода выключилась — то сообщения не пропадали": a
// transaction only ever really "arrives" once it's committed into the
// chain, so as long as it keeps getting rebroadcast, a momentary gossip
// drop, a node restart, or a temporarily-partitioned peer can never
// permanently strand it — it will keep reaching new peers on every pass
// until some validator finally proposes it into a block.
const mempoolRegossipInterval = 30 * time.Second

// runMempoolRegossip periodically re-publishes every transaction this
// node's mempool still considers pending. Safe to run on every node
// (validator or relay): gossipsub already delivers a node's own re-
// Publish back to its own local subscribers exactly once and is
// idempotent everywhere a transaction is handled (Mempool.Add is a
// dedup-by-hash no-op for an already-known tx), so redundant rebroadcasts
// are harmless, just occasionally wasteful bandwidth — an acceptable
// trade for this network's scale.
func runMempoolRegossip(ctx context.Context, pool *mempool.Mempool, gossip func(tx *blockchain.Transaction) error, log *zap.Logger) {
	ticker := time.NewTicker(mempoolRegossipInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		pending := pool.Propose(mempool.MaxPoolSize)
		for _, tx := range pending {
			if err := gossip(tx); err != nil {
				log.Debug("node: mempool re-gossip publish failed (will retry next pass)", zap.Error(err))
			}
		}
	}
}
