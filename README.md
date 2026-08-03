# OpenChat — decentralized E2EE messenger on a custom blockchain

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
[beanode.md](beanode.md) for how to run one, and
[deploy.md](deploy.md) for standing up a whole new network's validators.

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
- **Composition root** (`cmd/node`, `cmd/client`, `cmd/app`): pure wiring,
  no logic.
- **Shared client library** (`pkg/client`, `pkg/crypto`, `pkg/authproof`):
  has no CLI-, desktop- or mobile-specific dependencies. It's used
  unmodified by the CLI (`cmd/client`) *and* by the GUI messenger
  (`cmd/app`, `internal/app`) described below.

### Directory layout

```
cmd/node/            validator node entrypoint (composition root)
cmd/client/           CLI entrypoint (thin wrapper over pkg/client)
cmd/app/               Fyne GUI messenger — builds for macOS/Windows/Linux/iOS/Android
cmd/keytool/            derive a validator's address + libp2p peer ID from a seed
internal/app/          GUI's local storage + service glue (wallet, contacts, history)
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
deployments/docker/   Dockerfiles, dev TLS cert script, compose secrets
deployments/k8s/      StatefulSet + Services + ConfigMap/Secret examples
docker-compose.yml    2-validator + 2-client mini network
scripts/              build/tag/push helpers (Docker images, cross-platform client/GUI binaries)
deploy.md             standing up your own independent OpenChat network
beanode.md            running a relay node to join someone else's network
```

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

