// Package pb contains the wire types for the NodeGateway gRPC service
// defined in api/protobuf/sms.proto.
//
// These structs are hand-written to exactly mirror the .proto contract
// (field names/JSON names match `json=` tags protoc-gen-go would emit),
// because this project was authored in an offline sandbox with no protoc
// toolchain available. They are marshaled with the JSON codec registered
// in codec.go under the gRPC "proto" content-subtype, so wire behaviour
// (streaming, TLS, HTTP/2 multiplexing, deadlines) is identical to a real
// generated client/server pair. To switch to canonical protoc-gen-go /
// protoc-gen-go-grpc output later: run the command documented at the top
// of sms.proto and delete this package — field names were kept identical
// on purpose so calling code in internal/grpcserver and pkg/client does
// not need to change.
package pb

// Empty is the request type for GetAddress.
type Empty struct{}

// AddressResponse is returned by GetAddress.
type AddressResponse struct {
	ValidatorAddress string `json:"validator_address"`
	NodeVersion      string `json:"node_version"`
	ChainHeight      uint64 `json:"chain_height"`
	PeerCount        uint32 `json:"peer_count"`
	// Role is "validator" or "relay" — lets a client (or another node
	// probing this one) tell at a glance whether it's talking to a
	// consensus-voting node or a community-run relay/gateway.
	Role string `json:"role,omitempty"`
}

// SMSRequest carries one signed, E2EE-sealed message transaction.
type SMSRequest struct {
	FromAddress     string `json:"from_address"`
	ToAddress       string `json:"to_address"`
	Ciphertext      []byte `json:"ciphertext"`
	NonceAEAD       []byte `json:"nonce_aead"`
	EphemeralPubkey []byte `json:"ephemeral_pubkey"`
	Nonce           uint64 `json:"nonce"`
	Timestamp       int64  `json:"timestamp"`
	Signature       []byte `json:"signature"`
}

// SMSResponse is returned both by SendSMS (ack) and pushed by
// StreamIncomingSMS (delivery).
type SMSResponse struct {
	Accepted    bool        `json:"accepted"`
	TxHash      string      `json:"tx_hash"`
	Error       string      `json:"error,omitempty"`
	Message     *SMSRequest `json:"message,omitempty"`
	BlockHeight uint64      `json:"block_height,omitempty"`
}

// StreamRequest opens a StreamIncomingSMS subscription, proving ownership
// of `Address` via a signature over Address||Timestamp.
type StreamRequest struct {
	Address   string `json:"address"`
	Timestamp int64  `json:"timestamp"`
	Signature []byte `json:"signature"`
}

// DiscoveryRequest asks for a fresh gateway list.
type DiscoveryRequest struct {
	MaxResults uint32 `json:"max_results"`
}

// DiscoveryResponse is the client's failover cache seed.
type DiscoveryResponse struct {
	Gateways []*GatewayInfo `json:"gateways"`
}

// GatewayInfo describes one live RPC gateway known to the responding
// node — this is also the payload the client-facing node-status feature
// is built on: every node network-wide converges on the same set of
// GatewayInfo records via TopicNodeAnnounce gossip, and GetNodesDiscovery
// simply returns whatever the responding node currently holds.
type GatewayInfo struct {
	GRPCAddress      string `json:"grpc_address"`
	ValidatorAddress string `json:"validator_address"`
	LastSeenUnix     int64  `json:"last_seen_unix"`

	// Role is "validator" or "relay". Empty means unknown (e.g. a bare
	// seed entry that hasn't announced itself yet).
	Role string `json:"role,omitempty"`
	// NodeVersion is the self-reported build version of the node at
	// GRPCAddress (informational for clients; the *validator's* own
	// version-allowlist check at registration time is what actually
	// gates network participation, not this display field).
	NodeVersion string `json:"node_version,omitempty"`
	// Healthy reflects the most recent functional probe result for relay
	// nodes (see RegisterRelayResponse / the health-check job in
	// cmd/node). Always true for validators, which don't get probed —
	// their liveness is directly observable through consensus itself.
	Healthy bool `json:"healthy"`
	// LastProbeUnix is when Healthy was last updated by an actual
	// round-trip probe (0 if this node has never been probed, e.g. a
	// validator or a relay that just announced but hasn't been checked
	// yet).
	LastProbeUnix int64 `json:"last_probe_unix,omitempty"`
}

// RegisterRelayRequest is sent by a community-run relay node to a
// validator ("trusted center") to join the network's discovery/health
// system. It never grants consensus membership — a validator only ever
// adds the caller to its discovery Registry, never to its
// consensus.Config.Validators set.
type RegisterRelayRequest struct {
	RelayAddress     string `json:"relay_address"`      // hex ed25519 pubkey identifying this relay
	RelayGRPCAddress string `json:"relay_grpc_address"`  // host:port other peers/clients should dial to reach it
	NodeVersion      string `json:"node_version"`        // self-reported build version
	NodeCommit       string `json:"node_commit"`         // self-reported build commit hash
	Timestamp        int64  `json:"timestamp"`           // unix millis, bounds replay of old registrations
	Signature        []byte `json:"signature"`           // ed25519(RelayAddress) over relayproof.Bytes(...)
}

// RegisterRelayResponse tells the relay whether it was accepted into the
// validator's (and, via announce-gossip, the network's) discovery set.
type RegisterRelayResponse struct {
	Accepted             bool   `json:"accepted"`
	Reason               string `json:"reason,omitempty"`
	ProbeIntervalSeconds  uint32 `json:"probe_interval_seconds,omitempty"`
	RegistrationTTLSeconds uint32 `json:"registration_ttl_seconds,omitempty"`
}

// GetBlocksRequest asks for a contiguous range of committed blocks,
// starting at FromHeight (inclusive) — this is how a relay node (which
// never participates in the propose/vote gossip) catches up from a cold
// start or after being offline for longer than gossip retains messages.
type GetBlocksRequest struct {
	FromHeight uint64 `json:"from_height"`
	MaxBlocks  uint32 `json:"max_blocks"`
}

// GetBlocksResponse returns as many contiguous blocks starting at
// FromHeight as the responding node currently has, up to MaxBlocks.
// ChainHeight reports the responder's current tip so the caller knows
// whether it has fully caught up yet.
type GetBlocksResponse struct {
	Blocks      []*BlockPB `json:"blocks"`
	ChainHeight uint64     `json:"chain_height"`
}

// ValidatorSigPB mirrors blockchain.ValidatorSig on the wire.
type ValidatorSigPB struct {
	Validator string `json:"validator"`
	Round     uint32 `json:"round"`
	Signature []byte `json:"signature"`
}

// BlockPB mirrors blockchain.Block on the wire (see the note atop this
// file on why these are hand-written instead of protoc-generated).
// Transactions reuse SMSRequest, which already mirrors
// blockchain.Transaction field-for-field.
type BlockPB struct {
	Height       uint64            `json:"height"`
	PrevHash     string            `json:"prev_hash"`
	Timestamp    int64             `json:"timestamp"`
	Proposer     string            `json:"proposer"`
	Transactions []*SMSRequest     `json:"transactions"`
	MerkleRoot   string            `json:"merkle_root"`
	CommitSigs   []*ValidatorSigPB `json:"commit_sigs"`
}
