package mcp

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// advancePhaseInput is the argument shape for advance_phase.
type advancePhaseInput struct {
	DocID string `json:"doc_id" jsonschema:"document id"`
}

// advancePhaseOutput tells the caller which phase index just finished
// and which (if any) is now wip.
type advancePhaseOutput struct {
	CompletedIndex int              `json:"completed_index"`
	NextIndex      *int             `json:"next_index,omitempty"`
	Doc            *models.Document `json:"doc"`
}

// advancePhase flips the currently-wip entry in meta.phases to done
// and the next entry from todo to wip. If there's no wip entry yet
// (e.g. brand-new plan), the first todo entry is promoted to wip
// without anything being completed.
func (t *tools) advancePhase(ctx context.Context, _ *sdk.CallToolRequest, in advancePhaseInput) (*sdk.CallToolResult, *advancePhaseOutput, error) {
	if in.DocID == "" {
		return nil, nil, errors.New("doc_id is required")
	}
	doc, err := t.loadDoc(ctx, in.DocID)
	if err != nil {
		return nil, nil, err
	}

	phasesRaw, ok := doc.Meta["phases"].([]any)
	if !ok || len(phasesRaw) == 0 {
		return nil, nil, errors.New("meta.phases is missing or empty — nothing to advance")
	}

	// Locate the currently-wip phase.
	wipIdx := -1
	for i, p := range phasesRaw {
		pm, ok := p.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("meta.phases[%d] must be an object", i)
		}
		if status, _ := pm["status"].(string); status == "wip" {
			if wipIdx != -1 {
				return nil, nil, fmt.Errorf("meta.phases has multiple wip entries (indexes %d and %d) — fix manually before advancing", wipIdx, i)
			}
			wipIdx = i
		}
	}

	out := &advancePhaseOutput{CompletedIndex: -1}
	if wipIdx >= 0 {
		phasesRaw[wipIdx].(map[string]any)["status"] = "done"
		out.CompletedIndex = wipIdx
	}

	// Promote the next todo entry to wip (next after the just-completed
	// one, or the first if none was wip yet).
	startFrom := wipIdx + 1
	for i := startFrom; i < len(phasesRaw); i++ {
		pm := phasesRaw[i].(map[string]any)
		if status, _ := pm["status"].(string); status == "todo" || status == "" {
			pm["status"] = "wip"
			idx := i
			out.NextIndex = &idx
			break
		}
	}

	// Bump typed phase_current. Convention: 1-based, matches the
	// meta.session semantic used in the document format
	// (sp-1-1 = session 1).
	switch {
	case out.NextIndex != nil:
		cur := *out.NextIndex + 1
		doc.PhaseCurrent = &cur
	case wipIdx >= 0:
		// Final phase just completed — leave phase_current pointing at
		// the last index + 1 and mark the doc complete.
		cur := wipIdx + 1
		doc.PhaseCurrent = &cur
		if doc.Status == models.StatusInProgress || doc.Status == models.StatusTodo {
			doc.Status = models.StatusComplete
		}
	default:
		// No wip phase and no todo phase to promote — caller's plan is
		// already fully advanced or in a state advance_phase shouldn't
		// touch.
		return nil, nil, errors.New("nothing to advance — no wip or todo phase found")
	}

	total := len(phasesRaw)
	doc.PhaseTotal = &total

	doc.Meta["phases"] = phasesRaw
	doc.Meta["phase_current"] = float64(*doc.PhaseCurrent)
	doc.Meta["phase_total"] = float64(total)

	if err := t.saveDoc(ctx, doc); err != nil {
		return nil, nil, err
	}
	out.Doc = doc
	return nil, out, nil
}
