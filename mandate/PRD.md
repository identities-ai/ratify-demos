# Mandate
## Winning PRD for the Google for Startups AI Agents Challenge

**Track:** Track 1: Build (Net-New Agents)  
**Deadline:** June 5, 2026 at 5:00 PM PT  
**Owner:** Identities AI  
**Tagline:** The agent is autonomous. The authority is not.  
**One-line pitch:** Mandate lets enterprises give AI agents bounded, portable authority instead of broad API keys; every high-impact action is independently verified before execution.

---

## 1. Product Thesis

AI agents are moving from chat to action, but the authorization model has not changed. Most production agents still rely on API keys, service accounts, RBAC, and prompt instructions. Those mechanisms can say which service is calling, but they cannot prove that a specific agent was delegated a specific action by a specific human under specific constraints.

Mandate is an autonomous infrastructure incident-response system where Gemini-powered agents can act across cloud, vendor, and physical systems only when they carry a cryptographically valid mandate.

The winning claim:

> Enterprises cannot safely move agents from prototype to production until every agent action is independently verifiable.

Mandate proves that model end to end.

---

## 2. Why This Wins

This challenge is about moving agents from prototype to production. Most submissions will show agents that can do useful work. Mandate shows the missing production boundary: an agent can decide what to do, but the receiving system independently decides whether the agent is authorized to do it.

The demo must make three things obvious:

1. **Autonomy:** Gemini receives an incident, plans remediation, and calls tools without a human click.
2. **Real action:** MCP tools provision cloud infrastructure, coordinate a third-party hardware vendor, and authorize physical access.
3. **Production safety:** An unsafe or out-of-bounds action is denied by verification logic outside the model.

The primary product moment is a denial:

```text
MANDATE CONSTRAINT VIOLATED
requested: provision 3 nodes in us-east1
allowed:   max 1 node in us-central1
verdict:   denied before execution
```

If the demo only shows approvals, it does not win. The denial is the proof that Mandate is production infrastructure, not a scripted agent workflow.

---

## 3. User And Buyer

### Primary User

Platform, security, and SRE teams at companies deploying autonomous agents into production operations.

### Buyer

The security or infrastructure leader who owns risk for autonomous systems.

### Urgent Job

"Let agents remediate incidents without handing them broad credentials or forcing a human approval on every action."

### Existing Alternatives

| Alternative | Why it fails for production agents |
|---|---|
| API keys | Broad, bearer-style authority; hard to prove human delegation per action. |
| RBAC/service accounts | Authorizes a service identity, not a specific delegated agent action. |
| Prompt instructions | Not enforceable by receiving systems. |
| Human approval queues | Reduce risk but destroy autonomy and incident-response speed. |
| Centralized auth callback | Breaks offline/cross-org/physical verification and creates a dependency on the issuer. |

Mandate is different because the proof travels with the agent action and can be verified locally by any Ratify-aware system.

---

## 4. Demo Narrative: "The 3am Cascade Failure"

### Setup

A TechCorp SRE pre-authorizes an Incident Commander agent before going offline:

- Scope: `custom:infra:provision`
- Scope: `custom:physical:access`
- Constraint: only `us-central1`
- Constraint: at most `1` replacement node per incident
- Expiry: 8 hours

The SRE does not stay in the loop. The system is allowed to act, but only inside the mandate.

### Act 1: Incident

Monitoring emits:

```text
CRITICAL: db-prod-7 and db-prod-8 degraded
Impact: 40% read replica capacity lost
Suggested action: provision replacement capacity and prepare hardware replacement
```

Gemini ADK agent reads the incident, identifies remediation, and plans:

1. Provision one cloud replacement node.
2. Contact the hardware vendor for replacement hardware.
3. Prepare physical access for rack maintenance.

Dashboard shows the active mandate and the agent plan.

### Act 2: Authorized Cloud Action

The agent calls the CloudOps MCP server:

```text
cloud_provision(region="us-central1", instance_type="n2-standard-4", count=1)
```

