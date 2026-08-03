// Package config loads node configuration from the process environment
// and Docker/Kubernetes secret files only — never from disk-persisted
// application config containing key material, and never logs secret
// values. This is the "Zero-Trust: приватные ключи считываются из
// переменных окружения/Docker Secrets и хранятся только в памяти
// процесса" requirement.
package config

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type NodeConfig struct {
	// Identity
	ValidatorPriv ed25519.PrivateKey // in-memory only, never persisted by this process
	ValidatorAddr string

	// Storage
	DBPath string

	// P2P (libp2p)
	P2PListenAddrs    []string
	P2PBootstrapPeers []string
	ValidatorSet      []string // hex addresses of all validators (static set)

	// gRPC gateway
	GRPCListenAddr   string
	ExternalGRPCAddr string // host:port other peers/clients dial (may differ from bind addr behind NAT/k8s Service)
	GRPCTLSEnabled   bool   // false when a trusted reverse proxy (e.g. nginx with a real cert) terminates TLS in front of this node
	GRPCTLSCert      string
	GRPCTLSKey       string
	GatewayPeers     []string // seed list of other known gateway "host:port" entries, for GetNodesDiscovery

	// Metrics
	MetricsListenAddr string

	// Consensus
	RoundTimeout time.Duration
	BlockMaxTxs  int

	// Misc
	LogLevel    string
	NodeVersion string
	NodeCommit  string // self-reported build commit hash, used in relay registration

	// Node role: "validator" (default, runs consensus and is expected to
	// be a member of ValidatorSet) or "relay" (a community-run node that
	// only ever relays/gateways messages — libp2p + gRPC gateway + chain
	// sync, but NEVER runs the consensus engine and must NEVER appear in
	// ValidatorSet; see LoadNodeConfig's validation below).
	NodeRole string

	// Relay registration/health-check — meaningful on a validator (the
	// "trusted center" that accepts registrations and probes relays).
	RelayAllowedVersions   []string      // RELAY_ALLOWED_VERSIONS; empty => only NodeVersion itself is accepted
	RelayProbeInterval     time.Duration // RELAY_PROBE_INTERVAL: how often to functionally probe each registered relay
	RelayProbeTimeout      time.Duration // RELAY_PROBE_TIMEOUT: max wait for a single probe round-trip
	RelayMaxFailures       int           // RELAY_MAX_CONSECUTIVE_FAILURES before a relay is marked unhealthy
	RelayRegistrationTTL   time.Duration // RELAY_REGISTRATION_TTL: advertised to relays as how often to re-register
	RelayProbeInsecureTLS  bool          // RELAY_PROBE_INSECURE_TLS: skip TLS verification when dialing relays to probe them (dev/self-signed only)

	// Relay-side: which validator gateways to register with and sync the
	// chain from — meaningful on a relay (ignored by validators).
	RelayRegisterWith []string // RELAY_REGISTER_WITH
}

