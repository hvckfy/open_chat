// Package consensus implements a simplified, PBFT/Tendermint-style BFT
// consensus engine: round-robin proposer rotation plus a two-phase
// PREVOTE/PRECOMMIT vote exchange over gossip. With a validator set of
// size n = 3f+1, the engine only ever commits a block once it collects
// matching votes from a 2f+1 quorum, which is the standard safety
// threshold tolerating up to f byzantine or crashed validators.
//
// This is intentionally simplified relative to production Tendermint: no
// formal view-change certificates, no light-client proofs, no slashing.
// A stalled/byzantine proposer is handled by a round timeout that simply
// rotates to the next validator, which is sufficient to keep the network
// live as long as fewer than f validators are simultaneously faulty.
package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"

	"openchat/internal/blockchain"
	"openchat/internal/mempool"
)

// Proposal is the leader's block proposal for one (height, round).
type Proposal struct {
	Block       *blockchain.Block
	Round       uint32
	ProposerSig []byte
}

func (p *Proposal) signingBytes() []byte {
	buf := append([]byte{}, p.Block.HeaderBytes()...)
	return appendU32(buf, p.Round)
}

type Config struct {
	Validators    []string // hex addresses, static set for this simplified engine
	BlockMaxTxs   int
	RoundTimeout  time.Duration
	EmptyBlockGap time.Duration // min wait before proposing an empty block, avoids busy-looping
}

type Engine struct {
	cfg     Config
	self    Identity
	chain   *blockchain.Chain
	pool    *mempool.Mempool
	network PubSub
	log     *zap.Logger

	quorumSize int

	mu         sync.Mutex
	proposals  map[string]*Proposal        // "height/round" -> proposal
	prevotes   map[string]map[string]*Vote // "height/round" -> voter -> vote
	precommits map[string]map[string]*Vote
}

func NewEngine(cfg Config, self Identity, chain *blockchain.Chain, pool *mempool.Mempool, network PubSub, log *zap.Logger) *Engine {
	validators := append([]string{}, cfg.Validators...)
	sort.Strings(validators) // deterministic ordering across all nodes
	cfg.Validators = validators

	if cfg.RoundTimeout == 0 {
		cfg.RoundTimeout = 4 * time.Second
	}
	if cfg.BlockMaxTxs == 0 {
		cfg.BlockMaxTxs = 500
	}
	if cfg.EmptyBlockGap == 0 {
		cfg.EmptyBlockGap = 1 * time.Second
	}

	return &Engine{
		cfg:        cfg,
		self:       self,
		chain:      chain,
		pool:       pool,
		network:    network,
		log:        log,
		quorumSize: quorum(len(validators)),
		proposals:  make(map[string]*Proposal),
		prevotes:   make(map[string]map[string]*Vote),
		precommits: make(map[string]map[string]*Vote),
	}
}

// QuorumSize exposes 2f+1 for the current static validator set (used by
// metrics and by cmd/node for sanity logging at startup).
func (e *Engine) QuorumSize() int { return e.quorumSize }

// Run drives the consensus loop until ctx is canceled. It should be
// started in its own goroutine by cmd/node.
func (e *Engine) Run(ctx context.Context) {
	go e.readLoop(ctx, TopicPropose, e.handleProposal)
	go e.readLoop(ctx, TopicPrevote, e.handleVote(VotePrevote))
	go e.readLoop(ctx, TopicPrecommit, e.handleVote(VotePrecommit))

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := e.runHeight(ctx, e.chain.Height()+1); err != nil {
			if ctx.Err() != nil {
				return
			}
			e.log.Warn("consensus: height round failed, retrying", zap.Error(err))
		}
	}
}

func (e *Engine) proposerFor(height uint64, round uint32) string {
	n := len(e.cfg.Validators)
	if n == 0 {
		return ""
	}
	idx := int((height + uint64(round)) % uint64(n))
	return e.cfg.Validators[idx]
}

func (e *Engine) runHeight(ctx context.Context, height uint64) error {
	for round := uint32(0); ; round++ {
		roundCtx, cancel := context.WithTimeout(ctx, e.cfg.RoundTimeout)
		committed, err := e.runRound(roundCtx, height, round)
		cancel()
		if committed {
			e.gc(height)
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			e.log.Debug("consensus: round did not commit, advancing view", zap.Uint64("height", height), zap.Uint32("round", round), zap.Error(err))
		}
	}
}

