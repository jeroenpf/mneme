// Package bundle assembles a session-start context bundle for a project
// from the existing Mneme read surfaces (memory, the active plan,
// decisions, snippets, journal). It writes nothing and owns no schema.
package bundle

import (
	"context"
	"errors"
	"fmt"

	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

const (
	maxDecisions = 5
	maxSnippets  = 10
	maxJournal   = 3
)

// PlanSummary is the active plan's status line — title, lifecycle status,
// and phase progress. The plan body is intentionally omitted (too heavy
// for a session preamble).
type PlanSummary struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	PhaseCurrent *int   `json:"phase_current,omitempty"`
	PhaseTotal   *int   `json:"phase_total,omitempty"`
}

// Bundle is everything a session needs to start work on a project:
// structured fields plus a paste-ready markdown digest.
type Bundle struct {
	Project    string                 `json:"project"`
	Area       *string                `json:"area,omitempty"`
	Memory     map[string]string      `json:"memory"`
	Env        []*models.EnvEntry     `json:"env"`
	ActivePlan *PlanSummary           `json:"active_plan"`
	Decisions  []*models.Decision     `json:"decisions"`
	Snippets   []*models.Snippet      `json:"snippets"`
	Journal    []*models.JournalEntry `json:"journal"`
	Markdown   string                 `json:"markdown"`
}

// Assembler composes the store read methods into a Bundle.
type Assembler struct{ store store.Store }

// New returns an Assembler backed by st.
func New(st store.Store) *Assembler { return &Assembler{store: st} }

func ptr[T any](v T) *T { return &v }

func firstN[T any](xs []T, n int) []T {
	if len(xs) > n {
		return xs[:n]
	}
	return xs
}

// projectOrGlobal reports whether a nullable project pointer belongs in a
// bundle scoped to `project`: the project's own rows plus global (nil) rows.
func projectOrGlobal(p *string, project string) bool {
	return p == nil || *p == project
}

// mergeMemory folds memory groups least-specific-first into a flat map;
// later groups win on key collision. Kept local (the mcp package's copy is
// unexported) so bundle has no import edge into mcp.
func mergeMemory(groups ...[]*models.Memory) map[string]string {
	out := map[string]string{}
	for _, g := range groups {
		for _, m := range g {
			out[m.Key] = m.Value
		}
	}
	return out
}

// Assemble builds the bundle for a project (area optional). Returns
// store.ErrInvalidProject when the project slug does not exist.
func (a *Assembler) Assemble(ctx context.Context, project string, area *string) (*Bundle, error) {
	if _, err := a.store.GetProject(ctx, project); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, store.ErrInvalidProject
		}
		return nil, fmt.Errorf("bundle: verify project: %w", err)
	}

	memory, err := a.assembleMemory(ctx, project, area)
	if err != nil {
		return nil, err
	}
	env, err := a.store.ListEnv(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("bundle: env: %w", err)
	}
	plan, err := a.activePlan(ctx, project)
	if err != nil {
		return nil, err
	}
	decisions, err := a.recentDecisions(ctx, project)
	if err != nil {
		return nil, err
	}
	snippets, err := a.relevantSnippets(ctx, project)
	if err != nil {
		return nil, err
	}
	journal, err := a.recentJournal(ctx, project)
	if err != nil {
		return nil, err
	}

	b := &Bundle{
		Project:    project,
		Area:       area,
		Memory:     memory,
		Env:        env,
		ActivePlan: plan,
		Decisions:  decisions,
		Snippets:   snippets,
		Journal:    journal,
	}
	b.Markdown = renderMarkdown(b)
	return b, nil
}

func (a *Assembler) assembleMemory(ctx context.Context, project string, area *string) (map[string]string, error) {
	globalRows, err := a.store.ListMemory(ctx, store.MemoryFilter{Scope: ptr(models.ScopeGlobal)})
	if err != nil {
		return nil, fmt.Errorf("bundle: global memory: %w", err)
	}
	projRows, err := a.store.ListMemory(ctx, store.MemoryFilter{Scope: ptr(models.ScopeProject), Project: &project})
	if err != nil {
		return nil, fmt.Errorf("bundle: project memory: %w", err)
	}
	groups := [][]*models.Memory{globalRows, projRows}
	if area != nil && *area != "" {
		areaRows, err := a.store.ListMemory(ctx, store.MemoryFilter{Scope: ptr(models.ScopeArea), Project: &project, Area: area})
		if err != nil {
			return nil, fmt.Errorf("bundle: area memory: %w", err)
		}
		groups = append(groups, areaRows)
	}
	return mergeMemory(groups...), nil
}

func (a *Assembler) activePlan(ctx context.Context, project string) (*PlanSummary, error) {
	plans, err := a.store.ListDocuments(ctx, store.Filter{
		Project: &project,
		Type:    ptr(models.TypePlan),
		Status:  ptr(models.StatusInProgress),
		Limit:   1,
	})
	if err != nil {
		return nil, fmt.Errorf("bundle: active plan: %w", err)
	}
	if len(plans) == 0 {
		return nil, nil
	}
	d := plans[0]
	return &PlanSummary{
		ID:           d.ID,
		Title:        d.Title,
		Status:       d.Status,
		PhaseCurrent: d.PhaseCurrent,
		PhaseTotal:   d.PhaseTotal,
	}, nil
}

func (a *Assembler) recentDecisions(ctx context.Context, project string) ([]*models.Decision, error) {
	all, err := a.store.ListDecisions(ctx, store.DecisionFilter{})
	if err != nil {
		return nil, fmt.Errorf("bundle: decisions: %w", err)
	}
	out := []*models.Decision{}
	for _, d := range all {
		if projectOrGlobal(d.Project, project) {
			out = append(out, d)
		}
	}
	return firstN(out, maxDecisions), nil
}

func (a *Assembler) relevantSnippets(ctx context.Context, project string) ([]*models.Snippet, error) {
	all, err := a.store.ListSnippets(ctx, store.SnippetFilter{})
	if err != nil {
		return nil, fmt.Errorf("bundle: snippets: %w", err)
	}
	out := []*models.Snippet{}
	for _, s := range all {
		if projectOrGlobal(s.Project, project) {
			out = append(out, s)
		}
	}
	return firstN(out, maxSnippets), nil
}

func (a *Assembler) recentJournal(ctx context.Context, project string) ([]*models.JournalEntry, error) {
	all, err := a.store.ListJournalEntries(ctx, store.JournalFilter{})
	if err != nil {
		return nil, fmt.Errorf("bundle: journal: %w", err)
	}
	out := []*models.JournalEntry{}
	for _, j := range all {
		if projectOrGlobal(j.Project, project) {
			out = append(out, j)
		}
	}
	return firstN(out, maxJournal), nil
}
