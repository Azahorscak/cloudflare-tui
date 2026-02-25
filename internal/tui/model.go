// Package tui implements the Bubble Tea terminal UI.
package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Azahorscak/cloudflare-tui/internal/api"
)

// View represents which screen is currently active.
type View int

const (
	ViewZones   View = iota
	ViewRecords
	ViewEdit
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
	edit        EditModel
	width       int
	height      int
	readOnly    bool
}

// New creates a new root Model with the given API client.
// When readOnly is true, all mutating operations (editing records) are disabled.
func New(client *api.Client, readOnly bool) Model {
	return Model{
		currentView: ViewZones,
		client:      client,
		zones:       NewZonesModel(client),
		readOnly:    readOnly,
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
		m.records = NewRecordsModel(m.client, msg.zone, m.width, m.height, m.readOnly)
		return m, m.records.Init()

	case backToZonesMsg:
		m.currentView = ViewZones
		return m, nil

	case editRecordMsg:
		if m.readOnly {
			return m, nil
		}
		m.currentView = ViewEdit
		m.edit = NewEditModel(m.client, m.records.zone.ID, m.records.zone.Name, msg.record, m.width, m.height)
		return m, m.edit.Init()

	case cancelEditMsg:
		m.currentView = ViewRecords
		return m, nil

	case editDoneMsg:
		m.currentView = ViewRecords
		m.records.statusMsg = fmt.Sprintf("Record %q saved successfully", msg.record.Name)
		return m, tea.Batch(m.records.fetchRecords(), clearStatusAfter(5*time.Second))
	}

	var cmd tea.Cmd
	switch m.currentView {
	case ViewZones:
		m.zones, cmd = m.zones.Update(msg)
	case ViewRecords:
		m.records, cmd = m.records.Update(msg)
	case ViewEdit:
		m.edit, cmd = m.edit.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	switch m.currentView {
	case ViewRecords:
		return m.records.View()
	case ViewEdit:
		return m.edit.View()
	default:
		return m.zones.View()
	}
}
