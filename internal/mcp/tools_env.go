package mcp

import (
	"context"
	"errors"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpf/mneme/internal/live"
	"github.com/jeroenpf/mneme/internal/models"
)

// flattenEnv projects env records into a flat key→value map — the get_env
// contract ("just knows the port"). Pure and DB-free so it's unit-testable,
// mirroring mergeMemory.
func flattenEnv(entries []*models.EnvEntry) map[string]string {
	out := map[string]string{}
	for _, e := range entries {
		out[e.Key] = e.Value
	}
	return out
}

// --- get_env ----------------------------------------------------------

type getEnvInput struct {
	Project string `json:"project" jsonschema:"project slug whose env to load"`
}

type getEnvOutput struct {
	Project string            `json:"project"`
	Values  map[string]string `json:"values"`
}

func (t *tools) getEnv(ctx context.Context, _ *sdk.CallToolRequest, in getEnvInput) (*sdk.CallToolResult, *getEnvOutput, error) {
	project := strings.TrimSpace(in.Project)
	if project == "" {
		return nil, nil, errors.New("project is required")
	}
	entries, err := t.store.ListEnv(ctx, project)
	if err != nil {
		return nil, nil, translateStoreErr(err)
	}
	return nil, &getEnvOutput{Project: project, Values: flattenEnv(entries)}, nil
}

// --- set_env ----------------------------------------------------------

type setEnvInput struct {
	Project     string `json:"project" jsonschema:"project slug"`
	Key         string `json:"key" jsonschema:"env key, e.g. API_PORT"`
	Value       string `json:"value" jsonschema:"env value (non-secret)"`
	Description string `json:"description,omitempty" jsonschema:"optional note on what this configures"`
}

func (t *tools) setEnv(ctx context.Context, _ *sdk.CallToolRequest, in setEnvInput) (*sdk.CallToolResult, *models.EnvEntry, error) {
	project := strings.TrimSpace(in.Project)
	key := strings.TrimSpace(in.Key)
	if project == "" {
		return nil, nil, errors.New("project is required")
	}
	if key == "" {
		return nil, nil, errors.New("key is required")
	}
	if in.Value == "" {
		return nil, nil, errors.New("value is required")
	}
	e := &models.EnvEntry{Project: project, Key: key, Value: in.Value}
	if d := strings.TrimSpace(in.Description); d != "" {
		e.Description = &d
	}
	if err := t.store.SetEnv(ctx, e); err != nil {
		return nil, nil, translateStoreErr(err)
	}
	t.broadcast(live.Event{Type: "env", ID: key, Project: project})
	return nil, e, nil
}

// --- list_env ---------------------------------------------------------

type listEnvInput struct {
	Project string `json:"project" jsonschema:"project slug"`
}

type listEnvOutput struct {
	Items []*models.EnvEntry `json:"items"`
}

func (t *tools) listEnv(ctx context.Context, _ *sdk.CallToolRequest, in listEnvInput) (*sdk.CallToolResult, *listEnvOutput, error) {
	project := strings.TrimSpace(in.Project)
	if project == "" {
		return nil, nil, errors.New("project is required")
	}
	entries, err := t.store.ListEnv(ctx, project)
	if err != nil {
		return nil, nil, translateStoreErr(err)
	}
	return nil, &listEnvOutput{Items: entries}, nil
}