// LoadNodeConfig reads everything from env vars (12-factor style), which
// is exactly what the Dockerfile/docker-compose/Kubernetes manifests in
// this repo populate — Kubernetes Secrets and Docker secrets both surface
// as either env vars or files under /run/secrets, both handled below.
func LoadNodeConfig() (*NodeConfig, error) {
	priv, addr, err := loadValidatorKey()
	if err != nil {
		return nil, err
	}

	cfg := &NodeConfig{
		ValidatorPriv:     priv,
		ValidatorAddr:     addr,
		DBPath:            getEnv("DB_PATH", "/data/openchat-db"),
		P2PListenAddrs:    splitCSV(getEnv("P2P_LISTEN_ADDRS", "/ip4/0.0.0.0/tcp/4001")),
		P2PBootstrapPeers: splitCSV(getEnv("P2P_BOOTSTRAP_PEERS", "")),
		ValidatorSet:      splitCSV(mustGetEnv("VALIDATOR_SET")),
		GRPCListenAddr:    getEnv("GRPC_LISTEN_ADDR", "0.0.0.0:9090"),
		ExternalGRPCAddr:  getEnv("GRPC_EXTERNAL_ADDR", getEnv("GRPC_LISTEN_ADDR", "0.0.0.0:9090")),
		GRPCTLSEnabled:    getEnvBool("GRPC_TLS_ENABLED", true),
		GRPCTLSCert:       getEnv("GRPC_TLS_CERT_FILE", "/etc/openchat/tls/tls.crt"),
		GRPCTLSKey:        getEnv("GRPC_TLS_KEY_FILE", "/etc/openchat/tls/tls.key"),
		GatewayPeers:      splitCSV(getEnv("GATEWAY_PEERS", "")),
		MetricsListenAddr: getEnv("METRICS_LISTEN_ADDR", "0.0.0.0:2112"),
		RoundTimeout:      getEnvDuration("CONSENSUS_ROUND_TIMEOUT", 4*time.Second),
		BlockMaxTxs:       getEnvInt("CONSENSUS_BLOCK_MAX_TXS", 500),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		NodeVersion:       getEnv("NODE_VERSION", "dev"),
		NodeCommit:        getEnv("NODE_COMMIT", "unknown"),

		NodeRole:              strings.ToLower(getEnv("NODE_ROLE", "validator")),
		RelayAllowedVersions:  splitCSV(getEnv("RELAY_ALLOWED_VERSIONS", "")),
		RelayProbeInterval:    getEnvDuration("RELAY_PROBE_INTERVAL", 60*time.Second),
		RelayProbeTimeout:     getEnvDuration("RELAY_PROBE_TIMEOUT", 15*time.Second),
		RelayMaxFailures:      getEnvInt("RELAY_MAX_CONSECUTIVE_FAILURES", 3),
		RelayRegistrationTTL:  getEnvDuration("RELAY_REGISTRATION_TTL", 10*time.Minute),
		RelayProbeInsecureTLS: getEnvBool("RELAY_PROBE_INSECURE_TLS", false),
		RelayRegisterWith:     splitCSV(getEnv("RELAY_REGISTER_WITH", "")),
	}

	if len(cfg.ValidatorSet) == 0 {
		return nil, fmt.Errorf("config: VALIDATOR_SET must list at least one validator address")
	}

	switch cfg.NodeRole {
	case "validator":
		if !containsAddr(cfg.ValidatorSet, cfg.ValidatorAddr) {
			return nil, fmt.Errorf("config: NODE_ROLE=validator but this node's address %s is not a member of VALIDATOR_SET — it would never propose or vote; either add it to VALIDATOR_SET or set NODE_ROLE=relay", cfg.ValidatorAddr)
		}
	case "relay":
		if containsAddr(cfg.ValidatorSet, cfg.ValidatorAddr) {
			return nil, fmt.Errorf("config: NODE_ROLE=relay but this node's address %s IS a member of VALIDATOR_SET — a relay must never be able to vote in consensus; use a different keypair for relay nodes", cfg.ValidatorAddr)
		}
	default:
		return nil, fmt.Errorf("config: NODE_ROLE must be %q or %q, got %q", "validator", "relay", cfg.NodeRole)
	}

	return cfg, nil
}

func containsAddr(set []string, addr string) bool {
	for _, a := range set {
		if a == addr {
			return true
		}
	}
	return false
}

// loadValidatorKey implements the zero-trust key sourcing order:
//  1. VALIDATOR_PRIVATE_KEY_FILE — a Docker/Kubernetes secret file path
//     containing a 64-char hex Ed25519 seed (preferred: never touches
//     the process's own env, only the orchestrator-managed secret mount).
//  2. VALIDATOR_PRIVATE_KEY_SEED — the same hex seed passed directly as
//     an env var (e.g. for docker-compose demo/dev use).
//
// The key is decoded once into memory and never written back to disk or
// logged.
func loadValidatorKey() (ed25519.PrivateKey, string, error) {
	var seedHex string

	if path := os.Getenv("VALIDATOR_PRIVATE_KEY_FILE"); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, "", fmt.Errorf("config: read VALIDATOR_PRIVATE_KEY_FILE: %w", err)
		}
		seedHex = strings.TrimSpace(string(b))
	} else {
		seedHex = strings.TrimSpace(os.Getenv("VALIDATOR_PRIVATE_KEY_SEED"))
	}

	if seedHex == "" {
		return nil, "", fmt.Errorf("config: no validator key supplied (set VALIDATOR_PRIVATE_KEY_FILE or VALIDATOR_PRIVATE_KEY_SEED)")
	}

	seed, err := hex.DecodeString(seedHex)
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, "", fmt.Errorf("config: validator key must be a %d-byte hex-encoded Ed25519 seed", ed25519.SeedSize)
	}

	priv := ed25519.NewKeyFromSeed(seed)
	addr := hex.EncodeToString(priv.Public().(ed25519.PublicKey))
	return priv, addr, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustGetEnv(key string) string {
	return os.Getenv(key)
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
