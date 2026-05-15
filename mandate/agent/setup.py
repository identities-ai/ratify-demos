"""
Mandate Demo — Setup Script

Generates all keys and delegation certs needed for the multi-agent demo.
Run this ONCE before running main.py.

What it creates:
  keys/sre_root_priv.bin      SRE root private key (ed25519 seed [32] + ml_dsa_65 full key [4032])
  keys/sre_root.json          SRE root public identity
  keys/commander_priv.bin     Commander private key (same format)
  keys/commander_bundle.json  Commander ProofBundle (SRE → Commander, depth 1)

Usage:
    pip install ratify-protocol  (or: pip install -e ../../ratify-protocol/sdks/python)
    python setup.py

Re-run any time to regenerate. Bundle is valid for 24 hours.
"""

import json
import os
import time
from pathlib import Path

from ratify_protocol import (
    generate_human_root,
    generate_agent,
    issue_delegation,
    sign_challenge,
    generate_challenge,
)
from ratify_protocol.types import DelegationCert
from ratify_protocol.canonical import canonical_json

KEYS_DIR = Path("keys")
SCOPES = [
    "custom:infra:provision",
    "custom:infra:deprovision",
    "custom:physical:access",
    "identity:delegate",
]
TTL_SECONDS = 86400  # 24 hours


def save_priv(priv, path: Path) -> None:
    """Save HybridPrivateKey as ed25519_seed[32] + ml_dsa_65_full[N] bytes."""
    path.write_bytes(priv.ed25519 + priv.ml_dsa_65)


def main():
    KEYS_DIR.mkdir(exist_ok=True)
    now = int(time.time())

    print("Generating SRE root identity...")
    sre_root, sre_priv = generate_human_root()
    save_priv(sre_priv, KEYS_DIR / "sre_root_priv.bin")
    (KEYS_DIR / "sre_root.json").write_bytes(canonical_json(sre_root))
    print(f"  SRE root ID: {sre_root.id}")

    print("Generating Incident Commander keypair...")
    commander, commander_priv = generate_agent("Incident Commander", "mcp_server")
    save_priv(commander_priv, KEYS_DIR / "commander_priv.bin")
    print(f"  Commander ID: {commander.id}")

    print("Issuing delegation: SRE → Commander...")
    cert = DelegationCert(
        cert_id=f"cmd-{now}",
        version=1,
        issuer_id=sre_root.id,
        issuer_pub_key=sre_root.public_key,
        subject_id=commander.id,
        subject_pub_key=commander.public_key,
        scope=SCOPES,
        constraints=[],
        issued_at=now,
        expires_at=now + TTL_SECONDS,
        signature=None,
    )
    issue_delegation(cert, sre_priv)
    print(f"  Cert ID: {cert.cert_id}  expires: +{TTL_SECONDS}s")

    print("Assembling Commander ProofBundle...")
    challenge = generate_challenge()
    from ratify_protocol import sign_challenge
    challenge_sig = sign_challenge(challenge, now, commander_priv)

    from ratify_protocol.types import ProofBundle
    bundle = ProofBundle(
        agent_id=commander.id,
        agent_pub_key=commander.public_key,
        delegations=[cert],
        challenge=challenge,
        challenge_at=now,
        challenge_sig=challenge_sig,
    )
    bundle_json = canonical_json(bundle)
    (KEYS_DIR / "commander_bundle.json").write_bytes(bundle_json)

    print()
    print("Setup complete. Files written to keys/:")
    for f in sorted(KEYS_DIR.iterdir()):
        print(f"  {f.name}  ({f.stat().st_size} bytes)")
    print()
    print("Run the demo:")
    print("  python main.py")
    print()
    print("Note: commander_bundle.json is valid for 24 hours.")
    print("      Re-run setup.py if the challenge expires (>5 min old).")


if __name__ == "__main__":
    main()
