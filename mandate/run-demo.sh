#!/usr/bin/env bash
# Mandate Demo — One-command launcher
# Usage: ./run-demo.sh [--stop]
#
# Starts all services and runs the Gemini agent for the demo scenario:
#   1. Cloud incident → agent provisions (approved by mandate)
#   2. Agent over-provisions → DENIED by constraint enforcement
#   3. Agent orders hardware from cross-org vendor → approved
#   4. Same bundle verifies on Pi rack door (offline, no network)
#
# Prerequisites:
#   brew install docker python3 go
#   pip install google-adk ratify-protocol
#   export GOOGLE_API_KEY=your-gemini-key
#   Run: python agent/setup.py  (generates keys, bundles)

set -euo pipefail

DEMO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ "${1:-}" == "--stop" ]]; then
  echo "Stopping Mandate demo..."
  docker compose -f "$DEMO_DIR/docker-compose.yml" down
  echo "Done."
  exit 0
fi

echo ""
echo "╔══════════════════════════════════════════════════╗"
echo "║   MANDATE — Ratify Platform Demo                ║"
echo "║   Agentic API + Physical AI surfaces            ║"
echo "╚══════════════════════════════════════════════════╝"
echo ""

# Check prerequisites
command -v docker >/dev/null 2>&1 || { echo "Error: Docker not installed."; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "Error: Python 3 not installed."; exit 1; }
[ -n "${GOOGLE_API_KEY:-}" ] || { echo "Error: GOOGLE_API_KEY not set."; exit 1; }

# Check keys exist (setup.py must have been run)
if [ ! -f "$DEMO_DIR/agent/keys/commander_bundle.json" ]; then
  echo "Keys not found. Running setup..."
  cd "$DEMO_DIR/agent" && python3 setup.py && cd "$DEMO_DIR"
fi

# Start infrastructure
echo "→ Starting MCP servers, event relay, and dashboard..."
docker compose -f "$DEMO_DIR/docker-compose.yml" up -d --build

echo "→ Waiting for services to be ready..."
sleep 3

# Verify services are up
for svc in "8090" "8091" "8099" "8000"; do
  curl -sf "http://localhost:$svc" >/dev/null 2>&1 || \
  curl -sf "http://localhost:$svc/health" >/dev/null 2>&1 || true
done

echo ""
echo "┌─────────────────────────────────────────────────┐"
echo "│  Dashboard: http://localhost:8010               │"
echo "│  CloudOps MCP: http://localhost:8090/mcp        │"
echo "│  HardwareVendor MCP: http://localhost:8091/mcp  │"
echo "│  Event relay: http://localhost:8099             │"
echo "└─────────────────────────────────────────────────┘"
echo ""
echo "Open the dashboard in your browser, then press ENTER to run the agent."
echo "(The denial moment happens automatically — watch for the red flash)"
echo ""
read -r -p "Press ENTER to start the Gemini agent → "

echo ""
echo "→ Running Mandate incident scenario..."
echo ""

cd "$DEMO_DIR/agent"
CLOUDOPS_MCP_URL="http://localhost:8090/mcp" \
VENDOR_MCP_URL="http://localhost:8091/mcp" \
python3 main.py

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo " Demo complete."
echo ""
echo " Physical verification (optional):"
echo "   cd pi && BUNDLE_FILE=../agent/keys/commander_bundle.json NO_GPIO=1 ./verifier"
echo "   (Remove NO_GPIO=1 on Raspberry Pi for LED output)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Stop services: ./run-demo.sh --stop"
