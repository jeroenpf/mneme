package api

import (
	"strings"
	"unicode"
)

// slugify converts a free-form title into a kebab-case ASCII slug.
// Non-alphanumeric characters become "-", runs of "-" collapse, and
// leading/trailing "-" are trimmed. Empty input (or input that produces
// an empty slug, e.g. "***") yields "doc" as a stable fallback.
func slugify(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevDash := true // suppresses a leading dash
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if r > unicode.MaxASCII {
				// Non-ASCII letters/digits — drop rather than try to
				// transliterate. Mneme document IDs come from titles
				// authored by Claude Code, which are ASCII in practice.
				if !prevDash {
					b.WriteByte('-')
					prevDash = true
				}
				continue
			}
			b.WriteRune(unicode.ToLower(r))
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if out == "" {
		return "doc"
	}
	return out
}
