// This file wires the "permissionless relay node" feature into the
// composition root: chain sync for relay nodes (which never run
// consensus), registration + version verification against a trusted
// validator, network-wide status dissemination (TopicNodeAnnounce), and
// the validator-side functional health-check that actually sends itself
// a message through each registered relay and confirms it arrives.
//
// Design recap (see internal/consensus/types.go's TopicBlockCommit /
// TopicNodeAnnounce doc comments for the wire-level half):
//
//   - A relay node runs libp2p + the gRPC gateway + chain sync, but NEVER
//     the consensus engine, and its address must never appear in
//     VALIDATOR_SET (enforced both here and, defensively, again by
//     config.LoadNodeConfig and grpcserver.Server.RegisterRelay).
//   - Only validators ever publish to TopicNodeAnnounce (whether
//     announcing themselves or a relay they just accepted); every node
//     subscribes and verifies the announcer's signature against its own
//     locally-configured VALIDATOR_SET before trusting an update. This
//     mirrors how tx/vote/proposal gossip in this codebase is already
//     trusted purely by per-message signature, never by transport.
//   - A relay registers (and periodically re-registers) with one or more
//     validators via the signed RegisterRelay RPC; the validator checks
//     the relay's self-reported NodeVersion against an allowlist before
//     accepting it into its discovery Registry.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"openchat/internal/blockchain"
	"openchat/internal/config"
	"openchat/internal/consensus"
	"openchat/internal/grpcserver"
	"openchat/internal/grpcserver/pb"
	"openchat/internal/mempool"
	"openchat/internal/p2p"
	"openchat/pkg/authproof"
	"openchat/pkg/relayproof"
)

// ---------------------------------------------------------------------
// TopicNodeAnnounce: signed dissemination of GatewayInfo updates.
// ---------------------------------------------------------------------

// nodeAnnounce is the wire envelope published on TopicNodeAnnounce.
// InfoJSON is kept as raw bytes (rather than a nested struct) so the
// exact bytes a validator signs are the exact bytes every receiver
// verifies against — no risk of a re-marshal producing different bytes.
type nodeAnnounce struct {
	InfoJSON    []byte `json:"info_json"`
	AnnouncedBy string `json:"announced_by"` // hex address of the validator vouching for this info
	Timestamp   int64  `json:"timestamp"`
	Signature   []byte `json:"signature"`
}

func (a *nodeAnnounce) signingBytes() []byte {
	buf := append([]byte{}, a.InfoJSON...)
	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], uint64(a.Timestamp))
	return append(buf, ts[:]...)
}

// newAnnounceFunc returns a grpcserver.AnnounceFunc that signs info with
// this validator's own identity and publishes it on TopicNodeAnnounce.
// Only ever wired into a validator-role Server — see cmd/node's run().
func newAnnounceFunc(ctx context.Context, node *p2p.Node, self consensus.Identity, log *zap.Logger) grpcserver.AnnounceFunc {
	return func(info *pb.GatewayInfo) error {
		infoJSON, err := json.Marshal(info)
		if err != nil {
			return fmt.Errorf("relay: marshal announce info: %w", err)
		}
		ann := &nodeAnnounce{
			InfoJSON:    infoJSON,
			AnnouncedBy: self.Address,
			Timestamp:   time.Now().UnixMilli(),
		}
		ann.Signature = ed25519.Sign(self.Priv, ann.signingBytes())
		data, err := json.Marshal(ann)
		if err != nil {
			return fmt.Errorf("relay: marshal announce envelope: %w", err)
		}
		return node.Publish(ctx, consensus.TopicNodeAnnounce, data)
	}
}

