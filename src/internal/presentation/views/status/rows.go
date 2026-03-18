package status

import (
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/components"
)

// loadingIndicator returns a spinner while loading, or a static dot after
func (m Model) loadingIndicator() string {
	if m.allLoaded {
		return components.Muted("·")
	}
	return m.spinner.View()
}
