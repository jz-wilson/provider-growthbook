#!/usr/bin/env bash
# Run a local OCI registry that both the host and a kind cluster can reach.
#
# Crossplane requires a fully qualified package reference, and kind nodes
# cannot see images that only exist in the host's docker daemon. A registry
# reachable as 127.0.0.1:5001, wired into the kind network and into each
# node's containerd host config, satisfies both. Follows the pattern from
# https://kind.sigs.k8s.io/docs/user/local-registry/.
#
# Two names for one registry:
#   127.0.0.1:5001         host side, for pushing with the Crossplane CLI
#   <kind-net-ip>:5000     cluster side, for Crossplane's spec.package
# The cluster-side name has to be an address pods can dial directly, because
# Crossplane's package manager fetches the package over HTTP itself rather
# than through containerd (so a containerd hosts.toml redirect alone is not
# enough), and it has to contain a dot to pass the spec.package CEL rule
# (^[^./]+(\.[^./]+)+/...), which rules out "localhost". The registry
# container's IP on the kind docker network satisfies both. The provider
# pod image is then pulled by containerd from the same name, so each node
# also gets a hosts.toml for it.
#
# Prints the cluster-side registry address (host:port) on stdout.
#
# Usage: e2e/kind-registry.sh <kind-cluster-name>
set -euo pipefail

CLUSTER="${1:?kind cluster name}"
REG_NAME="${KIND_REGISTRY_NAME:-kind-registry}"
REG_PORT="${KIND_REGISTRY_PORT:-5001}"

if [ "$(docker inspect -f '{{.State.Running}}' "$REG_NAME" 2>/dev/null || true)" != "true" ]; then
  # Pull separately so image download chatter never lands on stdout.
  docker pull -q registry:2 >&2
  docker run -d --restart=always -p "127.0.0.1:${REG_PORT}:5000" --network bridge --name "$REG_NAME" registry:2 >/dev/null
fi

if [ "$(docker inspect -f '{{json .NetworkSettings.Networks.kind}}' "$REG_NAME")" = 'null' ]; then
  docker network connect kind "$REG_NAME"
fi
REG_IP=$(docker inspect -f '{{.NetworkSettings.Networks.kind.IPAddress}}' "$REG_NAME")
[ -n "$REG_IP" ] || { echo "registry has no address on the kind network" >&2; exit 1; }
CLUSTER_REG="${REG_IP}:5000"

# Each node gets a hosts.toml telling containerd to use plain HTTP for the
# cluster-side name.
REGISTRY_DIR="/etc/containerd/certs.d/${CLUSTER_REG}"
for node in $(kind get nodes --name "$CLUSTER"); do
  docker exec "$node" mkdir -p "$REGISTRY_DIR"
  cat <<EOF | docker exec -i "$node" cp /dev/stdin "${REGISTRY_DIR}/hosts.toml"
server = "http://${CLUSTER_REG}"

[host."http://${CLUSTER_REG}"]
EOF
done

# Advertise the registry to in-cluster tooling (kind convention).
kubectl apply -f - >&2 <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "127.0.0.1:${REG_PORT}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

echo "registry ready: push to 127.0.0.1:${REG_PORT}, cluster pulls from ${CLUSTER_REG}" >&2
printf '%s\n' "$CLUSTER_REG"