// runAnnounceListener subscribes to TopicNodeAnnounce and merges every
// validly-signed update into registry. Run on every node, regardless of
// role — this is how relay nodes (and other validators) learn about newly
// accepted relays and about validators' own self-announced liveness.
func runAnnounceListener(ctx context.Context, node *p2p.Node, registry *grpcserver.Registry, validatorSet []string, log *zap.Logger) {
	sub, err := node.Subscribe(consensus.TopicNodeAnnounce)
	if err != nil {
		log.Error("relay: subscribe TopicNodeAnnounce failed", zap.Error(err))
		return
	}
	defer sub.Cancel()

	validators := make(map[string]bool, len(validatorSet))
	for _, v := range validatorSet {
		validators[v] = true
	}

	for {
		data, err := sub.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		var ann nodeAnnounce
		if err := json.Unmarshal(data, &ann); err != nil {
			continue
		}
		// Only ever trust an announcement vouched for by a real,
		// locally-known validator — never by an arbitrary relay or
		// unknown address, regardless of whether the signature itself
		// checks out.
		if !validators[ann.AnnouncedBy] {
			continue
		}
		pub, err := hex.DecodeString(ann.AnnouncedBy)
		if err != nil || len(pub) != ed25519.PublicKeySize {
			continue
		}
		if !ed25519.Verify(ed25519.PublicKey(pub), ann.signingBytes(), ann.Signature) {
			log.Warn("relay: dropped announce with invalid signature", zap.String("announced_by", ann.AnnouncedBy))
			continue
		}
		var info pb.GatewayInfo
		if err := json.Unmarshal(ann.InfoJSON, &info); err != nil {
			continue
		}
		registry.Upsert(&info)
	}
}

// runSelfAnnounceHeartbeat periodically re-announces this validator's own
// GatewayInfo, so its liveness/version stays fresh network-wide even when
// no relay has recently (re-)registered. Validator-role only.
func runSelfAnnounceHeartbeat(ctx context.Context, cfg *config.NodeConfig, announce grpcserver.AnnounceFunc, registry *grpcserver.Registry, interval time.Duration, log *zap.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		info := &pb.GatewayInfo{
			GRPCAddress:      cfg.ExternalGRPCAddr,
			ValidatorAddress: cfg.ValidatorAddr,
			Role:             "validator",
			NodeVersion:      cfg.NodeVersion,
			Healthy:          true,
		}
		registry.Upsert(info)
		if err := announce(info); err != nil {
			log.Debug("relay: self-announce heartbeat failed (non-fatal)", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// ---------------------------------------------------------------------
// Outbound dialing (used by the relay-side registrar/sync and the
// validator-side health-check prober — both dial gateways as an ordinary
// gRPC *client*, exactly like a real end-user client would).
// ---------------------------------------------------------------------

func dialGateway(ctx context.Context, addr string, insecureTLS bool) (*grpc.ClientConn, error) {
	var creds credentials.TransportCredentials
	if insecureTLS {
		creds = insecure.NewCredentials()
	} else {
		creds = credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})
	}
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return grpc.DialContext(dialCtx, addr, grpc.WithTransportCredentials(creds), grpc.WithBlock())
}

// ---------------------------------------------------------------------
// Relay-role: registration with trusted validators + chain sync.
// ---------------------------------------------------------------------

// runRelayRegistrar registers (and periodically re-registers, well before
// each registration's advertised TTL expires) this relay with every
// validator gateway in cfg.RelayRegisterWith. Relay-role only.
func runRelayRegistrar(ctx context.Context, cfg *config.NodeConfig, log *zap.Logger) {
	if len(cfg.RelayRegisterWith) == 0 {
		log.Warn("relay: RELAY_REGISTER_WITH is empty — this relay will never be discoverable by clients or health-checked by any validator")
		return
	}

	interval := cfg.RelayRegistrationTTL / 2
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	register := func() {
		for _, addr := range cfg.RelayRegisterWith {
			if err := registerWith(ctx, cfg, addr, log); err != nil {
				log.Warn("relay: registration attempt failed", zap.String("validator", addr), zap.Error(err))
			}
		}
	}

	register()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			register()
		}
	}
}

