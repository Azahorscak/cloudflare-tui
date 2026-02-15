// Package tui implements the Bubble Tea terminal UI.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// View represents which screen is currently active.
type View int

const (
	ViewZones   View = iota
	ViewRecords
)

// Model is the root Bubble Tea model.
type Model struct {
	currentView View
}

// New creates a new root Model.
func New() Model {
	return Model{currentView: ViewZones}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	return "cloudflare-tui: not yet implemented\nPress Ctrl+C to quit."
}
