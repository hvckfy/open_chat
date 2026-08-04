# OpenChat — decentralized E2EE messenger on a custom blockchain

OpenChat is a small blockchain whose transactions *are* encrypted SMS
messages. A static set of validator nodes gossip over libp2p, agree on
blocks via a simplified BFT consensus, and expose a TLS gRPC gateway that
clients use to send and receive messages in real time. All message
content is end-to-end encrypted client-side — the network only ever sees
ciphertext, and there's no token or fee: sending a message is free.

This repo is organized around **one shared core, many frontends**: a
backend/core Go module (`app/golang`) builds the validator/relay node and
the shared client library, and each UI is its own separate module that
depends on that core — today, a working cross-platform GUI client (Fyne,
`app/fyne`). Native shells per OS (Xcode for macOS/iOS, and so on) will
join it the same way, talking to the same node network through the same
core instead of shipping the Fyne UI everywhere.

## Layout

```
app/
  golang/    the backend/core Go module — validator/relay node, CLI
             client, keytool, and the shared client library (pkg/client,
             pkg/crypto) every UI reuses. No UI code, no Fyne dependency.
             See app/golang/README.md.
  fyne/      the current cross-platform GUI (Fyne) — its own Go module,
             depends on app/golang's pkg/client/pkg/crypto via a replace
             directive. Expected to be deprecated as native shells land.
             See app/fyne/README.md.
  xcode/     native macOS/iOS app (SwiftUI). Screen-complete against
             docs/design-spec.md as a Swift package (OpenChatKit); the
             Xcode project itself is a short manual setup step — see
             app/xcode/README.md.
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
go.work      ties app/golang + app/fyne together for local dev (each is
             still a fully standalone Go module on its own)
```

`app/golang` and `app/fyne` are two separate Go modules on purpose: the
GUI's dependency tree (Fyne, cgo/OpenGL, mobile toolchain support) has no
reason to touch the backend's `go.sum` or the node/client Docker build
context, and `app/golang` has no idea `app/fyne` exists — only `app/fyne`
depends on `app/golang`, never the other way around. `cicd/` holds the
deployment artifacts that consume `app/golang`'s build output (Docker
images, k8s manifests) separately from application source, and `docs/`
holds the operator- and design-facing documentation that isn't source
code.

## Where to go next

- **Understand or modify the backend:** [app/golang/README.md](app/golang/README.md)
  — architecture, Clean Architecture layering, backend + client protocol
  details.
- **Understand or modify the GUI app:** [app/fyne/README.md](app/fyne/README.md)
  — what it does, how it depends on `app/golang`, and how to build it for
  each platform.
- **Deploy a network:** [docs/deploy.md](docs/deploy.md) (your own
  validators) or [docs/beanode.md](docs/beanode.md) (join someone else's
  as a relay), backed by the configs in `cicd/`.
- **Design new UI (native or otherwise):** [docs/design-code.md](docs/design-code.md)
  has the functional breakdown of every current screen/dialog.
- **Native macOS/iOS app:** [app/xcode/README.md](app/xcode/README.md) —
  the SwiftUI package (`OpenChatKit`) and the exact steps to wire it into
  a real Xcode project.
