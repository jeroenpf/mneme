package models

import (
	"fmt"
	"time"
)

// Document mirrors the documents row. Nullable text/int columns are
// represented as pointers so JSON encoding produces null rather than "".
type Document struct {
	ID           string         `json:"id"`
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
