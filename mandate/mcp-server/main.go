// Mandate MCP Server — Ratify-Aware cloud infrastructure tools.
//
// Exposes two MCP tools over HTTP+SSE:
//
//	cloud_provision(bundle, region, instance_type, count)
//	cloud_deprovision(bundle, instance_id)
//
// Every call requires a valid Ratify ProofBundle in the `bundle` parameter.
// Verification is offline, stateless, and completes in <1ms.
// An invalid or expired bundle returns status="denied" with the reason.
//
// Usage:
//
//	go run . [-addr :8090]
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	ratify "github.com/identities-ai/ratify-protocol"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var eventsURL = os.Getenv("EVENTS_URL") // e.g. http://localhost:8099/event

func publishEvent(tool, status, certID, agentID, humanID, scope, reason string, chainDepth int, extra ...string) {
	if eventsURL == "" {
		return
	}
	payload := map[string]any{
		"server":      "cloudops",
		"tool":        tool,
		"status":      status,
		"cert_id":     certID,
		"agent_id":    agentID,
		"human_id":    humanID,
		"scope":       scope,
		"chain_depth": chainDepth,
		"cross_org":   false,
		"reason":      reason,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}
	if len(extra) >= 2 {
		payload["requested"] = extra[0]
		payload["allowed"] = extra[1]
	}
	b, _ := json.Marshal(payload)
	go http.Post(eventsURL, "application/json", bytes.NewReader(b)) //nolint
}

const (
	scopeProvision   = "custom:infra:provision"
	scopeDeprovision = "custom:infra:deprovision"

	// Business constraints enforced by this verifier, independent of Ratify crypto.
	// The mandate authorizes the scope; the verifier enforces operational bounds.
	allowedRegion = "us-central1"
	maxNodeCount  = 1
)

