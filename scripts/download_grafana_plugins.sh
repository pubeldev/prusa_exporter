#!/usr/bin/env bash
set -euo pipefail

# Base dir = repo root (scripts/..)
BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PLUGINS_DIR="${BASE_DIR}/docs/config/grafana/plugins"

# Temp dir for downloads
TMP_DIR="$(mktemp -d)"
cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT

mkdir -p "$PLUGINS_DIR"

plugins=(
  dalvany-image-panel
  grafana-xyzchart-panel
  grafana-polystat-panel
  yesoreyeram-infinity-datasource
  volkovlabs-image-panel
  betatech-qrcode-panel
)

for id in "${plugins[@]}"; do
  echo "Downloading $id (latest)…"
  curl -fL -# "https://grafana.com/api/plugins/${id}/versions/latest/download" \
    -o "${TMP_DIR}/${id}.zip"
  # ensure per-plugin dir exists (unzip can create it, but being explicit is nice)
  mkdir -p "${PLUGINS_DIR}/${id}"
  unzip -q -o "${TMP_DIR}/${id}.zip" -d "${PLUGINS_DIR}/${id}"
done

chown -R 472:root "$PLUGINS_DIR"
chmod -R 750 "$PLUGINS_DIR"
echo "Done. Plugins in: $PLUGINS_DIR"

