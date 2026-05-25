package mcp

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// taskOutput is the structured payload returned from task-editing
// tools. The doc field carries the freshly-saved document so the
// caller can avoid a follow-up get_document.
type taskOutput struct {
	Task map[string]any   `json:"task"`
	Doc  *models.Document `json:"doc"`
}

// --- tick_task --------------------------------------------------------

type tickTaskInput struct {
	DocID  string `json:"doc_id" jsonschema:"document id"`
	TaskID string `json:"task_id" jsonschema:"task id within the document body"`
}

func (t *tools) tickTask(ctx context.Context, _ *sdk.CallToolRequest, in tickTaskInput) (*sdk.CallToolResult, *taskOutput, error) {
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
	return nil, &taskOutput{Task: task, Doc: doc}, nil
}

// --- update_task ------------------------------------------------------

type updateTaskInput struct {
	DocID  string         `json:"doc_id" jsonschema:"document id"`
	TaskID string         `json:"task_id" jsonschema:"task id"`
	Patch  map[string]any `json:"patch" jsonschema:"fields to overwrite — supports title, content, done, tags"`
}

var taskUpdatableFields = map[string]bool{
	"title":   true,
	"content": true,
	"done":    true,
	"tags":    true,
}

func (t *tools) updateTask(ctx context.Context, _ *sdk.CallToolRequest, in updateTaskInput) (*sdk.CallToolResult, *taskOutput, error) {
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
	return nil, &taskOutput{Task: task, Doc: doc}, nil
}

// --- add_task ---------------------------------------------------------

type addTaskInput struct {
	DocID       string         `json:"doc_id" jsonschema:"document id"`
	SectionID   string         `json:"section_id" jsonschema:"subphase id this task belongs to"`
	Task        map[string]any `json:"task" jsonschema:"task object — must include id; typical fields: title, content, done, tags"`
	AfterTaskID string         `json:"after_task_id,omitempty" jsonschema:"insert immediately after this task id (otherwise appends)"`
}

func (t *tools) addTask(ctx context.Context, _ *sdk.CallToolRequest, in addTaskInput) (*sdk.CallToolResult, *taskOutput, error) {
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
	sp := findSubphase(sections, in.SectionID)
	if sp == nil {
		return nil, nil, fmt.Errorf("subphase %q not found", in.SectionID)
	}

	tasks, _ := sp["tasks"].([]any)
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
	sp["tasks"] = tasks

	setSections(doc.Body, sections)
	if err := t.saveDoc(ctx, doc); err != nil {
		return nil, nil, err
	}
	return nil, &taskOutput{Task: in.Task, Doc: doc}, nil
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
