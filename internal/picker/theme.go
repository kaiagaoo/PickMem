// Package picker implements the pickmem TUI: a full-screen grouped
// multi-select over the vault's active notes, with a lens overlay and
// live filter. Bubble Tea drives the model; Lip Gloss owns styling.
package picker

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Theme collects every styled surface the picker uses. Keeping them in one
// struct makes it trivial to add a --theme flag or drop in a Catppuccin
// palette later — swap the constructor, not the callsites.
type Theme struct {
	Title       lipgloss.Style
	GroupHeader lipgloss.Style
	Item        lipgloss.Style
	Cursor      lipgloss.Style
	Selected    lipgloss.Style
	Checkbox    lipgloss.Style // filled state
	CheckboxDim lipgloss.Style // empty state
	Tag         lipgloss.Style
	Footer      lipgloss.Style
	FooterKey   lipgloss.Style
	FilterBar   lipgloss.Style
	OverlayBox  lipgloss.Style
	OverlayItem lipgloss.Style
	Dim         lipgloss.Style
	Accent      lipgloss.Style
	Danger      lipgloss.Style
}

// NordTheme is the default. Cool, restrained, works on both light and dark
// terminals. Colors picked by ANSI name where reasonable so 8-color TERMs
// still render sensibly.
func NordTheme() Theme {
	var (
		accent = lipgloss.Color("#88C0D0") // frost — cursor, accents
		sel    = lipgloss.Color("#A3BE8C") // aurora green — selected
		dim    = lipgloss.Color("#4C566A") // polar night — dimmed text
		fg     = lipgloss.AdaptiveColor{Light: "#2E3440", Dark: "#ECEFF4"}
		muted  = lipgloss.AdaptiveColor{Light: "#4C566A", Dark: "#D8DEE9"}
		warn   = lipgloss.Color("#BF616A") // aurora red
	)
	return Theme{
		Title:       lipgloss.NewStyle().Foreground(accent).Bold(true),
		GroupHeader: lipgloss.NewStyle().Foreground(muted).Bold(true).MarginTop(1),
		Item:        lipgloss.NewStyle().Foreground(fg),
		Cursor:      lipgloss.NewStyle().Foreground(fg).Background(lipgloss.Color("#3B4252")).Bold(true),
		Selected:    lipgloss.NewStyle().Foreground(sel).Bold(true),
		Checkbox:    lipgloss.NewStyle().Foreground(sel),
		CheckboxDim: lipgloss.NewStyle().Foreground(dim),
		Tag:         lipgloss.NewStyle().Foreground(muted),
		Footer:      lipgloss.NewStyle().Foreground(muted).MarginTop(1),
		FooterKey:   lipgloss.NewStyle().Foreground(accent).Bold(true),
		FilterBar:   lipgloss.NewStyle().Foreground(fg).Bold(true),
		OverlayBox:  lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(0, 1),
		OverlayItem: lipgloss.NewStyle().Foreground(fg),
		Dim:         lipgloss.NewStyle().Foreground(dim),
		Accent:      lipgloss.NewStyle().Foreground(accent),
		Danger:      lipgloss.NewStyle().Foreground(warn),
	}
}

// PlainTheme strips all color. Used when $NO_COLOR is set (per
// https://no-color.org) or when the terminal reports as monochrome.
func PlainTheme() Theme {
	plain := lipgloss.NewStyle()
	bold := lipgloss.NewStyle().Bold(true)
	return Theme{
		Title:       bold,
		GroupHeader: bold.MarginTop(1),
		Item:        plain,
		Cursor:      lipgloss.NewStyle().Reverse(true),
		Selected:    bold,
		Checkbox:    bold,
		CheckboxDim: plain,
		Tag:         plain,
		Footer:      plain.MarginTop(1),
		FooterKey:   bold,
		FilterBar:   bold,
		OverlayBox:  lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1),
		OverlayItem: plain,
		Dim:         plain,
		Accent:      bold,
		Danger:      bold,
	}
}

// DefaultTheme returns Nord unless $NO_COLOR is set. Callers can override
// via the picker's WithTheme option.
func DefaultTheme() Theme {
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return PlainTheme()
	}
	return NordTheme()
}