**Relay nodes (`NODE_ROLE=relay`, see `beanode.md`):** a node can run the
full libp2p host and gRPC gateway — accepting messages, gossiping them,
syncing the chain, delivering incoming messages — without ever running
the consensus engine, and its address can never appear in a validator's
`VALIDATOR_SET` (`internal/config.LoadNodeConfig` and
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

## Running the demo network

```bash
./deployments/docker/gen-dev-certs.sh   # self-signed TLS cert for node1+node2
docker compose up --build
docker compose logs -f client1          # watch it receive client2's message
```

This starts 2 validators (`node1`, `node2`) gossiping over libp2p and
reaching consensus (n=2, f=0 — bump the validator set to 4 for a real
f=1 byzantine-fault demo, see `deployments/k8s/30-statefulset.yaml` for a
4-validator layout), plus 2 isolated CLI "phone" containers that connect
to different nodes and exchange one E2EE message.

### CLI usage

```bash
go build -o openchat-node ./cmd/node      # requires `go mod tidy` first, see note below
go build -o openchat-client ./cmd/client

./openchat-client keygen                  # prints mnemonic + address + x25519 pubkey
./openchat-client send  -mnemonic "..." -to <hex addr> -to-x25519 <hex pub> -text "hi" \
    -bootstrap node1:9090 -ca deployments/docker/certs/tls.crt
./openchat-client listen -mnemonic "..." -bootstrap node1:9090 -ca deployments/docker/certs/tls.crt
```

## Desktop & mobile app (GUI)

`cmd/app` is a graphical messenger built with [Fyne](https://fyne.io) — one
Go codebase, natively compiled for macOS, Windows, Linux, iOS and Android
(no Electron/webview, no per-platform rewrite). It reuses `pkg/client` and
`pkg/crypto` unmodified; `internal/app` adds only what a GUI needs on top:

- **Onboarding** (`cmd/app/onboarding.go`): create a new identity (shows
  the 24-word recovery phrase once, for backup) or import an existing one,
  then set a local PIN.
- **Wallet storage** (`internal/app/wallet_store.go`): the mnemonic is
  encrypted at rest with a key derived from that PIN via `scrypt` +
  AES-256-GCM, written through Fyne's cross-platform storage API (works
  identically in a macOS/Windows app-support folder and an iOS/Android app
  sandbox — raw `os.UserConfigDir()` isn't reliably usable on mobile).
- **Contacts** (`internal/app/contactcode.go`): since a recipient needs
  both your address *and* X25519 key, the app combines them into one
  shareable `oc1:...` code (`Service.MyContactCode`) instead of making
  users copy two separate hex strings.
- **Chat list & conversation view** (`cmd/app/mainscreen.go`): local
  message history (`internal/app/chat_store.go`) plus a live
  `StreamIncomingSMS` subscription; incoming messages update the UI via
  `fyne.Do` (Fyne's thread-safe main-goroutine dispatch, since the stream
  read happens on a background goroutine).
- **Free to send:** there's no token, fee, or gas — `SendSMS` just needs a
  validly signed, rate-limited transaction (see the mempool rules in Part
  1). Sending a message costs nothing beyond the recipient having their
  gateway reachable.
- **Network settings** (`cmd/app/settings.go`): a small dialog (gear icon)
  to point the app at your own gateways instead of the placeholder
  `DefaultBootstrapGateways`, e.g. `localhost:9091,localhost:9092` against
  the docker-compose demo, with an optional CA cert / insecure-TLS toggle
  for self-signed certs during local testing.

### Building it

Install the Fyne CLI once (needs your platform's normal cgo toolchain —
Xcode command line tools on macOS, a C compiler + graphics headers on
Linux, etc., since Fyne's OpenGL backend uses cgo):

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
go mod tidy   # see the "honest caveats" note below — required first
```

**macOS** (run on a Mac):

```bash
cd cmd/app
fyne package -os darwin -icon Icon.png
# -> OpenChat.app, drag into /Applications or notarize for distribution
```

**Windows** (run natively on Windows, or cross-compile from macOS/Linux
with `mingw-w64` installed):

```bash
cd cmd/app
fyne package -os windows -icon Icon.png
# -> OpenChat.exe
```

**Android** (from any OS, needs the Android SDK + NDK):

```bash
cd cmd/app
fyne package -os android -appID network.openchat.app -icon Icon.png
# -> OpenChat.apk
```

**iOS** (must run on macOS with Xcode + an Apple Developer account for
signing):

```bash
cd cmd/app
fyne package -os ios -appID network.openchat.app -icon Icon.png
# -> OpenChat.app / .ipa via Xcode's archive/export step
```

**Cross-compiling all four from one Linux CI box:** use
[`fyne-cross`](https://github.com/fyne-io/fyne-cross) instead — it runs
each target's build in a matching Docker container, so you don't need
Xcode/Android Studio installed locally for every platform:

```bash
go install github.com/fyne-io/fyne-cross@latest
fyne-cross darwin -arch=amd64,arm64 -app-id network.openchat.app ./cmd/app
fyne-cross windows -arch=amd64 -app-id network.openchat.app ./cmd/app
fyne-cross android -app-id network.openchat.app ./cmd/app
fyne-cross ios -app-id network.openchat.app ./cmd/app   # still needs a Mac for signing
```

`cmd/app/FyneApp.toml` carries the app ID/name/icon metadata both `fyne
package` and `fyne-cross` read by default; `cmd/app/Icon.png` is a
placeholder — swap it for real artwork before shipping.

### Kubernetes

`deployments/k8s/` has a 4-validator `StatefulSet` (headless Service for
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

1. **Get each validator's identity.** `cmd/keytool` prints the address
   (for `VALIDATOR_SET`) and libp2p peer ID (for bootstrap multiaddrs)
   from a seed — generate one per validator:
   ```bash
   docker run --rm -v $(pwd):/src -w /src golang:1.22-bookworm go run ./cmd/keytool
   ```
   Keep the printed seed secret; it's what you set as
   `VALIDATOR_PRIVATE_KEY_SEED` for that node.
2. **Point nginx at both nodes.** `deployments/nginx/openchat-grpc.conf.example`
   terminates TLS per-domain (via `certbot`) and reverse-proxies gRPC
   (`grpc_pass`) to each node's private address.
   `deployments/nginx/openchat-stream.conf.example` does raw TCP
   passthrough for libp2p on two external ports, one per node, since
   libp2p already encrypts itself and doesn't need nginx to touch TLS.
3. **Deploy `deployments/docker/portainer-stack.yml` twice** (once per
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
- `deployments/docker/secrets/*.txt` and the two client mnemonics in
  `docker-compose.yml` are well-known/throwaway demo values — never reuse
  them outside this local demo.
- `cmd/app` needs `fyne.Do`, added in Fyne v2.5 — `go.mod` pins
  `fyne.io/fyne/v2 v2.5.3`; don't downgrade it without replacing the
  `fyne.Do` calls in `cmd/app/mainscreen.go` with your own main-goroutine
  dispatch.
- Fyne's desktop backend uses cgo/OpenGL, so building `cmd/app` needs a
  real C toolchain (Xcode CLT / a Linux C compiler + Mesa dev headers /
  MSVC or mingw on Windows) — `CGO_ENABLED=0` (used for the node/CLI
  Docker builds above) will not work for it.
- `cmd/app/Icon.png` is a placeholder generated for this reference
  implementation — replace it before shipping to a store.
