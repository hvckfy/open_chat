// Package p2p is the networking adapter: it wires go-libp2p (host, Noise/
// TLS1.3 transport security, NAT traversal) and Kademlia DHT peer
// discovery, and exposes a small consensus.PubSub-shaped API so the
// consensus and mempool-gossip layers stay libp2p-agnostic.
package p2p

import (
	"context"
	stded25519 "crypto/ed25519"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	noise "github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	ma "github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
)

// Rendezvous is the DHT advertise/discover tag identifying the OpenChat
// validator network (distinct networks/chain-ids should use distinct tags).
const Rendezvous = "openchat-network/v1"

type Config struct {
	ListenAddrs    []string // multiaddrs, e.g. "/ip4/0.0.0.0/tcp/4001"
	BootstrapPeers []string // multiaddrs of known peers to dial at startup
	PrivKeyBytes   []byte   // 32-byte Ed25519 seed OR 64-byte (seed||pub) key, reused as the libp2p node identity
}

// Node bundles the libp2p host, Kademlia DHT and gossipsub router.
type Node struct {
	Host   host.Host
	DHT    *dht.IpfsDHT
	pubsub *pubsub.PubSub
	log    *zap.Logger

	topics map[string]*pubsub.Topic
}

func New(ctx context.Context, cfg Config, log *zap.Logger) (*Node, error) {
	priv, err := crypto.UnmarshalEd25519PrivateKey(expandSeed(cfg.PrivKeyBytes))
	if err != nil {
		return nil, fmt.Errorf("p2p: derive libp2p identity: %w", err)
	}

	listenAddrs := make([]ma.Multiaddr, 0, len(cfg.ListenAddrs))
	for _, a := range cfg.ListenAddrs {
		maddr, err := ma.NewMultiaddr(a)
		if err != nil {
			return nil, fmt.Errorf("p2p: bad listen addr %q: %w", a, err)
		}
		listenAddrs = append(listenAddrs, maddr)
	}

	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrs(listenAddrs...),
		// Traffic encryption: both Noise and TLS 1.3 are offered; peers
		// negotiate whichever they have in common (spec: "Noise/TLS 1.3").
		libp2p.Security(noise.ID, noise.New),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		// NAT traversal: UPnP/NAT-PMP port mapping + hole punching + the
		// AutoNAT/identify-based NAT service so peers behind NAT can still
		// be dialed.
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelay(),
	)
	if err != nil {
		return nil, fmt.Errorf("p2p: create libp2p host: %w", err)
	}

	kad, err := dht.New(ctx, h, dht.Mode(dht.ModeServer))
	if err != nil {
		return nil, fmt.Errorf("p2p: create kademlia dht: %w", err)
	}
	if err := kad.Bootstrap(ctx); err != nil {
		return nil, fmt.Errorf("p2p: dht bootstrap: %w", err)
	}

	for _, addr := range cfg.BootstrapPeers {
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			log.Warn("p2p: skipping invalid bootstrap addr", zap.String("addr", addr), zap.Error(err))
			continue
		}
		info, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			log.Warn("p2p: skipping unparsable bootstrap addr", zap.String("addr", addr), zap.Error(err))
			continue
		}
		go func(info peer.AddrInfo) {
			dialCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()
			if err := h.Connect(dialCtx, info); err != nil {
				log.Warn("p2p: bootstrap peer dial failed", zap.String("peer", info.ID.String()), zap.Error(err))
			}
		}(*info)
	}

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("p2p: create gossipsub: %w", err)
	}

	n := &Node{Host: h, DHT: kad, pubsub: ps, log: log, topics: make(map[string]*pubsub.Topic)}

	routingDiscovery := routing.NewRoutingDiscovery(kad)
	dutil.Advertise(ctx, routingDiscovery, Rendezvous)
	go n.findPeersLoop(ctx, routingDiscovery)

	return n, nil
}

// findPeersLoop periodically re-queries the DHT for other rendezvous'd
// peers and dials any not already connected. This is how a freshly
// started node, given only a couple of bootstrap addresses, discovers the
// rest of the validator set (Kademlia DHT peer discovery).
func (n *Node) findPeersLoop(ctx context.Context, disc *routing.RoutingDiscovery) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		peersCh, err := disc.FindPeers(ctx, Rendezvous)
		if err != nil {
			n.log.Warn("p2p: dht find peers failed", zap.Error(err))
		} else {
			for p := range peersCh {
				if p.ID == n.Host.ID() || len(p.Addrs) == 0 {
					continue
				}
				go func(pi peer.AddrInfo) {
					dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
					defer cancel()
					if n.Host.Network().Connectedness(pi.ID) != network.Connected {
						_ = n.Host.Connect(dialCtx, pi)
					}
				}(p)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// PeerCount reports the number of currently connected peers (exported for
// Prometheus metrics and the GetAddress gRPC method).
func (n *Node) PeerCount() int {
	return len(n.Host.Network().Peers())
}

func (n *Node) Close() error {
	if err := n.DHT.Close(); err != nil {
		return err
	}
	return n.Host.Close()
}

// expandSeed turns a 32-byte Ed25519 seed into the 64-byte
// (seed||pubkey) form both crypto/ed25519 and libp2p's crypto package
// expect as "private key bytes".
func expandSeed(seed []byte) []byte {
	if len(seed) == stded25519.PrivateKeySize {
		return seed
	}
	return stded25519.NewKeyFromSeed(seed)
}

// IdentityFromSeed derives the libp2p private key a node built from this
// seed will use — exported so cmd/keytool can print the resulting peer ID
// without duplicating the seed-expansion logic here.
func IdentityFromSeed(seed []byte) (crypto.PrivKey, error) {
	return crypto.UnmarshalEd25519PrivateKey(expandSeed(seed))
}

// PeerIDFromSeed derives the libp2p peer ID a node started with this seed
// will advertise — this is what goes in a bootstrap multiaddr's trailing
// `/p2p/<peer-id>` component.
func PeerIDFromSeed(seed []byte) (peer.ID, error) {
	priv, err := IdentityFromSeed(seed)
	if err != nil {
		return "", err
	}
	return peer.IDFromPrivateKey(priv)
}
