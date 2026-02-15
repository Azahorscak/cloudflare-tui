package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Azahorscak/cloudflare-tui/internal/api"
)

func TestNew(t *testing.T) {
	m := New(nil)
	if m.currentView != ViewZones {
		t.Errorf("initial view = %d, want ViewZones (%d)", m.currentView, ViewZones)
	}
}

func TestModel_ViewRouting(t *testing.T) {
	m := New(nil)
	// Default should render zones view.
	out := m.View()
	if out == "" {
		t.Fatal("View() returned empty string")
	}
}

func TestModel_SelectZoneMsg(t *testing.T) {
	m := New(nil)
	zone := api.Zone{ID: "z1", Name: "example.com"}

	updated, _ := m.Update(selectZoneMsg{zone: zone})
	model := updated.(Model)

	if model.currentView != ViewRecords {
		t.Errorf("view after selectZoneMsg = %d, want ViewRecords (%d)", model.currentView, ViewRecords)
	}
}

func TestModel_BackToZonesMsg(t *testing.T) {
	m := New(nil)
	// Simulate being on the records view.
	m.currentView = ViewRecords

	updated, _ := m.Update(backToZonesMsg{})
	model := updated.(Model)

	if model.currentView != ViewZones {
		t.Errorf("view after backToZonesMsg = %d, want ViewZones (%d)", model.currentView, ViewZones)
	}
}

func TestModel_CtrlCQuits(t *testing.T) {
	m := New(nil)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected a quit command from Ctrl+C, got nil")
	}
	// Execute the cmd to get the message; tea.Quit returns a tea.QuitMsg.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestModel_WindowSizeMsg(t *testing.T) {
	m := New(nil)
	sizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}

	updated, _ := m.Update(sizeMsg)
	model := updated.(Model)

	if model.width != 120 {
		t.Errorf("width = %d, want 120", model.width)
	}
	if model.height != 40 {
		t.Errorf("height = %d, want 40", model.height)
	}
}
