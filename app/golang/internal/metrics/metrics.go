// Package metrics exposes the node's Prometheus metrics: chain height,
// mempool size, connected P2P peers and TPS, per the spec.
package metrics

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ChainHeight = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "openchat",
		Name:      "chain_height",
		Help:      "Current committed block height.",
	})

	MempoolSize = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "openchat",
		Name:      "mempool_size",
		Help:      "Number of transactions currently pending in the mempool.",
	})

	PeerCount = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "openchat",
		Name:      "p2p_peers",
		Help:      "Number of currently connected libp2p peers.",
	})

	CommittedTxTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "openchat",
		Name:      "committed_tx_total",
		Help:      "Total transactions committed into the chain since process start.",
	})

	TPS = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "openchat",
		Name:      "tps",
		Help:      "Transactions committed per second (rolling window).",
	})

	BlocksCommittedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "openchat",
		Name:      "blocks_committed_total",
		Help:      "Total blocks committed since process start.",
	})

	GRPCRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "openchat",
		Name:      "grpc_requests_total",
		Help:      "gRPC gateway requests by method and outcome.",
	}, []string{"method", "outcome"})
)

// committedTxShadow mirrors CommittedTxTotal as a plain int64 so
// StartTPSSampler can compute a delta-per-window without needing to
// decode the Prometheus wire format just to read our own counter back.
var committedTxShadow int64

// RecordCommit updates counters after a block is committed.
func RecordCommit(txCount int) {
	BlocksCommittedTotal.Inc()
	CommittedTxTotal.Add(float64(txCount))
	atomic.AddInt64(&committedTxShadow, int64(txCount))
}

// StartTPSSampler recomputes the TPS gauge every `window` from the delta
// in committed tx count, until ctx is canceled.
func StartTPSSampler(ctx context.Context, window time.Duration) {
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		var last int64
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cur := atomic.LoadInt64(&committedTxShadow)
				TPS.Set(float64(cur-last) / window.Seconds())
				last = cur
			}
		}
	}()
}

// Handler returns the standard Prometheus scrape HTTP handler, to be
// mounted at /metrics.
func Handler() http.Handler {
	return promhttp.Handler()
}
