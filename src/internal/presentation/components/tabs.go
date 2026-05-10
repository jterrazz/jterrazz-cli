package components

import (
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/theme"
)

// Tabs is a horizontal tab strip — one line of labels with the active tab
// highlighted in the primary accent and inactive tabs muted. Shared by every
// tabbed TUI (j status, j config) so the look stays unified.
//
// Tabs is value-type semantically; methods that mutate take a pointer
// receiver. Render is read-only.
type Tabs struct {
	Labels []string
	Active int
}

// Render returns the tab strip as a single line of styled text. Caller
// supplies the rendering width but the strip itself only spans as far as
// its labels — extra width is unused.
func (t Tabs) Render(width int) string {
	_ = width // currently informational; reserved for future centring
	var b strings.Builder
	b.WriteString(" ")
	for i, label := range t.Labels {
		if i > 0 {
			b.WriteString("  ")
		}
		if i == t.Active {
			b.WriteString(theme.Selected.Render("● " + label))
		} else {
			b.WriteString(theme.Muted.Render("○ " + label))
		}
	}
	return b.String()
}

// Next moves the active tab right. No-op at the end.
func (t *Tabs) Next() {
	if t.Active < len(t.Labels)-1 {
		t.Active++
	}
}

// Prev moves the active tab left. No-op at the start.
func (t *Tabs) Prev() {
	if t.Active > 0 {
		t.Active--
	}
}

// SetActive jumps directly to an index, clamped to bounds.
func (t *Tabs) SetActive(i int) {
	if i < 0 {
		i = 0
	}
	if i >= len(t.Labels) {
		i = len(t.Labels) - 1
	}
	t.Active = i
}
