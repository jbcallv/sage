#!/usr/bin/env bash
set -euo pipefail

echo "starting tetragon..."
sudo systemctl start tetragon
sleep 3

echo "loading policies..."
for f in policies/*.yaml; do
    sudo tetra tracingpolicy add "$f" && echo "  loaded $f"
done

echo "starting containers..."
docker compose up -d --build

echo "waiting for proxy..."
until docker inspect tetragon-proxy-1 --format '{{.State.Health.Status}}' 2>/dev/null | grep -q healthy; do
    sleep 1
done

echo "enabling bridge netfilter..."
sudo modprobe br_netfilter
sudo sysctl -w net.bridge.bridge-nf-call-iptables=1

echo "applying iptables redirect..."
BRIDGE_ID=$(docker network inspect tetragon_agent-net --format '{{.Id}}' | cut -c1-12)
BRIDGE_IF="br-${BRIDGE_ID}"
PROXY_IP=$(docker inspect tetragon-proxy-1 --format '{{json .NetworkSettings.Networks}}' \
    | python3 -c "import sys,json; nets=json.load(sys.stdin); print(next(v['IPAddress'] for k,v in nets.items() if 'agent' in k))")

# redirect all TCP on agent-net to proxy, except traffic to/from proxy itself
sudo iptables -t nat -A PREROUTING \
    -i "$BRIDGE_IF" -p tcp \
    ! -d "$PROXY_IP" ! -s "$PROXY_IP" \
    -j DNAT --to-destination "${PROXY_IP}:8080"

echo "  ${BRIDGE_IF} → ${PROXY_IP}:8080"
echo "ready."
