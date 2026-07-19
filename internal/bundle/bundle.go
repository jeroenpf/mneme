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
	maxNextTasks = 6
	maxBlockers  = 5
)

// DefaultTokenBudget bounds the estimated size of the rendered digest. A
// session preamble should be scannable, not a whole-document dump; when the
// assembled content would exceed this, expendable sections are trimmed
// deterministically (see applyBudget). ~900 tokens ≈ a screen of markdown.
const DefaultTokenBudget = 900

// Section floors: the minimum items an expendable section keeps under budget
// pressure, so the digest never loses its most-recent signal entirely.
const (
	journalFloor   = 1
	decisionsFloor = 1
	blockersFloor  = 1
)

// Options tunes bundle assembly. The zero value selects defaults.
type Options struct {
	// TokenBudget caps the estimated tokens of the rendered markdown digest.
	// <= 0 selects DefaultTokenBudget.
	TokenBudget int
}

// PlanSummary is the active plan's status line — title, lifecycle status,
// phase progress, and the current (wip) phase title. The plan body is
// intentionally omitted (too heavy for a session preamble); its next tasks are
// lifted out separately into Bundle.NextTasks.
type PlanSummary struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	ActivePhase  string `json:"active_phase,omitempty"`
	PhaseCurrent *int   `json:"phase_current,omitempty"`
	PhaseTotal   *int   `json:"phase_total,omitempty"`
}

// NextTask is one incomplete task lifted from the active plan, carrying the
// stable id a surgical tool (tick_task) needs and its owning phase for context.
type NextTask struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Phase string `json:"phase,omitempty"`
}

// Blocker is a document parked in the blocked state — work the session may need
// to unstick before proceeding.
type Blocker struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Bundle is everything a session needs to start work on a project:
// structured fields plus a paste-ready markdown digest.
type Bundle struct {
	Project    string                 `json:"project"`
	Area       *string                `json:"area,omitempty"`
	Memory     map[string]string      `json:"memory"`
	Env        []*models.EnvEntry     `json:"env"`
	ActivePlan *PlanSummary           `json:"active_plan"`
	NextTasks  []NextTask             `json:"next_tasks"`
	Blockers   []Blocker              `json:"blockers"`
	Deferred   []string               `json:"deferred"`
	Decisions  []*models.Decision     `json:"decisions"`
	Snippets   []*models.Snippet      `json:"snippets"`
	Journal    []*models.JournalEntry `json:"journal"`
	Markdown   string                 `json:"markdown"`

	// Budget accounting for the rendered digest (road-p5-t3). TokenBudget is the
	// cap applied; EstimatedTokens is the ~4-chars-per-token estimate of the
	// final Markdown; Truncated reports whether any section was dropped to fit.
	TokenBudget     int  `json:"token_budget"`
	EstimatedTokens int  `json:"estimated_tokens"`
	Truncated       bool `json:"truncated"`
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

// Assemble builds the bundle for a project (area optional) under the default
// token budget. Returns store.ErrInvalidProject when the project slug does not
// exist.
func (a *Assembler) Assemble(ctx context.Context, project string, area *string) (*Bundle, error) {
	return a.AssembleWithOptions(ctx, project, area, Options{})
}

// AssembleWithOptions is Assemble with an explicit token budget (Options).
// After gathering every section it renders the digest and, if the estimate
// exceeds the budget, trims expendable sections in priority order until it
// fits or only core content remains.
func (a *Assembler) AssembleWithOptions(ctx context.Context, project string, area *string, opts Options) (*Bundle, error) {
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
	plan, nextTasks, err := a.activePlan(ctx, project)
	if err != nil {
		return nil, err
	}
	blockers, err := a.blockers(ctx, project)
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
		NextTasks:  nextTasks,
		Blockers:   blockers,
		Deferred:   deferredFrom(journal),
		Decisions:  decisions,
		Snippets:   snippets,
		Journal:    journal,
	}
	budget := opts.TokenBudget
	if budget <= 0 {
		budget = DefaultTokenBudget
	}
	applyBudget(b, budget)
	return b, nil
}

// estimateTokens approximates the token count of s using the standard
// ~4-characters-per-token heuristic for English prose. It is intentionally
// backend-independent — no tokenizer dependency for a rough budget check.
func estimateTokens(s string) int { return (len(s) + 3) / 4 }

