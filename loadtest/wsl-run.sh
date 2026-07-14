#!/bin/bash
set -e

RATE=${1:-25}
DURATION=${2:-30s}
NAME=${3:-baseline}

HOST_IP=$(grep nameserver /etc/resolv.conf | awk '{print $2}')
TARGET_URL="http://${HOST_IP}:8080/ingest"
RESULTS_DIR="$(dirname "$0")/results"
mkdir -p "$RESULTS_DIR"

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BIN="${RESULTS_DIR}/${NAME}-${TIMESTAMP}.bin"
REPORT="${RESULTS_DIR}/${NAME}-${TIMESTAMP}.txt"

BODY_FILE="$(dirname "$0")/ingest-body.json"

echo "Target: ${TARGET_URL}"
echo "Rate: ${RATE}/s | Duration: ${DURATION} | Name: ${NAME}"
echo "---"

BODY=$(cat "$BODY_FILE")

vegeta attack \
  -rate "${RATE}/s" \
  -duration "${DURATION}" \
  -output "$BIN" \
  "${TARGET_URL}" \
  -method POST \
  -header "Content-Type: application/json" \
  -body "$BODY"

vegeta report "$BIN" | tee "$REPORT"

echo ""
echo "Saved: ${BIN}"
echo "Saved: ${REPORT}"
