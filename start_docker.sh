#!/bin/bash

if [[ "$(uname)" == "Darwin" ]]; then
    # Added | head -n 1 to grab only the first matching IP
    export HOST_IP=$(ifconfig | grep 'inet ' | grep -v '127.0.0.1' | grep -v 'docker' | grep -v 'veth' | awk '$2 !~ /^172\./ {print $2}' | head -n 1)
else
    # Added | head -n 1 to grab only the first matching IP
    export HOST_IP=$(ip a | grep 'inet ' | grep -v '127.0.0.1' | grep -v 'docker' | grep -v 'veth' | grep -v 'br-' | awk '{print $2}' | cut -d'/' -f1 | head -n 1)
fi

if [ -z "$HOST_IP" ]; then
    echo "Could not find a valid host IP address."
    exit 1
fi

echo "Using host IP: $HOST_IP"

docker compose up