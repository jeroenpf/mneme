package mcp

import (
	"context"
	"errors"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpf/mneme/internal/ids"
	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

type appendJournalInput struct {
	ID           string   `json:"id,omitempty" jsonschema:"journal entry id to update; omit to append a new entry"`
	Project      string   `json:"project,omitempty" jsonschema:"project slug; omit for a global (cross-project) entry"`
	SessionRef   string   `json:"session_ref,omitempty" jsonschema:"free-text session reference, e.g. a phase id like sp-2-4"`
	Summary      string   `json:"summary,omitempty" jsonschema:"one- or two-line summary of what the session did; required when creating"`
	Accomplished []string `json:"accomplished,omitempty" jsonschema:"what got done this session; on update pass [] to clear, omit to leave unchanged"`
	Deferred     []string `json:"deferred,omitempty" jsonschema:"what was consciously left for later; on update pass [] to clear, omit to leave unchanged"`
}

func (t *tools) appendJournal(ctx context.Context, _ *sdk.CallToolRequest, in appendJournalInput) (*sdk.CallToolResult, *idResult, error) {
	project := strings.TrimSpace(in.Project)
	var projectPtr *string
	if project != "" {
		projectPtr = &project
	}
	sessionRef := strings.TrimSpace(in.SessionRef)

	// Update path: id given -> load, apply provided fields, save. The ack
	// carries public ids only, so a jrnl_ public id is accepted alongside
	// the internal id.
	if id := strings.TrimSpace(in.ID); id != "" {
		var e *models.JournalEntry
		var err error
		if ids.ValidFor(ids.KindJournal, id) {
			e, err = t.store.GetJournalEntryByPublicID(ctx, id)
		} else {
			e, err = t.store.GetJournalEntry(ctx, id)
		}
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
		t.enqueue("journal", e.ID)
		return nil, &idResult{PublicID: e.PublicID}, nil
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
	t.enqueue("journal", e.ID)
	return nil, &idResult{PublicID: e.PublicID}, nil
}

