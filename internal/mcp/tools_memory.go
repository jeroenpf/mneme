package mcp

import (
	"context"
	"errors"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpf/mneme/internal/live"
	"github.com/jeroenpf/mneme/internal/models"
)

// --- set_memory -------------------------------------------------------

type setMemoryInput struct {
	Scope   string `json:"scope" jsonschema:"one of global, project, area"`
	Key     string `json:"key" jsonschema:"memory key (stable identifier)"`
	Value   string `json:"value" jsonschema:"memory value (free text)"`
	Project string `json:"project,omitempty" jsonschema:"project slug; required for project/area scope"`
	Area    string `json:"area,omitempty" jsonschema:"area name; required for area scope"`
}

// setMemoryResult is the lean ack for set_memory: set returns {key},
// delete (empty value) returns {key, deleted:true}. Write acks never
// echo payloads.
type setMemoryResult struct {
	Key     string `json:"key"`
	Deleted bool   `json:"deleted,omitempty"`
}

func (t *tools) setMemory(ctx context.Context, _ *sdk.CallToolRequest, in setMemoryInput) (*sdk.CallToolResult, *setMemoryResult, error) {
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
	if strings.TrimSpace(in.Value) == "" {
		if err := t.store.DeleteMemory(ctx, scope, pp, ap, in.Key); err != nil {
			return nil, nil, translateStoreErr(err)
		}
		t.broadcast(live.Event{Type: "memory", ID: in.Key, Project: project})
		return nil, &setMemoryResult{Key: in.Key, Deleted: true}, nil
	}
	m := &models.Memory{Scope: scope, Key: in.Key, Value: in.Value, Project: pp, Area: ap}
	if err := t.store.SetMemory(ctx, m); err != nil {
		return nil, nil, translateStoreErr(err)
	}
	t.broadcast(live.Event{Type: "memory", ID: in.Key, Project: project})
	return nil, &setMemoryResult{Key: m.Key}, nil
}
