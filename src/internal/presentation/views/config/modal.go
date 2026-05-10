package configview

import (
	"github.com/charmbracelet/huh"
	"github.com/jterrazz/jterrazz-cli/src/internal/config"
)

// modalActive reports whether the input-collection modal is currently
// being shown.
func (m Model) modalActive() bool {
	return m.form != nil
}

// buildModal initialises a huh.Form for the script's Inputs and stashes the
// bindings on the model so values can be read back when the form completes.
//
// One *string per input is allocated so huh can write directly into it.
func (m *Model) buildModal(s *config.Script) {
	m.formScript = s
	m.formBindings = make([]*string, len(s.Inputs))

	fields := make([]huh.Field, 0, len(s.Inputs))
	for i, in := range s.Inputs {
		val := in.Default
		m.formBindings[i] = &val
		fields = append(fields, buildField(in, m.formBindings[i]))
	}

	m.form = huh.NewForm(huh.NewGroup(fields...)).
		WithTheme(huh.ThemeBase()).
		WithShowHelp(false)
}

// buildField turns a ScriptInput into the matching huh.Field, with its
// value pointer wired up so writes flow back to the binding.
func buildField(in config.ScriptInput, ptr *string) huh.Field {
	switch in.Kind {
	case config.InputPassword:
		f := huh.NewInput().
			Title(in.Label).
			Description(in.Help).
			EchoMode(huh.EchoModePassword).
			Value(ptr)
		if in.Validate != nil {
			f.Validate(in.Validate)
		}
		return f
	case config.InputSelect:
		opts := make([]huh.Option[string], 0, len(in.Options))
		for _, o := range in.Options {
			opts = append(opts, huh.NewOption(o, o))
		}
		return huh.NewSelect[string]().
			Title(in.Label).
			Description(in.Help).
			Options(opts...).
			Value(ptr)
	case config.InputConfirm:
		// huh's Confirm binds to *bool; we stringify "yes"/"no" so the
		// binding stays uniformly *string. The InstallFn reads via
		// values.Get(name) and compares against "yes" / "no".
		return huh.NewSelect[string]().
			Title(in.Label).
			Description(in.Help).
			Options(
				huh.NewOption("Yes", "yes"),
				huh.NewOption("No", "no"),
			).
			Value(ptr)
	default: // InputText
		f := huh.NewInput().
			Title(in.Label).
			Description(in.Help).
			Value(ptr)
		if in.Validate != nil {
			f.Validate(in.Validate)
		}
		return f
	}
}

// collectModalValues reads the live form bindings back into a config.InputValues
// map keyed by Input.Name.
func (m Model) collectModalValues() config.InputValues {
	values := config.InputValues{}
	if m.formScript == nil {
		return values
	}
	for i, in := range m.formScript.Inputs {
		if i < len(m.formBindings) && m.formBindings[i] != nil {
			values[in.Name] = *m.formBindings[i]
		}
	}
	return values
}

// closeModal clears modal state. Called on completion (after queueing the
// install action) or on abort (esc).
func (m *Model) closeModal() {
	m.form = nil
	m.formScript = nil
	m.formBindings = nil
}
