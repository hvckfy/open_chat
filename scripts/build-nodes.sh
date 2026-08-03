#!/usr/bin/env bash
#
# build-nodes.sh — builds and (optionally) pushes the openchat-node and
# openchat-client Docker images, tagged consistently with everything else
# this repo builds (scripts/build-client.sh derives the same
# `git describe`-based version string for its binary/zip artifacts).
#
# Every image gets tagged TWICE: once with the resolved version (so a
# specific build is always addressable/pinnable — this matters a lot once
# relay nodes are in the picture, since a validator's RELAY_ALLOWED_VERSIONS
# allowlist checks a relay's self-reported NODE_VERSION against exact
# strings) and once with `:latest` (so "just pull the newest" keeps working
# for local/dev use). Never deploy production validators or relays against
# a bare `:latest` tag — pin the version tag instead, so you know exactly
# what's running and can roll back deterministically.
#
# Usage:
#   ./scripts/build-nodes.sh                # build only
#   ./scripts/build-nodes.sh --push         # build and push both tags
#   TAG=v1.2.3 ./scripts/build-nodes.sh      # override the version tag
#   REGISTRY=myregistry.example.com/openchat ./scripts/build-nodes.sh
#
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

if [ ! -f go.mod ]; then
  echo "error: $REPO_ROOT doesn't look like the openchat repo root (no go.mod)" >&2
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "error: docker not found on PATH" >&2
  exit 1
fi

PUSH=0
for arg in "$@"; do
  case "$arg" in
    --push) PUSH=1 ;;
    *) echo "unknown argument: $arg" >&2; exit 1 ;;
  esac
done

# Same version-derivation scheme as scripts/build-client.sh, so a binary
# build and the Docker images built from the same commit always carry the
# same version string.
TAG="${TAG:-$(git describe --tags --always --dirty 2>/dev/null || date +%Y%m%d%H%M%S)}"
REGISTRY="${REGISTRY:-registry.lan.mftkhv.ru/openchat}"

echo "==> version tag: $TAG"
echo "==> registry:    $REGISTRY"

build_and_tag() {
  local dockerfile="$1" name="$2"
  echo "==> building $REGISTRY/$name:$TAG"
  docker build -f "$dockerfile" -t "$REGISTRY/$name:$TAG" -t "$REGISTRY/$name:latest" .
}

# openchat-node: the validator/relay binary — see internal/config for the
# NODE_VERSION/NODE_COMMIT env vars a running container should be given so
# what it reports over gRPC (GetAddress, RegisterRelay) actually matches
# this image's tag.
build_and_tag deployments/docker/Dockerfile.node node

# openchat-client: CLI client image, used by the docker-compose demo
# network — not required for a production deployment, but kept in sync
# with the same tagging scheme.
build_and_tag deployments/docker/Dockerfile.client client

if [ "$PUSH" -eq 1 ]; then
  for name in node client; do
    echo "==> pushing $REGISTRY/$name:$TAG"
    docker push "$REGISTRY/$name:$TAG"
    echo "==> pushing $REGISTRY/$name:latest"
    docker push "$REGISTRY/$name:latest"
  done
else
  echo "==> built (not pushed — pass --push to also push both tags)"
fi

echo
echo "Done. Images:"
echo "  $REGISTRY/node:$TAG   (and :latest)"
echo "  $REGISTRY/client:$TAG (and :latest)"
echo
echo "Remember to also set NODE_VERSION=$TAG (and ideally NODE_COMMIT=\$(git rev-parse --short HEAD))"
echo "in whatever compose/Portainer/k8s config deploys this image, so the"
echo "running process reports the same version it was actually built from."