func registerWith(ctx context.Context, cfg *config.NodeConfig, validatorAddr string, log *zap.Logger) error {
	conn, err := dialGateway(ctx, validatorAddr, cfg.RelayProbeInsecureTLS)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	now := time.Now().UnixMilli()
	sig := ed25519.Sign(cfg.ValidatorPriv, relayproof.Bytes(cfg.ValidatorAddr, cfg.ExternalGRPCAddr, cfg.NodeVersion, cfg.NodeCommit, now))

	rpcCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	resp, err := pb.NewNodeGatewayClient(conn).RegisterRelay(rpcCtx, &pb.RegisterRelayRequest{
		RelayAddress:     cfg.ValidatorAddr,
		RelayGRPCAddress: cfg.ExternalGRPCAddr,
		NodeVersion:      cfg.NodeVersion,
		NodeCommit:       cfg.NodeCommit,
		Timestamp:        now,
		Signature:        sig,
	})
	if err != nil {
		return fmt.Errorf("RegisterRelay rpc: %w", err)
	}
	if !resp.Accepted {
		return fmt.Errorf("rejected: %s", resp.Reason)
	}
	log.Info("relay: registered with validator", zap.String("validator", validatorAddr))
	return nil
}

// runRelaySync keeps a relay node's local chain up to date without ever
// running consensus: it ingests blocks broadcast in real time on
// TopicBlockCommit, and separately runs a periodic backfill against
// cfg.RelayRegisterWith to close any gap (cold start, or an outage longer
// than gossip retains). Relay-role only.
func runRelaySync(ctx context.Context, cfg *config.NodeConfig, chain *blockchain.Chain, pool *mempool.Mempool, node *p2p.Node, log *zap.Logger) {
	quorum := consensus.Quorum(len(cfg.ValidatorSet))
	go runBlockCommitListener(ctx, chain, pool, node, quorum, log)
	runBackfillLoop(ctx, cfg, chain, pool, quorum, log)
}

func runBlockCommitListener(ctx context.Context, chain *blockchain.Chain, pool *mempool.Mempool, node *p2p.Node, quorum int, log *zap.Logger) {
	sub, err := node.Subscribe(consensus.TopicBlockCommit)
	if err != nil {
		log.Error("relay: subscribe TopicBlockCommit failed", zap.Error(err))
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
		var b blockchain.Block
		if err := json.Unmarshal(data, &b); err != nil {
			continue
		}
		if err := chain.CommitBlock(&b, quorum); err != nil {
			// Most commonly just "already applied" or "out of order,
			// waiting for backfill" — both benign and self-healing.
			log.Debug("relay: gossip block not applied (will backfill if needed)", zap.Uint64("height", b.Height), zap.Error(err))
			continue
		}
		pool.Remove(b.Transactions)
	}
}

func runBackfillLoop(ctx context.Context, cfg *config.NodeConfig, chain *blockchain.Chain, pool *mempool.Mempool, quorum int, log *zap.Logger) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		backfillOnce(ctx, cfg, chain, pool, quorum, log)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func backfillOnce(ctx context.Context, cfg *config.NodeConfig, chain *blockchain.Chain, pool *mempool.Mempool, quorum int, log *zap.Logger) {
	for _, addr := range cfg.RelayRegisterWith {
		conn, err := dialGateway(ctx, addr, cfg.RelayProbeInsecureTLS)
		if err != nil {
			log.Debug("relay: backfill dial failed", zap.String("validator", addr), zap.Error(err))
			continue
		}
		gw := pb.NewNodeGatewayClient(conn)

		for {
			from := chain.Height() + 1
			rpcCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			resp, err := gw.GetBlocks(rpcCtx, &pb.GetBlocksRequest{FromHeight: from, MaxBlocks: 200})
			cancel()
			if err != nil {
				log.Debug("relay: backfill GetBlocks failed", zap.String("validator", addr), zap.Error(err))
				break
			}
			if len(resp.Blocks) == 0 {
				break // fully caught up against this validator
			}
			for _, bp := range resp.Blocks {
				b, err := grpcserver.BlockFromPB(bp)
				if err != nil {
					log.Warn("relay: backfill block decode failed", zap.Error(err))
					conn.Close()
					return
				}
				if err := chain.CommitBlock(b, quorum); err != nil {
					log.Warn("relay: backfill block rejected", zap.Uint64("height", b.Height), zap.Error(err))
					conn.Close()
					return
				}
				pool.Remove(b.Transactions)
			}
		}
		conn.Close()
	}
}

