# app/golang — OpenChat's core Go module (backend node + client library)

This directory is the backend/core Go module: the validator/relay node
and the shared client library (`pkg/client`, `pkg/crypto`) the CLI client
uses. It is deliberately **UI-free** — the cross-platform GUI app (Fyne)
lives in its own separate module at [`../fyne`](../fyne), which depends on
this module's `pkg/client`/`pkg/crypto` via a `replace` directive rather
than living inside it, so that Fyne's large cgo/OpenGL/mobile-toolchain
dependency tree never touches this module's `go.sum` or the node/client
Docker build context. For where everything fits in the repo as a whole
(deployment configs in `../../cicd`, docs in `../../docs`, the future
native macOS/iOS shell in `../xcode`), see the repo root
[README.md](../../README.md).

OpenChat is a small blockchain whose transactions *are* encrypted SMS
messages. A static set of validator nodes gossip over libp2p, agree on
blocks via a simplified BFT consensus, and expose a TLS gRPC gateway that
clients use to send messages and subscribe to new ones in real time. All
message content is end-to-end encrypted client-side; the network only ever
sees ciphertext.

Anyone can additionally run a **relay node** (`NODE_ROLE=relay`) to help
the network without needing a vote: it relays messages and syncs the
chain like a validator's gateway does, but never proposes, never votes,
and can never be added to a validator set. See
[../../docs/beanode.md](../../docs/beanode.md) for how to run one, and
[../../docs/deploy.md](../../docs/deploy.md) for standing up a whole new
network's validators.

## Architecture

```
                                   ┌─────────────────────────────────────────┐
                                   │              validator node              │
                                   │                                           │
 ┌──────────┐   TLS gRPC          │  gRPC gateway (internal/grpcserver)       │
 │  client  │◄───────────────────►│      │                                    │
 │ (CLI /   │  SendSMS             │      ▼                                    │
 │  mobile  │  StreamIncomingSMS   │  mempool (internal/mempool)               │
 │  lib)    │  GetAddress          │   rate-limit 5tx/s/key, nonce/replay      │
 │          │  GetNodesDiscovery   │      │                                    │
 └──────────┘                     │      ▼                                    │
                                   │  consensus engine (internal/consensus)    │
                                   │   simplified PBFT: propose/prevote/       │
                                   │   precommit over gossip, 2f+1 quorum      │
                                   │      │                                    │
                                   │      ▼                                    │
                                   │  chain (internal/blockchain)              │
                                   │   blocks, tx verification, nonce index    │
                                   │      │                                    │
                                   │      ▼                                    │
                                   │  storage (internal/storage, BadgerDB)     │
                                   │                                           │
                                   │  p2p (internal/p2p): libp2p host, Noise/  │
                                   │   TLS1.3, Kademlia DHT, gossipsub         │
                                   └─────────────────┬─────────────────────────┘
                                                      │ libp2p (gossip + DHT)
                                                      ▼
                                   ┌─────────────────────────────────────────┐
                                   │           other validator nodes          │
                                   └─────────────────────────────────────────┘
```

### Clean Architecture layering

- **Domain** (`internal/blockchain`): `Transaction`, `Block`, `Chain`. Defines
  the `BlockStore` port it needs from persistence — zero imports of gRPC,
  libp2p, or BadgerDB.
- **Use cases** (`internal/consensus`, `internal/mempool`): BFT block
  production and spam/replay defense. Depend only on the domain layer plus
  two tiny self-defined ports (`PubSub`/`Subscription`, `NonceSource`).
- **Adapters** (`internal/p2p`, `internal/storage`, `internal/grpcserver`):
  implement those ports on top of go-libp2p, BadgerDB, and gRPC
  respectively. None of them are imported by the domain/use-case layers —
  dependencies point inward.
- **Composition root** (`cmd/node`, `cmd/client`, `cmd/keytool`): pure
  wiring, no logic.
