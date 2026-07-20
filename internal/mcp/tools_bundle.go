package mcp

import (
	"context"
	"errors"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpf/mneme/internal/bundle"
)

type getContextBundleInput struct {
	Project string `json:"project" jsonschema:"the project slug to assemble a session context bundle for"`
	Area    string `json:"area,omitempty" jsonschema:"optional area within the project; adds area-scoped memory"`
	Budget  int    `json:"budget,omitempty" jsonschema:"optional token budget for the digest; expendable sections are trimmed to fit (default ~900)"`
}

// contextBundleOutput is the MCP-facing shape of get_context_bundle: only
// the pre-rendered markdown digest. The structured bundle.Bundle (with
// separate memory/decisions/snippets/env/journal fields) is still served
// over REST at /api/v1/bundle for the Vue viewer — here we ship just the
// digest, since duplicating it as structured data doubled the tokens.
type contextBundleOutput struct {
	Markdown string `json:"markdown"`
}

func (t *tools) getContextBundle(ctx context.Context, _ *sdk.CallToolRequest, in getContextBundleInput) (*sdk.CallToolResult, *contextBundleOutput, error) {
	project := strings.TrimSpace(in.Project)
	if project == "" {
		return nil, nil, errors.New("project is required")
	}
	var areaPtr *string
	if a := strings.TrimSpace(in.Area); a != "" {
		areaPtr = &a
	}
	b, err := bundle.New(t.store).AssembleWithOptions(ctx, project, areaPtr, bundle.Options{TokenBudget: in.Budget})
	if err != nil {
		return nil, nil, translateStoreErr(err)
	}
	return nil, &contextBundleOutput{Markdown: b.Markdown}, nil
}
