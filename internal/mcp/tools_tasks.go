package mcp

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// tickResult is the lean payload from tick_task: just the flipped
// state. taskResult carries the edited task for update/add. Both attach
// the full document only when the caller passes return_doc, so a routine
// edit no longer re-serializes the whole plan into the session context.
type tickResult struct {
	TaskID string           `json:"task_id"`
	Done   bool             `json:"done"`
	Doc    *models.Document `json:"doc,omitempty"`
}

type taskResult struct {
	Task map[string]any   `json:"task"`
	Doc  *models.Document `json:"doc,omitempty"`
}

// --- tick_task --------------------------------------------------------

type tickTaskInput struct {
	DocID     string `json:"doc_id" jsonschema:"document id"`
	TaskID    string `json:"task_id" jsonschema:"task id within the document body"`
	ReturnDoc bool   `json:"return_doc,omitempty" jsonschema:"when true, also return the full updated document; default false"`
}

func (t *tools) tickTask(ctx context.Context, _ *sdk.CallToolRequest, in tickTaskInput) (*sdk.CallToolResult, *tickResult, error) {
	if in.TaskID == "" {
		return nil, nil, errors.New("task_id is required")
	}
	doc, sections, task, err := t.loadDocAndFindTask(ctx, in.DocID, in.TaskID)
	if err != nil {
		return nil, nil, err
	}

	current, _ := task["done"].(bool)
	task["done"] = !current

	setSections(doc.Body, sections)
	if err := t.saveDoc(ctx, doc); err != nil {
		return nil, nil, err
	}
	out := &tickResult{TaskID: in.TaskID, Done: !current}
	if in.ReturnDoc {
		out.Doc = doc
	}
	return nil, out, nil
}

// --- update_task ------------------------------------------------------

type updateTaskInput struct {
	DocID     string         `json:"doc_id" jsonschema:"document id"`
	TaskID    string         `json:"task_id" jsonschema:"task id"`
	Patch     map[string]any `json:"patch" jsonschema:"fields to overwrite — supports title, content, done, tags"`
	ReturnDoc bool           `json:"return_doc,omitempty" jsonschema:"when true, also return the full updated document; default false"`
}

var taskUpdatableFields = map[string]bool{
	"title":   true,
	"content": true,
	"done":    true,
	"tags":    true,
}

func (t *tools) updateTask(ctx context.Context, _ *sdk.CallToolRequest, in updateTaskInput) (*sdk.CallToolResult, *taskResult, error) {
	if in.TaskID == "" {
		return nil, nil, errors.New("task_id is required")
	}
	if len(in.Patch) == 0 {
		return nil, nil, errors.New("patch must contain at least one field")
	}
	for k := range in.Patch {
		if !taskUpdatableFields[k] {
			return nil, nil, fmt.Errorf("patch field %q is not editable — allowed: title, content, done, tags", k)
		}
	}

	doc, sections, task, err := t.loadDocAndFindTask(ctx, in.DocID, in.TaskID)
	if err != nil {
		return nil, nil, err
	}
	for k, v := range in.Patch {
		task[k] = v
	}

	setSections(doc.Body, sections)
	if err := t.saveDoc(ctx, doc); err != nil {
		return nil, nil, err
	}
	out := &taskResult{Task: task}
	if in.ReturnDoc {
		out.Doc = doc
	}
	return nil, out, nil
}

// --- add_task ---------------------------------------------------------

type addTaskInput struct {
	DocID       string         `json:"doc_id" jsonschema:"document id"`
	SectionID   string         `json:"section_id" jsonschema:"id of the subphase or task-list to append the task into"`
	Task        map[string]any `json:"task" jsonschema:"task object — must include id; typical fields: title, content, done, tags"`
	AfterTaskID string         `json:"after_task_id,omitempty" jsonschema:"insert immediately after this task id (otherwise appends)"`
	ReturnDoc   bool           `json:"return_doc,omitempty" jsonschema:"when true, also return the full updated document; default false"`
}