- **Shared client library** (`pkg/client`, `pkg/crypto`, `pkg/authproof`):
  has no CLI-, desktop- or mobile-specific dependencies — it doesn't even
  import anything else in this module beyond `internal/grpcserver/pb`'s
  wire types. It's used unmodified by the CLI (`cmd/client`) *and*, from
  the separate `../fyne` module, by the GUI messenger (see that module's
  `cmd/app`, `internal/app`). This is also the "core" meant for future
  native shells (see `../xcode`) — a Swift/Kotlin app would talk to the
  same node network the same way, just through its own UI instead of
  Fyne's.

### Directory layout

```
cmd/node/            validator node entrypoint (composition root)
cmd/client/           CLI entrypoint (thin wrapper over pkg/client)
cmd/keytool/            derive a validator's address + libp2p peer ID from a seed
api/protobuf/         sms.proto — the authoritative gRPC contract
internal/blockchain/  Transaction, Block, Chain (domain)
internal/consensus/   simplified PBFT engine (use case)
internal/mempool/     rate limiting + replay protection (use case)
internal/storage/     BadgerDB adapter implementing blockchain.BlockStore
internal/p2p/         libp2p host, Kademlia DHT, gossipsub adapter
internal/grpcserver/  TLS gRPC gateway adapter + hand-rolled pb/ package
internal/config/      env/Docker-secret config loading (zero-trust keys)
internal/logger/      zap JSON structured logging
internal/metrics/     Prometheus metrics
pkg/crypto/           client-side BIP-39 keygen + E2EE (ECDH + AES-256-GCM)
pkg/client/           resilient discovery/failover + high-level client API
pkg/authproof/        tiny shared "prove you own this address" byte layout
pkg/relayproof/        tiny shared "prove you own this relay identity" byte layout
pkg/tlsutil/           gRPC TLS credentials (server + client), public so
                       every consumer (CLI, ../fyne, ../mobile) can use it
mobile/                gomobile-bindable facade over pkg/client/pkg/crypto,
                       for native shells (see ../xcode) — see its package
                       doc comment for the bind command and what it does
                       and doesn't cover
```

No `cmd/app`, `internal/app`, or Fyne dependency here — that's all in the
separate [`../fyne`](../fyne) module, which imports `pkg/client`/
`pkg/crypto` from this one via a `replace` directive (see
[`../../go.work`](../../go.work) for local multi-module dev). Everything
deploy-related (Dockerfiles, docker-compose, Portainer stack, Kubernetes
manifests, nginx configs, the Docker-image build/push script) lives in
[`../../cicd`](../../cicd) — see that directory and
[../../docs/deploy.md](../../docs/deploy.md) /
[../../docs/beanode.md](../../docs/beanode.md) for everything about
running this code, as opposed to what's in this README about how it's
built.

## Part 1 — backend (validator node)

**Keys (zero-trust):** `internal/config.LoadNodeConfig` reads the
validator's Ed25519 seed from `VALIDATOR_PRIVATE_KEY_FILE` (a mounted
Docker/Kubernetes secret — preferred) or `VALIDATOR_PRIVATE_KEY_SEED` (env
var, for compose/dev). It's decoded once into an in-memory
`ed25519.PrivateKey` and never written back to disk or logged.

**Storage:** `internal/storage` wraps BadgerDB, storing blocks
(`blk:<height>`), the chain head pointer, and a durable per-sender nonce
high-water mark used for replay protection. The in-flight mempool itself
is in-memory (`internal/mempool`), since pending transactions aren't
durable state.

**Logs/metrics:** `internal/logger` builds a JSON `zap.Logger`.
`internal/metrics` exposes `openchat_chain_height`, `openchat_mempool_size`,
`openchat_p2p_peers`, `openchat_tps`, `openchat_committed_tx_total`,
`openchat_blocks_committed_total`, and `openchat_grpc_requests_total` on
`/metrics` (Prometheus text format).