func (e *Engine) runRound(ctx context.Context, height uint64, round uint32) (bool, error) {
	proposer := e.proposerFor(height, round)
	key := roundKey(height, round)

	if proposer == e.self.Address {
		if err := e.propose(ctx, height, round); err != nil {
			return false, fmt.Errorf("propose: %w", err)
		}
	}

	prop, ok := waitFor(ctx, func() (*Proposal, bool) {
		e.mu.Lock()
		defer e.mu.Unlock()
		p, ok := e.proposals[key]
		return p, ok
	})
	if !ok {
		return false, fmt.Errorf("no valid proposal received from %s before timeout", proposer)
	}
	if err := e.validateProposal(prop, proposer, height); err != nil {
		return false, err
	}

	blockHash := prop.Block.Hash()

	prevote := &Vote{Type: VotePrevote, Height: height, Round: round, BlockHash: blockHash, Voter: e.self.Address}
	prevote.Sign(e.self.Priv)
	if err := e.broadcastVote(ctx, TopicPrevote, prevote); err != nil {
		return false, err
	}

	if _, ok := waitFor(ctx, func() (struct{}, bool) {
		return struct{}{}, e.countVotes(e.prevotes, key, blockHash) >= e.quorumSize
	}); !ok {
		return false, fmt.Errorf("prevote quorum not reached")
	}

	precommit := &Vote{Type: VotePrecommit, Height: height, Round: round, BlockHash: blockHash, Voter: e.self.Address}
	precommit.Sign(e.self.Priv)
	if err := e.broadcastVote(ctx, TopicPrecommit, precommit); err != nil {
		return false, err
	}

	if _, ok := waitFor(ctx, func() (struct{}, bool) {
		return struct{}{}, e.countVotes(e.precommits, key, blockHash) >= e.quorumSize
	}); !ok {
		return false, fmt.Errorf("precommit quorum not reached")
	}

	e.mu.Lock()
	sigs := make([]blockchain.ValidatorSig, 0, e.quorumSize)
	for voter, v := range e.precommits[key] {
		if v.BlockHash == blockHash {
			sigs = append(sigs, blockchain.ValidatorSig{Validator: voter, Round: round, Signature: v.Signature})
		}
	}
	e.mu.Unlock()

	prop.Block.CommitSigs = sigs
	if err := e.chain.CommitBlock(prop.Block, e.quorumSize); err != nil {
		return false, fmt.Errorf("commit: %w", err)
	}
	e.pool.Remove(prop.Block.Transactions)
	e.log.Info("consensus: block committed",
		zap.Uint64("height", prop.Block.Height),
		zap.Int("txs", len(prop.Block.Transactions)),
		zap.String("proposer", prop.Block.Proposer),
		zap.Uint32("round", round),
	)
	e.announceCommit(ctx, prop.Block)
	return true, nil
}

// announceCommit best-effort broadcasts a just-committed block on
// TopicBlockCommit, purely so non-voting relay nodes can sync (see the
// topic's doc comment in types.go). It deliberately never fails runRound:
// the block is already durably committed locally by the time this runs,
// and every validator's own next-height propose/vote cycle is completely
// independent of whether this gossip send succeeds.
func (e *Engine) announceCommit(ctx context.Context, b *blockchain.Block) {
	data, err := json.Marshal(b)
	if err != nil {
		e.log.Warn("consensus: marshal committed block for relay-sync gossip failed", zap.Error(err))
		return
	}
	if err := e.network.Publish(ctx, TopicBlockCommit, data); err != nil {
		e.log.Debug("consensus: announce committed block failed (non-fatal, relays will catch up via GetBlocks)", zap.Uint64("height", b.Height), zap.Error(err))
	}
}

