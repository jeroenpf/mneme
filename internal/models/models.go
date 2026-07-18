package models

import (
	"fmt"
	"slices"
	"time"
)

// Document mirrors the documents row. Nullable text/int columns are
// represented as pointers so JSON encoding produces null rather than "".
type Document struct {
	ID           string         `json:"id"`
	PublicID     string         `json:"public_id"`
	Title        string         `json:"title"`
	Project      *string        `json:"project,omitempty"`
	Category     *string        `json:"category,omitempty"`
	Type         string         `json:"type"`
	Status       string         `json:"status"`
	Ticket       *string        `json:"ticket,omitempty"`
	Repo         *string        `json:"repo,omitempty"`
	Tags         []string       `json:"tags"`
	PhaseCurrent *int           `json:"phase_current,omitempty"`
	PhaseTotal   *int           `json:"phase_total,omitempty"`
	Meta         map[string]any `json:"meta"`
	Body         map[string]any `json:"body"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// Project mirrors the projects row.
type Project struct {
	ID          string    `json:"id"`
	PublicID    string    `json:"public_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ProjectCounts breaks down a project's document counts by status. The
// JSON keys mirror the status enum values exactly so the UI can index
// by status without a translation table.
type ProjectCounts struct {
	Todo       int `json:"todo"`
	InProgress int `json:"in-progress"`
	Complete   int `json:"complete"`
	Blocked    int `json:"blocked"`
	Archived   int `json:"archived"`
	Total      int `json:"total"`
}

// ProjectStats is the aggregated row returned by Store.ListProjects.
type ProjectStats struct {
	Project
	Counts ProjectCounts `json:"counts"`
}

// Embedding mirrors the embeddings row. Phase 2.8 populates this; the
// shape is fixed in 1.2 so dependent migrations stay stable.
type Embedding struct {
	ID          string    `json:"id"`
	SourceType  string    `json:"source_type"`
	SourceID    string    `json:"source_id"`
	ChunkID     string    `json:"chunk_id"`
	ChunkText   string    `json:"chunk_text"`
	Embedding   []float32 `json:"embedding,omitempty"`
	Project     *string   `json:"project,omitempty"`
	SourceTitle string    `json:"source_title"`
	Model       string    `json:"model"`
	CreatedAt   time.Time `json:"created_at"`
}

// MemoryScope is the hierarchy level a memory entry lives at. More
// specific scopes override less specific ones when merged: global <
// project < area. Keep the values in sync with the memories_scope_shape
// CHECK in migrations/007_memories.up.sql.
type MemoryScope string

const (
	ScopeGlobal  MemoryScope = "global"
	ScopeProject MemoryScope = "project"
	ScopeArea    MemoryScope = "area"
)

// Memory mirrors a memories row. Project/Area are pointers so the JSON
// (and the DB) can represent "absent" for scopes that don't use them.
type Memory struct {
	ID        string      `json:"id"`
	Scope     MemoryScope `json:"scope"`
	Project   *string     `json:"project,omitempty"`
	Area      *string     `json:"area,omitempty"`
	Key       string      `json:"key"`
	Value     string      `json:"value"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// EnvEntry mirrors an env_entries row — non-secret project-scoped config
// (ports, service names, local URLs). Description is a pointer so "absent"
// round-trips as SQL NULL / omitted JSON. Every entry has a project (unlike
// memory, which has un-scoped global keys).
type EnvEntry struct {
	ID          string    `json:"id"`
	Project     string    `json:"project"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description *string   `json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ValidateMemoryScoping enforces the memories_scope_shape invariants at
// the API/MCP boundary so callers get a friendly message instead of a
// raw CHECK violation. project/area are trimmed inputs; "" means absent.
func ValidateMemoryScoping(scope MemoryScope, project, area string) error {
	switch scope {
	case ScopeGlobal:
		if project != "" || area != "" {
			return fmt.Errorf("global scope takes no project or area")
		}
	case ScopeProject:
		if project == "" {
			return fmt.Errorf("project scope requires a project")
		}
		if area != "" {
			return fmt.Errorf("project scope takes no area")
		}
	case ScopeArea:
		if project == "" || area == "" {
			return fmt.Errorf("area scope requires both project and area")
		}
	default:
		return fmt.Errorf("scope must be one of global, project, area")
	}
	return nil
}

// Document type / status constants — keep in sync with the CHECK
// constraints in migrations/002_documents.up.sql.
const (
	TypePlan       = "plan"
	TypeReport     = "report"
	TypeSpec       = "spec"
	TypeADR        = "adr"
	TypeBrainstorm = "brainstorm"
	TypeJournal    = "journal"

	StatusTodo       = "todo"
	StatusInProgress = "in-progress"
	StatusComplete   = "complete"
	StatusBlocked    = "blocked"
	StatusArchived   = "archived"
)

// ValidStatuses returns the closed set of document lifecycle statuses,
// in enum order. Matches the CHECK constraint in
// migrations/002_documents.up.sql — keep the two in sync.
func ValidStatuses() []string {
	return []string{StatusTodo, StatusInProgress, StatusComplete, StatusBlocked, StatusArchived}
}

// ValidDocTypes returns the closed set of document types, in enum order.
// Matches the same CHECK constraint.
func ValidDocTypes() []string {
	return []string{TypePlan, TypeReport, TypeSpec, TypeADR, TypeBrainstorm, TypeJournal}
}

// IsValidStatus reports whether s is an accepted document status.
func IsValidStatus(s string) bool { return slices.Contains(ValidStatuses(), s) }

// IsValidType reports whether s is an accepted document type.
func IsValidType(s string) bool { return slices.Contains(ValidDocTypes(), s) }

// DecisionStatus is the lifecycle state of a decision-log entry. A
// decision starts proposed, becomes accepted once ratified, and is
// deprecated when a later decision supersedes it.
type DecisionStatus string

const (
	DecisionProposed   DecisionStatus = "proposed"
	DecisionAccepted   DecisionStatus = "accepted"
	DecisionDeprecated DecisionStatus = "deprecated"
)

// Decision mirrors the decisions row — the mutable ADR log Claude Code
// writes as a session side-effect. Project is nil for a global
// (cross-project) decision.
type Decision struct {
	ID           string         `json:"id"`
	PublicID     string         `json:"public_id"`
	Title        string         `json:"title"`
	Project      *string        `json:"project,omitempty"`
	Decision     string         `json:"decision"`
	Rationale    string         `json:"rationale"`
	Alternatives string         `json:"alternatives"`
	Consequences string         `json:"consequences"`
	Status       DecisionStatus `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// ValidateDecisionStatus enforces the status CHECK at the API/MCP
// boundary so callers get a friendly message instead of a raw CHECK
// violation. "" is invalid — callers wanting a default substitute
// DecisionAccepted before calling.
func ValidateDecisionStatus(s DecisionStatus) error {
	switch s {
	case DecisionProposed, DecisionAccepted, DecisionDeprecated:
		return nil
	default:
		return fmt.Errorf("status must be one of proposed, accepted, deprecated")
	}
}

// Snippet mirrors the snippets row — a reusable code pattern or project
// convention Claude Code saves with save_snippet and retrieves with
// get_snippets / search_snippets. Project is nil for a global
// (cross-project) snippet. Language is free-text, lowercased at the write
// boundary; the Vue viewer maps it to a Prism grammar, falling back to
// plaintext for unknown languages.
type Snippet struct {
	ID          string    `json:"id"`
	PublicID    string    `json:"public_id"`
	Title       string    `json:"title"`
	Project     *string   `json:"project,omitempty"`
	Language    string    `json:"language"`
	Content     string    `json:"content"`
	Tags        []string  `json:"tags"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// JournalEntry mirrors the journal_entries row — a per-session dev-journal
// entry Claude Code writes with append_journal and retrieves with
// get_journal. Project is nil for a global (cross-project) entry.
// Summary is required; session_ref is free-text (e.g. "sp-2-4").
type JournalEntry struct {
	ID           string    `json:"id"`
	PublicID     string    `json:"public_id"`
	Project      *string   `json:"project,omitempty"`
	SessionRef   string    `json:"session_ref"`
	Summary      string    `json:"summary"`
	Accomplished []string  `json:"accomplished"`
	Deferred     []string  `json:"deferred"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ParseSince parses a since-filter value: either an RFC3339 timestamp or a
// plain ISO date (YYYY-MM-DD). Used by the journal REST + MCP surfaces to
// turn a query/arg string into a created_at lower bound.
func ParseSince(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.DateOnly, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("since must be an ISO date (YYYY-MM-DD) or an RFC3339 timestamp")
}

// Solution mirrors the solutions row — an error and the fix that worked,
// logged with log_solution and retrieved with find_solution. Project is
// nil for a global (cross-project) gotcha. ErrorDescription + Solution are
// required; SourceURL is an optional link (empty = none).
type Solution struct {
	ID               string    `json:"id"`
	PublicID         string    `json:"public_id"`
	Project          *string   `json:"project,omitempty"`
	ErrorDescription string    `json:"error_description"`
	Solution         string    `json:"solution"`
	Tags             []string  `json:"tags"`
	SourceURL        string    `json:"source_url"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// SearchHit is one ranked result from the unified Search across content
// types. Title/Excerpt are per-type projections; Score is the reciprocal-
// rank fusion score (higher = more relevant). Similarity is the raw cosine
// similarity (0–1, higher = closer) of the best-matching chunk when the hit
// was reached via the vector side; nil for FTS-only hits. It makes the
// semantic relevance floor observable.
type SearchHit struct {
	Type       string    `json:"type"`
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Excerpt    string    `json:"excerpt"`
	Project    *string   `json:"project,omitempty"`
	Score      float64   `json:"score"`
	Similarity *float64  `json:"similarity,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}
