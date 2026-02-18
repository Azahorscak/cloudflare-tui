package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Azahorscak/cloudflare-tui/internal/api"
)

// zoneItem implements list.DefaultItem for display in a bubbles/list.
type zoneItem struct {
	zone api.Zone
}

func (z zoneItem) Title() string       { return sanitize(z.zone.Name) }
func (z zoneItem) Description() string { return sanitize(z.zone.ID) }
func (z zoneItem) FilterValue() string { return sanitize(z.zone.Name) }

// zonesLoadedMsg carries the result of loading zones from the API.
type zonesLoadedMsg struct {
	zones []api.Zone
	err   error
}

// ZonesModel handles the zone-selection list view.
type ZonesModel struct {
	client  *api.Client
	list    list.Model
	spinner spinner.Model
	loading bool
	err     error
}

// NewZonesModel creates a new zone-selection model.
func NewZonesModel(client *api.Client) ZonesModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 80, 24)
	l.Title = "Cloudflare Zones"

	return ZonesModel{
		client:  client,
		list:    l,
		spinner: sp,
		loading: true,
	}
}

// Init starts the spinner and fires the zone-loading command.
func (m ZonesModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchZones())
}

func (m ZonesModel) fetchZones() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		zones, err := client.ListZones(ctx)
		return zonesLoadedMsg{zones: zones, err: err}
	}
}

// Update handles messages for the zones view.
func (m ZonesModel) Update(msg tea.Msg) (ZonesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case zonesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		items := make([]list.Item, len(msg.zones))
		for i, z := range msg.zones {
			items[i] = zoneItem{zone: z}
		}
		cmd := m.list.SetItems(items)
		return m, cmd

	case tea.KeyMsg:
		if !m.loading && m.err == nil {
			if msg.String() == "enter" && m.list.FilterState() != list.Filtering {
				if selected := m.list.SelectedItem(); selected != nil {
					zi := selected.(zoneItem)
					return m, func() tea.Msg {
						return selectZoneMsg{zone: zi.zone}
					}
				}
			}
		}
	}

	if !m.loading && m.err == nil {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View renders the zones view.
func (m ZonesModel) View() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Loading zones...\n", m.spinner.View())
	}
	if m.err != nil {
		return fmt.Sprintf("\n  Error loading zones: %v\n\n  Press Ctrl+C to quit.\n", m.err)
	}
	return m.list.View()
}
