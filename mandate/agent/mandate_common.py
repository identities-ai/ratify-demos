import json
import time
from pathlib import Path

import httpx

from ratify_protocol import (
    generate_agent,
    issue_delegation,
    sign_challenge,
    generate_challenge,
)
from ratify_protocol.types import (
    DelegationCert,
    ProofBundle,
    HybridPrivateKey,
    HybridPublicKey,
    HybridSignature,
)
from ratify_protocol.canonical import canonical_json, base64_standard_decode


def load_priv(path: Path) -> HybridPrivateKey:
    data = path.read_bytes()
    return HybridPrivateKey(ed25519=data[:32], ml_dsa_65=data[32:])


def _decode_hybrid_pub(raw: dict) -> HybridPublicKey:
    return HybridPublicKey(
        ed25519=base64_standard_decode(raw["ed25519"]),
        ml_dsa_65=base64_standard_decode(raw["ml_dsa_65"]),
    )


def _decode_hybrid_sig(raw: dict) -> HybridSignature:
    return HybridSignature(
        ed25519=base64_standard_decode(raw["ed25519"]),
        ml_dsa_65=base64_standard_decode(raw["ml_dsa_65"]),
    )


def _decode_cert(raw: dict) -> DelegationCert:
    return DelegationCert(
        cert_id=raw["cert_id"],
        version=raw["version"],
        issuer_id=raw["issuer_id"],
        issuer_pub_key=_decode_hybrid_pub(raw["issuer_pub_key"]),
        subject_id=raw["subject_id"],
        subject_pub_key=_decode_hybrid_pub(raw["subject_pub_key"]),
        scope=list(raw["scope"]),
        constraints=raw.get("constraints") or [],
        issued_at=raw["issued_at"],
        expires_at=raw["expires_at"],
        signature=_decode_hybrid_sig(raw["signature"]),
    )


def issue_specialist_bundle(
    commander_bundle_raw: dict,
    commander_priv: HybridPrivateKey,
    specialist_name: str,
    scopes: list[str],
    ttl_seconds: int = 7200,
) -> str:
    now = int(time.time())
    commander_cert = _decode_cert(commander_bundle_raw["delegations"][0])
    commander_id = commander_bundle_raw["agent_id"]
    commander_pub = _decode_hybrid_pub(commander_bundle_raw["agent_pub_key"])

    specialist, specialist_priv = generate_agent(specialist_name, "mcp_server")

    cert = DelegationCert(
        cert_id=f"spec-{specialist_name.lower().replace(' ', '-')}-{now}",
        version=1,
        issuer_id=commander_id,
        issuer_pub_key=commander_pub,
        subject_id=specialist.id,
        subject_pub_key=specialist.public_key,
        scope=scopes,
        constraints=[],
        issued_at=now,
        expires_at=now + ttl_seconds,
        signature=None,
    )
    issue_delegation(cert, commander_priv)

    challenge = generate_challenge()
    challenge_sig = sign_challenge(challenge, now, specialist_priv)

    bundle = ProofBundle(
        agent_id=specialist.id,
        agent_pub_key=specialist.public_key,
        delegations=[cert, commander_cert],
        challenge=challenge,
        challenge_at=now,
        challenge_sig=challenge_sig,
    )
    return canonical_json(bundle).decode()


def _decode_jsonrpc_response(resp: httpx.Response) -> dict:
    text = resp.text.strip()
    if not text:
        return {}
    if text.startswith("{"):
        return resp.json()

    data_messages = []
    for line in text.splitlines():
        line = line.strip()
        if line.startswith("data:"):
            payload = line.removeprefix("data:").strip()
            if payload and payload != "[DONE]":
                data_messages.append(json.loads(payload))
    if data_messages:
        return data_messages[-1]
    raise ValueError(f"unexpected MCP response: {text[:200]}")


def call_mcp_tool(base_url: str, tool_name: str, arguments: dict) -> dict:
    """Call an MCP tool via Streamable HTTP. Returns the tool result dict."""
    with httpx.Client(timeout=30) as client:
        init_resp = client.post(
            base_url,
            headers={
                "Content-Type": "application/json",
                "Accept": "application/json, text/event-stream",
            },
            json={
                "jsonrpc": "2.0",
                "id": 1,
                "method": "initialize",
                "params": {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {},
                    "clientInfo": {"name": "mandate-agent", "version": "1.0"},
                },
            },
        )
        init_resp.raise_for_status()
        session_id = init_resp.headers.get("Mcp-Session-Id", "")

        headers = {
            "Content-Type": "application/json",
            "Accept": "application/json, text/event-stream",
        }
        if session_id:
            headers["Mcp-Session-Id"] = session_id

        resp = client.post(
            base_url,
            headers=headers,
            json={
                "jsonrpc": "2.0",
                "id": 2,
                "method": "tools/call",
                "params": {"name": tool_name, "arguments": arguments},
            },
        )
        resp.raise_for_status()
        data = _decode_jsonrpc_response(resp)
        content = data.get("result", {}).get("content", [{}])
        text = content[0].get("text", "{}") if content else "{}"
        return json.loads(text)
