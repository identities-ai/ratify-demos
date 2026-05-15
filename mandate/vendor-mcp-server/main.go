// HardwareVendor MCP Server — Cross-Org Ratify Verification
//
// Simulates a hardware supplier in a DIFFERENT organization from the agent.
// Has no prior relationship with the agent's organization. No shared auth system.
// No API key exchange. Verifies the agent's ProofBundle offline using only
// the Ratify Go SDK and the issuer's public key embedded in the cert chain.
//
// This is the cross-org trust demonstration: any Ratify-issued mandate
// from any organization is verifiable by any other Ratify-aware system.
//
// Tools:
//   request_hardware(bundle, item, quantity, site)
//   check_inventory(bundle, item)
//
// Emits SSE events to the Ratify Verify dashboard with cross_org=true.
//
// Usage:
//
//	go run . [-addr :8091]
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	ratify "github.com/identities-ai/ratify-protocol"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var eventsURL = os.Getenv("EVENTS_URL")

func publishEvent(tool, status, certID, agentID, humanID, scope, reason string, chainDepth int, crossOrg bool) {
	if eventsURL == "" {
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"server":      "hardwarevendor",
		"tool":        tool,
		"status":      status,
		"cert_id":     certID,
		"agent_id":    agentID,
		"human_id":    humanID,
		"scope":       scope,
		"chain_depth": chainDepth,
		"cross_org":   crossOrg,
		"reason":      reason,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	})
	go http.Post(eventsURL, "application/json", bytes.NewReader(payload)) //nolint
}

const (
	scopePhysicalAccess = "custom:physical:access"
	orgName             = "HardwareVendor Inc."
)

