package tui

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Azahorscak/cloudflare-tui/internal/api"
)

// recordsLoadedMsg carries the result of loading DNS records from the API.
type recordsLoadedMsg struct {
	records []api.DNSRecord
	err     error
}

// RecordsModel handles the DNS records table view.
type RecordsModel struct {
	client  *api.Client
	zone    api.Zone
	table   table.Model
	spinner spinner.Model
	loading bool
	err     error
	width   int
	height  int
}

// NewRecordsModel creates a new DNS records table model for the given zone.
func NewRecordsModel(client *api.Client, zone api.Zone, width, height int) RecordsModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return RecordsModel{
		client:  client,
		zone:    zone,
		spinner: sp,
		loading: true,
		width:   width,
		height:  height,
	}
}

// Init starts the spinner and fires the record-loading command.
func (m RecordsModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchRecords())
}

func (m RecordsModel) fetchRecords() tea.Cmd {
	client := m.client
	zoneID := m.zone.ID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		records, err := client.ListDNSRecords(ctx, zoneID)
		return recordsLoadedMsg{records: records, err: err}
	}
}

// buildTable creates a table model from loaded DNS records.
func (m RecordsModel) buildTable(records []api.DNSRecord) table.Model {
	columns := []table.Column{
		{Title: "Type", Width: 8},
		{Title: "Name", Width: 64},
		{Title: "Content", Width: 64},
		{Title: "TTL", Width: 8},
		{Title: "Proxied", Width: 8},
	}

	rows := make([]table.Row, len(records))
	for i, r := range records {
		proxied := "No"
		if r.Proxied {
			proxied = "Yes"
		}
		ttl := strconv.Itoa(r.TTL)
		if r.TTL == 1 {
			ttl = "Auto"
		}
		rows[i] = table.Row{r.Type, r.Name, r.Content, ttl, proxied}
	}

	h := m.height
	if h == 0 {
		h = 24
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(h-4),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57"))
	t.SetStyles(s)
	t.Focus()

	return t
}

// Update handles messages for the records view.
func (m RecordsModel) Update(msg tea.Msg) (RecordsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.loading && m.err == nil {
			m.table.SetWidth(msg.Width)
			m.table.SetHeight(msg.Height - 4)
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case recordsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = m.buildTable(msg.records)
		return m, nil

	case tea.KeyMsg:
		key := msg.String()
		if key == "q" || key == "esc" {
			return m, func() tea.Msg { return backToZonesMsg{} }
		}
	}

	if !m.loading && m.err == nil {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View renders the records view.
func (m RecordsModel) View() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Loading DNS records for %s...\n", m.spinner.View(), m.zone.Name)
	}
	if m.err != nil {
		return fmt.Sprintf("\n  Error loading records: %v\n\n  Press q to go back.\n", m.err)
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 0, 1, 2).
		Render(fmt.Sprintf("DNS Records - %s", m.zone.Name))

	help := lipgloss.NewStyle().
		Faint(true).
		Padding(1, 0, 0, 2).
		Render("q/Esc: back | Ctrl+C: quit")

	return header + "\n" + m.table.View() + "\n" + help
}