**P2P & consensus:** `internal/p2p` wires go-libp2p with both Noise and
TLS 1.3 transport security, UPnP/NAT-PMP port mapping, hole punching and
relay for NAT traversal, and Kademlia DHT (`go-libp2p-kad-dht`) rendezvous
discovery so a node only needs 1-2 bootstrap peers to find the rest of the
validator set. `internal/consensus` runs a simplified PBFT/Tendermint-style
engine: round-robin proposer rotation, PREVOTE/PRECOMMIT gossip rounds,
and a 2f+1 quorum requirement (n=3f+1 validators tolerate f byzantine or
crashed nodes). A stalled or byzantine proposer just causes a round
timeout and view rotation to the next validator — no formal view-change
certificates, matching the "упрощенный" requirement.

**Spam/replay defense:** `internal/mempool` enforces a 5 tx/sec token
bucket per sender public key and rejects any transaction whose nonce isn't
strictly greater than both the sender's last *committed* nonce (durable,
from storage) and the highest nonce already *pending* for that sender —
plus a timestamp freshness window (±2 min) to bound stockpiled signed
transactions.

**gRPC gateway:** `api/protobuf/sms.proto` is the authoritative contract
for `GetAddress` / `SendSMS` / `StreamIncomingSMS` (server streaming) /
`GetNodesDiscovery` / `RegisterRelay` / `GetBlocks`, served over TLS 1.3
(`internal/grpcserver`).

