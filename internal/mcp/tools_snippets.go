package mcp

import (
	"context"
	"errors"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

type saveSnippetInput struct {
	ID          string   `json:"id,omitempty" jsonschema:"snippet id to update; omit to create a new snippet"`
	Title       string   `json:"title,omitempty" jsonschema:"short snippet title; required when creating"`
	Project     string   `json:"project,omitempty" jsonschema:"project slug; omit for a global (cross-project) snippet"`
	Language    string   `json:"language,omitempty" jsonschema:"source language, e.g. go, typescript, sql (free-text, lowercased)"`
	Content     string   `json:"content,omitempty" jsonschema:"the code/pattern itself; required when creating"`
	Tags        []string `json:"tags,omitempty" jsonschema:"free-form tags for filtering; on update pass [] to clear, omit to leave unchanged"`
	Description string   `json:"description,omitempty" jsonschema:"what the snippet is for and when to use it"`
}

func (t *tools) saveSnippet(ctx context.Context, _ *sdk.CallToolRequest, in saveSnippetInput) (*sdk.CallToolResult, *models.Snippet, error) {
	project := strings.TrimSpace(in.Project)
	var projectPtr *string
	if project != "" {
		projectPtr = &project
	}
	language := strings.ToLower(strings.TrimSpace(in.Language))

	// Update path: id given -> load, apply provided fields, save.
	if id := strings.TrimSpace(in.ID); id != "" {
		sn, err := t.store.GetSnippet(ctx, id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, nil, errors.New("snippet not found — omit id to save a new snippet")
			}
			return nil, nil, translateStoreErr(err)
		}
		if v := strings.TrimSpace(in.Title); v != "" {
			sn.Title = v
		}
		if v := strings.TrimSpace(in.Content); v != "" {
			sn.Content = v
		}
		if language != "" {
			sn.Language = language
		}
		if in.Description != "" {
			sn.Description = in.Description
		}
		if in.Project != "" {
			sn.Project = projectPtr
		}
		// tags: nil = omitted (leave unchanged); non-nil (incl. []) = replace.
		if in.Tags != nil {
			sn.Tags = in.Tags
		}
		if err := t.store.UpdateSnippet(ctx, sn); err != nil {
			return nil, nil, translateStoreErr(err)
		}
		t.enqueue("snippets", sn.ID)
		return nil, sn, nil
	}

	// Create path.
	title := strings.TrimSpace(in.Title)
	content := strings.TrimSpace(in.Content)
	if title == "" {
		return nil, nil, errors.New("title is required")
	}
	if content == "" {
		return nil, nil, errors.New("content is required")
	}
	sn := &models.Snippet{
		Title:       title,
		Project:     projectPtr,
		Language:    language,
		Content:     content,
		Tags:        in.Tags,
		Description: in.Description,
	}
	if err := t.store.CreateSnippet(ctx, sn); err != nil {
		return nil, nil, translateStoreErr(err)
	}
	t.enqueue("snippets", sn.ID)
	return nil, sn, nil
}

type snippetsOutput struct {
	Snippets []*models.Snippet `json:"snippets"`
}

// snippetFilterFrom builds a SnippetFilter from the shared MCP filter
// args, normalizing language to lowercase and lifting a single tag into
// the containment slice.
func snippetFilterFrom(project, language, tag string, limit int) store.SnippetFilter {
	f := store.SnippetFilter{Limit: limit}
	if p := strings.TrimSpace(project); p != "" {
		f.Project = &p
	}
	if l := strings.ToLower(strings.TrimSpace(language)); l != "" {
		f.Language = &l
	}
	if tg := strings.TrimSpace(tag); tg != "" {
		f.Tags = []string{tg}
	}
	return f
}

type getSnippetsInput struct {
	Project  string `json:"project,omitempty" jsonschema:"filter to a project slug; omit for all snippets incl. global"`
	Language string `json:"language,omitempty" jsonschema:"filter by source language (case-insensitive)"`
	Tag      string `json:"tag,omitempty" jsonschema:"filter to snippets carrying this tag"`
	Limit    int    `json:"limit,omitempty" jsonschema:"max rows (newest first); 0 = no limit"`
}

func (t *tools) getSnippets(ctx context.Context, _ *sdk.CallToolRequest, in getSnippetsInput) (*sdk.CallToolResult, *snippetsOutput, error) {
	f := snippetFilterFrom(in.Project, in.Language, in.Tag, in.Limit)
	sns, err := t.store.ListSnippets(ctx, f)
	if err != nil {
		return nil, nil, translateStoreErr(err)
	}
	return nil, &snippetsOutput{Snippets: sns}, nil
}

type searchSnippetsInput struct {
	Query    string `json:"query" jsonschema:"natural-language search over title, description, content"`
	Project  string `json:"project,omitempty" jsonschema:"optional project slug to scope the search"`
	Language string `json:"language,omitempty" jsonschema:"optional source language filter (case-insensitive)"`
	Tag      string `json:"tag,omitempty" jsonschema:"optional tag filter"`
	Limit    int    `json:"limit,omitempty" jsonschema:"max ranked results; 0 = no limit"`
}

func (t *tools) searchSnippets(ctx context.Context, _ *sdk.CallToolRequest, in searchSnippetsInput) (*sdk.CallToolResult, *snippetsOutput, error) {
	q := strings.TrimSpace(in.Query)
	if q == "" {
		return nil, nil, errors.New("query is required")
	}
	f := snippetFilterFrom(in.Project, in.Language, in.Tag, in.Limit)
	sns, err := t.store.SearchSnippets(ctx, q, f)
	if err != nil {
		return nil, nil, translateStoreErr(err)
	}
	return nil, &snippetsOutput{Snippets: sns}, nil
}
