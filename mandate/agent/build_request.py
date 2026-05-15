"""Build a test MCP tool-call request. Usage: python build_request.py"""
import json, sys, os

bundle_file = sys.argv[1] if len(sys.argv) > 1 else "/tmp/bundle.json"
out_file    = sys.argv[2] if len(sys.argv) > 2 else "/tmp/mcp_request.json"
tool        = sys.argv[3] if len(sys.argv) > 3 else "cloud_provision"

bundle = json.load(open(bundle_file))

if tool == "cloud_provision":
    args = {"bundle": json.dumps(bundle), "region": "us-central1", "instance_type": "n2-standard-4", "count": 1}
elif tool == "cloud_deprovision":
    args = {"bundle": json.dumps(bundle), "instance_id": "inst-00001"}
elif tool == "request_hardware":
    args = {"bundle": json.dumps(bundle), "item": "server-node-2u", "quantity": 1, "site": "seattle-dc-01"}
else:
    print(f"Unknown tool: {tool}"); sys.exit(1)

req = {"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": {"name": tool, "arguments": args}}
open(out_file, "w").write(json.dumps(req))
print(f"Written: {out_file}  (tool={tool})")
