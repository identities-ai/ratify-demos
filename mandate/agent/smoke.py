"""
Deterministic Mandate smoke demo.

This drives the exact winning sequence without relying on Gemini behavior:
1. CloudOps approved provision.
2. CloudOps denied over-provision before execution.
3. HardwareVendor cross-org approval.
"""

import json
import os
import sys
from pathlib import Path

from mandate_common import call_mcp_tool, issue_specialist_bundle, load_priv


KEYS_DIR = Path(os.environ.get("KEYS_DIR", "keys"))
CLOUDOPS_URL = os.environ.get("CLOUDOPS_MCP_URL", "http://localhost:8090/mcp")
VENDOR_URL = os.environ.get("VENDOR_MCP_URL", "http://localhost:8091/mcp")


def require_status(name: str, result: dict, expected: str) -> None:
    status = result.get("status")
    verdict = result.get("verdict")
    actual = verdict if verdict else status
    if expected == "constraint_denied":
        ok = status == "denied" and verdict == "constraint_denied"
    else:
        ok = status == expected
    marker = "PASS" if ok else "FAIL"
    print(f"[{marker}] {name}: {actual}")
    if not ok:
        print(json.dumps(result, indent=2))
        raise SystemExit(1)


def main() -> None:
    for f in ["commander_bundle.json", "commander_priv.bin"]:
        if not (KEYS_DIR / f).exists():
            print(f"ERROR: keys/{f} not found. Run: python setup.py", file=sys.stderr)
            raise SystemExit(1)

    commander_bundle_raw = json.loads((KEYS_DIR / "commander_bundle.json").read_text())
    commander_priv = load_priv(KEYS_DIR / "commander_priv.bin")

    print("[COMMANDER] Issuing fresh specialist mandates...")
    infra_bundle = issue_specialist_bundle(
        commander_bundle_raw,
        commander_priv,
        "Infrastructure Specialist",
        ["custom:infra:provision", "custom:infra:deprovision"],
    )
    procurement_bundle = issue_specialist_bundle(
        commander_bundle_raw,
        commander_priv,
        "Procurement Specialist",
        ["custom:physical:access"],
    )

    print("[1/3] Authorized cloud provision")
    approved = call_mcp_tool(
        CLOUDOPS_URL,
        "cloud_provision",
        {
            "bundle": infra_bundle,
            "region": "us-central1",
            "instance_type": "n2-standard-4",
            "count": 1,
        },
    )
    require_status("cloud provision within mandate", approved, "approved")

    print("[2/3] Out-of-bounds cloud provision")
    denied = call_mcp_tool(
        CLOUDOPS_URL,
        "cloud_provision",
        {
            "bundle": infra_bundle,
            "region": "us-east1",
            "instance_type": "n2-standard-8",
            "count": 3,
        },
    )
    require_status("cloud over-provision denied", denied, "constraint_denied")
    print(f"      requested: {denied.get('requested')}")
    print(f"      allowed:   {denied.get('allowed')}")
    print(f"      reason:    {denied.get('reason')}")

    print("[3/3] Cross-org hardware request")
    vendor = call_mcp_tool(
        VENDOR_URL,
        "request_hardware",
        {
            "bundle": procurement_bundle,
            "item": "server-node-2u",
            "quantity": 1,
            "site": "seattle-dc-01",
        },
    )
    require_status("hardware vendor cross-org approval", vendor, "approved")
    if not vendor.get("cross_org"):
        print(json.dumps(vendor, indent=2))
        raise SystemExit("FAIL: vendor response did not include cross_org=true")

    print()
    print("MANDATE SMOKE DEMO PASSED")
    print("The dashboard should show: green approval, red denial, cross-org approval.")


if __name__ == "__main__":
    main()
