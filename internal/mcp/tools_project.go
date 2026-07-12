package mcp

import (
	"context"
	"errors"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/slug"
)

// --- create_project ---------------------------------------------------

type createProjectInput struct {
	Slug        string `json:"slug" jsonschema:"project slug in kebab-case; normalized on save and echoed back in the response"`
	Name        string `json:"name" jsonschema:"human-friendly project name, e.g. \"TradeGod\""`
	Description string `json:"description,omitempty" jsonschema:"optional one-line description"`
}

func (t *tools) createProject(ctx context.Context, _ *sdk.CallToolRequest, in createProjectInput) (*sdk.CallToolResult, *models.Project, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, nil, errors.New("name is required")
	}
	if strings.TrimSpace(in.Slug) == "" {
		return nil, nil, errors.New("slug is required")
	}
	p := &models.Project{Name: name, Slug: slug.Make(in.Slug)}
	if d := strings.TrimSpace(in.Description); d != "" {
		p.Description = &d
	}
	if err := t.store.CreateProject(ctx, p); err != nil {
		return nil, nil, translateStoreErr(err)
	}
	return nil, p, nil
}