func main() {
	addr := flag.String("addr", ":8091", "HTTP listen address")
	flag.Parse()

	s := server.NewMCPServer("hardwarevendor-mcp-server", "1.0.0",
		server.WithToolCapabilities(false),
	)

	s.AddTool(
		mcp.NewTool("request_hardware",
			mcp.WithDescription("Request hardware procurement. Requires a Ratify mandate with scope custom:physical:access. This API belongs to HardwareVendor Inc. — a separate organization. It verifies your mandate offline with no callback."),
			mcp.WithString("bundle", mcp.Required(), mcp.Description("JSON-encoded Ratify ProofBundle.")),
			mcp.WithString("item", mcp.Required(), mcp.Description("Hardware item SKU, e.g. server-node-2u.")),
			mcp.WithNumber("quantity", mcp.Required(), mcp.Description("Number of units.")),
			mcp.WithString("site", mcp.Required(), mcp.Description("Delivery site ID, e.g. seattle-dc-01.")),
		),
		handleRequestHardware,
	)

	s.AddTool(
		mcp.NewTool("check_inventory",
			mcp.WithDescription("Check hardware inventory. Requires a Ratify mandate with scope custom:physical:access."),
			mcp.WithString("bundle", mcp.Required(), mcp.Description("JSON-encoded Ratify ProofBundle.")),
			mcp.WithString("item", mcp.Required(), mcp.Description("Hardware item SKU to check.")),
		),
		handleCheckInventory,
	)

	httpServer := server.NewStreamableHTTPServer(s)
	log.Printf("HardwareVendor MCP Server  addr=%s  org=%q  endpoint=http://localhost%s/mcp", *addr, orgName, *addr)
	log.Printf("  request_hardware   scope=%s  [cross-org]", scopePhysicalAccess)
	log.Printf("  check_inventory    scope=%s  [cross-org]", scopePhysicalAccess)
	log.Printf("  Verifies Ratify mandates offline — no phone-home to issuer org.")
	if err := httpServer.Start(*addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func handleRequestHardware(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	bundleJSON := req.GetString("bundle", "")
	item := req.GetString("item", "")
	quantity := req.GetFloat("quantity", 1)
	site := req.GetString("site", "")

	result, err := verify(bundleJSON, scopePhysicalAccess)
	if err != nil {
		return errResult("bundle error: " + err.Error()), nil
	}
	if !result.Valid {
		return deniedResult(result, true), nil
	}

	orderID := fmt.Sprintf("ORD-%05d", rand.Intn(99999))
	eta := time.Now().Add(4 * time.Hour).UTC().Format(time.RFC3339)

	log.Printf("APPROVED  request_hardware  cross_org=true  agent=%s  item=%s  qty=%.0f  site=%s  order=%s",
		result.AgentID, item, quantity, site, orderID)
	publishEvent("request_hardware", "approved", firstCertID(bundleJSON), result.AgentID, result.HumanID, scopePhysicalAccess, "", chainDepth(bundleJSON), true)

	return okResult(map[string]any{
		"status":    "approved",
		"order_id":  orderID,
		"item":      item,
		"quantity":  int(quantity),
		"site":      site,
		"eta":       eta,
		"vendor":    orgName,
		"cross_org": true,
		"agent_id":  result.AgentID,
		"human_id":  result.HumanID,
		"cert_id":   firstCertID(bundleJSON),
		"chain_depth": chainDepth(bundleJSON),
		"verified_at": time.Now().UTC().Format(time.RFC3339),
		"note": "Verified offline. No callback to issuer organization.",
	}), nil
}

func handleCheckInventory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	bundleJSON := req.GetString("bundle", "")
	item := req.GetString("item", "")

	result, err := verify(bundleJSON, scopePhysicalAccess)
	if err != nil {
		return errResult("bundle error: " + err.Error()), nil
	}
	if !result.Valid {
		return deniedResult(result, true), nil
	}

	log.Printf("APPROVED  check_inventory  cross_org=true  agent=%s  item=%s", result.AgentID, item)

	return okResult(map[string]any{
		"status":      "approved",
		"item":        item,
		"in_stock":    true,
		"quantity":    12,
		"lead_time":   "4 hours",
		"cross_org":   true,
		"verified_at": time.Now().UTC().Format(time.RFC3339),
	}), nil
}

func verify(bundleJSON, requiredScope string) (ratify.VerifyResult, error) {
	if bundleJSON == "" {
		return ratify.VerifyResult{}, fmt.Errorf("bundle is required")
	}
	var bundle ratify.ProofBundle
	if err := json.Unmarshal([]byte(bundleJSON), &bundle); err != nil {
		return ratify.VerifyResult{}, fmt.Errorf("invalid bundle JSON: %w", err)
	}
	return ratify.Verify(&bundle, ratify.VerifyOptions{
		RequiredScope: requiredScope,
	}), nil
}

func firstCertID(bundleJSON string) string {
	var b ratify.ProofBundle
	if err := json.Unmarshal([]byte(bundleJSON), &b); err != nil || len(b.Delegations) == 0 {
		return ""
	}
	return b.Delegations[0].CertID
}

func chainDepth(bundleJSON string) int {
	var b ratify.ProofBundle
	if err := json.Unmarshal([]byte(bundleJSON), &b); err != nil {
		return 0
	}
	return len(b.Delegations)
}

func okResult(v any) *mcp.CallToolResult {
	b, _ := json.MarshalIndent(v, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: string(b)}},
	}
}

func deniedResult(r ratify.VerifyResult, crossOrg bool) *mcp.CallToolResult {
	log.Printf("DENIED  cross_org=%v  status=%s  reason=%s", crossOrg, r.IdentityStatus, r.ErrorReason)
	publishEvent("tool", "denied", "", r.AgentID, r.HumanID, "", r.ErrorReason, 0, crossOrg)
	b, _ := json.Marshal(map[string]any{
		"status":          "denied",
		"identity_status": r.IdentityStatus,
		"reason":          r.ErrorReason,
		"cross_org":       crossOrg,
		"vendor":          orgName,
	})
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: string(b)}},
		IsError: true,
	}
}

func errResult(msg string) *mcp.CallToolResult {
	b, _ := json.Marshal(map[string]string{"status": "error", "message": msg})
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: string(b)}},
		IsError: true,
	}
}

func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ltime)
}
