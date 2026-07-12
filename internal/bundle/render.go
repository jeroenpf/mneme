package bundle

import (
	"fmt"
	"sort"
	"strings"
)

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

	sb.WriteString("\n## Active plan\n")
	if b.ActivePlan == nil {
		sb.WriteString("_none_\n")
	} else {
		p := b.ActivePlan
		phase := "—"
		if p.PhaseCurrent != nil && p.PhaseTotal != nil {
			phase = fmt.Sprintf("%d/%d", *p.PhaseCurrent, *p.PhaseTotal)
		}
		fmt.Fprintf(&sb, "**%s** — phase %s (%s)\n", p.Title, phase, p.Status)
	}

	sb.WriteString("\n## Recent decisions\n")
	if len(b.Decisions) == 0 {
		sb.WriteString("_none_\n")
	} else {
		for _, d := range b.Decisions {
			fmt.Fprintf(&sb, "- %s — %s (%s)\n", d.Title, d.Status, d.CreatedAt.Format("2006-01-02"))
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
			fmt.Fprintf(&sb, "- %s [%s]\n", s.Title, lang)
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
