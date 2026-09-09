#!/usr/bin/env bash
# Run a local OCI registry that both the host and a kind cluster can reach.
#
# Crossplane requires a fully qualified package reference, and kind nodes
# cannot see images that only exist in the host's docker daemon. A registry
# on localhost:5001, wired into the kind network and into each node's
# containerd host config, satisfies both. Follows the pattern from
# https://kind.sigs.k8s.io/docs/user/local-registry/.
#
# Usage: e2e/kind-registry.sh <kind-cluster-name>
set -euo pipefail

CLUSTER="${1:?kind cluster name}"
REG_NAME="${KIND_REGISTRY_NAME:-kind-registry}"
REG_PORT="${KIND_REGISTRY_PORT:-5001}"

if [ "$(docker inspect -f '{{.State.Running}}' "$REG_NAME" 2>/dev/null || true)" != "true" ]; then
  docker run -d --restart=always -p "127.0.0.1:${REG_PORT}:5000" --network bridge --name "$REG_NAME" registry:2 >/dev/null
fi

# Each node gets a hosts.toml that maps localhost:<port> to the registry
# container's in-network address.
REGISTRY_DIR="/etc/containerd/certs.d/localhost:${REG_PORT}"
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
    host: "localhost:${REG_PORT}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

echo "registry ready at localhost:${REG_PORT}" >&2
