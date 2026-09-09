#!/usr/bin/env bash
# Run a local OCI registry that both the host and a kind cluster can reach.
#
# Crossplane requires a fully qualified package reference, and kind nodes
# cannot see images that only exist in the host's docker daemon. A registry
# reachable as 127.0.0.1:5001, wired into the kind network and into each
# node's containerd host config, satisfies both. Follows the pattern from
# https://kind.sigs.k8s.io/docs/user/local-registry/.
#
# The reference host is 127.0.0.1 rather than localhost on purpose:
# Crossplane's spec.package CEL rule demands a dot in the registry host
# (^[^./]+(\.[^./]+)+/...), which "localhost:5001" fails and an IP passes.
# Nodes never dial 127.0.0.1:5001 themselves; hosts.toml redirects that
# name to the registry container.
#
# Usage: e2e/kind-registry.sh <kind-cluster-name>
set -euo pipefail

CLUSTER="${1:?kind cluster name}"
REG_NAME="${KIND_REGISTRY_NAME:-kind-registry}"
REG_HOST="${KIND_REGISTRY_HOST:-127.0.0.1}"
REG_PORT="${KIND_REGISTRY_PORT:-5001}"

if [ "$(docker inspect -f '{{.State.Running}}' "$REG_NAME" 2>/dev/null || true)" != "true" ]; then
  docker run -d --restart=always -p "127.0.0.1:${REG_PORT}:5000" --network bridge --name "$REG_NAME" registry:2 >/dev/null
fi

# Each node gets a hosts.toml that maps localhost:<port> to the registry
# container's in-network address.
REGISTRY_DIR="/etc/containerd/certs.d/${REG_HOST}:${REG_PORT}"
for node in $(kind get nodes --name "$CLUSTER"); do
  docker exec "$node" mkdir -p "$REGISTRY_DIR"
  cat <<EOF | docker exec -i "$node" cp /dev/stdin "${REGISTRY_DIR}/hosts.toml"
[host."http://${REG_NAME}:5000"]
EOF
done

if [ "$(docker inspect -f '{{json .NetworkSettings.Networks.kind}}' "$REG_NAME")" = 'null' ]; then
  docker network connect kind "$REG_NAME"
fi

# Advertise the registry to in-cluster tooling (kind convention).
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "${REG_HOST}:${REG_PORT}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

echo "registry ready at ${REG_HOST}:${REG_PORT}" >&2
