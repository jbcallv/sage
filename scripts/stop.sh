#!/usr/bin/env bash
set -euo pipefail

echo "removing iptables rules..."
BRIDGE_ID=$(docker network inspect tetragon_agent-net --format '{{.Id}}' 2>/dev/null | cut -c1-12 || true)
if [ -n "$BRIDGE_ID" ]; then
    BRIDGE_IF="br-${BRIDGE_ID}"
    PROXY_IP=$(docker inspect tetragon-proxy-1 --format '{{json .NetworkSettings.Networks}}' 2>/dev/null \
        | python3 -c "import sys,json; nets=json.load(sys.stdin); print(next(v['IPAddress'] for k,v in nets.items() if 'agent' in k))" 2>/dev/null || true)
    if [ -n "$PROXY_IP" ]; then
        sudo iptables -t nat -D PREROUTING \
            -i "$BRIDGE_IF" -p tcp \
            ! -d "$PROXY_IP" ! -s "$PROXY_IP" \
            -j DNAT --to-destination "${PROXY_IP}:8080" 2>/dev/null && echo "  removed" || echo "  not found, skipping"
    fi
fi

echo "resetting bridge netfilter..."
sudo sysctl -w net.bridge.bridge-nf-call-iptables=0 2>/dev/null || true

docker compose down

for f in policies/*.yaml; do
    name=$(grep 'name:' "$f" | awk '{print $2}')
    sudo tetra tracingpolicy delete "$name" 2>/dev/null && echo "removed policy $name" || true
done

sudo systemctl stop tetragon
