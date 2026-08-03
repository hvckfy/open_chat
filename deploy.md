# Deploying your own OpenChat network

This is a guide for standing up a brand new, independent OpenChat network
that you control end to end: your own validators, your own domains, your
own `VALIDATOR_SET`. If you instead want to join *someone else's* existing
network as a community-run relay node, see [beanode.md](beanode.md) — this
document is for the operator(s) running the validators themselves.

Everything here builds on what's already described in the main
[README.md](README.md) ("Manual deploy behind nginx" and "Kubernetes"
sections) — this doc goes deeper into the parts operators most often get
stuck on: picking a validator count, generating keys, building/tagging
images consistently, and wiring TLS.

## 1. Decide your validator count

OpenChat's consensus quorum is `2f+1` out of `n = 3f+1` validators, where
`f` is how many byzantine/crashed validators the network tolerates:

| n (validators) | f (tolerated faults) | quorum |
|---|---|---|
| 1 | 0 | 1 |
| 2 | 0 | 1 |
| 3 | 0 | 1 |
| 4 | 1 | 3 |
| 7 | 2 | 5 |

Notice `n=2` and `n=3` both give `f=0` — no real byzantine-fault tolerance,
just a working demo. If you want the network to survive one validator
going rogue or crashing, you need **at least 4**. Start with 2 for local
testing (`docker compose up --build` — see the README), then grow to 4+
before anything you'd call production.

## 2. Generate each validator's identity

Every validator needs an Ed25519 keypair. `cmd/keytool` derives the
address (for `VALIDATOR_SET`) and libp2p peer ID (for bootstrap
multiaddrs) from a seed, generating a fresh one if you don't supply one:

```bash
go run ./cmd/keytool
# or, without a local Go toolchain:
docker run --rm -v $(pwd):/src -w /src golang:1.22-bookworm go run ./cmd/keytool
```

Do this once per validator. Keep each printed seed secret — it's what you
set as `VALIDATOR_PRIVATE_KEY_SEED` (or write to a
`VALIDATOR_PRIVATE_KEY_FILE`-mounted secret) for that specific node. Write
down every validator's **address** too; the exact same comma-separated
list of all of them becomes `VALIDATOR_SET` on *every* validator.

`VALIDATOR_SET` is read purely from local config on each node — there is
no on-chain vote to add a validator. All validators must be updated and
redeployed with the same new list at the same time whenever it changes, or
they will disagree about quorum size and proposer rotation and stop
committing blocks together.

## 3. Build and tag your images

```bash
./scripts/build-nodes.sh                 # builds openchat-node + openchat-client
REGISTRY=myregistry.example.com/openchat ./scripts/build-nodes.sh --push
```

