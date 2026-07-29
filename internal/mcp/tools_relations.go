package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/relations"
)

type linkInput struct {
	From    string `json:"from" jsonschema:"source entity: a public id (doc_/dec_/snip_/sol_/jrnl_) or a document id/slug"`
	To      string `json:"to" jsonschema:"target entity: same forms as from"`
	RelType string `json:"rel_type" jsonschema:"one of relates-to, implements, supersedes, depends-on, blocks. Directional from->to (A implements B). Inline mentions register automatically on document writes — link is for semantics a mention cannot carry"`
}

type linkResult struct {
	Relation *models.Relation `json:"relation"`
}

func (t *tools) link(ctx context.Context, _ *sdk.CallToolRequest, in linkInput) (*sdk.CallToolResult, *linkResult, error) {
	rel, err := t.rel.Link(ctx, in.From, in.To, in.RelType)
	if err != nil {
		return nil, nil, err
	}
	return nil, &linkResult{Relation: rel}, nil
}

type unlinkInput struct {
	From    string `json:"from" jsonschema:"source entity: public id or document id/slug"`
	To      string `json:"to" jsonschema:"target entity: public id or document id/slug"`
	RelType string `json:"rel_type,omitempty" jsonschema:"remove only this rel type; omitted removes every explicit link between the pair. Auto mentions are scanner-owned and never removed here"`
}

type unlinkResult struct {
	Removed int64 `json:"removed"`
}

func (t *tools) unlink(ctx context.Context, _ *sdk.CallToolRequest, in unlinkInput) (*sdk.CallToolResult, *unlinkResult, error) {
	var rt *string
	if in.RelType != "" {
		rt = &in.RelType
	}
	n, err := t.rel.Unlink(ctx, in.From, in.To, rt)
	if err != nil {
		return nil, nil, err
	}
	return nil, &unlinkResult{Removed: n}, nil
}

type getRelatedInput struct {
	Ref string `json:"ref" jsonschema:"entity to inspect: a public id (doc_/dec_/snip_/sol_/jrnl_) or a document id/slug"`
}

func (t *tools) getRelated(ctx context.Context, _ *sdk.CallToolRequest, in getRelatedInput) (*sdk.CallToolResult, *relations.Bundle, error) {
	b, err := t.rel.Related(ctx, in.Ref)
	if err != nil {
		return nil, nil, err
	}
	return nil, b, nil
}