// applyBudget renders the digest and, when the estimate exceeds budget, drops
// tail items from expendable sections in a fixed priority order — snippets
// first, then journal, decisions, deferred, and finally blockers — each toward
// its floor, re-rendering after every drop. Core sections (active plan, next
// tasks, memory, env) are never trimmed: the bundle always answers "what next".
func applyBudget(b *Bundle, budget int) {
	b.TokenBudget = budget
	render := func() {
		b.Markdown = renderMarkdown(b)
		b.EstimatedTokens = estimateTokens(b.Markdown)
	}
	render()
	if b.EstimatedTokens <= budget {
		return
	}
	// Each reducer drops one tail item from its section if above its floor.
	reducers := []func() bool{
		func() bool { return dropTail(&b.Snippets, 0) },
		func() bool { return dropTail(&b.Journal, journalFloor) },
		func() bool { return dropTail(&b.Decisions, decisionsFloor) },
		func() bool { return dropTail(&b.Deferred, 0) },
		func() bool { return dropTail(&b.Blockers, blockersFloor) },
	}
	for _, reduce := range reducers {
		for b.EstimatedTokens > budget && reduce() {
			b.Truncated = true
			render()
		}
		if b.EstimatedTokens <= budget {
			return
		}
	}
}

// dropTail removes the last element of *xs when its length exceeds floor,
// reporting whether it dropped anything.
func dropTail[T any](xs *[]T, floor int) bool {
	if len(*xs) > floor {
		*xs = (*xs)[:len(*xs)-1]
		return true
	}
	return false
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

// activePlan returns the in-progress plan's status line plus the next
// incomplete tasks lifted from its body — the "what to do next" the compiler
// exists to surface. Returns (nil, nil, nil) when the project has no
// in-progress plan.
func (a *Assembler) activePlan(ctx context.Context, project string) (*PlanSummary, []NextTask, error) {
	plans, err := a.store.ListDocuments(ctx, store.Filter{
		Project: &project,
		Type:    ptr(models.TypePlan),
		Status:  ptr(models.StatusInProgress),
		Limit:   1,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("bundle: active plan: %w", err)
	}
	if len(plans) == 0 {
		return nil, nil, nil
	}
	d := plans[0]
	sum := &PlanSummary{
		ID:           d.ID,
		Title:        d.Title,
		Status:       d.Status,
		ActivePhase:  wipPhase(d.Meta),
		PhaseCurrent: d.PhaseCurrent,
		PhaseTotal:   d.PhaseTotal,
	}
	return sum, firstN(incompleteTasks(d.Body), maxNextTasks), nil
}

// wipPhase returns the title of the phase currently in progress, read from
// meta.phases[].status == "wip"; empty when there is no wip phase.
func wipPhase(meta map[string]any) string {
	phases, _ := meta["phases"].([]any)
	for _, raw := range phases {
		p, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if s, _ := p["status"].(string); s == "wip" {
			t, _ := p["title"].(string)
			return t
		}
	}
	return ""
}

// incompleteTasks walks a plan body and returns every not-done task in document
// order, tagged with its owning subphase/task-list title. Document order puts
// earlier (usually current-phase) work first, so the leading entries are the
// natural next steps.
func incompleteTasks(body map[string]any) []NextTask {
	sections, _ := body["sections"].([]any)
	var out []NextTask
	var walk func(blocks []any, phase string)
	walk = func(blocks []any, phase string) {
		for _, raw := range blocks {
			b, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			ph := phase
			if t, _ := b["type"].(string); t == "subphase" || t == "task-list" {
				if title, _ := b["title"].(string); title != "" {
					ph = title
				}
				tasks, _ := b["tasks"].([]any)
				for _, traw := range tasks {
					tm, ok := traw.(map[string]any)
					if !ok {
						continue
					}
					if done, _ := tm["done"].(bool); done {
						continue
					}
					id, _ := tm["id"].(string)
					title, _ := tm["title"].(string)
					out = append(out, NextTask{ID: id, Title: title, Phase: ph})
				}
			}
			if children, _ := b["children"].([]any); children != nil {
				walk(children, ph)
			}
		}
	}
	walk(sections, "")
	return out
}

// blockers lists documents parked in the blocked state for the project.
func (a *Assembler) blockers(ctx context.Context, project string) ([]Blocker, error) {
	docs, err := a.store.ListDocuments(ctx, store.Filter{
		Project: &project,
		Status:  ptr(models.StatusBlocked),
	})
	if err != nil {
		return nil, fmt.Errorf("bundle: blockers: %w", err)
	}
	out := []Blocker{}
	for _, d := range firstN(docs, maxBlockers) {
		out = append(out, Blocker{ID: d.ID, Title: d.Title})
	}
	return out, nil
}

// deferredFrom lifts the deferred-work list from the most recent journal entry
// (the bundle's journal is newest-first) — the work a prior session parked.
func deferredFrom(journal []*models.JournalEntry) []string {
	if len(journal) == 0 {
		return nil
	}
	return journal[0].Deferred
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
