package grpcserver

import (
	"sync"
	"time"

	"openchat/internal/grpcserver/pb"
)

// Registry tracks RPC gateways this node currently believes are live, so
// it can answer GetNodesDiscovery. Entries come from this node's own
// static config (seed gateways) plus, in a fuller deployment, gossip/DHT
// records refreshed by peers announcing themselves — the interface here
// deliberately doesn't care where entries came from.
type Registry struct {
	mu       sync.RWMutex
	gateways map[string]*pb.GatewayInfo // keyed by grpc_address
}

func NewRegistry(seed ...*pb.GatewayInfo) *Registry {
	r := &Registry{gateways: make(map[string]*pb.GatewayInfo)}
	for _, g := range seed {
		r.Upsert(g)
	}
	return r
}

func (r *Registry) Upsert(info *pb.GatewayInfo) {
	if info == nil || info.GRPCAddress == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	info.LastSeenUnix = time.Now().Unix()
	r.gateways[info.GRPCAddress] = info
}

// Get looks up a single known gateway by its grpc_address.
func (r *Registry) Get(grpcAddress string) (*pb.GatewayInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g, ok := r.gateways[grpcAddress]
	return g, ok
}

// SetHealth records the outcome of a functional health-check probe
// (see cmd/node's relay health-check job) against an already-known
// gateway, without disturbing its other fields (Role, NodeVersion, ...).
// A no-op if grpcAddress isn't registered yet — the probe job only ever
// probes entries it already pulled from AllRelays.
func (r *Registry) SetHealth(grpcAddress string, healthy bool, probedAtUnix int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	g, ok := r.gateways[grpcAddress]
	if !ok {
		return
	}
	updated := *g
	updated.Healthy = healthy
	updated.LastProbeUnix = probedAtUnix
	r.gateways[grpcAddress] = &updated
}

// AllRelays returns every currently-known gateway whose Role is "relay" —
// used by the validator-side health-check job to know who to probe, and
// by the announce-gossip disseminator to decide what's worth
// re-broadcasting.
func (r *Registry) AllRelays() []*pb.GatewayInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*pb.GatewayInfo, 0, len(r.gateways))
	for _, g := range r.gateways {
		if g.Role == "relay" {
			out = append(out, g)
		}
	}
	return out
}

// List returns up to max known gateways (0/negative -> a sane default).
func (r *Registry) List(max int) []*pb.GatewayInfo {
	if max <= 0 {
		max = 20
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*pb.GatewayInfo, 0, len(r.gateways))
	for _, g := range r.gateways {
		out = append(out, g)
		if len(out) >= max {
			break
		}
	}
	return out
}

// Snapshot returns every currently-known gateway, uncapped — used by the
// announce-gossip disseminator (unlike List, which exists to answer the
// client-facing, deliberately-capped GetNodesDiscovery RPC).
func (r *Registry) Snapshot() []*pb.GatewayInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*pb.GatewayInfo, 0, len(r.gateways))
	for _, g := range r.gateways {
		out = append(out, g)
	}
	return out
}