This tags every image with **both** a resolved version (`git describe`,
e.g. `v1.2.0` or a commit-based fallback if you haven't tagged a release)
and `:latest`. Deploy validators pinned to the resolved version tag, not
`:latest` — `imagePullPolicy: IfNotPresent` + `:latest` in Kubernetes in
particular will silently keep running a stale image after you push a new
one under the same tag.

Set `NODE_VERSION` (and ideally `NODE_COMMIT`) on every running container
to match the image tag you actually deployed:

```bash
NODE_VERSION=$(git describe --tags --always --dirty)
NODE_COMMIT=$(git rev-parse --short HEAD)
```

This matters beyond bookkeeping: if you ever accept relay nodes (see
beanode.md), a validator's default policy is to only accept a relay whose
self-reported `NODE_VERSION` matches its **own** `NODE_VERSION` exactly. A
validator whose `NODE_VERSION` doesn't reflect what it's actually running
will reject relays for the wrong reason, or silently accept ones you didn't
mean to allow.

The Fyne desktop/mobile client app is built separately, with one script per
host OS since only a real Mac can produce the macOS/iOS builds:
`scripts/source-macos.sh` (builds everything — macOS, iOS, Windows, Linux,
Android), `scripts/source-linux.sh` and `scripts/source-windows.sh` (build
everything except macOS/iOS from those hosts). All three derive the same
`git describe`-based version string for their zip/bundle filenames, so a
given commit's server images and client binaries are always identifiable
by the same version. (`scripts/build-client.sh` still works as an alias for
`scripts/source-macos.sh`.)

## 4. Pick a TLS strategy

Two supported shapes, both already wired into `internal/config`:

- **Node terminates TLS itself** (`GRPC_TLS_ENABLED=true`, the default):
  point `GRPC_TLS_CERT_FILE`/`GRPC_TLS_KEY_FILE` at a real keypair (or use
  `deployments/docker/gen-dev-certs.sh` for a throwaway self-signed one,
  local testing only). Simple, but every validator needs its own
  certificate and you're managing renewal per-node.
- **A reverse proxy (nginx) terminates TLS, node serves plaintext gRPC
  behind it** (`GRPC_TLS_ENABLED=false`): one certificate per public
  domain, managed centrally (e.g. via `certbot` or a proxy manager like
  NPM). This is what the README's "Manual deploy behind nginx" walkthrough
  and `deployments/nginx/*.conf.example` assume. If you go this route, the
  node's gRPC port **must never** be reachable from anywhere except the
  proxy host — bind it to a private/VPN interface or firewall it, since a
  plaintext, publicly-reachable gRPC port lets anyone read/submit
  transactions without TLS.

## 5. Deploy

Three supported paths, pick one:

- **docker-compose** (`docker-compose.yml`) — 2-validator local demo, not
  meant for real deployment. Good for verifying your build works at all
  before touching real infrastructure.
- **Portainer, one stack per validator** (`deployments/docker/portainer-stack.yml`)
  — for running validators by hand across one or more Docker hosts, each
  behind its own domain via nginx. Fill in the `${...}` placeholders via
  Portainer's per-stack "Environment variables" editor; the `ports:` block
  needs manual editing per validator if multiple validators share a host
  (see the comments at the top of the file).
- **Kubernetes** (`deployments/k8s/`) — a 4-validator `StatefulSet` with
  per-replica keys sourced from one multi-entry `Secret`
  (`20-secret.example.yaml`), a headless Service for stable libp2p
  identity, and `volumeClaimTemplates` for BadgerDB. Apply in numeric
  order (`00-` through `40-`) after filling in `10-configmap.yaml`'s
  `VALIDATOR_SET`/`P2P_BOOTSTRAP_PEERS`/`NODE_VERSION` and
  `30-statefulset.yaml`'s image tag.

## 6. Verify it's actually committing blocks

```bash
curl http://<node-host>:2112/metrics | grep openchat_chain_height
```

should climb over time. In the logs, look for `consensus: block committed`
firing repeatedly, with the `proposer` field rotating between your
validators. If instead you see `no valid proposal received` or `prevote
quorum not reached` timeouts on every round, the validators most likely
can't reach each other over libp2p yet — double check
`P2P_BOOTSTRAP_PEERS` (needs the *other* node's libp2p peer ID from step
2, not its gRPC address) and that the P2P port is actually reachable
between hosts.

Then send a real message end to end with the CLI client (`cmd/client`) or
the GUI app (`cmd/app`) — see the README's "CLI usage" section — to
confirm the full path (mempool → consensus → chain → delivery) works, not
just that blocks are being produced.

## 7. Adding more validators later

1. Generate the new validator's identity (step 2 above).
2. Redeploy **every existing validator** with the new, complete
   `VALIDATOR_SET` (all addresses, old and new) and the new validator added
   to `GATEWAY_PEERS`. Do this before step 3 — a new validator joining
   before the others know about it just gets ignored by their still-stale
   validator set.
3. Deploy the new validator's own stack, with `VALIDATOR_SET` already
   including itself, `GATEWAY_PEERS` pointing at the existing validators,
   and `P2P_BOOTSTRAP_PEERS` pointing at any one of them (Kademlia DHT
   discovery finds the rest).
4. Point DNS/your reverse proxy at it and open its P2P port through
   whatever firewall sits in front of it.

There is no minimum-downtime or rolling-upgrade tooling here beyond "update
everyone's config and redeploy" — this is a small, operator-coordinated
network, not a permissionless validator set.