**Relay nodes (`NODE_ROLE=relay`, see `../../docs/beanode.md`):** a node
can run the full libp2p host and gRPC gateway — accepting messages,
gossiping them, syncing the chain, delivering incoming messages — without
ever running the consensus engine, and its address can never appear in a
validator's `VALIDATOR_SET` (`internal/config.LoadNodeConfig` and
`grpcserver.Server.RegisterRelay` both refuse it independently). A relay
registers with one or more validators via a signed `RegisterRelay` RPC
(`pkg/relayproof` + `internal/grpcserver/server.go`); the validator checks
the signature and a self-reported `NODE_VERSION` allowlist before
accepting. Chain sync for relays doesn't depend on ever having voted:
validators broadcast every committed block on a dedicated gossip topic in
real time, and `GetBlocks` backfills any gap (cold start, or an outage
longer than gossip retains). A validator that accepted a relay
periodically dials it back as an ordinary gRPC *client*, sends itself a
message through it, and confirms it arrives via that same relay's
`StreamIncomingSMS` — a real functional check, not a port-open ping — and
the result (plus every node's self-reported version/role) is disseminated
network-wide over a signed announce-gossip topic, which is what backs the
`Role`/`NodeVersion`/`Healthy` fields `GetNodesDiscovery` returns to
clients.

> **Why `internal/grpcserver/pb` isn't `protoc`-generated:** this project
> was authored in a sandbox with no network access to the Go module proxy
> or `protoc`/plugin binaries. `internal/grpcserver/pb` hand-mirrors
> exactly what `protoc-gen-go` + `protoc-gen-go-grpc` would emit from
> `sms.proto` (message structs, a `NodeGatewayServer`/`NodeGatewayClient`
> pair, a `grpc.ServiceDesc`), using a small custom `encoding.Codec`
> (`pb/codec.go`) registered under the standard `"proto"` content-subtype
> name so it plugs into a completely normal gRPC/HTTP2/TLS stack — the
> client and server in this repo interoperate exactly as generated code
> would. To switch to canonical generated code once you have `protoc`
> available: run the command documented at the top of `sms.proto`, then
> delete `internal/grpcserver/pb`; field names were kept identical so no
> call site needs to change.

## Part 2 — client & interaction

**Key derivation (client-side only, `pkg/crypto`):** 256 bits of
`crypto/rand` entropy → BIP-39 mnemonic (`GenerateMnemonic`) → 64-byte
BIP-39 seed → two *independently* HKDF-SHA512-derived keys: an Ed25519
signing keypair (`DeriveKeyPair`) and an X25519 ECDH keypair, so
compromising one doesn't trivially expose the other. The Ed25519 public
key, hex-encoded, is the user's network address ("phone number").

**E2EE (`pkg/crypto/cipher.go`):** `Encrypt(senderPriv, senderPub,
recipientPub, plaintext)` does ECDH → HKDF-SHA256 → AES-256-GCM seal, and
includes the sender's X25519 public key in the output so the recipient can
recompute the same shared secret with only their own private key
(ECDH is commutative: `priv_a·pub_b == priv_b·pub_a`). The network only
ever transports the sealed ciphertext, nonce, and sender ephemeral pubkey.

**Discovery & failover (`pkg/client/discovery.go`):** a hardcoded
`DefaultBootstrapGateways` list (3-5 host:port entries) is baked into the
client. `Discovery.Connect` dials the first candidate; on failure it backs
off exponentially (500ms → 15s cap) and tries the next, cycling the whole
candidate list until one succeeds. On success it immediately calls
`GetNodesDiscovery` and caches the returned gateway list (≥20 entries in a
real network). If the active connection breaks mid-session,
`Discovery.Failover` transparently drops it and reconnects to a different
cached (or bootstrap) candidate — callers of `Client.SendMessage` /
`Client.Listen` never see the underlying reconnect.

**Message sync (`pkg/client/client.go`):** `Listen` opens
`StreamIncomingSMS`, proving address ownership with an Ed25519 signature
over `address || timestamp` (`pkg/authproof`, verified server-side in
`grpcserver.Server.StreamIncomingSMS`). The node subscribes to its own
`Chain`'s commit event bus (`internal/blockchain/chain.go`); the instant a
block commits, every transaction addressed to that client's pubkey is
pushed down the open stream. The client decrypts each arrival with its
X25519 private key and hands plaintext to the application callback.
`FetchHistory` (used by the GUI module's `Service.SyncHistory` — see
`../fyne/internal/app/service.go`) walks `GetBlocks` from a checkpoint
height for the same thing on demand, so a wallet that was offline for a
while catches up instead of only ever seeing messages sent *after* it
reconnects.

## Running the demo network

The demo's docker-compose file now lives in `../../cicd/docker/`:

```bash
cd ../../cicd/docker
./gen-dev-certs.sh          # self-signed TLS cert for node1+node2
docker compose up --build
docker compose logs -f client1   # watch it receive client2's message
```

This starts 2 validators (`node1`, `node2`) gossiping over libp2p and
reaching consensus (n=2, f=0 — bump the validator set to 4 for a real
f=1 byzantine-fault demo, see `../../cicd/k8s/30-statefulset.yaml` for a
4-validator layout), plus 2 isolated CLI "phone" containers that connect
to different nodes and exchange one E2EE message.

### CLI usage

```bash
go build -o openchat-node ./cmd/node
go build -o openchat-client ./cmd/client

./openchat-client keygen                  # prints mnemonic + address + x25519 pubkey
./openchat-client send  -mnemonic "..." -to <hex addr> -to-x25519 <hex pub> -text "hi" \
    -bootstrap node1:9090 -ca ../../cicd/docker/certs/tls.crt
./openchat-client listen -mnemonic "..." -bootstrap node1:9090 -ca ../../cicd/docker/certs/tls.crt
```

## Desktop & mobile app (GUI)

The GUI messenger — onboarding, chat list/conversation, contacts, network
settings — lives in the separate [`../fyne`](../fyne) module, built with
[Fyne](https://fyne.io) (one Go codebase, natively compiled for macOS,
Windows, Linux, iOS and Android). It reuses this module's `pkg/client` and
`pkg/crypto` unmodified via a `replace` directive; see
[`../fyne/README.md`](../fyne/README.md) for what it does and how to
build it for each platform, and
[../../docs/design-code.md](../../docs/design-code.md) for the functional
breakdown of every screen/dialog (useful if you're designing new UI,
native or otherwise).

## Deploying a validator/relay network

### Kubernetes

`../../cicd/k8s/` has a 4-validator `StatefulSet` (headless Service for
libp2p peer identity, an ordinal-aware init container that hands each
replica its own key from a multi-entry `Secret`, `volumeClaimTemplates`
for BadgerDB, a `ClusterIP` Service clients dial for gRPC) plus a
`ConfigMap` for non-secret settings. Apply in order (`00-` → `40-`) after
filling in `10-configmap.yaml`'s `VALIDATOR_SET`/`P2P_BOOTSTRAP_PEERS` and
`20-secret.example.yaml`'s key material.

### Manual deploy behind nginx (Portainer, 1 entry IP + per-node domains)

For running validators by hand via Portainer instead of Kubernetes, with
one public IP fronting nginx and a separate domain (+ real TLS cert) per
node:

1. **Get each validator's identity.** `cmd/keytool` (run from this
   directory) prints the address (for `VALIDATOR_SET`) and libp2p peer ID
   (for bootstrap multiaddrs) from a seed — generate one per validator:
   ```bash
   docker run --rm -v $(pwd):/src -w /src golang:1.22-bookworm go run ./cmd/keytool
   ```
   Keep the printed seed secret; it's what you set as
   `VALIDATOR_PRIVATE_KEY_SEED` for that node.
