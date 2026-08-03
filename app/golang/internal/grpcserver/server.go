// Package grpcserver is the client-facing gateway (Clean Architecture:
// an inbound adapter). It translates gRPC calls into calls against the
// domain layer (internal/blockchain, internal/mempool) and never contains
// consensus or storage logic itself.
package grpcserver

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"time"

	"go.uber.org/zap"

	"openchat/internal/blockchain"
	"openchat/internal/grpcserver/pb"
	"openchat/internal/mempool"
	"openchat/pkg/authproof"
	"openchat/pkg/relayproof"
)

// StreamAuthFreshness bounds how old/new a StreamIncomingSMS ownership
// proof's timestamp may be.
const StreamAuthFreshness = 5 * time.Minute

// RegisterRelayFreshness bounds how old/new a RegisterRelay request's
// timestamp may be, for the same replay-defense reason as
// StreamAuthFreshness above.
const RegisterRelayFreshness = 5 * time.Minute

// GossipFunc broadcasts a locally-accepted transaction to the rest of the
// network (wired to p2p pubsub by cmd/node).
type GossipFunc func(*blockchain.Transaction) error

// AnnounceFunc broadcasts a GatewayInfo update (new/changed node status)
// to the rest of the network over TopicNodeAnnounce (wired to p2p pubsub
// by cmd/node), so every node's Registry converges on the same picture.
type AnnounceFunc func(*pb.GatewayInfo) error

type Server struct {
	pb.UnimplementedNodeGatewayServer

	SelfAddress string
	NodeVersion string
	Role        string // "validator" or "relay" — reported by GetAddress
	PeerCount   func() int

	Chain    *blockchain.Chain
	Pool     *mempool.Mempool
	Gossip   GossipFunc
	Registry *Registry

	// Relay registration ("trusted center" side — see RegisterRelay).
	// ValidatorSet is the real, consensus-voting address set: a
	// registration claiming one of these addresses as the relay's own
	// identity is always rejected, since a genuine validator never needs
	// to (and must never) register as a relay.
	ValidatorSet []string
	// AllowedVersions gates which self-reported NodeVersion strings a
	// relay may register with. Empty means "only my own NodeVersion" —
	// deliberately conservative, since an empty allowlist arriving from
	// misconfiguration should not silently mean "allow anything".
	AllowedVersions []string
	NodeCommit      string
	// ProbeIntervalSeconds/RegistrationTTLSeconds are advertised back to
	// a registering relay purely as guidance (how often the trusted
	// center will probe it / how often it should re-register); they don't
	// change this server's own behaviour.
	ProbeIntervalSeconds   uint32
	RegistrationTTLSeconds uint32
	// AnnounceGossip disseminates an accepted registration (or any other
	// Registry update worth telling the rest of the network about) —
	// nil is fine (registration still succeeds locally), it just means
	// only this node currently knows about the new relay.
	AnnounceGossip AnnounceFunc

	Log *zap.Logger
}

func (s *Server) GetAddress(ctx context.Context, _ *pb.Empty) (*pb.AddressResponse, error) {
	peers := 0
	if s.PeerCount != nil {
		peers = s.PeerCount()
	}
	return &pb.AddressResponse{
		ValidatorAddress: s.SelfAddress,
		NodeVersion:      s.NodeVersion,
		ChainHeight:      s.Chain.Height(),
		PeerCount:        uint32(peers),
		Role:             s.Role,
	}, nil
}

func (s *Server) SendSMS(ctx context.Context, req *pb.SMSRequest) (*pb.SMSResponse, error) {
	tx, err := txFromPB(req)
	if err != nil {
		return &pb.SMSResponse{Accepted: false, Error: err.Error()}, nil
	}

	if err := s.Pool.Add(tx); err != nil {
		s.Log.Debug("grpcserver: rejected tx", zap.String("from", tx.From), zap.Error(err))
		return &pb.SMSResponse{Accepted: false, Error: err.Error()}, nil
	}

	if s.Gossip != nil {
		if err := s.Gossip(tx); err != nil {
			s.Log.Warn("grpcserver: gossip broadcast failed", zap.Error(err))
			// still accepted locally; gossip is best-effort, the tx will
			// also propagate once this node proposes/relays a block.
		}
	}

	return &pb.SMSResponse{Accepted: true, TxHash: tx.Hash()}, nil
}