// ---------------------------------------------------------------------
// Validator-role: the functional relay health-check.
//
// "доверенный центр попробовал сам себе сообщение отправить и
// подтвердил, что сообщение дошло" — this validator connects to a
// registered relay exactly as an ordinary end-user client would, submits
// a self-addressed probe message through it (SendSMS), and confirms that
// message actually arrives back out through that SAME relay's
// StreamIncomingSMS — proving the relay's entire inbound+outbound path
// works, not just that its TCP port is open.
// ---------------------------------------------------------------------

// runRelayHealthProber periodically probes every relay currently known to
// registry and records the result. Validator-role only.
func runRelayHealthProber(ctx context.Context, cfg *config.NodeConfig, registry *grpcserver.Registry, announce grpcserver.AnnounceFunc, log *zap.Logger) {
	ticker := time.NewTicker(cfg.RelayProbeInterval)
	defer ticker.Stop()

	failures := make(map[string]int)

	for {
		for _, relay := range registry.AllRelays() {
			addr := relay.GRPCAddress
			probeCtx, cancel := context.WithTimeout(ctx, cfg.RelayProbeTimeout)
			err := probeRelay(probeCtx, cfg, addr, log)
			cancel()

			healthy := err == nil
			if !healthy {
				failures[addr]++
				log.Warn("relay: health probe failed", zap.String("relay", addr), zap.Int("consecutive_failures", failures[addr]), zap.Error(err))
				if failures[addr] < cfg.RelayMaxFailures {
					// Not yet over the threshold: keep reporting the
					// last-known-good status rather than flapping.
					continue
				}
			} else {
				if failures[addr] > 0 {
					log.Info("relay: health probe recovered", zap.String("relay", addr))
				}
				failures[addr] = 0
			}

			registry.SetHealth(addr, healthy, time.Now().Unix())
			if updated, ok := registry.Get(addr); ok && announce != nil {
				if err := announce(updated); err != nil {
					log.Debug("relay: announcing updated relay health failed (non-fatal)", zap.Error(err))
				}
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func probeRelay(ctx context.Context, cfg *config.NodeConfig, relayAddr string, log *zap.Logger) error {
	conn, err := dialGateway(ctx, relayAddr, cfg.RelayProbeInsecureTLS)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	gw := pb.NewNodeGatewayClient(conn)

	// Build and sign a harmless self-addressed probe transaction exactly
	// as a real client would (see blockchain.Transaction.SigningBytes) —
	// the "ciphertext" here is opaque probe filler, never decrypted by
	// anyone, since nothing about this codepath needs plaintext, only a
	// tx_hash that round-trips.
	tx := &blockchain.Transaction{
		From:      cfg.ValidatorAddr,
		To:        cfg.ValidatorAddr,
		Ciphertext: []byte("openchat-relay-health-probe"),
		Nonce:     uint64(time.Now().UnixNano()),
		Timestamp: time.Now().UnixMilli(),
	}
	tx.Sign(cfg.ValidatorPriv)

	sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	sendResp, err := gw.SendSMS(sendCtx, &pb.SMSRequest{
		FromAddress:     tx.From,
		ToAddress:       tx.To,
		Ciphertext:      tx.Ciphertext,
		NonceAEAD:       make([]byte, 12),
		EphemeralPubkey: make([]byte, 32),
		Nonce:           tx.Nonce,
		Timestamp:       tx.Timestamp,
		Signature:       tx.Signature,
	})
	cancel()
	if err != nil {
		return fmt.Errorf("SendSMS: %w", err)
	}
	if !sendResp.Accepted {
		return fmt.Errorf("relay rejected probe message: %s", sendResp.Error)
	}

	now := time.Now().UnixMilli()
	authSig := ed25519.Sign(cfg.ValidatorPriv, authproof.Bytes(cfg.ValidatorAddr, now))
	stream, err := gw.StreamIncomingSMS(ctx, &pb.StreamRequest{
		Address:   cfg.ValidatorAddr,
		Timestamp: now,
		Signature: authSig,
	})
	if err != nil {
		return fmt.Errorf("StreamIncomingSMS: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("waiting for probe delivery: %w", err)
		}
		if resp.TxHash == sendResp.TxHash {
			return nil
		}
		// Some other message addressed to this validator arrived first
		// (unlikely for a validator identity, but not impossible) — keep
		// waiting until ctx (bounded by RelayProbeTimeout) expires.
	}
}