CloudOps does not trust the model. It verifies the Ratify proof bundle locally:

- Signature chain valid.
- Scope includes `custom:infra:provision`.
- Region constraint passes.
- Count constraint passes.
- Challenge is fresh.

Result:

```text
APPROVED: replacement node provisioned
```

Dashboard row is green.

### Act 3: Constraint Violation

The incident gets worse. The agent attempts a more aggressive action:

```text
cloud_provision(region="us-east1", instance_type="n2-standard-8", count=3)
```

CloudOps verifies the same way, but the mandate does not allow this action.

Result:

```text
DENIED: mandate constraint violated
reason: requested region us-east1 outside allowed region us-central1; requested count 3 exceeds max 1
```

Dashboard flashes red:

```text
MANDATE CONSTRAINT VIOLATED
```

The agent adapts:

```text
Escalating to on-call SRE because remediation requires authority outside the active mandate.
```

This is the core proof: the agent remains autonomous, but authority remains bounded.

### Act 4: Cross-Organization Verification

The agent contacts HardwareVendor Inc. through a separate MCP server:

```text
request_hardware(item="server-node-2u", quantity=1, site="seattle-dc-01")
```

HardwareVendor has no API key from TechCorp and no shared auth system. It verifies the mandate offline using Ratify.

Result:

```text
APPROVED: hardware order accepted
cross_org=true
```

Dashboard shows a cross-org badge. This proves the mandate is portable authority, not a local permission check.

### Act 5: Physical Verification

The same physical-access mandate is presented to a Raspberry Pi verifier running the Ratify C SDK.

No cloud call. No issuer callback. No network dependency.

Result:

```text
PHYSICAL_ACCESS GRANTED
```

Green LED turns on.

This is the visceral close: the same authorization model spans cloud software, a third-party vendor, and a physical device.

---

## 5. Product Requirements

### P0: Required To Win

| Requirement | Description | Verification |
|---|---|---|
| Gemini ADK incident agent | Agent receives incident and chooses tool calls. | Console or dashboard shows plan and tool sequence. |
| MCP CloudOps server | Tool verifies mandate before provisioning. | Valid call approved; invalid call denied before execution. |
| Real constraint denial | Region/count/rate constraint must fail visibly. | Dashboard shows red denial row with reason. |
| Cross-org vendor MCP server | Separate server verifies same proof without shared credentials. | Response includes `cross_org=true`; dashboard shows badge. |
| Live verification dashboard | Shows every verification event in real time. | Green approval and red denial visible during video. |
| One-command demo | Local demo starts predictably. | `./bootstrap-demo.sh` launches services. |
| 3-minute video script | Narrative maps directly to judging criteria. | Video covers autonomy, action, denial, portability. |

### P1: Strong Differentiators

| Requirement | Description | Verification |
|---|---|---|
| Runtime sub-delegation | Commander issues short-lived specialist mandates. | Dashboard shows chain depth `2`. |
| Physical verifier | Pi or laptop C verifier checks physical-access bundle offline. | Terminal prints granted/denied; LED optional. |
| Signed audit receipt | Each verification emits a tamper-evident receipt hash. | Dashboard shows receipt hash for each row. |
| Failure-mode library | Demo can trigger wrong scope, expired cert, and tampered bundle. | CLI or UI scenario selector. |

### P2: Only If Time Remains

| Requirement | Description |
|---|---|
| Polished ratify-web integration | Move dashboard from static demo page into full product UI. |
| Cloud Run deployment | Host demo services publicly. |
| Marketplace packaging | Useful later, not needed for Track 1 win. |

---

## 6. Technical Architecture

```text
TechCorp SRE
  signs mandate for Incident Commander
        |
        v
Gemini ADK Incident Commander
  receives alert
  plans remediation
  calls MCP tools with proof bundles
        |
        +--> CloudOps MCP Server
        |      verifies scope + constraints locally
        |      approves valid provision
        |      denies out-of-bounds provision
        |
        +--> HardwareVendor MCP Server
        |      separate org
        |      verifies same mandate offline
        |      accepts hardware request
        |
        +--> Physical Verifier
               C SDK
               no network required
               grants/denies access

All verification events stream to Mandate Dashboard.
```

