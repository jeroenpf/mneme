package mcp

import (
	"context"
	"errors"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpfeil/mneme/internal/live"
	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// mergeMemory folds groups into a flat key→value map. Pass groups
// least-specific-first (global, project, area); later groups win on
// collision. Pure and DB-free so the precedence rule is unit-testable.
func mergeMemory(groups ...[]*models.Memory) map[string]string {
	out := map[string]string{}
	for _, g := range groups {
		for _, m := range g {
			out[m.Key] = m.Value
		}
	}
	return out
}

func ptrScope(s models.MemoryScope) *models.MemoryScope { return &s }

// --- get_memory -------------------------------------------------------

type getMemoryInput struct {
	Scope   string `json:"scope" jsonschema:"one of global, project, area"`
	Project string `json:"project,omitempty" jsonschema:"project slug; required for project/area scope"`
	Area    string `json:"area,omitempty" jsonschema:"area name; required for area scope"`
}

type getMemoryOutput struct {
	Scope   string            `json:"scope"`
	Project string            `json:"project,omitempty"`
	Area    string            `json:"area,omitempty"`
	Values  map[string]string `json:"values"`
}

func (t *tools) getMemory(ctx context.Context, _ *sdk.CallToolRequest, in getMemoryInput) (*sdk.CallToolResult, *getMemoryOutput, error) {
	scope := models.MemoryScope(in.Scope)
	project := strings.TrimSpace(in.Project)
	area := strings.TrimSpace(in.Area)
	if err := models.ValidateMemoryScoping(scope, project, area); err != nil {
		return nil, nil, err
	}

	globalRows, err := t.store.ListMemory(ctx, store.MemoryFilter{Scope: ptrScope(models.ScopeGlobal)})
	if err != nil {
		return nil, nil, translateStoreErr(err)
	}
	groups := [][]*models.Memory{globalRows}
	if scope == models.ScopeProject || scope == models.ScopeArea {
		projRows, err := t.store.ListMemory(ctx, store.MemoryFilter{Scope: ptrScope(models.ScopeProject), Project: &project})
		if err != nil {
			return nil, nil, translateStoreErr(err)
		}
		groups = append(groups, projRows)
	}
	if scope == models.ScopeArea {
		areaRows, err := t.store.ListMemory(ctx, store.MemoryFilter{Scope: ptrScope(models.ScopeArea), Project: &project, Area: &area})
		if err != nil {
			return nil, nil, translateStoreErr(err)
		}
		groups = append(groups, areaRows)
	}
	return nil, &getMemoryOutput{Scope: in.Scope, Project: project, Area: area, Values: mergeMemory(groups...)}, nil
}

// --- set_memory -------------------------------------------------------

type setMemoryInput struct {
	Scope   string `json:"scope" jsonschema:"one of global, project, area"`
	Key     string `json:"key" jsonschema:"memory key (stable identifier)"`
	Value   string `json:"value" jsonschema:"memory value (free text)"`
	Project string `json:"project,omitempty" jsonschema:"project slug; required for project/area scope"`
	Area    string `json:"area,omitempty" jsonschema:"area name; required for area scope"`
}

func (t *tools) setMemory(ctx context.Context, _ *sdk.CallToolRequest, in setMemoryInput) (*sdk.CallToolResult, *models.Memory, error) {
	scope := models.MemoryScope(in.Scope)
	project := strings.TrimSpace(in.Project)
	area := strings.TrimSpace(in.Area)
	if strings.TrimSpace(in.Key) == "" {
		return nil, nil, errors.New("key is required")
	}
	if in.Value == "" {
		return nil, nil, errors.New("value is required")
	}
	if err := models.ValidateMemoryScoping(scope, project, area); err != nil {
		return nil, nil, err
	}
	m := &models.Memory{Scope: scope, Key: in.Key, Value: in.Value}
	if project != "" {
		m.Project = &project
	}
	if area != "" {
		m.Area = &area
	}
	if err := t.store.SetMemory(ctx, m); err != nil {
		return nil, nil, translateStoreErr(err)
	}
	t.broadcast(live.Event{Type: "memory", ID: in.Key, Project: project})
	return nil, m, nil
}

// --- delete_memory ----------------------------------------------------

type deleteMemoryInput struct {
	Scope   string `json:"scope" jsonschema:"one of global, project, area"`
	Key     string `json:"key" jsonschema:"memory key to delete"`
	Project string `json:"project,omitempty"`
	Area    string `json:"area,omitempty"`
}

func (t *tools) deleteMemory(ctx context.Context, _ *sdk.CallToolRequest, in deleteMemoryInput) (*sdk.CallToolResult, *okResult, error) {
	scope := models.MemoryScope(in.Scope)
	project := strings.TrimSpace(in.Project)
	area := strings.TrimSpace(in.Area)
	if strings.TrimSpace(in.Key) == "" {
		return nil, nil, errors.New("key is required")
	}
	if err := models.ValidateMemoryScoping(scope, project, area); err != nil {
		return nil, nil, err
	}
	var pp, ap *string
	if project != "" {
		pp = &project
	}
	if area != "" {
		ap = &area
	}
	if err := t.store.DeleteMemory(ctx, scope, pp, ap, in.Key); err != nil {
		return nil, nil, translateStoreErr(err)
	}
	t.broadcast(live.Event{Type: "memory", ID: in.Key, Project: project, Op: "delete_memory"})
	return nil, &okResult{OK: true}, nil
}
