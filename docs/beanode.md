# Running a relay node (joining someone else's OpenChat network)

This guide is for a third party who wants to run an OpenChat node on their
own server and join an *existing* network — without becoming a validator.
If you're standing up a brand new, independent network instead, see
[deploy.md](deploy.md).

## What a relay node is (and isn't)

A relay node runs the same `cmd/node` binary as a validator, with
`NODE_ROLE=relay`. It participates in the network almost the same way a
validator's gRPC gateway does — it accepts messages from clients
(`SendSMS`), gossips them to the network, syncs a full copy of the chain,
and delivers incoming messages to clients (`StreamIncomingSMS`). What it
**never** does is participate in consensus: it never proposes a block,
never votes, and its address is never added to any validator's
`VALIDATOR_SET`. Core decisions about what goes into the chain stay
entirely with the network's own validators, exactly as they were before
your node joined. Running a relay node is how you help the network — more
gateways, better geographic/ISP diversity, more resilience if one gateway
goes down — without the network needing to trust you with a vote.

Concretely, this is enforced in three independent places, not just by
convention:

- `internal/config.LoadNodeConfig` refuses to start a `NODE_ROLE=relay`
  node whose own address is present in its configured `VALIDATOR_SET`.
- The validator you register with (`RegisterRelay`) checks the exact same
  thing server-side and rejects the registration if your claimed address
  matches one of its real validators.
- A relay's own `cmd/node` process never constructs or runs the consensus
  engine at all — the code path simply doesn't exist for that role.

## What you'll need from the network operator

Before starting, get from whoever runs the network you're joining:

1. **One or more validator gRPC addresses** to register with and sync
   from — e.g. `openchat.node1.example.com:443`. This becomes
   `RELAY_REGISTER_WITH`.
2. **The current `VALIDATOR_SET`** (comma-separated hex addresses) — your
   node needs this to independently verify blocks' commit-quorum
   signatures; it doesn't need to be secret, it's the same public value
   every validator already runs with.
3. **The expected `NODE_VERSION` string.** By default, a validator only
   accepts a `RegisterRelay` request whose self-reported `NODE_VERSION`
   matches its own exactly (`RELAY_ALLOWED_VERSIONS`, if the operator set
   one, may widen this to a short list of acceptable versions). Ask what
   value to use — usually the same git tag/version they built their
   validators from.
4. Whether their gRPC endpoint expects **real TLS** (normal case — you
   connect the same way any client would, trusting the system CA store)
   or a private/self-signed CA (they'll give you a CA file, or tell you to
   run with `RELAY_PROBE_INSECURE_TLS`-equivalent settings — see below).

## Generate your own identity

Your relay needs its own Ed25519 keypair — **never reuse a validator's
key**, and never reuse a chat wallet's key either; this is a distinct
node identity:

```bash
cd app/golang
go run ./cmd/keytool
```

Keep the printed seed secret (`VALIDATOR_PRIVATE_KEY_SEED` /
`VALIDATOR_PRIVATE_KEY_FILE`, same zero-trust sourcing every node uses —
never written to disk by the process itself). Note the printed address and
libp2p peer ID.

## Configuration

Relay-specific environment variables (on top of the ones every node needs
— `VALIDATOR_PRIVATE_KEY_SEED`/`_FILE`, `VALIDATOR_SET`,
`P2P_LISTEN_ADDRS`, `GRPC_LISTEN_ADDR`, `GRPC_EXTERNAL_ADDR`, TLS settings
— see `app/golang/internal/config/config.go` and
[app/golang/README.md](../app/golang/README.md)):

| Variable | Required | Meaning |
|---|---|---|
| `NODE_ROLE` | yes | Must be `relay`. |
| `RELAY_REGISTER_WITH` | yes | Comma-separated validator gRPC addresses (`host:port`) to register with and sync the chain from. |
| `NODE_VERSION` | yes | Self-reported version string, checked against the validator's allowlist at registration time. |
| `NODE_COMMIT` | no | Self-reported commit hash — informational only. |
| `RELAY_REGISTRATION_TTL` | no (default `10m`) | How often re-registration happens is half of this; only meaningful if you also run as the "trusted center" side, otherwise informational. |
| `RELAY_PROBE_INSECURE_TLS` | no (default `false`) | Set `true` only if the validators you dial use a private/self-signed cert you haven't pinned a CA for — dev/test networks only. |