func (e *Engine) propose(ctx context.Context, height uint64, round uint32) error {
	head, err := e.chain.Head()
	if err != nil {
		return err
	}
	txs := e.pool.Propose(e.cfg.BlockMaxTxs)
	block := &blockchain.Block{
		Height:       height,
		PrevHash:     head.Hash(),
		Timestamp:    time.Now().UnixMilli(),
		Proposer:     e.self.Address,
		Transactions: txs,
	}
	block.ComputeMerkleRoot()

	prop := &Proposal{Block: block, Round: round}
	prop.ProposerSig = signWith(e.self.Priv, prop.signingBytes())

	data, err := json.Marshal(prop)
	if err != nil {
		return err
	}
	return e.network.Publish(ctx, TopicPropose, data)
}

func (e *Engine) validateProposal(prop *Proposal, expectedProposer string, expectedHeight uint64) error {
	if prop.Block.Proposer != expectedProposer {
		return fmt.Errorf("proposal from unexpected proposer %s (want %s)", prop.Block.Proposer, expectedProposer)
	}
	if prop.Block.Height != expectedHeight {
		return fmt.Errorf("proposal height mismatch: got %d want %d", prop.Block.Height, expectedHeight)
	}
	pub, err := hexDecodePubkey(expectedProposer)
	if err != nil {
		return err
	}
	if !verifyWith(pub, prop.signingBytes(), prop.ProposerSig) {
		return fmt.Errorf("invalid proposer signature")
	}
	for _, tx := range prop.Block.Transactions {
		if err := tx.Verify(); err != nil {
			return fmt.Errorf("proposal contains invalid tx %s: %w", tx.Hash(), err)
		}
	}
	return nil
}

func (e *Engine) broadcastVote(ctx context.Context, topic string, v *Vote) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return e.network.Publish(ctx, topic, data)
}

func (e *Engine) handleProposal(data []byte) {
	var p Proposal
	if err := json.Unmarshal(data, &p); err != nil || p.Block == nil {
		return
	}
	key := roundKey(p.Block.Height, p.Round)
	e.mu.Lock()
	if _, exists := e.proposals[key]; !exists {
		e.proposals[key] = &p
	}
	e.mu.Unlock()
}

func (e *Engine) handleVote(want VoteType) func([]byte) {
	return func(data []byte) {
		var v Vote
		if err := json.Unmarshal(data, &v); err != nil || v.Type != want || !v.Verify() {
			return
		}
		key := roundKey(v.Height, v.Round)
		bucket := e.prevotes
		if want == VotePrecommit {
			bucket = e.precommits
		}
		e.mu.Lock()
		m, ok := bucket[key]
		if !ok {
			m = make(map[string]*Vote)
			bucket[key] = m
		}
		if _, seen := m[v.Voter]; !seen { // ignore equivocating re-votes
			m[v.Voter] = &v
		}
		e.mu.Unlock()
	}
}

func (e *Engine) countVotes(bucket map[string]map[string]*Vote, key, blockHash string) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	m, ok := bucket[key]
	if !ok {
		return 0
	}
	n := 0
	for _, v := range m {
		if v.BlockHash == blockHash {
			n++
		}
	}
	return n
}

func (e *Engine) gc(height uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	prefix := fmt.Sprintf("%d/", height)
	for k := range e.proposals {
		if hasPrefix(k, prefix) {
			delete(e.proposals, k)
		}
	}
	for k := range e.prevotes {
		if hasPrefix(k, prefix) {
			delete(e.prevotes, k)
		}
	}
	for k := range e.precommits {
		if hasPrefix(k, prefix) {
			delete(e.precommits, k)
		}
	}
}

// waitFor polls check() until it returns ok=true or ctx expires. It is a
// free function (not a method) because Go does not support type parameters
// on methods.
func waitFor[T any](ctx context.Context, check func() (T, bool)) (T, bool) {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		if v, ok := check(); ok {
			return v, true
		}
		select {
		case <-ctx.Done():
			var zero T
			return zero, false
		case <-ticker.C:
		}
	}
}

func (e *Engine) readLoop(ctx context.Context, topic string, handle func([]byte)) {
	sub, err := e.network.Subscribe(topic)
	if err != nil {
		e.log.Error("consensus: subscribe failed", zap.String("topic", topic), zap.Error(err))
		return
	}
	defer sub.Cancel()
	for {
		data, err := sub.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		handle(data)
	}
}

func roundKey(height uint64, round uint32) string {
	return fmt.Sprintf("%d/%d", height, round)
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