### Components

| Component | Stack | Role |
|---|---|---|
| Agent | Python, Gemini, Google ADK | Plans and orchestrates incident response. |
| Ratify SDK | Python + Go + C | Issues and verifies cryptographic mandates. |
| CloudOps MCP | Go, MCP HTTP | Simulated cloud provisioning API. |
| HardwareVendor MCP | Go, MCP HTTP | Simulated third-party vendor API. |
| Event relay | Go, SSE | Streams verification events to dashboard. |
| Dashboard | HTML/JS or Next.js | Judge-facing operational view. |
| Physical verifier | C SDK | Offline physical authorization check. |

---

## 7. Mandate Model

The mandate answers five questions:

1. Who authorized the agent?
2. Which agent was authorized?
3. What action is allowed?
4. Under what constraints?
5. Is the proof fresh and valid right now?

### Required Scopes

| Scope | Meaning |
|---|---|
| `custom:infra:provision` | Provision replacement cloud capacity. |
| `custom:physical:access` | Authorize physical infrastructure access. |

### Required Constraints

For the winning demo, constraints should be simple and visible:

| Constraint | Allowed | Denied scenario |
|---|---|---|
| Region | `us-central1` | Agent requests `us-east1`. |
| Count | max `1` node | Agent requests `3` nodes. |
| Expiry | 8-hour mandate | Expired bundle fails closed. |

If Ratify's built-in constraint vocabulary does not directly support region/count, implement this as a demo-specific extension constraint evaluator in the CloudOps MCP server. The point is not the name of the constraint; the point is that the verifier, not the model, enforces the boundary.

---

## 8. Dashboard Requirements

The dashboard is the judge's mental model. It must avoid crypto jargon until the moment the user asks for details.

### Primary View

Show a live event feed with these columns:

| Column | Example |
|---|---|
| Time | `03:14:08` |
| Actor | `Incident Commander` |
| Tool | `cloud_provision` |
| Requested | `3 nodes / us-east1` |
| Mandate | `max 1 node / us-central1` |
| Verdict | `APPROVED` or `DENIED` |
| Reason | `count exceeds max` |
| Chain | `SRE -> Commander -> Specialist` |
| Mode | `local`, `cross-org`, or `offline` |

### Required Visual States

- Green row for approved actions.
- Red row for denied actions.
- Cross-org badge for vendor verification.
- Offline badge for physical verification.
- Large red denial banner for the constraint violation.

### Copy Rules

Use outcome language:

- "Denied before execution"
- "Verified locally"
- "No API key exchanged"
- "No issuer callback"
- "Human authority preserved"

Avoid leading with:

- "post-quantum"
- "hybrid signatures"
- "canonical JSON"
- "delegation cert"

Those details can appear in a technical drawer, not the main story.

---

## 9. Judging Alignment

| Criterion | How Mandate Scores |
|---|---|
| Technical implementation | Gemini ADK agent, MCP tools, local verification, cross-org verification, constraint enforcement, optional C SDK physical verifier. |
| Business impact | Solves the production blocker for autonomous enterprise agents: bounded authority and auditability. |
| Innovation | Portable delegated authority that works across cloud, organizations, and physical devices. |
| Demo quality | Clear incident story, visible autonomous action, dramatic denial moment, live dashboard, optional hardware close. |

The submission should explicitly say:

> Mandate is not another agent that can use tools. Mandate is the production authorization pattern that lets agents use tools safely.

---

## 10. Build Plan

### Milestone 1: Make The Denial Real

**Goal:** CloudOps MCP approves an allowed provision and denies an out-of-bounds provision.

**Verify:**

```text
allowed:  cloud_provision(us-central1, count=1) -> approved
denied:   cloud_provision(us-east1, count=3) -> constraint_denied
```

