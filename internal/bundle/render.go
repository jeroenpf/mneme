package bundle

import (
	"fmt"
	"sort"
	"strings"
)

const (
	decisionExcerptLen = 180
	snippetExcerptLen  = 120
)

// excerpt collapses all whitespace in s to single spaces and truncates the
// result to at most n runes on a word boundary, appending an ellipsis when it
// had to cut. Returns "" for empty/whitespace-only input. Used to lift a small,
// scannable line of context out of long rationale or snippet bodies.
func excerpt(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len([]rune(s)) <= n {
		return s
	}
	cut := string([]rune(s)[:n])
	if i := strings.LastIndex(cut, " "); i > 0 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,.;:") + "…"
}

// renderMarkdown produces the paste-ready session digest. Every section
// renders "_none_" when empty so the shape is stable and auditable.
func renderMarkdown(b *Bundle) string {
	var sb strings.Builder
	title := b.Project
	if b.Area != nil && *b.Area != "" {
		title += " / " + *b.Area
	}
	fmt.Fprintf(&sb, "# Context bundle — %s\n\n", title)

	sb.WriteString("## Memory\n")
	if len(b.Memory) == 0 {
		sb.WriteString("_none_\n")
	} else {
		keys := make([]string, 0, len(b.Memory))
		for k := range b.Memory {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&sb, "- **%s**: %s\n", k, b.Memory[k])
		}
	}

	sb.WriteString("\n## Env\n")
	if len(b.Env) == 0 {
		sb.WriteString("_none_\n")
	} else {
		for _, e := range b.Env {
			if e.Description != nil && *e.Description != "" {
				fmt.Fprintf(&sb, "- %s = %s — %s\n", e.Key, e.Value, *e.Description)
			} else {
				fmt.Fprintf(&sb, "- %s = %s\n", e.Key, e.Value)
			}
		}
	}

	sb.WriteString("\n## Active plan\n")
	if b.ActivePlan == nil {
		sb.WriteString("_no in-progress plan — start one, or pick up a todo plan_\n")
	} else {
		p := b.ActivePlan
		phase := "—"
		if p.PhaseCurrent != nil && p.PhaseTotal != nil {
			phase = fmt.Sprintf("%d/%d", *p.PhaseCurrent, *p.PhaseTotal)
		}
		fmt.Fprintf(&sb, "**%s** — phase %s (%s)\n", p.Title, phase, p.Status)
		if p.ActivePhase != "" {
			fmt.Fprintf(&sb, "Current phase: **%s**\n", p.ActivePhase)
		}
	}

	sb.WriteString("\n## Next tasks\n")
	if len(b.NextTasks) == 0 {
		if b.ActivePlan == nil {
			sb.WriteString("_none_\n")
		} else {
			sb.WriteString("_none incomplete — the active plan's tasks are all done_\n")
		}
	} else {
		for _, t := range b.NextTasks {
			if t.Phase != "" {
				fmt.Fprintf(&sb, "- [ ] %s — %s `%s`\n", t.Title, t.Phase, t.ID)
			} else {
				fmt.Fprintf(&sb, "- [ ] %s `%s`\n", t.Title, t.ID)
			}
		}
	}

	if len(b.Blockers) > 0 {
		sb.WriteString("\n## Blockers\n")
		for _, bl := range b.Blockers {
			fmt.Fprintf(&sb, "- %s `%s`\n", bl.Title, bl.ID)
		}
	}

	if len(b.Deferred) > 0 {
		sb.WriteString("\n## Deferred (from last session)\n")
		for _, d := range b.Deferred {
			fmt.Fprintf(&sb, "- %s\n", d)
		}
	}

	sb.WriteString("\n## Recent decisions\n")
	if len(b.Decisions) == 0 {
		sb.WriteString("_none_\n")
	} else {
		for _, d := range b.Decisions {
			fmt.Fprintf(&sb, "- **%s** — %s (%s)\n", d.Title, d.Status, d.CreatedAt.Format("2006-01-02"))
			if r := excerpt(d.Rationale, decisionExcerptLen); r != "" {
				fmt.Fprintf(&sb, "  %s\n", r)
			}
		}
	}

	sb.WriteString("\n## Snippets\n")
	if len(b.Snippets) == 0 {
		sb.WriteString("_none_\n")
	} else {
		for _, s := range b.Snippets {
			lang := s.Language
			if lang == "" {
				lang = "text"
			}
			ex := excerpt(s.Description, snippetExcerptLen)
			if ex == "" {
				ex = excerpt(s.Content, snippetExcerptLen)
			}
			if ex != "" {
				fmt.Fprintf(&sb, "- **%s** [%s] — %s\n", s.Title, lang, ex)
			} else {
				fmt.Fprintf(&sb, "- **%s** [%s]\n", s.Title, lang)
			}
		}
	}

	sb.WriteString("\n## Recent journal\n")
	if len(b.Journal) == 0 {
		sb.WriteString("_none_\n")
	} else {
		for _, j := range b.Journal {
			ref := j.SessionRef
			if ref == "" {
				ref = "—"
			}
			fmt.Fprintf(&sb, "- %s: %s (%s)\n", ref, j.Summary, j.CreatedAt.Format("2006-01-02"))
		}
	}

	return strings.TrimRight(sb.String(), "\n") + "\n"
}
