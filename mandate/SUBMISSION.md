# Mandate: Bounded Authority for Autonomous Agents

## Elevator Pitch

Mandate is a Gemini ADK incident-response agent that can provision cloud infrastructure, coordinate a third-party hardware vendor, and authorize physical access only when each action carries a valid cryptographic mandate.

Instead of giving agents broad API keys, Mandate gives them bounded, portable authority that every MCP tool can verify before execution.

## Track

Track 1: Build (Net-New Agents)

## The Problem

AI agents are ready to act, but production authorization is still built for static services. API keys and service accounts can identify a caller, but they do not prove that a specific agent was authorized by a specific human to perform a specific action under specific constraints.

That is why many enterprise agents stay trapped in prototype. The agent may be capable, but the receiving system cannot independently verify what the agent was allowed to do.

## What We Built

Mandate is an autonomous infrastructure operations demo built with:

- Gemini and Google ADK for the incident-response agent.
- MCP servers for cloud operations and third-party hardware procurement.
- Ratify Protocol for cryptographic delegated authority.
- Go, Python, and an optional C SDK physical verifier.
- A live dashboard showing every authorization decision.

## Most Important Demo Moment

The agent successfully provisions one replacement node:

```text
cloud_provision(region="us-central1", count=1) -> approved
```

Then the agent attempts a larger out-of-bounds remediation:

```text
cloud_provision(region="us-east1", count=3) -> denied
```

The MCP server blocks the action before execution and the dashboard shows:

```text
MANDATE CONSTRAINT VIOLATED
```

The model asked. The verifier said no. The action never executed.

## Why It Matters

Mandate demonstrates the missing production boundary for autonomous agents:

- Agents can remain autonomous.
- Authority stays bounded.
- Every high-impact action is independently verifiable.
- The proof travels across internal systems, outside vendors, and offline devices.

## Innovation

Mandate is not another tool-using agent. It is a production authorization pattern for agent networks.

The same delegated authority model works across:

- Internal cloud APIs.
- A separate vendor organization with no shared API key.
- A physical/offline verifier.

## Business Case

The first buyer is a security or infrastructure leader deploying agents into production operations. Their urgent question is:

> Can we let agents act without giving them broad standing credentials?

Mandate answers yes. Each verification produces an auditable decision: who authorized the agent, what scope was granted, what was requested, and why the action was approved or denied.

## Demo Command

```bash
cd ratify-demos/mandate
./bootstrap-demo.sh --gemini
```

Dashboard:

```text
http://localhost:8010
```

The Gemini ADK demo produces the three required proof points: approved action, denied action, and cross-org verification. For local verifier testing without Gemini, use `./bootstrap-demo.sh --smoke --yes`.
