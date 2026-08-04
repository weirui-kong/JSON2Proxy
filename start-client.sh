#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CONFIG_PATH="${1:-$SCRIPT_DIR/.private/configs/client.json}"

if [[ ! -f "$CONFIG_PATH" ]]; then
  echo "Client config not found: $CONFIG_PATH" >&2
  echo "Create it from the public template:" >&2
  echo "  mkdir -p \"$SCRIPT_DIR/.private/configs\"" >&2
  echo "  cp \"$SCRIPT_DIR/configs/client.example.json\" \"$SCRIPT_DIR/.private/configs/client.json\"" >&2
  exit 1
fi

echo "Starting JSON2Proxy client in the foreground (not installed as a service)."
exec go run "$SCRIPT_DIR/cmd/json2proxy-client" -config "$CONFIG_PATH"
