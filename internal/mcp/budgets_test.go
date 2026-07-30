package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

// Performance budgets (roadmap P8-t6). These are regression tripwires, not
// micro-benchmarks: each names a ceiling that catches a gross regression
// (payload bloat, a full-table scan, needless re-embedding) without being
// sensitive to machine speed. The four roadmap dimensions and where each budget
// lives:
//
//   - startup bundle .... internal/bundle TestBundleSnapshots (token bounds)
//   - budgetBundleBytes below (the MCP digest payload)
//   - MCP payload size .. this file (search / list_documents / bundle payloads)
//   - search latency .... this file (budgetSearchLatency)
//   - reconciliation .... internal/embed TestWarmReconcileMakesNoProviderCalls
//     (a warm pass makes zero provider calls → seconds)
const (
	// The JSON payload an MCP tool returns to the model. Slim outputs keep
	// context cost bounded: list omits bodies, search returns excerpts (not raw
	// bodies), and the bundle is token-budgeted. Ceilings sized with headroom
	// over observed payloads so ordinary growth does not trip them.
	budgetListDocsBytes = 16 * 1024
	budgetSearchBytes   = 24 * 1024
	budgetBundleBytes   = 12 * 1024

	// A search over a seeded corpus must return well under this. It is a gross-
	// regression bound (a missing index / full scan / hang), not a latency SLO —
	// the FTS query is milliseconds, leaving ~100x headroom.
	budgetSearchLatency = 2 * time.Second

	// The definition-side cost (spec-mcp-surface-v2): tool definitions and
	// instructions ride along with every API request in every connected
	// session, so the surface itself is budgeted. If one of these trips,
	// fix the surface — do not raise the ceiling.
	budgetToolCount        = 22
	budgetToolsListBytes   = 15 * 1024
	budgetInstructionBytes = 4 * 1024
)

// TestToolSurfaceBudget pins the advertised surface: exactly 22 tools, the
// tools/list JSON under its byte ceiling, and the instructions block under
// its own. This is the regression tripwire for the v2 surface diet.
func TestToolSurfaceBudget(t *testing.T) {
	cs := newClient(t)

	result, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(result.Tools) != budgetToolCount {
		names := make([]string, 0, len(result.Tools))
		for _, tool := range result.Tools {
			names = append(names, tool.Name)
		}
		t.Errorf("tool count = %d, want %d: %s", len(result.Tools), budgetToolCount, strings.Join(names, ", "))
	}

	raw, err := json.Marshal(result.Tools)
	if err != nil {
		t.Fatalf("marshal tools/list: %v", err)
	}
	t.Logf("tools/list = %d bytes (budget %d)", len(raw), budgetToolsListBytes)
	if len(raw) > budgetToolsListBytes {
		t.Errorf("tools/list JSON %d bytes exceeds budget %d — the definition diet regressed", len(raw), budgetToolsListBytes)
	}

	instructions := cs.InitializeResult().Instructions
	t.Logf("instructions = %d bytes (budget %d)", len(instructions), budgetInstructionBytes)
	if len(instructions) > budgetInstructionBytes {
		t.Errorf("instructions %d bytes exceed budget %d", len(instructions), budgetInstructionBytes)
	}
}

// seedBudgetCorpus creates a representative corpus for apollo: one document with
// a large body (to prove list/search payloads do not carry bodies) plus several
// small documents and decisions the search and bundle tools draw on.
func seedBudgetCorpus(t *testing.T, proj string) {
	t.Helper()
	st := store.NewWithPool(testPool)
	ctx := context.Background()

	bigSections := make([]any, 30)
	for i := range bigSections {
		bigSections[i] = map[string]any{
			"type": "section", "id": fmt.Sprintf("s%d", i), "title": fmt.Sprintf("Section %d", i),
			"content": strings.Repeat("coordinator body text ", 60),
		}
	}
	if err := st.CreateDocument(ctx, &models.Document{
		ID: "big-plan", Title: "Big Plan", Project: &proj,
		Type: models.TypePlan, Status: models.StatusInProgress, Tags: []string{}, Meta: map[string]any{},
		Body: map[string]any{"sections": bigSections},
	}); err != nil {
		t.Fatalf("seed big doc: %v", err)
	}
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("plan-%d", i)
		if err := st.CreateDocument(ctx, &models.Document{
			ID: id, Title: fmt.Sprintf("Plan %d", i), Project: &proj,
			Type: models.TypePlan, Status: models.StatusTodo, Tags: []string{"coordinator"}, Meta: map[string]any{},
			Body: map[string]any{"sections": []any{
				map[string]any{"type": "section", "id": "o", "title": "Overview", "content": "coordinator overview"},
			}},
		}); err != nil {
			t.Fatalf("seed doc %s: %v", id, err)
		}
	}
	for i := 0; i < 5; i++ {
		if err := st.CreateDecision(ctx, &models.Decision{
			Title: fmt.Sprintf("Decision %d", i), Project: &proj,
			Decision: "use the coordinator", Status: models.DecisionAccepted,
		}); err != nil {
			t.Fatalf("seed decision %d: %v", i, err)
		}
	}
}

// payloadBytes calls a tool and returns the size of the JSON payload the model
// receives (the text content), which is what the payload budgets bound.
func payloadBytes(t *testing.T, cs *sdk.ClientSession, name string, args any) int {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("call %s tool error: %s", name, contentText(res))
	}
	return len(contentText(res))
}

func TestMCPPayloadBudgets(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	seedBudgetCorpus(t, "apollo")

	cases := []struct {
		name   string
		tool   string
		args   any
		budget int
	}{
		{"list_documents", "list_documents", map[string]any{"project": "apollo"}, budgetListDocsBytes},
		{"search", "search", map[string]any{"q": "coordinator"}, budgetSearchBytes},
		{"get_context_bundle", "get_context_bundle", map[string]any{"project": "apollo"}, budgetBundleBytes},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			size := payloadBytes(t, cs, tc.tool, tc.args)
			t.Logf("%s payload = %d bytes (budget %d)", tc.tool, size, tc.budget)
			if size > tc.budget {
				t.Errorf("%s payload %d bytes exceeds budget %d — outputs must stay slim", tc.tool, size, tc.budget)
			}
		})
	}
}

// list_documents must not carry document bodies: a doc with a large body still
// yields a compact list entry. This is the structural guarantee behind the
// payload budget (a growing body must never grow the list payload).
func TestListDocumentsOmitsBodies(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	seedBudgetCorpus(t, "apollo")

	res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{
		Name: "list_documents", Arguments: map[string]any{"project": "apollo"},
	})
	if err != nil {
		t.Fatalf("list_documents: %v", err)
	}
	if body := contentText(res); strings.Contains(body, "coordinator body text") {
		t.Error("list_documents payload leaked a document body — bodies must be omitted")
	}
}

func TestSearchLatencyBudget(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	seedBudgetCorpus(t, "apollo")

	start := time.Now()
	payloadBytes(t, cs, "search", map[string]any{"q": "coordinator"})
	elapsed := time.Since(start)
	t.Logf("search latency = %v (budget %v)", elapsed, budgetSearchLatency)
	if elapsed > budgetSearchLatency {
		t.Errorf("search took %v, over budget %v — check for a full scan or missing index", elapsed, budgetSearchLatency)
	}
}
