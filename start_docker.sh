#!/bin/bash

if [[ "$(uname)" == "Darwin" ]]; then
    INTERFACES=$(ifconfig)
    export HOST_IP=$(echo "$INTERFACES" | grep 'inet ' | grep -v '127.0.0.1' | grep -v 'docker' | grep -v 'veth' | awk '$2 !~ /^172\./ {print $2}')
else
    INTERFACES=$(ip a)
    export HOST_IP=$(echo "$INTERFACES" | grep 'inet ' | grep -v '127.0.0.1' | grep -v 'docker' | grep -v 'veth' | grep -v 'br-' | awk '{print $2}' | cut -d'/' -f1)
fi

if [ -z "$HOST_IP" ]; then
    echo "Could not find a valid host IP address."
    exit 1
fi

echo "Using host IP: $HOST_IP"

docker compose up