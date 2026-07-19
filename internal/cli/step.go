package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mattn/go-isatty"
)

// spinnerFrames is a braille spinner cycle used for long-step feedback.
var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

// runStep runs action while showing progress for a potentially slow operation
// (mkcert, the first migration). On a real terminal it animates a spinner in
// place; on a non-TTY writer (tests, piped output) it prints the title once and
// runs the action. The action's error is returned unchanged.
func runStep(out io.Writer, title string, action func() error) error {
	if !isTerminal(out) {
		fmt.Fprintf(out, "→ %s\n", title)
		return action()
	}

	done := make(chan struct{})
	var actionErr error
	go func() {
		actionErr = action()
		close(done)
	}()

	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	for i := 0; ; i++ {
		select {
		case <-done:
			mark := "✓"
			if actionErr != nil {
				mark = "✗"
			}
			fmt.Fprintf(out, "\r\033[K%s %s\n", mark, title)
			return actionErr
		case <-ticker.C:
			fmt.Fprintf(out, "\r%c %s", spinnerFrames[i%len(spinnerFrames)], title)
		}
	}
}

// isTerminal reports whether out is an interactive terminal (so animation is
// appropriate). Non-*os.File writers are never terminals.
func isTerminal(out io.Writer) bool {
	f, ok := out.(*os.File)
	return ok && isatty.IsTerminal(f.Fd())
}