func (s *Server) StreamIncomingSMS(req *pb.StreamRequest, stream pb.NodeGateway_StreamIncomingSMSServer) error {
	if err := verifyStreamAuth(req); err != nil {
		return fmt.Errorf("grpcserver: stream auth failed: %w", err)
	}

	events, cancel := s.Chain.Subscribe()
	defer cancel()

	ctx := stream.Context()
	s.Log.Info("grpcserver: client subscribed", zap.String("address", req.Address))

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			for _, tx := range ev.Txs {
				if tx.To != req.Address {
					continue
				}
				resp := &pb.SMSResponse{
					Accepted:    true,
					TxHash:      tx.Hash(),
					Message:     pbFromTx(tx),
					BlockHeight: ev.Height,
				}
				if err := stream.Send(resp); err != nil {
					return err
				}
			}
		}
	}
}

func (s *Server) GetNodesDiscovery(ctx context.Context, req *pb.DiscoveryRequest) (*pb.DiscoveryResponse, error) {
	return &pb.DiscoveryResponse{Gateways: s.Registry.List(int(req.MaxResults))}, nil
}

// RegisterRelay is how a community-run relay node joins the network's
// discovery/health system ("проверка что нода поднимает именно то, что
// мы хотим"): it must present a valid ed25519 signature over its claimed
// identity/endpoint/version (proving key ownership without transmitting
// the key), and its self-reported NodeVersion must be on this node's
// allowlist. Acceptance only ever adds the caller to the discovery
// Registry with Role "relay" — it is structurally incapable of granting
// consensus membership, since that lives entirely in
// consensus.Config.Validators, which this package never touches.
func (s *Server) RegisterRelay(ctx context.Context, req *pb.RegisterRelayRequest) (*pb.RegisterRelayResponse, error) {
	// Only a validator ("trusted center") is authoritative here. A relay
	// forwarding/accepting registrations on another relay's behalf would
	// let anyone bootstrap network-wide announce-gossip entries without
	// ever passing a real validator's version-allowlist check — reject
	// outright rather than let that spoofing path exist.
	if s.Role != "validator" {
		return &pb.RegisterRelayResponse{Accepted: false, Reason: "this node is a relay, not a trusted validator — register with a validator gateway instead"}, nil
	}
	if req.RelayGRPCAddress == "" {
		return &pb.RegisterRelayResponse{Accepted: false, Reason: "relay_grpc_address is required"}, nil
	}

	now := time.Now().UnixMilli()
	skew := now - req.Timestamp
	if skew < 0 {
		skew = -skew
	}
	if time.Duration(skew)*time.Millisecond > RegisterRelayFreshness {
		return &pb.RegisterRelayResponse{Accepted: false, Reason: "stale or future-dated timestamp"}, nil
	}

	pubBytes, err := hex.DecodeString(req.RelayAddress)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return &pb.RegisterRelayResponse{Accepted: false, Reason: "invalid relay_address"}, nil
	}
	msg := relayproof.Bytes(req.RelayAddress, req.RelayGRPCAddress, req.NodeVersion, req.NodeCommit, req.Timestamp)
	if !ed25519.Verify(ed25519.PublicKey(pubBytes), msg, req.Signature) {
		return &pb.RegisterRelayResponse{Accepted: false, Reason: "signature does not match relay_address"}, nil
	}

	for _, v := range s.ValidatorSet {
		if v == req.RelayAddress {
			s.Log.Warn("grpcserver: rejected RegisterRelay claiming a real validator address", zap.String("address", req.RelayAddress))
			return &pb.RegisterRelayResponse{Accepted: false, Reason: "this address is a consensus validator, not a relay"}, nil
		}
	}

	if !s.versionAllowed(req.NodeVersion) {
		s.Log.Info("grpcserver: rejected RegisterRelay with disallowed version",
			zap.String("relay", req.RelayAddress), zap.String("version", req.NodeVersion))
		return &pb.RegisterRelayResponse{Accepted: false, Reason: fmt.Sprintf("node_version %q is not on the allowlist", req.NodeVersion)}, nil
	}

	info := &pb.GatewayInfo{
		GRPCAddress:      req.RelayGRPCAddress,
		ValidatorAddress: req.RelayAddress,
		Role:             "relay",
		NodeVersion:      req.NodeVersion,
		Healthy:          true, // optimistic until the first probe says otherwise
	}
	s.Registry.Upsert(info)
	s.Log.Info("grpcserver: relay registered", zap.String("relay", req.RelayAddress), zap.String("addr", req.RelayGRPCAddress), zap.String("version", req.NodeVersion))

	if s.AnnounceGossip != nil {
		if err := s.AnnounceGossip(info); err != nil {
			s.Log.Warn("grpcserver: announcing new relay to the network failed (still registered locally)", zap.Error(err))
		}
	}

	return &pb.RegisterRelayResponse{
		Accepted:               true,
		ProbeIntervalSeconds:   s.ProbeIntervalSeconds,
		RegistrationTTLSeconds: s.RegistrationTTLSeconds,
	}, nil
}