2. **Point nginx at both nodes.** `../../cicd/nginx/openchat-grpc.conf.example`
   terminates TLS per-domain (via `certbot`) and reverse-proxies gRPC
   (`grpc_pass`) to each node's private address.
   `../../cicd/nginx/openchat-stream.conf.example` does raw TCP
   passthrough for libp2p on two external ports, one per node, since
   libp2p already encrypts itself and doesn't need nginx to touch TLS.
3. **Deploy `../../cicd/docker/portainer-stack.yml` twice** (once per
   validator, wherever that validator's container actually runs), filling
   in Portainer's stack environment editor per the comments at the top of
   that file. Each node runs with `GRPC_TLS_ENABLED=false` — nginx is now
   the one holding the real, browser/mobile-trusted certificate, so
   clients connect with plain `grpcserver.ClientTLS("", false)` (system
   trust store), no `-insecure` flag or custom CA file needed.

Because `GRPC_TLS_ENABLED=false` makes the node serve plaintext gRPC,
that port must **never** be reachable except from the nginx host — bind
the published port to a private/VPN interface (see the comments in
`portainer-stack.yml`) or firewall it.

## Honest caveats

This was built in an offline sandbox (no `go`, `protoc`, or module-proxy
network access), so nothing here was compiled or run on the machine that
wrote it — the code was written and manually cross-checked for API
correctness against each library's documented interfaces instead. `go.sum`
*has* since been generated and committed via a real `go mod tidy` run
(with the `google.golang.org/genproto` split pinned in `go.mod` to avoid
the "ambiguous import" MVS conflict that shows up otherwise), and
`docker build` for `cmd/node`/`cmd/client` has gotten past dependency
resolution. Still, before relying on it further:

- Run `go build ./...` and `go vet ./...` (or just keep building the
  Docker images) and fix whatever surfaces —
  hand-written code at this scale without a compiler in the loop should
  be expected to have a few rough edges (an import path off, a signature
  drift against the exact library versions `go mod tidy` picks, etc.). The
  relay-node feature (`NODE_ROLE=relay`, `RegisterRelay`/`GetBlocks`,
  announce-gossip) is the newest, largest piece of this and has had the
  least real-world exercise — build and test it against a real 2+
  validator network before depending on it.
- The consensus engine is intentionally simplified (documented at the top
  of `internal/consensus/pbft.go`): no formal view-change certificates, no
  slashing, static validator set. Treat it as a correct-in-spirit
  reference, not a production BFT implementation.
- `../../cicd/docker/secrets/*.txt` and the two client mnemonics in
  `../../cicd/docker/docker-compose.yml` are well-known/throwaway demo
  values — never reuse them outside this local demo.
- This `go.mod` still lists a chunk of Fyne-only indirect dependencies
  left over from before the GUI module split (see the comment above the
  `google.golang.org/genproto` pin) — run `go mod tidy` here once to prune
  them; see `../fyne/README.md` for the GUI module's own caveats
  (Fyne version pin, cgo toolchain requirement, placeholder icon).
- `api/protobuf/sms.proto` documents the original 4 RPCs but hasn't been
  updated to list `RegisterRelay`/`GetBlocks` and their message types yet
  — the hand-rolled `internal/grpcserver/pb` package fully implements
  them regardless; the `.proto` is just stale as a reference document.
