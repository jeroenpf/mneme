package mcp

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/jeroenpf/mneme/internal/store"
)

// LintHit is one violation found by lintBody: where (block id, walk path,
// field) and what (the label the detectors return, or a structural
// problem), plus a short excerpt for triage without fetching the doc.
type LintHit struct {
	BlockID string `json:"block_id,omitempty"`
	Path    string `json:"path"`
	Field   string `json:"field"`
	Found   string `json:"found"`
	Excerpt string `json:"excerpt,omitempty"`
}

// LintDocHit is a LintHit stamped with the owning document — doc_id plus
// block_id are exactly what update_section/update_task need to fix it.
type LintDocHit struct {
	LintHit
	DocID     string `json:"doc_id"`
	DocTitle  string `json:"doc_title"`
	DocStatus string `json:"doc_status"`
}

// LintReport is the result of a store-wide lint sweep.
type LintReport struct {
	Hits         []LintDocHit `json:"hits"`
	DocsScanned  int          `json:"docs_scanned"`
	DocsWithHits int          `json:"docs_with_hits"`
}

// LintStore sweeps every stored document — all projects and statuses,
// archived included — through lintBody. Read-only: it reports what the
// write path would reject today and changes nothing. Consumed by the
// `mneme lint` CLI subcommand (the admin home of the former MCP tool).
func LintStore(ctx context.Context, st store.Store) (*LintReport, error) {
	docs, err := st.ListDocuments(ctx, store.Filter{})
	if err != nil {
		return nil, err
	}
	out := &LintReport{Hits: []LintDocHit{}, DocsScanned: len(docs)}
	for _, d := range docs {
		hs := lintBody(d.Body)
		if len(hs) == 0 {
			continue
		}
		out.DocsWithHits++
		for _, h := range hs {
			out.Hits = append(out.Hits, LintDocHit{LintHit: h, DocID: d.ID, DocTitle: d.Title, DocStatus: d.Status})
		}
	}
	return out, nil
}

// lintBody is the collect-mode twin of validateBody: same field maps and
// detectors, but it accumulates every violation instead of failing fast,
// and also reports structural problems (unknown types, unknown fields) —
// the silent-drop trap in documents that predate write-path validation.
func lintBody(body map[string]any) []LintHit {
	if body == nil {
		return nil
	}
	arr, _ := body["sections"].([]any)
	return lintBlocks(arr, "body.sections")
}

func lintBlocks(blocks []any, path string) []LintHit {
	var hits []LintHit
	for i, raw := range blocks {
		p := fmt.Sprintf("%s[%d]", path, i)
		b, ok := raw.(map[string]any)
		if !ok {
			hits = append(hits, LintHit{Path: p, Field: "type", Found: "not an object"})
			continue
		}
		id, _ := b["id"].(string)
		t, _ := b["type"].(string)
		if !validBlockTypes[t] {
			hits = append(hits, LintHit{BlockID: id, Path: p, Field: "type", Found: fmt.Sprintf("unknown block type %q", t)})
			continue
		}
		for k := range b {
			if k != "id" && k != "type" && !slices.Contains(blockFields[t], k) {
				hits = append(hits, LintHit{BlockID: id, Path: p, Field: k, Found: "unknown field"})
			}
		}
		for _, f := range paragraphProseFields[t] {
			s, _ := b[f].(string)
			if sig := detectBlockStructure(s); sig != "" {
				hits = append(hits, LintHit{BlockID: id, Path: p, Field: f, Found: sig, Excerpt: excerpt(s)})
			}
		}
		for _, f := range inlineProseFields[t] {
			s, _ := b[f].(string)
			if sig := detectBlockMarkdown(s); sig != "" {
				hits = append(hits, LintHit{BlockID: id, Path: p, Field: f, Found: sig, Excerpt: excerpt(s)})
			}
		}
		switch t {
		case "task-list", "subphase":
			tasks, _ := b["tasks"].([]any)
			for j, traw := range tasks {
				task, _ := traw.(map[string]any)
				tid, _ := task["id"].(string)
				for _, f := range []string{"title", "content"} {
					s, _ := task[f].(string)
					if sig := detectBlockMarkdown(s); sig != "" {
						hits = append(hits, LintHit{BlockID: tid, Path: fmt.Sprintf("%s.tasks[%d]", p, j), Field: f, Found: sig, Excerpt: excerpt(s)})
					}
				}
			}
		case "key-value":
			data, _ := b["data"].(map[string]any)
			for k, vraw := range data {
				s, _ := vraw.(string)
				if sig := detectBlockMarkdown(s); sig != "" {
					hits = append(hits, LintHit{BlockID: id, Path: p, Field: fmt.Sprintf("data[%q]", k), Found: sig, Excerpt: excerpt(s)})
				}
			}
		}
		if children, ok := b["children"].([]any); ok {
			hits = append(hits, lintBlocks(children, p+".children")...)
		}
	}
	return hits
}

// excerpt flattens newlines and caps at 80 runes — enough to triage a
// hit without fetching the document.
func excerpt(s string) string {
	s = strings.ReplaceAll(s, "\n", " ⏎ ")
	r := []rune(s)
	if len(r) > 80 {
		return string(r[:80]) + "…"
	}
	return s
}
