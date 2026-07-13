package mcp_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpfeil/mneme/internal/api"
	"github.com/jeroenpfeil/mneme/internal/config"
	mcpsrv "github.com/jeroenpfeil/mneme/internal/mcp"
	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// TestHTTPMount verifies that the production wiring works end-to-end:
// the same Router used by main.go exposes /mcp as a Streamable HTTP
// MCP endpoint that a real client can connect to.
func TestHTTPMount(t *testing.T) {
	resetDB(t)
	seedProject(t, "apollo")

	st := store.NewWithPool(testPool)
	mcpSrv := mcpsrv.New(st, nil, nil)
	cfg := &config.Config{CORSOrigins: []string{"http://localhost:5173"}}
	srv := httptest.NewServer(api.Router(cfg, st, mcpSrv.Handler(), nil, nil))
	t.Cleanup(srv.Close)

	// /mcp must accept a Streamable HTTP MCP session.
	mcpURL := srv.URL + "/mcp"
	if !strings.HasPrefix(mcpURL, "http://") {
		t.Fatalf("unexpected url: %s", mcpURL)
	}

	client := sdk.NewClient(&sdk.Implementation{Name: "smoke", Version: "0"}, nil)
	transport := &sdk.StreamableClientTransport{Endpoint: mcpURL}
	ctx := context.Background()
	cs, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("connect over http: %v", err)
	}
	defer cs.Close()

	// Round-trip a real tool call to confirm the HTTP path works.
	call(t, cs, "push_document", samplePlan("smoke-doc", "apollo"), nil)
	var doc models.Document
	call(t, cs, "get_document", map[string]any{"id": "smoke-doc"}, &doc)
	if doc.Title != "Vehicle Listing API" {
		t.Errorf("HTTP round-trip: title=%q", doc.Title)
	}
}
