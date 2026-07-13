package mcp

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// unifiedSearchInput is the arg shape for the `search` tool. Named to avoid
// colliding with searchInput (the search_documents arg type in
// tools_docs.go, same package — discovered during execution).
type unifiedSearchInput struct {
	Query   string   `json:"q" jsonschema:"the natural-language query — matched full-text across content types"`
	Types   []string `json:"types,omitempty" jsonschema:"restrict to these content types (documents|decisions|snippets|solutions|journal); omit for all"`
	Project string   `json:"project,omitempty" jsonschema:"optional project slug to scope the search"`
	Limit   *int     `json:"limit,omitempty" jsonschema:"max ranked results; omit for 10"`
}

type unifiedSearchOutput struct {
	Results []*models.SearchHit `json:"results"`
	// Note is a hint surfaced to the LLM caller when embedding is active but
	// nothing cleared the relevance floor — a signal to refine the query
	// rather than assume the knowledge doesn't exist. Empty otherwise.
	Note string `json:"note,omitempty"`
}

func (t *tools) search(ctx context.Context, _ *sdk.CallToolRequest, in unifiedSearchInput) (*sdk.CallToolResult, *unifiedSearchOutput, error) {
	q := strings.TrimSpace(in.Query)
	if q == "" {
		return nil, nil, errors.New("q is required")
	}
	f := store.SearchFilter{Types: in.Types}
	if p := strings.TrimSpace(in.Project); p != "" {
		f.Project = &p
	}
	f.Limit = 10
	if in.Limit != nil {
		f.Limit = *in.Limit
	}
	// Embed the query for hybrid ranking when a client is configured. Any
	// Voyage error degrades to FTS-only (Vector stays nil) — search never
	// hard-fails because embeddings are down.
	if t.client != nil {
		if vecs, err := t.client.Embed(ctx, []string{q}, "query"); err == nil && len(vecs) == 1 {
			f.Vector = vecs[0]
		} else if err != nil {
			slog.Warn("search query embed failed; falling back to FTS-only", "err", err)
		}
	}
	hits, err := t.store.Search(ctx, q, f)
	if err != nil {
		if errors.Is(err, store.ErrInvalidSearchType) {
			return nil, nil, err // surfaced verbatim to the caller
		}
		return nil, nil, translateStoreErr(err)
	}
	out := &unifiedSearchOutput{Results: hits}
	// When embedding is active and nothing came back, the query likely fell
	// below the relevance floor — nudge the caller to rephrase rather than
	// conclude the knowledge is absent.
	if t.client != nil && len(hits) == 0 {
		out.Note = "No results — try rephrasing with different or more specific terms."
	}
	return nil, out, nil
}