### Milestone 2: Make The Dashboard Tell The Story

**Goal:** Verification events become obvious to a non-crypto judge.

**Verify:**

```text
green approval row
red denial row
large MANDATE CONSTRAINT VIOLATED banner
reason text visible
```

### Milestone 3: Put Gemini Back In The Loop

**Goal:** The ADK agent chooses and calls the same tools used in manual tests.

**Verify:**

```text
incident prompt -> agent plan -> tool call -> verification event -> agent adaptation
```

### Milestone 4: Add Cross-Org Proof

**Goal:** HardwareVendor MCP verifies the proof independently.

**Verify:**

```text
request_hardware(...) -> approved
dashboard row includes cross_org=true
```

### Milestone 5: Add Physical Or Offline Close

**Goal:** Same model works without the cloud.

**Verify:**

```text
NO_GPIO=1 ./verifier < bundle.json -> PHYSICAL_ACCESS GRANTED
```

### Milestone 6: Package The Demo

**Goal:** A judge or teammate can run it reliably.

**Verify:**

```text
./bootstrap-demo.sh
open dashboard
python agent/main.py
```

---

## 11. Video Script

Target length: 3 minutes.

### 0:00-0:20 — Hook

"AI agents are stuck in prototype because companies cannot prove what an agent was authorized to do. Mandate fixes that. The agent is autonomous. The authority is not."

### 0:20-0:50 — Incident And Plan

Show Gemini receiving the incident and planning remediation.

### 0:50-1:20 — Approved Action

Show cloud provision approved. Dashboard green row.

### 1:20-1:55 — Winning Moment

Show agent attempting out-of-bounds action. Dashboard red banner:

```text
MANDATE CONSTRAINT VIOLATED
```

Say:

"The model asked. The verifier said no. The action never executed."

### 1:55-2:30 — Cross-Org

Show vendor MCP accepting the mandate without an API key or issuer callback.

### 2:30-2:50 — Offline/Physical

Show physical verifier terminal or LED.

### 2:50-3:00 — Close

"Mandate is the production safety layer for autonomous agents: bounded authority, independent verification, and auditability across every tool an agent touches."

---

## 12. Submission Copy

**Title:** Mandate: Bounded Authority for Autonomous Agents

**Elevator Pitch:** Mandate is a Gemini ADK incident-response agent that can provision cloud infrastructure, coordinate a third-party hardware vendor, and authorize physical access only when each action carries a valid cryptographic mandate. Instead of giving agents broad API keys, Mandate gives them bounded, portable authority that every MCP tool can verify before execution.

**Track:** Build (Net-New Agents)

**What We Built:** A multi-tool autonomous infrastructure operations system using Gemini, Google ADK, MCP, Ratify Protocol, Go, Python, and an optional C SDK physical verifier.

**Why It Matters:** Enterprises will not let agents take real actions unless they can prove who authorized the agent, what it was allowed to do, under what constraints, and whether the action was verified before execution. Mandate makes that proof portable across cloud, vendors, and offline systems.

**Most Important Demo Moment:** Gemini attempts an out-of-bounds remediation. The MCP server independently verifies the mandate, denies the request before execution, and the dashboard shows exactly which constraint was violated.

---

## 13. Non-Goals

- Do not build a broad agent marketplace.
- Do not lead with cryptography details.
- Do not spend time on Google Cloud Marketplace packaging for this submission.
- Do not make the physical verifier required for the primary demo path.
- Do not build features that do not appear in the 3-minute video.

---

## 14. Definition Of Done

Mandate is submission-ready when:

1. A fresh checkout can run the demo locally with documented commands.
2. Gemini ADK performs at least one approved tool action.
3. Gemini ADK performs or triggers one denied out-of-bounds action.
4. The denial is enforced outside the model by the MCP verifier.
5. The dashboard makes the approval and denial obvious.
6. Cross-org verification is shown through the vendor MCP server.
7. The final video can be understood by a non-crypto judge in under 3 minutes.

If all seven are true, Mandate has a real chance to win.
