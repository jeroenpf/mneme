package mcp

import (
	"context"
	"errors"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

type logDecisionInput struct {
	ID           string `json:"id,omitempty" jsonschema:"decision id to update; omit to create a new decision"`
	Title        string `json:"title,omitempty" jsonschema:"short decision title; required when creating"`
	Project      string `json:"project,omitempty" jsonschema:"project slug; omit for a global (cross-project) decision"`
	Decision     string `json:"decision,omitempty" jsonschema:"what was decided; required when creating"`
	Rationale    string `json:"rationale,omitempty" jsonschema:"why — the reasoning behind the decision"`
	Alternatives string `json:"alternatives,omitempty" jsonschema:"options considered and rejected"`
	Consequences string `json:"consequences,omitempty" jsonschema:"trade-offs and follow-on effects"`
	Status       string `json:"status,omitempty" jsonschema:"proposed | accepted | deprecated (default accepted)"`
}

func (t *tools) logDecision(ctx context.Context, _ *sdk.CallToolRequest, in logDecisionInput) (*sdk.CallToolResult, *models.Decision, error) {
	project := strings.TrimSpace(in.Project)
	var projectPtr *string
	if project != "" {
		projectPtr = &project
	}

	// Update path: id given -> load, apply provided fields, save.
	if id := strings.TrimSpace(in.ID); id != "" {
		d, err := t.store.GetDecision(ctx, id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, nil, errors.New("decision not found — omit id to log a new decision")
			}
			return nil, nil, translateStoreErr(err)
		}
		if v := strings.TrimSpace(in.Title); v != "" {
			d.Title = v
		}
		if v := strings.TrimSpace(in.Decision); v != "" {
			d.Decision = v
		}
		if in.Rationale != "" {
			d.Rationale = in.Rationale
		}
		if in.Alternatives != "" {
			d.Alternatives = in.Alternatives
		}
		if in.Consequences != "" {
			d.Consequences = in.Consequences
		}
		if in.Project != "" {
			d.Project = projectPtr
		}
		if s := strings.TrimSpace(in.Status); s != "" {
			st := models.DecisionStatus(s)
			if err := models.ValidateDecisionStatus(st); err != nil {
				return nil, nil, err
			}
			d.Status = st
		}
		if err := t.store.UpdateDecision(ctx, d); err != nil {
			return nil, nil, translateStoreErr(err)
		}
		t.enqueue("decisions", d.ID)
		return nil, d, nil
	}

	// Create path.
	title := strings.TrimSpace(in.Title)
	decision := strings.TrimSpace(in.Decision)
	if title == "" {
		return nil, nil, errors.New("title is required")
	}
	if decision == "" {
		return nil, nil, errors.New("decision is required")
	}
	status := models.DecisionStatus(strings.TrimSpace(in.Status))
	if status == "" {
		status = models.DecisionAccepted
	}
	if err := models.ValidateDecisionStatus(status); err != nil {
		return nil, nil, err
	}
	d := &models.Decision{
		Title:        title,
		Project:      projectPtr,
		Decision:     decision,
		Rationale:    in.Rationale,
		Alternatives: in.Alternatives,
		Consequences: in.Consequences,
		Status:       status,
	}
	if err := t.store.CreateDecision(ctx, d); err != nil {
		return nil, nil, translateStoreErr(err)
	}
	t.enqueue("decisions", d.ID)
	return nil, d, nil
}

type decisionsOutput struct {
	Decisions []*models.Decision `json:"decisions"`
}

type getDecisionsInput struct {
	Project string `json:"project,omitempty" jsonschema:"filter to a project slug; omit for all decisions incl. global"`
	Status  string `json:"status,omitempty" jsonschema:"filter by status: proposed | accepted | deprecated"`
	Limit   int    `json:"limit,omitempty" jsonschema:"max rows (newest first); 0 = no limit"`
}

func (t *tools) getDecisions(ctx context.Context, _ *sdk.CallToolRequest, in getDecisionsInput) (*sdk.CallToolResult, *decisionsOutput, error) {
	f := store.DecisionFilter{Limit: in.Limit}
	if p := strings.TrimSpace(in.Project); p != "" {
		f.Project = &p
	}
	if s := strings.TrimSpace(in.Status); s != "" {
		st := models.DecisionStatus(s)
		if err := models.ValidateDecisionStatus(st); err != nil {
			return nil, nil, err
		}
		f.Status = &st
	}
	ds, err := t.store.ListDecisions(ctx, f)
	if err != nil {
		return nil, nil, translateStoreErr(err)
	}
	return nil, &decisionsOutput{Decisions: ds}, nil
}

type queryDecisionsInput struct {
	Query   string `json:"query" jsonschema:"natural-language search over title, decision, rationale, alternatives, consequences"`
	Project string `json:"project,omitempty" jsonschema:"optional project slug to scope the search"`
	Limit   int    `json:"limit,omitempty" jsonschema:"max ranked results; 0 = no limit"`
}

func (t *tools) queryDecisions(ctx context.Context, _ *sdk.CallToolRequest, in queryDecisionsInput) (*sdk.CallToolResult, *decisionsOutput, error) {
	q := strings.TrimSpace(in.Query)
	if q == "" {
		return nil, nil, errors.New("query is required")
	}
	f := store.DecisionFilter{Limit: in.Limit}
	if p := strings.TrimSpace(in.Project); p != "" {
		f.Project = &p
	}
	ds, err := t.store.SearchDecisions(ctx, q, f)
	if err != nil {
		return nil, nil, translateStoreErr(err)
	}
	return nil, &decisionsOutput{Decisions: ds}, nil
}
