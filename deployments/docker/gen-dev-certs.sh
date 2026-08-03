#!/usr/bin/env bash
# Generates a throwaway self-signed CA + server certificate for the
# docker-compose demo network (node1/node2). NOT for production use — in
# production each validator gets a cert from a real/private CA and the
# private key never leaves the orchestrator's secret store.
set -euo pipefail

cd "$(dirname "$0")"
OUT=./certs
mkdir -p "$OUT"

if [[ -f "$OUT/tls.crt" ]]; then
  echo "certs already exist in $OUT, skipping (delete the directory to regenerate)"
  exit 0
fi

openssl req -x509 -newkey ed25519 -nodes \
  -keyout "$OUT/tls.key" \
  -out "$OUT/tls.crt" \
  -days 365 \
  -subj "/CN=openchat-dev" \
  -addext "subjectAltName=DNS:node1,DNS:node2,DNS:localhost,IP:127.0.0.1"

echo "wrote $OUT/tls.crt and $OUT/tls.key (shared by node1 + node2 for this demo)"
