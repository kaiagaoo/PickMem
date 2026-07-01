package picker

import "github.com/charmbracelet/bubbles/key"

// keys is the picker's key map. Named bindings live in one struct so the
// footer help renders exactly what Update handles — no drift.
type keys struct {
	Up       key.Binding
	Down     key.Binding
	Toggle   key.Binding
	Filter   key.Binding
	Lens     key.Binding
	SaveLens key.Binding
	Confirm  key.Binding
	Cancel   key.Binding
	// Overlay/prompt keys reuse Confirm/Cancel; filter mode uses Filter to
	// exit back to browse.
}

func defaultKeys() keys {
	return keys{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" ", "space"),
			key.WithHelp("space", "toggle"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Lens: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "lens"),
		),
		SaveLens: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "save lens"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q/esc", "cancel"),
		),
	}
}
