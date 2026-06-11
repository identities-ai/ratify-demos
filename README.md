# Mandate Demo

This repository contains **Mandate: Bounded Authority for Autonomous Agents**, a Google AI Agents Challenge demo powered by the open-source Ratify Protocol.

Mandate is the agent project. Ratify is the cryptographic authorization layer. The dashboard is branded **Ratify Verify** because it visualizes the live verification/audit feed from the MCP tools.

## Fastest Run

For the full Google ADK agent path:

```bash
cd mandate
./bootstrap-demo.sh --yes
```

If `GOOGLE_API_KEY` is not set, the script prompts you to paste a Gemini API key. Get one from:

```text
https://aistudio.google.com/app/apikey
```

Then open:

```text
http://localhost:8010
```

For a deterministic verification run without a Gemini key:

```bash
cd mandate
./bootstrap-demo.sh --smoke --yes
```

Then open:

```text
http://localhost:8010
```

The bootstrap script prefers Docker Compose. If no Docker daemon is running, it creates a local Python virtualenv and runs the Go MCP services directly. On macOS, if Go or Python 3 are missing, it prompts to install them with Homebrew.

When `GOOGLE_API_KEY` is set, the default path runs the Gemini ADK agent. Without a key, use `--smoke` to exercise the exact same MCP servers and Ratify verification logic without model quota, latency, or prompt variance.

## Self-Contained Docker Path

Use this path when Docker Desktop, Colima, or another Docker daemon is already running:

```bash
cd mandate
./bootstrap-demo.sh --smoke --yes
```

Stop services:

```bash
./bootstrap-demo.sh --stop
```

## Live Gemini ADK Path

The hackathon story uses Gemini/ADK for the autonomous incident commander. Run that path with:

```bash
cd mandate
export GOOGLE_API_KEY=your-gemini-api-key
./bootstrap-demo.sh --gemini --yes
```

You can also set `GEMINI_API_KEY`; the bootstrap script maps it to `GOOGLE_API_KEY` for the demo.

This is the primary hackathon demo path. The smoke path exists as the deterministic test harness for evaluators and CI.

## No-Docker Local Path

Requires Go and Python 3:

```bash
cd mandate
./bootstrap-demo.sh --local --yes
```

## Expected Output

```text
[PASS] cloud provision within mandate: approved
[PASS] cloud over-provision denied: constraint_denied
[PASS] hardware vendor cross-org approval: approved

MANDATE SMOKE DEMO PASSED
```

## What The Demo Proves

The agent can act autonomously through MCP tools, but the receiving system independently verifies whether each action is authorized. One safe cloud provisioning action succeeds, one out-of-bounds action is denied before execution, and a separate vendor MCP server verifies the same style of mandate across an organizational boundary.

## Where Ratify Is Used

Mandate is also a companion demo for the open-source Ratify Protocol:

- The Python agent code imports the Ratify Python SDK (`ratify_protocol`) to generate identities, issue delegated mandates, sign fresh challenges, and assemble proof bundles.
- The CloudOps MCP server imports the Ratify Go SDK (`github.com/identities-ai/ratify-protocol`) and calls `ratify.Verify(...)` before provisioning.
- The HardwareVendor MCP server imports the same Go SDK and verifies the mandate independently across an org boundary.
- The optional Pi verifier uses the C SDK path to show the same proof pattern in a physical/offline setting.

So the demo is not just a UI mock: every approve/deny row comes from SDK-backed verification in the MCP servers.