func (t *tools) addTask(ctx context.Context, _ *sdk.CallToolRequest, in addTaskInput) (*sdk.CallToolResult, *taskResult, error) {
	if in.SectionID == "" {
		return nil, nil, errors.New("section_id is required")
	}
	if in.Task == nil {
		return nil, nil, errors.New("task is required")
	}
	if id, _ := in.Task["id"].(string); id == "" {
		return nil, nil, errors.New("task.id is required")
	}

	doc, err := t.loadDoc(ctx, in.DocID)
	if err != nil {
		return nil, nil, err
	}
	sections, err := sectionsArray(doc.Body)
	if err != nil {
		return nil, nil, err
	}
	container := findTaskContainer(sections, in.SectionID)
	if container == nil {
		return nil, nil, fmt.Errorf("subphase or task-list %q not found", in.SectionID)
	}

	tasks, _ := container["tasks"].([]any)
	insertAt := len(tasks)
	if in.AfterTaskID != "" {
		found := false
		for i, traw := range tasks {
			tm, ok := traw.(map[string]any)
			if !ok {
				continue
			}
			if tid, _ := tm["id"].(string); tid == in.AfterTaskID {
				insertAt = i + 1
				found = true
				break
			}
		}
		if !found {
			return nil, nil, fmt.Errorf("after_task_id %q not found in subphase %q", in.AfterTaskID, in.SectionID)
		}
	}
	tasks = append(tasks, nil)
	copy(tasks[insertAt+1:], tasks[insertAt:])
	tasks[insertAt] = in.Task
	container["tasks"] = tasks

	setSections(doc.Body, sections)
	if err := t.saveDoc(ctx, doc); err != nil {
		return nil, nil, err
	}
	out := &taskResult{Task: in.Task}
	if in.ReturnDoc {
		out.Doc = doc
	}
	return nil, out, nil
}

// --- remove_task ------------------------------------------------------

type removeTaskInput struct {
	DocID  string `json:"doc_id" jsonschema:"document id"`
	TaskID string `json:"task_id" jsonschema:"task id"`
}

func (t *tools) removeTask(ctx context.Context, _ *sdk.CallToolRequest, in removeTaskInput) (*sdk.CallToolResult, *okResult, error) {
	if in.TaskID == "" {
		return nil, nil, errors.New("task_id is required")
	}
	doc, err := t.loadDoc(ctx, in.DocID)
	if err != nil {
		return nil, nil, err
	}
	sections, err := sectionsArray(doc.Body)
	if err != nil {
		return nil, nil, err
	}
	sp, idx, task := walkTaskByID(sections, in.TaskID)
	if task == nil {
		return nil, nil, fmt.Errorf("task %q not found", in.TaskID)
	}
	tasks, _ := sp["tasks"].([]any)
	sp["tasks"] = append(tasks[:idx], tasks[idx+1:]...)

	setSections(doc.Body, sections)
	if err := t.saveDoc(ctx, doc); err != nil {
		return nil, nil, err
	}
	return nil, &okResult{OK: true}, nil
}

// loadDocAndFindTask is the common preamble of tick/update_task: load
// the doc, parse sections, walk for the task. Returns the sections
// slice so the caller can write it back via setSections.
func (t *tools) loadDocAndFindTask(ctx context.Context, docID, taskID string) (
	doc *models.Document,
	sections []any,
	task map[string]any,
	err error,
) {
	doc, err = t.loadDoc(ctx, docID)
	if err != nil {
		return nil, nil, nil, err
	}
	sections, err = sectionsArray(doc.Body)
	if err != nil {
		return nil, nil, nil, err
	}
	_, _, task = walkTaskByID(sections, taskID)
	if task == nil {
		return nil, nil, nil, fmt.Errorf("task %q not found", taskID)
	}
	return doc, sections, task, nil
}