Everything else (`P2P_LISTEN_ADDRS`, `GRPC_LISTEN_ADDR`,
`GRPC_EXTERNAL_ADDR`, `DB_PATH`, `METRICS_LISTEN_ADDR`, TLS cert paths)
works exactly like a validator's — a relay runs the full libp2p host and
gRPC gateway, it just skips consensus.

Example (plain `docker run`, adjust for your own compose/Portainer/k8s
setup — see `cicd/docker/portainer-stack.yml` for the validator equivalent
this mirrors):

```bash
docker run -d --name openchat-relay \
  -e NODE_ROLE=relay \
  -e VALIDATOR_PRIVATE_KEY_SEED=<your seed> \
  -e VALIDATOR_SET=<comma-separated validator addresses from the operator> \
  -e RELAY_REGISTER_WITH=openchat.node1.example.com:443,openchat.node2.example.com:443 \
  -e GRPC_EXTERNAL_ADDR=my-relay.example.com:443 \
  -e NODE_VERSION=<version the operator told you to use> \
  -e GRPC_TLS_ENABLED=false \
  -p 4001:4001 -p 9090:9090 -p 2112:2112 \
  -v openchat-relay-data:/data \
  registry.example.com/openchat/node:<tag>
```

(`GRPC_TLS_ENABLED=false` here assumes you're putting your own reverse
proxy in front, same TLS-termination pattern as validators — see
deploy.md section 4. If you'd rather have the node terminate TLS itself,
set `GRPC_TLS_ENABLED=true` and provide `GRPC_TLS_CERT_FILE`/`_KEY_FILE`.)

## What happens after you start it

1. Your node starts libp2p, the gRPC gateway, and — because it's a relay —
   skips straight past consensus into two background jobs instead:
   registration and chain sync.
2. **Registration:** it signs a `RegisterRelay` request with your relay's
   own key and sends it to each address in `RELAY_REGISTER_WITH`. A
   validator accepts it if the signature checks out, your address isn't
   one of its real validators, and your `NODE_VERSION` is on its
   allowlist. Watch your logs for `relay: registered with validator`.
   Common rejection reasons (`relay: registration attempt failed`, with a
   `reason` from the validator): version not allowed (ask the operator
   what they expect), or a stale/skewed system clock (registration
   requests are timestamped and rejected if too old/new).
3. **Chain sync:** your node subscribes to a real-time block-commit feed
   and, in parallel, backfills any gap via the same validators' `GetBlocks`
   RPC — so a cold start (or catching up after downtime) doesn't depend on
   gossip timing. Watch `openchat_chain_height` in your own `/metrics`
   climb to match the network's.
4. **Health checks:** periodically, a validator that accepted your
   registration will connect to *your* node exactly like an ordinary
   client would, send itself a message through you, and confirm it
   arrives back out through your `StreamIncomingSMS` — proving your relay
   actually works end to end, not just that your port is open. You don't
   need to do anything for this; it's entirely validator-initiated. If it
   fails a few times in a row, the network marks your node unhealthy
   (visible to clients via node-status gossip) until a probe succeeds
   again.
5. **Re-registration:** your node re-registers automatically, well before
   its previous registration would be considered stale — no action needed
   as long as the process keeps running.

## Verifying it's working

```bash
curl http://localhost:2112/metrics | grep openchat_chain_height   # should track the network's height
```

Any client pointed at your relay's `GRPC_EXTERNAL_ADDR` should be able to
send and receive messages through it exactly as through a validator's
gateway — that's the actual point of the health-check loop above, so if
registration succeeded and a probe has passed at least once, this already
works.

## Troubleshooting

- **Stuck at height 0 / never climbs:** check that `RELAY_REGISTER_WITH`
  is reachable (TLS handshake succeeding — try `openssl s_client -connect
  host:port` or just watch for dial errors in your logs) and that
  `VALIDATOR_SET` exactly matches what the operator gave you — a wrong or
  incomplete set makes every incoming block fail its quorum-signature
  check.
- **Registration always rejected with a version reason:** your
  `NODE_VERSION` doesn't match what the validator expects — ask the
  operator for the exact string, don't guess.
- **"this address is a consensus validator, not a relay":** you
  accidentally used a validator's own keypair as your relay identity —
  generate a fresh one (see "Generate your own identity" above).
- **Clients can reach your relay but never receive anything:** check that
  chain sync is actually progressing (previous bullet) — messages only
  ever get delivered once the block containing them is committed on your
  node's own local chain copy, not the moment they're sent.
