// Mandate Event Relay
//
// Receives verification events from both MCP servers via POST /event,
// broadcasts them to all SSE subscribers on GET /events.
//
// Usage: go run . [-addr :8099]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type VerifyEvent struct {
	Timestamp  string `json:"timestamp"`
	Server     string `json:"server"`
	Tool       string `json:"tool"`
	Status     string `json:"status"` // "approved" | "denied" | "thinking"
	CertID     string `json:"cert_id"`
	AgentID    string `json:"agent_id"`
	HumanID    string `json:"human_id"`
	Scope      string `json:"scope"`
	CrossOrg   bool   `json:"cross_org"`
	ChainDepth int    `json:"chain_depth"`
	Reason     string `json:"reason,omitempty"`
	Requested  string `json:"requested,omitempty"` // what the agent asked for
	Allowed    string `json:"allowed,omitempty"`   // what the mandate permits
}

type broker struct {
	mu   sync.RWMutex
	subs map[chan []byte]struct{}
}

func newBroker() *broker { return &broker{subs: make(map[chan []byte]struct{})} }

func (b *broker) subscribe() chan []byte {
	ch := make(chan []byte, 20)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *broker) unsubscribe(ch chan []byte) {
	b.mu.Lock()
	delete(b.subs, ch)
	b.mu.Unlock()
	close(ch)
}

func (b *broker) publish(data []byte) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs {
		select {
		case ch <- data:
		default:
		}
	}
}

func main() {
	addr := flag.String("addr", ":8099", "HTTP listen address")
	flag.Parse()

	bus := newBroker()

	mux := http.NewServeMux()

	// POST /event — MCP servers call this after each verification
	mux.HandleFunc("POST /event", func(w http.ResponseWriter, r *http.Request) {
		var evt VerifyEvent
		if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		if evt.Timestamp == "" {
			evt.Timestamp = time.Now().UTC().Format(time.RFC3339)
		}
		data, _ := json.Marshal(evt)
		bus.publish(data)
		log.Printf("event  server=%-12s  tool=%-20s  status=%s  cert=%s...",
			evt.Server, evt.Tool, evt.Status, evt.CertID[:min(16, len(evt.CertID))])
		w.WriteHeader(http.StatusNoContent)
	})

	// GET /events — SSE stream for the dashboard
	mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Connection", "keep-alive")

		ch := bus.subscribe()
		defer bus.unsubscribe(ch)

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", 500)
			return
		}

		// Send a heartbeat immediately so the client knows it's connected
		fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
		flusher.Flush()

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case data, ok := <-ch:
				if !ok {
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			case <-ticker.C:
				fmt.Fprintf(w, ": heartbeat\n\n")
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})

	// GET /health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	log.Printf("Mandate Event Relay  addr=%s", *addr)
	log.Printf("  POST /event    — MCP servers push verification events here")
	log.Printf("  GET  /events   — SSE stream for the dashboard")
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("relay: %v", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
