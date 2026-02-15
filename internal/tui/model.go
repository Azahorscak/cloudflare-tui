// Package tui implements the Bubble Tea terminal UI.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Azahorscak/cloudflare-tui/internal/api"
)

// View represents which screen is currently active.
type View int

const (
	ViewZones   View = iota
	ViewRecords
)

// selectZoneMsg signals a transition from zones to the records view.
type selectZoneMsg struct {
	zone api.Zone
}

// backToZonesMsg signals a transition back to the zone-selection view.
type backToZonesMsg struct{}

// Model is the root Bubble Tea model.
type Model struct {
	currentView View
	client      *api.Client
	zones       ZonesModel
	records     RecordsModel
	width       int
	height      int
}

// New creates a new root Model with the given API client.
func New(client *api.Client) Model {
	return Model{
		currentView: ViewZones,
		client:      client,
		zones:       NewZonesModel(client),
	}
}

func (m Model) Init() tea.Cmd {
	return m.zones.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// fall through so the active sub-model also receives the resize

	case selectZoneMsg:
		m.currentView = ViewRecords
		m.records = NewRecordsModel(m.client, msg.zone, m.width, m.height)
		return m, m.records.Init()

	case backToZonesMsg:
		m.currentView = ViewZones
		return m, nil
	}

	var cmd tea.Cmd
	switch m.currentView {
	case ViewZones:
		m.zones, cmd = m.zones.Update(msg)
	case ViewRecords:
		m.records, cmd = m.records.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	switch m.currentView {
	case ViewRecords:
		return m.records.View()
	default:
		return m.zones.View()
	}
}