func main() {
	addr := flag.String("addr", ":8090", "HTTP listen address")
	flag.Parse()

	s := server.NewMCPServer("mandate-mcp-server", "1.0.0",
		server.WithToolCapabilities(false),
	)

	s.AddTool(
		mcp.NewTool("cloud_provision",
			mcp.WithDescription("Provision a cloud compute instance. Requires a valid Ratify mandate with scope custom:infra:provision."),
			mcp.WithString("bundle", mcp.Required(), mcp.Description("JSON-encoded Ratify ProofBundle.")),
			mcp.WithString("region", mcp.Required(), mcp.Description("Cloud region, e.g. us-central1.")),
			mcp.WithString("instance_type", mcp.Required(), mcp.Description("Instance type, e.g. n2-standard-4.")),
			mcp.WithNumber("count", mcp.Required(), mcp.Description("Number of instances (1–10).")),
		),
		handleProvision,
	)

	s.AddTool(
		mcp.NewTool("cloud_deprovision",
			mcp.WithDescription("Terminate a cloud compute instance. Requires a valid Ratify mandate with scope custom:infra:deprovision."),
			mcp.WithString("bundle", mcp.Required(), mcp.Description("JSON-encoded Ratify ProofBundle.")),
			mcp.WithString("instance_id", mcp.Required(), mcp.Description("ID of the instance to terminate.")),
		),
		handleDeprovision,
	)

	httpServer := server.NewStreamableHTTPServer(s)
	log.Printf("Mandate MCP Server  addr=%s  endpoint=http://localhost%s/mcp", *addr, *addr)
	log.Printf("  cloud_provision    scope=%s", scopeProvision)
	log.Printf("  cloud_deprovision  scope=%s", scopeDeprovision)
	if err := httpServer.Start(*addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// ---- tool handlers ----

func handleProvision(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	bundleJSON := req.GetString("bundle", "")
	region := req.GetString("region", "")
	instanceType := req.GetString("instance_type", "")
	count := req.GetFloat("count", 1)

	result, err := verify(bundleJSON, scopeProvision)
	if err != nil {
		return errResult("bundle error: " + err.Error()), nil
	}
	if !result.Valid {
		return deniedResult(result), nil
	}

	// Business constraint enforcement — happens after cryptographic verification.
	// The mandate grants the scope; this verifier enforces operational bounds.
	certID := firstCertID(bundleJSON)
	depth := chainDepth(bundleJSON)

	if region != allowedRegion {
		reason := fmt.Sprintf("region %q is outside the mandate — allowed: %s only", region, allowedRegion)
		log.Printf("CONSTRAINT_DENIED  region=%s  agent=%s  reason=%s", region, result.AgentID, reason)
		req := fmt.Sprintf("%d node(s) in %s", int(count), region)
		alw := fmt.Sprintf("max %d node in %s", maxNodeCount, allowedRegion)
		publishEvent("cloud_provision", "denied", certID, result.AgentID, result.HumanID, scopeProvision, reason, depth, req, alw)
		return constraintDeniedResult(req, alw, reason), nil
	}

	if int(count) > maxNodeCount {
		reason := fmt.Sprintf("requested %d nodes — mandate allows max %d per incident", int(count), maxNodeCount)
		log.Printf("CONSTRAINT_DENIED  count=%.0f  agent=%s  reason=%s", count, result.AgentID, reason)
		req := fmt.Sprintf("%d node(s) in %s", int(count), region)
		alw := fmt.Sprintf("max %d node in %s", maxNodeCount, allowedRegion)
		publishEvent("cloud_provision", "denied", certID, result.AgentID, result.HumanID, scopeProvision, reason, depth, req, alw)
		return constraintDeniedResult(req, alw, reason), nil
	}

	instanceID := fmt.Sprintf("inst-%d", time.Now().UnixNano()%100000)
	log.Printf("APPROVED  provision  agent=%s  region=%s  type=%s  count=%.0f  id=%s",
		result.AgentID, region, instanceType, count, instanceID)
	publishEvent("cloud_provision", "approved", certID, result.AgentID, result.HumanID, scopeProvision, "", depth)

	return okResult(map[string]any{
		"status":        "approved",
		"instance_id":   instanceID,
		"region":        region,
		"instance_type": instanceType,
		"count":         int(count),
		"agent_id":      result.AgentID,
		"human_id":      result.HumanID,
		"cert_id":       firstCertID(bundleJSON),
		"verified_at":   time.Now().UTC().Format(time.RFC3339),
	}), nil
}

func handleDeprovision(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	bundleJSON := req.GetString("bundle", "")
	instanceID := req.GetString("instance_id", "")

	result, err := verify(bundleJSON, scopeDeprovision)
	if err != nil {
		return errResult("bundle error: " + err.Error()), nil
	}
	if !result.Valid {
		return deniedResult(result), nil
	}

	log.Printf("APPROVED  deprovision  agent=%s  instance=%s", result.AgentID, instanceID)

	return okResult(map[string]any{
		"status":      "approved",
		"instance_id": instanceID,
		"agent_id":    result.AgentID,
		"human_id":    result.HumanID,
		"cert_id":     firstCertID(bundleJSON),
		"verified_at": time.Now().UTC().Format(time.RFC3339),
	}), nil
}

// ---- verification ----

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
	var bundle ratify.ProofBundle
	if err := json.Unmarshal([]byte(bundleJSON), &bundle); err != nil {
		return ""
	}
	if len(bundle.Delegations) > 0 {
		return bundle.Delegations[0].CertID
	}
	return ""
}

func chainDepth(bundleJSON string) int {
	var bundle ratify.ProofBundle
	if err := json.Unmarshal([]byte(bundleJSON), &bundle); err != nil {
		return 0
	}
	return len(bundle.Delegations)
}

// ---- result helpers ----

func okResult(v any) *mcp.CallToolResult {
	b, _ := json.MarshalIndent(v, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: string(b)}},
	}
}

func deniedResult(r ratify.VerifyResult) *mcp.CallToolResult {
	log.Printf("DENIED  status=%s  reason=%s", r.IdentityStatus, r.ErrorReason)
	publishEvent("tool", "denied", "", r.AgentID, r.HumanID, "", r.ErrorReason, 0)
	b, _ := json.Marshal(map[string]any{
		"status":          "denied",
		"identity_status": r.IdentityStatus,
		"reason":          r.ErrorReason,
	})
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: string(b)}},
		IsError: true,
	}
}

func constraintDeniedResult(requested, allowed, reason string) *mcp.CallToolResult {
	b, _ := json.Marshal(map[string]any{
		"status":    "denied",
		"verdict":   "constraint_denied",
		"requested": requested,
		"allowed":   allowed,
		"reason":    reason,
		"message":   "Action denied before execution. The mandate does not authorize this request.",
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
