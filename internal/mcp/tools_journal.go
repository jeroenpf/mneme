package mcp

import (
	"context"
	"errors"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

type appendJournalInput struct {
	ID           string   `json:"id,omitempty" jsonschema:"journal entry id to update; omit to append a new entry"`
	Project      string   `json:"project,omitempty" jsonschema:"project slug; omit for a global (cross-project) entry"`
	SessionRef   string   `json:"session_ref,omitempty" jsonschema:"free-text session reference, e.g. a phase id like sp-2-4"`
	Summary      string   `json:"summary,omitempty" jsonschema:"one- or two-line summary of what the session did; required when creating"`
	Accomplished []string `json:"accomplished,omitempty" jsonschema:"what got done this session; on update pass [] to clear, omit to leave unchanged"`
	Deferred     []string `json:"deferred,omitempty" jsonschema:"what was consciously left for later; on update pass [] to clear, omit to leave unchanged"`
}

func (t *tools) appendJournal(ctx context.Context, _ *sdk.CallToolRequest, in appendJournalInput) (*sdk.CallToolResult, *models.JournalEntry, error) {
	project := strings.TrimSpace(in.Project)
	var projectPtr *string
	if project != "" {
		projectPtr = &project
	}
	sessionRef := strings.TrimSpace(in.SessionRef)

	// Update path: id given -> load, apply provided fields, save.
	if id := strings.TrimSpace(in.ID); id != "" {
		e, err := t.store.GetJournalEntry(ctx, id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, nil, errors.New("journal entry not found — omit id to append a new entry")
			}
			return nil, nil, translateStoreErr(err)
		}
		if v := strings.TrimSpace(in.Summary); v != "" {
			e.Summary = v
		}
		if sessionRef != "" {
			e.SessionRef = sessionRef
		}
		if in.Project != "" {
			e.Project = projectPtr
		}
		// arrays: nil = omitted (leave unchanged); non-nil (incl. []) = replace.
		if in.Accomplished != nil {
			e.Accomplished = in.Accomplished
		}
		if in.Deferred != nil {
			e.Deferred = in.Deferred
		}
		if err := t.store.UpdateJournalEntry(ctx, e); err != nil {
			return nil, nil, translateStoreErr(err)
		}
		return nil, e, nil
	}

	// Create path.
	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		return nil, nil, errors.New("summary is required")
	}
	e := &models.JournalEntry{
		Project:      projectPtr,
		SessionRef:   sessionRef,
		Summary:      summary,
		Accomplished: in.Accomplished,
		Deferred:     in.Deferred,
	}
	if err := t.store.CreateJournalEntry(ctx, e); err != nil {
		return nil, nil, translateStoreErr(err)
	}
	return nil, e, nil
}

type journalOutput struct {
	Entries []*models.JournalEntry `json:"entries"`
}

type getJournalInput struct {
	Project string `json:"project,omitempty" jsonschema:"filter to a project slug; omit for all entries incl. global"`
	Since   string `json:"since,omitempty" jsonschema:"only entries on/after this ISO date (YYYY-MM-DD) or RFC3339 timestamp"`
	Limit   int    `json:"limit,omitempty" jsonschema:"max rows (newest first); 0 = no limit"`
}

func (t *tools) getJournal(ctx context.Context, _ *sdk.CallToolRequest, in getJournalInput) (*sdk.CallToolResult, *journalOutput, error) {
	f := store.JournalFilter{Limit: in.Limit}
	if p := strings.TrimSpace(in.Project); p != "" {
		f.Project = &p
	}
	if s := strings.TrimSpace(in.Since); s != "" {
		since, err := models.ParseSince(s)
		if err != nil {
			return nil, nil, err
		}
		f.Since = &since
	}
	es, err := t.store.ListJournalEntries(ctx, f)
	if err != nil {
		return nil, nil, translateStoreErr(err)
	}
	return nil, &journalOutput{Entries: es}, nil
}
