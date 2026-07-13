package mcp

import (
	"context"
	"errors"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

type logSolutionInput struct {
	ID               string   `json:"id,omitempty" jsonschema:"solution id to update; omit to log a new solution"`
	Project          string   `json:"project,omitempty" jsonschema:"project slug; omit for a global (cross-project) gotcha"`
	ErrorDescription string   `json:"error_description,omitempty" jsonschema:"the error/symptom as you would search for it later; required when creating"`
	Solution         string   `json:"solution,omitempty" jsonschema:"the fix that worked; required when creating"`
	Tags             []string `json:"tags,omitempty" jsonschema:"free-form tags for filtering; on update pass [] to clear, omit to leave unchanged"`
	SourceURL        string   `json:"source_url,omitempty" jsonschema:"optional link to the issue/StackOverflow/doc that helped"`
}

func (t *tools) logSolution(ctx context.Context, _ *sdk.CallToolRequest, in logSolutionInput) (*sdk.CallToolResult, *models.Solution, error) {
	project := strings.TrimSpace(in.Project)
	var projectPtr *string
	if project != "" {
		projectPtr = &project
	}

	// Update path: id given -> load, apply provided fields, save.
	if id := strings.TrimSpace(in.ID); id != "" {
		sol, err := t.store.GetSolution(ctx, id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, nil, errors.New("solution not found — omit id to log a new solution")
			}
			return nil, nil, translateStoreErr(err)
		}
		if v := strings.TrimSpace(in.ErrorDescription); v != "" {
			sol.ErrorDescription = v
		}
		if v := strings.TrimSpace(in.Solution); v != "" {
			sol.Solution = v
		}
		if in.SourceURL != "" {
			sol.SourceURL = in.SourceURL
		}
		if in.Project != "" {
			sol.Project = projectPtr
		}
		// tags: nil = omitted (leave unchanged); non-nil (incl. []) = replace.
		if in.Tags != nil {
			sol.Tags = in.Tags
		}
		if err := t.store.UpdateSolution(ctx, sol); err != nil {
			return nil, nil, translateStoreErr(err)
		}
		t.enqueue("solutions", sol.ID)
		return nil, sol, nil
	}

	// Create path.
	errDesc := strings.TrimSpace(in.ErrorDescription)
	solution := strings.TrimSpace(in.Solution)
	if errDesc == "" {
		return nil, nil, errors.New("error_description is required")
	}
	if solution == "" {
		return nil, nil, errors.New("solution is required")
	}
	sol := &models.Solution{
		Project:          projectPtr,
		ErrorDescription: errDesc,
		Solution:         solution,
		Tags:             in.Tags,
		SourceURL:        in.SourceURL,
	}
	if err := t.store.CreateSolution(ctx, sol); err != nil {
		return nil, nil, translateStoreErr(err)
	}
	t.enqueue("solutions", sol.ID)
	return nil, sol, nil
}

type solutionsOutput struct {
	Solutions []*models.Solution `json:"solutions"`
}

type findSolutionInput struct {
	Query   string `json:"query" jsonschema:"the error/symptom to search for — natural language over error_description + solution"`
	Project string `json:"project,omitempty" jsonschema:"optional project slug to scope the search"`
	Tag     string `json:"tag,omitempty" jsonschema:"optional tag filter"`
	Limit   *int   `json:"limit,omitempty" jsonschema:"max ranked results; omit for the top 3; 0 = no limit"`
}

func (t *tools) findSolution(ctx context.Context, _ *sdk.CallToolRequest, in findSolutionInput) (*sdk.CallToolResult, *solutionsOutput, error) {
	q := strings.TrimSpace(in.Query)
	if q == "" {
		return nil, nil, errors.New("query is required")
	}
	limit := 3 // default: top 3 (omitted). An explicit 0 means "no cap".
	if in.Limit != nil {
		limit = *in.Limit
	}
	f := store.SolutionFilter{Limit: limit}
	if p := strings.TrimSpace(in.Project); p != "" {
		f.Project = &p
	}
	if tg := strings.TrimSpace(in.Tag); tg != "" {
		f.Tags = []string{tg}
	}
	sols, err := t.store.SearchSolutions(ctx, q, f)
	if err != nil {
		return nil, nil, translateStoreErr(err)
	}
	return nil, &solutionsOutput{Solutions: sols}, nil
}
