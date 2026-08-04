// Package client is the OpenChat client library: resilient gateway
// discovery/failover, wallet/session management and E2EE message
// send/receive, all runnable both from cmd/client (CLI) and embedded in a
// mobile app via gomobile bindings around this same package.
package client

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"openchat/internal/grpcserver/pb"
)

// DefaultBootstrapGateways is the hardcoded fallback list baked into the
// client binary — the two real validators of this private network.
// Override per-device via the client's network settings (e.g. the
// mobile app's B6 "Network settings" sheet, stored in UserDefaults) if
// a client should point somewhere else; this is only the fallback used
// when no override is configured, so it must stay in sync with the
// actual deployment (see cicd/docker/portainer-stack.yml) rather than
// carrying unrelated placeholder domains no client can ever reach.
var DefaultBootstrapGateways = []string{
	"openchat.node1.mftkhv.ru:443",
	"openchat.node2.mftkhv.ru:443",
}

const (
	initialBackoff = 500 * time.Millisecond
	maxBackoff     = 15 * time.Second
	dialTimeout    = 5 * time.Second
)

// Discovery owns the resilient connection: a hardcoded bootstrap list, a
// dynamically-grown cache of gateways learned via GetNodesDiscovery, and
// the currently active *grpc.ClientConn. Every public method is safe for
// concurrent use.
type Discovery struct {
	tlsCreds credentials.TransportCredentials
	OnEvent  func(format string, args ...any) // optional logging hook

	mu          sync.Mutex
	bootstrap   []string
	cache       []string
	current     *grpc.ClientConn
	currentAddr string
}

func NewDiscovery(bootstrap []string, tlsCreds credentials.TransportCredentials) *Discovery {
	if len(bootstrap) == 0 {
		bootstrap = DefaultBootstrapGateways
	}
	return &Discovery{tlsCreds: tlsCreds, bootstrap: bootstrap}
}

func (d *Discovery) log(format string, args ...any) {
	if d.OnEvent != nil {
		d.OnEvent(format, args...)
	}
}

// candidates returns the dynamic cache (preferred: fresher, larger, likely
// closer/healthier) followed by the hardcoded bootstrap list, deduplicated,
// with the cache order shuffled so many clients don't all hammer the same
// "first" gateway simultaneously.
func (d *Discovery) candidates(exclude string) []string {
	d.mu.Lock()
	cache := append([]string{}, d.cache...)
	bootstrap := append([]string{}, d.bootstrap...)
	d.mu.Unlock()

	rand.Shuffle(len(cache), func(i, j int) { cache[i], cache[j] = cache[j], cache[i] })

	seen := map[string]bool{exclude: true}
	out := make([]string, 0, len(cache)+len(bootstrap))
	for _, a := range append(cache, bootstrap...) {
		if a == "" || seen[a] {
			continue
		}
		seen[a] = true
		out = append(out, a)
	}
	return out
}

// Connect implements the "Алгоритм живучести": try the first candidate,
// on failure back off exponentially and try the next, cycling through the
// whole candidate list (bootstrap, plus any cached discovery results) and
// repeating with growing backoff until ctx is canceled or a connection
// succeeds. On success it immediately refreshes the gateway cache via
// GetNodesDiscovery.
func (d *Discovery) Connect(ctx context.Context) error {
	backoff := initialBackoff
	for {
		for _, addr := range d.candidates("") {
			dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
			conn, err := grpc.DialContext(dialCtx, addr,
				grpc.WithTransportCredentials(d.tlsCreds),
				grpc.WithBlock(),
			)
			cancel()
			if err != nil {
				d.log("discovery: gateway %s unreachable: %v", addr, err)
				continue
			}

			d.mu.Lock()
			d.current = conn
			d.currentAddr = addr
			d.mu.Unlock()
			d.log("discovery: connected to gateway %s", addr)

			d.refreshCache(ctx)
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("client: no gateway reachable: %w", ctx.Err())
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// refreshCache calls GetNodesDiscovery on the currently active gateway and
// stores the returned peer list, so future Failover calls have a live,
// >=20-entry pool to pick from instead of only the 3-5 hardcoded seeds.
func (d *Discovery) refreshCache(ctx context.Context) {
	conn := d.Conn()
	if conn == nil {
		return
	}
	c := pb.NewNodeGatewayClient(conn)
	reqCtx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()
	resp, err := c.GetNodesDiscovery(reqCtx, &pb.DiscoveryRequest{MaxResults: 32})
	if err != nil {
		d.log("discovery: refresh failed: %v", err)
		return
	}
	addrs := make([]string, 0, len(resp.Gateways))
	for _, g := range resp.Gateways {
		if g.GRPCAddress != "" {
			addrs = append(addrs, g.GRPCAddress)
		}
	}
	d.mu.Lock()
	d.cache = addrs
	d.mu.Unlock()
	d.log("discovery: cached %d live gateways", len(addrs))
}

// Conn returns the currently active connection (nil if never connected).
func (d *Discovery) Conn() *grpc.ClientConn {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.current
}

// CurrentAddr returns the "host:port" of the currently active gateway
// ("" if never connected) — mainly for UI status display (e.g. the
// mobile bridge and network-settings screens showing what's actually
// connected right now, as opposed to what was configured).
func (d *Discovery) CurrentAddr() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.currentAddr
}

// Failover drops the current (presumably broken) connection and connects
// to a different candidate, invisibly to the caller — "клиент мгновенно и
// незаметно для пользователя переключается на рабочий узел из кэша".
func (d *Discovery) Failover(ctx context.Context) error {
	d.mu.Lock()
	bad := d.currentAddr
	if d.current != nil {
		_ = d.current.Close()
		d.current = nil
	}
	d.mu.Unlock()

	backoff := initialBackoff
	for {
		for _, addr := range d.candidates(bad) {
			dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
			conn, err := grpc.DialContext(dialCtx, addr,
				grpc.WithTransportCredentials(d.tlsCreds),
				grpc.WithBlock(),
			)
			cancel()
			if err != nil {
				d.log("discovery: failover candidate %s unreachable: %v", addr, err)
				continue
			}
			d.mu.Lock()
			d.current = conn
			d.currentAddr = addr
			d.mu.Unlock()
			d.log("discovery: failed over to gateway %s", addr)
			d.refreshCache(ctx)
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("client: failover exhausted candidates: %w", ctx.Err())
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (d *Discovery) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.current != nil {
		return d.current.Close()
	}
	return nil
}
