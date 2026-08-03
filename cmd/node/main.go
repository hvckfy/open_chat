// Command node runs one OpenChat validator: a libp2p network participant,
// a BFT consensus voter, a BadgerDB-backed chain, and a TLS gRPC gateway
// for clients. This file is pure wiring (Clean Architecture's outermost
// "composition root") — no business logic lives here.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"openchat/internal/blockchain"
	"openchat/internal/config"
	"openchat/internal/consensus"
	"openchat/internal/grpcserver"
	"openchat/internal/grpcserver/pb"
	"openchat/internal/logger"
	"openchat/internal/mempool"
	"openchat/internal/metrics"
	"openchat/internal/p2p"
	"openchat/internal/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "node: fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadNodeConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log, err := logger.New(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer log.Sync()

	log.Info("node: starting",
		zap.String("validator_address", cfg.ValidatorAddr),
		zap.Int("validator_set_size", len(cfg.ValidatorSet)),
		zap.String("version", cfg.NodeVersion),
		zap.String("role", cfg.NodeRole),
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- Storage -----------------------------------------------------
	store, err := storage.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}
	defer store.Close()

	chain, err := blockchain.NewChain(store)
	if err != nil {
		return fmt.Errorf("init chain: %w", err)
	}

	pool := mempool.New(store)

	// --- P2P (libp2p + Kademlia DHT + gossipsub) ----------------------
	p2pNode, err := p2p.New(ctx, p2p.Config{
		ListenAddrs:    cfg.P2PListenAddrs,
		BootstrapPeers: cfg.P2PBootstrapPeers,
		PrivKeyBytes:   []byte(cfg.ValidatorPriv),
	}, log)
	if err != nil {
		return fmt.Errorf("init p2p: %w", err)
	}
	defer p2pNode.Close()

	log.Info("node: libp2p host up", zap.String("peer_id", p2pNode.Host.ID().String()))

	// Relay locally-accepted transactions to the network, and admit
	// transactions gossiped in from peers into our own mempool.
	gossipTx := func(tx *blockchain.Transaction) error {
		data, err := json.Marshal(tx)
		if err != nil {
			return err
		}
		return p2pNode.Publish(ctx, consensus.TopicTxGossip, data)
	}
	go relayIncomingTxGossip(ctx, p2pNode, pool, log)
	go runMempoolRegossip(ctx, pool, gossipTx, log)

	identity := consensus.Identity{Address: cfg.ValidatorAddr, Priv: cfg.ValidatorPriv}

	// --- Discovery/status (network-wide, both roles) ---------------------
	registry := grpcserver.NewRegistry(seedGateways(cfg)...)
	go runAnnounceListener(ctx, p2pNode, registry, cfg.ValidatorSet, log)

	// --- Consensus (validator role) / chain sync + registration (relay
	// role) — mutually exclusive: a relay NEVER runs the consensus engine
	// and can therefore never propose or vote, no matter what; a
	// validator never needs relay chain-sync since it already gets every
	// block firsthand by voting on it. ----------------------------------
	var announce grpcserver.AnnounceFunc
	switch cfg.NodeRole {
	case "validator":
		engine := consensus.NewEngine(consensus.Config{
			Validators:   cfg.ValidatorSet,
			BlockMaxTxs:  cfg.BlockMaxTxs,
			RoundTimeout: cfg.RoundTimeout,
		}, identity, chain, pool, p2pNode, log)

		log.Info("node: consensus configured", zap.Int("quorum_2f+1", engine.QuorumSize()))
		go engine.Run(ctx)

		announce = newAnnounceFunc(ctx, p2pNode, identity, log)
		go runSelfAnnounceHeartbeat(ctx, cfg, announce, registry, cfg.RelayProbeInterval, log)
		go runRelayHealthProber(ctx, cfg, registry, announce, log)

	case "relay":
		log.Info("node: running as a relay — never proposes or votes; relaying + syncing only",
			zap.Strings("register_with", cfg.RelayRegisterWith))
		go runRelaySync(ctx, cfg, chain, pool, p2pNode, log)
		go runRelayRegistrar(ctx, cfg, log)
	}

	// --- Metrics --------------------------------------------------------
	metrics.StartTPSSampler(ctx, 10*time.Second)
	go sampleGauges(ctx, chain, pool, p2pNode)
	go recordCommits(ctx, chain, log)

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", metrics.Handler())
	metricsSrv := &http.Server{Addr: cfg.MetricsListenAddr, Handler: metricsMux}
	go func() {
		log.Info("node: metrics server listening", zap.String("addr", cfg.MetricsListenAddr))
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("node: metrics server error", zap.Error(err))
		}
	}()

	// --- gRPC gateway ----------------------------------------------------
	gwServer := &grpcserver.Server{
		SelfAddress: cfg.ValidatorAddr,
		NodeVersion: cfg.NodeVersion,
		Role:        cfg.NodeRole,
		PeerCount:   p2pNode.PeerCount,
		Chain:       chain,
		Pool:        pool,
		Gossip:      gossipTx,
		Registry:    registry,

		ValidatorSet:           cfg.ValidatorSet,
		AllowedVersions:        cfg.RelayAllowedVersions,
		NodeCommit:             cfg.NodeCommit,
		ProbeIntervalSeconds:   uint32(cfg.RelayProbeInterval.Seconds()),
		RegistrationTTLSeconds: uint32(cfg.RelayRegistrationTTL.Seconds()),
		AnnounceGossip:         announce,

		Log: log,
	}

	var grpcSrv *grpc.Server
	if cfg.GRPCTLSEnabled {
		tlsCreds, err := grpcserver.ServerTLS(cfg.GRPCTLSCert, cfg.GRPCTLSKey)
		if err != nil {
			return fmt.Errorf("load gRPC TLS credentials: %w", err)
		}
		grpcSrv = grpc.NewServer(grpc.Creds(tlsCreds))
	} else {
		// GRPC_TLS_ENABLED=false: this node expects to sit behind a
		// trusted reverse proxy (e.g. nginx with a real Let's Encrypt
		// cert) that terminates TLS and forwards plaintext HTTP/2 via
		// grpc_pass on an isolated/private network. Never set this on a
		// node whose gRPC port is reachable directly from untrusted
		// networks.
		log.Warn("node: GRPC_TLS_ENABLED=false — gateway is serving PLAINTEXT gRPC, only safe behind a trusted TLS-terminating proxy")
		grpcSrv = grpc.NewServer()
	}
	pb.RegisterNodeGatewayServer(grpcSrv, gwServer)

	lis, err := net.Listen("tcp", cfg.GRPCListenAddr)
	if err != nil {
		return fmt.Errorf("listen gRPC: %w", err)
	}
	go func() {
		log.Info("node: gRPC gateway listening", zap.String("addr", cfg.GRPCListenAddr))
		if err := grpcSrv.Serve(lis); err != nil {
			log.Error("node: gRPC server stopped", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("node: shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	grpcSrv.GracefulStop()
	_ = metricsSrv.Shutdown(shutdownCtx)

	return nil
}

func seedGateways(cfg *config.NodeConfig) []*pb.GatewayInfo {
	out := []*pb.GatewayInfo{{
		GRPCAddress:      cfg.ExternalGRPCAddr,
		ValidatorAddress: cfg.ValidatorAddr,
		Role:             cfg.NodeRole,
		NodeVersion:      cfg.NodeVersion,
		Healthy:          true,
	}}
	for _, addr := range cfg.GatewayPeers {
		out = append(out, &pb.GatewayInfo{GRPCAddress: addr})
	}
	return out
}

func relayIncomingTxGossip(ctx context.Context, node *p2p.Node, pool *mempool.Mempool, log *zap.Logger) {
	sub, err := node.Subscribe(consensus.TopicTxGossip)
	if err != nil {
		log.Error("node: subscribe tx gossip failed", zap.Error(err))
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
		var tx blockchain.Transaction
		if err := json.Unmarshal(data, &tx); err != nil {
			continue
		}
		if err := pool.Add(&tx); err != nil {
			log.Debug("node: rejected gossiped tx", zap.Error(err))
		}
	}
}

func sampleGauges(ctx context.Context, chain *blockchain.Chain, pool *mempool.Mempool, node *p2p.Node) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics.ChainHeight.Set(float64(chain.Height()))
			metrics.MempoolSize.Set(float64(pool.Size()))
			metrics.PeerCount.Set(float64(node.PeerCount()))
		}
	}
}

func recordCommits(ctx context.Context, chain *blockchain.Chain, log *zap.Logger) {
	events, cancel := chain.Subscribe()
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			metrics.RecordCommit(len(ev.Txs))
			log.Debug("node: committed", zap.Uint64("height", ev.Height), zap.Int("txs", len(ev.Txs)))
		}
	}
}
