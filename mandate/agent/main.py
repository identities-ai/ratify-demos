"""
Mandate Demo — Autonomous Infrastructure Incident Response
Gemini + Google ADK + Ratify Protocol

The agent carries two cryptographic sub-mandates (depth-2 delegation chain:
SRE → Commander → Specialist). These are injected automatically into every
MCP tool call — Gemini never sees the bundle JSON directly.

Run setup.py first to generate keys.

Usage:
    python setup.py     # once, generates keys/
    python main.py
"""

import asyncio
import json
import os
import sys
import time
from pathlib import Path

from google.adk.agents import LlmAgent
from google.adk.runners import InMemoryRunner
from google.adk.tools import FunctionTool
from google.genai import types as genai_types

from mandate_common import call_mcp_tool, issue_specialist_bundle, load_priv

KEYS_DIR = Path(os.environ.get("KEYS_DIR", "keys"))
CLOUDOPS_URL = os.environ.get("CLOUDOPS_MCP_URL", "http://localhost:8090/mcp")
VENDOR_URL   = os.environ.get("VENDOR_MCP_URL",   "http://localhost:8091/mcp")
MODEL        = os.environ.get("MODEL", "gemini-3.1-flash-lite")

INCIDENT = """
CRITICAL ALERT — {timestamp}
Nodes: db-prod-7, db-prod-8
Status: DEGRADED (cascade failure)
Errors: Disk I/O timeouts, connection drops, failed health checks
Impact: 40% read replica capacity lost
Required actions:
  1. Provision 2 replacement cloud nodes (region: us-central1, type: n2-standard-4)
  2. Order replacement physical hardware for rack slot rack-07
"""

INSTRUCTION = """\
You are an autonomous infrastructure operations agent for TechCorp responding to a critical incident.

You must execute the following remediation plan in order:

Step 1 — Conservative provision (within mandate):
  Call provision_cloud_node(region="us-central1", instance_type="n2-standard-4", count=1)
  This recovers partial capacity. Report the instance_id.

Step 2 — Aggressive provision (attempt full recovery):
  The incident is severe — two nodes are degraded. Attempt to provision more capacity:
  Call provision_cloud_node(region="us-east1", instance_type="n2-standard-8", count=3)
  If this is denied, report the denial reason and say:
  "Escalating to on-call SRE — full remediation requires authority outside the active mandate."

Step 3 — Hardware order:
  Call order_hardware(item="server-node-2u", quantity=1, site="seattle-dc-01")
  Report the order_id.

Step 4 — Final summary:
  Report what was accomplished and what required escalation.
"""

# ---- ADK tools (bundles injected automatically) ----

def make_tools(infra_bundle: str, procurement_bundle: str):

    def provision_cloud_node(region: str, instance_type: str, count: int) -> str:
        """Provision a replacement cloud compute node. Returns instance_id and cert_id."""
        result = call_mcp_tool(CLOUDOPS_URL, "cloud_provision", {
            "bundle": infra_bundle,
            "region": region,
            "instance_type": instance_type,
            "count": count,
        })
        print(f"  [MCP cloudops] {result.get('status')}  instance={result.get('instance_id')}  cert={result.get('cert_id','')[:16]}...")
        return json.dumps(result)

    def order_hardware(item: str, quantity: int, site: str) -> str:
        """Order replacement physical hardware. Returns order_id and cert_id."""
        result = call_mcp_tool(VENDOR_URL, "request_hardware", {
            "bundle": procurement_bundle,
            "item": item,
            "quantity": quantity,
            "site": site,
        })
        print(f"  [MCP vendor]   {result.get('status')}  order={result.get('order_id')}  cross_org={result.get('cross_org')}  cert={result.get('cert_id','')[:16]}...")
        return json.dumps(result)

    return [FunctionTool(provision_cloud_node), FunctionTool(order_hardware)]

# ---- main ----

async def run():
    for f in ["commander_bundle.json", "commander_priv.bin"]:
        if not (KEYS_DIR / f).exists():
            print(f"ERROR: keys/{f} not found. Run: python setup.py", file=sys.stderr)
            sys.exit(1)

    commander_bundle_raw = json.loads((KEYS_DIR / "commander_bundle.json").read_text())
    commander_priv       = load_priv(KEYS_DIR / "commander_priv.bin")
    now_ts               = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())

    print(f"[COMMANDER]  agent_id={commander_bundle_raw['agent_id'][:16]}...")
    print(f"[MODEL]      {MODEL}")
    print(f"[MCP]        cloudops={CLOUDOPS_URL}  vendor={VENDOR_URL}")
    print()

    print("[COMMANDER]  Issuing sub-mandates (chain depth: 2)...")
    infra_bundle       = issue_specialist_bundle(commander_bundle_raw, commander_priv,
                            "Infrastructure Specialist",
                            ["custom:infra:provision", "custom:infra:deprovision"])
    procurement_bundle = issue_specialist_bundle(commander_bundle_raw, commander_priv,
                            "Procurement Specialist",
                            ["custom:physical:access"])
    print("[COMMANDER]  Sub-mandates issued.\n")

    agent = LlmAgent(
        name="incident_commander",
        model=MODEL,
        instruction=INSTRUCTION,
        tools=make_tools(infra_bundle, procurement_bundle),
    )

    runner = InMemoryRunner(agent=agent, app_name="mandate-demo")
    session = await runner.session_service.create_session(
        app_name="mandate-demo", user_id="sre-oncall")

    incident = INCIDENT.format(timestamp=now_ts)
    print(f"[INCIDENT]\n{incident}")
    print("[COMMANDER] Dispatching response...\n")

    async for event in runner.run_async(
        user_id="sre-oncall",
        session_id=session.id,
        new_message=genai_types.Content(
            role="user",
            parts=[genai_types.Part(text=incident)],
        ),
    ):
        if event.is_final_response():
            print(f"\n[RESOLVED]\n{event.content.parts[0].text}")

if __name__ == "__main__":
    asyncio.run(run())
