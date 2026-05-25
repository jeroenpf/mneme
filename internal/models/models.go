package models

import "time"

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
