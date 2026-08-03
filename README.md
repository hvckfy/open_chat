# OpenChat — decentralized E2EE messenger on a custom blockchain

OpenChat is a small blockchain whose transactions *are* encrypted SMS
messages. A static set of validator nodes gossip over libp2p, agree on
blocks via a simplified BFT consensus, and expose a TLS gRPC gateway that
clients use to send and receive messages in real time. All message
content is end-to-end encrypted client-side — the network only ever sees
ciphertext, and there's no token or fee: sending a message is free.

This repo is organized around **one shared core, many frontends**: the
Go module today builds both the validator/relay node *and* a working
cross-platform GUI client (Fyne). The plan going forward is to keep the
Go module as the network/protocol core and add native shells per OS
(Xcode for macOS/iOS, and so on) that talk to the same node network
instead of shipping the Fyne UI everywhere.

## Layout

```
app/
  golang/    the Go module — validator/relay node, CLI client, shared
             client library (pkg/client, pkg/crypto), and the current
             cross-platform GUI app (Fyne). See app/golang/README.md.
  xcode/     future native macOS/iOS app. Empty for now — see
             app/xcode/README.md for the plan.
cicd/
  docker/    Dockerfiles, docker-compose demo, Portainer stack, dev certs
  k8s/       Kubernetes manifests (StatefulSet, ConfigMap, Secret, Services)
  nginx/     nginx config examples for TLS-terminating reverse-proxy setups
  scripts/   build-nodes.sh — builds/tags/pushes the Docker images
docs/
  design-code.md   functional spec of every app screen/dialog, written to
                   feed into a design tool (claude-design) for visuals
  deploy.md        standing up your own independent OpenChat network
  beanode.md       running a relay node to join someone else's network
```

Nothing about this split changes how the code works — `app/golang` is
still a normal, self-contained Go module (`go build ./...` from inside
it); `cicd/` just holds the deployment artifacts that consume its build
output (Docker images, k8s manifests) separately from the application
source, and `docs/` holds the operator- and design-facing documentation
that isn't source code.

## Where to go next

- **Understand or modify the code:** [app/golang/README.md](app/golang/README.md)
  — architecture, Clean Architecture layering, backend + client protocol
  details, and how to build the GUI app for each platform.
- **Deploy a network:** [docs/deploy.md](docs/deploy.md) (your own
  validators) or [docs/beanode.md](docs/beanode.md) (join someone else's
  as a relay), backed by the configs in `cicd/`.
- **Design new UI (native or otherwise):** [docs/design-code.md](docs/design-code.md)
  has the functional breakdown of every current screen/dialog.
- **Native macOS/iOS app:** [app/xcode/README.md](app/xcode/README.md) —
  not started yet; read there for the plan and what's needed before work
  can begin.
