#!/bin/bash

set -eou pipefail

if [[ "$(uname)" == "Darwin" ]]; then
    INTERFACES=$(ifconfig)
    export HOST_IP=$(echo "$INTERFACES" | grep 'inet ' | grep -v '127.0.0.1' | grep -v 'docker' | grep -v 'veth' | awk '$2 !~ /^172\./ {print $2}')
else
    export HOST_IP=$(ip -o -4 addr show | grep -Ev '127.0.0.1|docker' | awk -F'[ /]+' '{print $4}')
fi

if [ -z "$HOST_IP" ]; then
    echo "Could not find a valid host IP address."
    exit 1
fi

echo "Using host IP: $HOST_IP"

cd "$(dirname "$0")"
docker compose -f ../docker-compose.yaml up -d
