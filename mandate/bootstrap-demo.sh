#!/usr/bin/env bash
# Mandate demo launcher.
#
# One script handles:
# - Gemini ADK mode by default.
# - Smoke mode only when explicitly requested.
# - Docker runtime when a daemon is available.
# - Local runtime fallback with prompted Go/Python install on macOS.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATE_DIR="$ROOT/.mandate-run"
PID_DIR="$STATE_DIR/pids"
LOG_DIR="$STATE_DIR/logs"

RUNTIME="auto"
DEMO_MODE="gemini"
ASSUME_YES="no"
STOP="no"

usage() {
  cat <<'EOF'
Usage:
  ./bootstrap-demo.sh [--yes]
  ./bootstrap-demo.sh --gemini [--yes]
  ./bootstrap-demo.sh --smoke [--yes]
  ./bootstrap-demo.sh --docker [--gemini|--smoke] [--yes]
  ./bootstrap-demo.sh --local [--gemini|--smoke] [--yes]
  ./bootstrap-demo.sh --stop

Default:
  - Uses Docker if a Docker daemon is running.
  - Otherwise runs local Go services + Python virtualenv.
  - Runs the Gemini ADK agent.
  - Prompts for a Gemini API key in interactive terminals and exits if no key is provided.

Get a Gemini API key:
  https://aistudio.google.com/app/apikey
EOF
}

for arg in "$@"; do
  case "$arg" in
    --docker) RUNTIME="docker" ;;
    --local) RUNTIME="local" ;;
    --gemini) DEMO_MODE="gemini" ;;
    --smoke) DEMO_MODE="smoke" ;;
    --yes|-y) ASSUME_YES="yes" ;;
    --stop) STOP="yes" ;;
    --help|-h) usage; exit 0 ;;
    *) usage; exit 1 ;;
  esac
done

docker_ready() {
  command -v docker >/dev/null 2>&1 && docker ps >/dev/null 2>&1
}

confirm() {
  local prompt="$1"
  if [[ "$ASSUME_YES" == "yes" ]]; then
    return 0
  fi
  read -r -p "$prompt [y/N] " answer
  [[ "$answer" == "y" || "$answer" == "Y" || "$answer" == "yes" || "$answer" == "YES" ]]
}

prompt_for_api_key() {
  if [[ -n "${GOOGLE_API_KEY:-}" ]]; then
    return 0
  fi
  if [[ -n "${GEMINI_API_KEY:-}" ]]; then
    export GOOGLE_API_KEY="$GEMINI_API_KEY"
    return 0
  fi
  if [[ "$ASSUME_YES" == "yes" || ! -t 0 ]]; then
    return 1
  fi

  echo ""
  echo "Gemini ADK mode requires a Gemini API key."
  echo "Get one here: https://aistudio.google.com/app/apikey"
  echo "Paste it below. Press Enter without a key to exit."
  read -r -s -p "Gemini API key: " key
  echo ""

  if [[ -n "$key" ]]; then
    export GOOGLE_API_KEY="$key"
    return 0
  fi
  return 1
}

resolve_demo_mode() {
  if [[ "$DEMO_MODE" == "gemini" ]]; then
    prompt_for_api_key || {
      echo "Error: Gemini ADK mode requires GOOGLE_API_KEY."
      echo "Create one at https://aistudio.google.com/app/apikey, then run:"
      echo "  export GOOGLE_API_KEY=your-key"
      echo "  ./bootstrap-demo.sh --gemini"
      echo ""
      echo "For local verifier testing only, run:"
      echo "  ./bootstrap-demo.sh --smoke --yes"
      exit 1
    }
  fi
}

resolve_runtime() {
  if [[ "$RUNTIME" == "auto" ]]; then
    if docker_ready; then
      RUNTIME="docker"
    else
      RUNTIME="local"
    fi
  fi

  if [[ "$RUNTIME" == "docker" ]] && ! docker_ready; then
    echo "Error: Docker runtime requested, but no Docker daemon is available."
    echo "Start Docker Desktop/Colima, or use: ./bootstrap-demo.sh --local"
    exit 1
  fi
}

