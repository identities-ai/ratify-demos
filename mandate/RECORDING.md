# Mandate Recording Script

Target length: 3 minutes.

## Before Recording

Run:

```bash
cd /Users/chuksy/projects/IdentitiesAI/ratify-demos/mandate
./bootstrap-demo.sh --stop
./bootstrap-demo.sh --gemini
```

Open:

```text
http://localhost:8010
```

Keep the dashboard and terminal visible.

## 0:00-0:20 Hook

"AI agents are stuck in prototype because companies cannot prove what an agent was authorized to do. Mandate fixes that. The agent is autonomous. The authority is not."

Show the dashboard before events arrive.

## 0:20-0:50 The Incident

"This is a 3am database incident. A TechCorp SRE pre-issued a narrow mandate: the agent can provision one replacement node in us-central1 and request physical hardware support."

Start the Gemini ADK demo.

## 0:50-1:20 Approved Action

When the green row appears:

"The agent presents its mandate to CloudOps. CloudOps verifies the cryptographic delegation chain locally, sees the request is inside the mandate, and provisions the node."

Point out:

- approved
- same org
- chain depth
- offline verification

## 1:20-1:55 Winning Moment

When the red flash appears:

"Now the agent attempts a larger remediation: three nodes in us-east1. This is exactly what enterprises fear from autonomous agents."

Then:

"The model asked. The verifier said no. The action never executed."

Point out:

- requested: 3 nodes in us-east1
- allowed: max 1 node in us-central1
- denied before execution

## 1:55-2:30 Cross-Org Proof

When the vendor row appears:

"Now the same authorization model crosses an organizational boundary. HardwareVendor does not have a TechCorp API key and does not call back to TechCorp. It verifies the proof locally and accepts the hardware request."

Point out:

- cross-org badge
- no API key exchanged
- no issuer callback

## 2:30-2:50 Optional Physical Close

If showing the C verifier:

"The same pattern works offline. A Raspberry Pi or physical access device can verify a mandate without WiFi or a cloud dependency."

## 2:50-3:00 Close

"Mandate is the production safety layer for autonomous agents: bounded authority, independent verification, and auditability across every tool an agent touches."

## Do Not Say

- Do not lead with post-quantum cryptography.
- Do not explain canonical JSON.
- Do not spend more than 20 seconds on Ratify internals.
- Do not imply the verifier trusts the model.

## Must Say

- "The agent is autonomous. The authority is not."
- "The model asked. The verifier said no."
- "The action never executed."