func (s *Server) versionAllowed(version string) bool {
	if len(s.AllowedVersions) == 0 {
		return version == s.NodeVersion
	}
	for _, v := range s.AllowedVersions {
		if v == version {
			return true
		}
	}
	return false
}

// maxGetBlocksBatch caps how many blocks a single GetBlocks call returns,
// regardless of what the caller asked for, so a misbehaving/careless
// caller can't force this node to serialize its whole chain in one RPC.
const maxGetBlocksBatch = 500

// GetBlocks returns a contiguous run of committed blocks starting at
// FromHeight, for relay-node chain sync/backfill (see TopicBlockCommit's
// doc comment in internal/consensus/types.go for the real-time half of
// that story — this RPC is the catch-up half, covering cold starts and
// gaps longer than gossip retains).
func (s *Server) GetBlocks(ctx context.Context, req *pb.GetBlocksRequest) (*pb.GetBlocksResponse, error) {
	limit := int(req.MaxBlocks)
	if limit <= 0 || limit > maxGetBlocksBatch {
		limit = maxGetBlocksBatch
	}

	blocks := make([]*pb.BlockPB, 0, limit)
	for h := req.FromHeight; len(blocks) < limit; h++ {
		b, found, err := s.Chain.GetBlock(h)
		if err != nil {
			return nil, fmt.Errorf("grpcserver: get block %d: %w", h, err)
		}
		if !found {
			break
		}
		blocks = append(blocks, BlockToPB(b))
	}
	return &pb.GetBlocksResponse{Blocks: blocks, ChainHeight: s.Chain.Height()}, nil
}

func verifyStreamAuth(req *pb.StreamRequest) error {
	now := time.Now().UnixMilli()
	skew := now - req.Timestamp
	if skew < 0 {
		skew = -skew
	}
	if time.Duration(skew)*time.Millisecond > StreamAuthFreshness {
		return fmt.Errorf("stale auth timestamp")
	}
	pubBytes, err := hex.DecodeString(req.Address)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid address")
	}
	if !ed25519.Verify(ed25519.PublicKey(pubBytes), authproof.Bytes(req.Address, req.Timestamp), req.Signature) {
		return fmt.Errorf("signature does not match address")
	}
	return nil
}

var _ pb.NodeGatewayServer = (*Server)(nil)