install_missing_local_deps() {
  local missing=()
  command -v go >/dev/null 2>&1 || missing+=("go")
  command -v python3 >/dev/null 2>&1 || missing+=("python")

  if [[ ${#missing[@]} -eq 0 ]]; then
    return
  fi

  if [[ "$(uname -s)" == "Darwin" && -x "$(command -v brew || true)" ]]; then
    echo "Missing local runtime package(s): ${missing[*]}"
    if confirm "Install with Homebrew now?"; then
      brew install "${missing[@]}"
      return
    fi
  fi

  echo "Error: local runtime requires Go and Python 3."
  echo "Install missing package(s): ${missing[*]}"
  exit 1
}

wait_for_port() {
  local name="$1"
  local port="$2"

  for _ in $(seq 1 40); do
    if nc -z localhost "$port" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.25
  done

  echo "Error: $name did not start on port $port."
  echo "Log: $LOG_DIR/$name.log"
  [[ -f "$LOG_DIR/$name.log" ]] && tail -40 "$LOG_DIR/$name.log"
  exit 1
}

start_local_service() {
  local name="$1"
  local port="$2"
  local dir="$3"
  shift 3

  mkdir -p "$PID_DIR" "$LOG_DIR"

  if nc -z localhost "$port" >/dev/null 2>&1; then
    echo "  $name already listening on :$port"
    return
  fi

  echo "  starting $name on :$port"
  (
    cd "$dir"
    nohup "$@" >"$LOG_DIR/$name.log" 2>&1 &
    echo $! >"$PID_DIR/$name.pid"
  )
  wait_for_port "$name" "$port"
}

stop_demo() {
  echo "Stopping Mandate demo..."

  if docker_ready; then
    docker compose -f "$ROOT/docker-compose.yml" down >/dev/null 2>&1 || true
  fi

  if [[ -d "$PID_DIR" ]]; then
    for pid_file in "$PID_DIR"/*.pid; do
      [[ -f "$pid_file" ]] || continue
      pid="$(cat "$pid_file")"
      kill "$pid" >/dev/null 2>&1 || true
      rm -f "$pid_file"
    done
  fi

  pkill -f "ratify-demos/mandate/.*/go run" >/dev/null 2>&1 || true
  pkill -f "ratify-demos/mandate/.*/http.server 8010" >/dev/null 2>&1 || true
  pkill -f "http.server 8010" >/dev/null 2>&1 || true

  if command -v lsof >/dev/null 2>&1; then
    for port in 8090 8091 8099 8010; do
      while IFS= read -r pid; do
        [[ -n "$pid" ]] && kill "$pid" >/dev/null 2>&1 || true
      done < <(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)
    done
  fi

  echo "Done."
}

install_python_deps() {
  local deps="agent/requirements.txt"
  if [[ "$DEMO_MODE" == "smoke" ]]; then
    deps="agent/requirements-smoke.txt"
  fi

  if [[ ! -x "$ROOT/.venv/bin/python" ]]; then
    echo "Creating Python virtualenv..."
    python3 -m venv "$ROOT/.venv"
  fi

  echo "Installing Python dependencies: $deps"
  "$ROOT/.venv/bin/python" -m pip install --upgrade pip >/dev/null
  "$ROOT/.venv/bin/python" -m pip install -r "$ROOT/$deps" >/dev/null
}

print_banner() {
  echo ""
  echo "╔══════════════════════════════════════════════════╗"
  echo "║   MANDATE — Ratify Platform Demo                ║"
  echo "║   Gemini ADK + MCP + Ratify verification        ║"
  echo "╚══════════════════════════════════════════════════╝"
  echo ""
  echo "Runtime: $RUNTIME"
  echo "Mode:    $DEMO_MODE"
  echo ""
}

print_urls() {
  echo ""
  echo "Dashboard:       http://localhost:8010"
  echo "CloudOps MCP:    http://localhost:8090/mcp"
  echo "Hardware MCP:    http://localhost:8091/mcp"
  echo "Event relay:     http://localhost:8099/events"
  echo ""
}

run_docker_demo() {
  docker compose -f "$ROOT/docker-compose.yml" up -d --build cloudops-mcp vendor-mcp event-relay dashboard
  print_urls

  if [[ "$ASSUME_YES" != "yes" ]]; then
    read -r -p "Open the dashboard, then press Enter to run the agent..."
  fi

  if [[ "$DEMO_MODE" == "smoke" ]]; then
    docker compose -f "$ROOT/docker-compose.yml" --profile runner run --rm agent \
      sh -lc "python setup.py >/dev/null && python smoke.py"
  else
    docker compose -f "$ROOT/docker-compose.yml" --profile runner run --rm agent \
      sh -lc "python -m pip install -r requirements.txt >/dev/null && python setup.py >/dev/null && python main.py"
  fi
}

run_local_demo() {
  install_missing_local_deps
  install_python_deps

  echo "Starting local services..."
  "$ROOT/.venv/bin/python" "$ROOT/agent/setup.py" >/dev/null
  start_local_service "event-relay" 8099 "$ROOT/events" go run .
  start_local_service "cloudops-mcp" 8090 "$ROOT/mcp-server" env EVENTS_URL=http://localhost:8099/event go run .
  start_local_service "vendor-mcp" 8091 "$ROOT/vendor-mcp-server" env EVENTS_URL=http://localhost:8099/event go run .
  start_local_service "dashboard" 8010 "$ROOT/dashboard-server" go run . -dir "$ROOT/dashboard"
  print_urls

  if [[ "$ASSUME_YES" != "yes" ]]; then
    read -r -p "Open the dashboard, then press Enter to run the agent..."
  fi

  if [[ "$DEMO_MODE" == "smoke" ]]; then
    CLOUDOPS_MCP_URL="http://localhost:8090/mcp" \
    VENDOR_MCP_URL="http://localhost:8091/mcp" \
    "$ROOT/.venv/bin/python" "$ROOT/agent/smoke.py"
  else
    CLOUDOPS_MCP_URL="http://localhost:8090/mcp" \
    VENDOR_MCP_URL="http://localhost:8091/mcp" \
    "$ROOT/.venv/bin/python" "$ROOT/agent/main.py"
  fi
}

if [[ "$STOP" == "yes" ]]; then
  stop_demo
  exit 0
fi

resolve_demo_mode
resolve_runtime
print_banner

if [[ "$RUNTIME" == "docker" ]]; then
  run_docker_demo
else
  run_local_demo
fi

echo ""
echo "Demo complete. Dashboard remains available until you run:"
echo "  ./bootstrap-demo.sh --stop"
