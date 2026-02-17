package tui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Azahorscak/cloudflare-tui/internal/api"
)

// editField identifies which form field is focused.
type editField int

const (
	fieldName editField = iota
	fieldContent
	fieldTTL
	fieldProxied
)

const editFieldCount = 4

// EditModel represents a form for editing a single DNS record.
type EditModel struct {
	client   *api.Client
	zoneID   string
	zoneName string
	record   api.DNSRecord

	nameInput    textinput.Model
	contentInput textinput.Model
	ttlInput     textinput.Model
	proxied      bool

	focused editField
	width   int
	height  int
}

// NewEditModel creates a new EditModel pre-filled with the given record's values.
func NewEditModel(client *api.Client, zoneID, zoneName string, record api.DNSRecord, width, height int) EditModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "Record name"
	nameInput.SetValue(record.Name)
	nameInput.CharLimit = 253
	nameInput.Width = 60
	nameInput.Focus()

	contentInput := textinput.New()
	contentInput.Placeholder = "Record content"
	contentInput.SetValue(record.Content)
	contentInput.CharLimit = 2048
	contentInput.Width = 60

	ttlInput := textinput.New()
	ttlInput.Placeholder = "TTL (1 = Auto)"
	if record.TTL == 1 {
		ttlInput.SetValue("Auto")
	} else {
		ttlInput.SetValue(strconv.Itoa(record.TTL))
	}
	ttlInput.CharLimit = 10
	ttlInput.Width = 20

	return EditModel{
		client:       client,
		zoneID:       zoneID,
		zoneName:     zoneName,
		record:       record,
		nameInput:    nameInput,
		contentInput: contentInput,
		ttlInput:     ttlInput,
		proxied:      record.Proxied,
		focused:      fieldName,
		width:        width,
		height:       height,
	}
}

// Init returns nil; no initial commands needed.
func (m EditModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the edit view.
func (m EditModel) Update(msg tea.Msg) (EditModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.focused = (m.focused + 1) % editFieldCount
			m.updateFocus()
			return m, nil
		case "shift+tab":
			m.focused = (m.focused - 1 + editFieldCount) % editFieldCount
			m.updateFocus()
			return m, nil
		}
	}

	// Delegate to the focused text input.
	var cmd tea.Cmd
	switch m.focused {
	case fieldName:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case fieldContent:
		m.contentInput, cmd = m.contentInput.Update(msg)
	case fieldTTL:
		m.ttlInput, cmd = m.ttlInput.Update(msg)
	case fieldProxied:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == " " || keyMsg.String() == "enter" {
				m.proxied = !m.proxied
			}
		}
	}
	return m, cmd
}

// updateFocus sets the focused state on each text input.
func (m *EditModel) updateFocus() {
	m.nameInput.Blur()
	m.contentInput.Blur()
	m.ttlInput.Blur()

	switch m.focused {
	case fieldName:
		m.nameInput.Focus()
	case fieldContent:
		m.contentInput.Focus()
	case fieldTTL:
		m.ttlInput.Focus()
	}
}

// View renders the edit form.
func (m EditModel) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Padding(0, 1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(1, 0, 1, 2)

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Width(10).
		Padding(0, 1, 0, 2)

	readOnlyStyle := lipgloss.NewStyle().
		Faint(true).
		Padding(0, 1)

	fieldStyle := lipgloss.NewStyle().
		Padding(0, 0, 0, 0)

	focusedLabelStyle := labelStyle.
		Foreground(lipgloss.Color("205"))

	proxiedStyle := lipgloss.NewStyle().
		Padding(0, 1)

	helpStyle := lipgloss.NewStyle().
		Faint(true).
		Padding(1, 0, 0, 2)

	// Header
	title := titleStyle.Render(fmt.Sprintf(" Edit %s Record ", m.record.Type))
	subtitle := headerStyle.Render(fmt.Sprintf("%s  %s", title, m.zoneName))

	// Type (read-only)
	typeRow := lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Type"),
		readOnlyStyle.Render(m.record.Type+" (read-only)"),
	)

	// Name
	nameLbl := labelStyle
	if m.focused == fieldName {
		nameLbl = focusedLabelStyle
	}
	nameRow := lipgloss.JoinHorizontal(lipgloss.Top,
		nameLbl.Render("Name"),
		fieldStyle.Render(m.nameInput.View()),
	)

	// Content
	contentLbl := labelStyle
	if m.focused == fieldContent {
		contentLbl = focusedLabelStyle
	}
	contentRow := lipgloss.JoinHorizontal(lipgloss.Top,
		contentLbl.Render("Content"),
		fieldStyle.Render(m.contentInput.View()),
	)

	// TTL
	ttlLbl := labelStyle
	if m.focused == fieldTTL {
		ttlLbl = focusedLabelStyle
	}
	ttlRow := lipgloss.JoinHorizontal(lipgloss.Top,
		ttlLbl.Render("TTL"),
		fieldStyle.Render(m.ttlInput.View()),
	)

	// Proxied toggle
	proxiedLbl := labelStyle
	if m.focused == fieldProxied {
		proxiedLbl = focusedLabelStyle
	}
	proxiedValue := "[ ] No"
	if m.proxied {
		proxiedValue = "[x] Yes"
	}
	if m.focused == fieldProxied {
		proxiedValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Render(proxiedValue)
	}
	proxiedRow := lipgloss.JoinHorizontal(lipgloss.Top,
		proxiedLbl.Render("Proxied"),
		proxiedStyle.Render(proxiedValue),
	)

	help := helpStyle.Render("Tab/Shift+Tab: navigate | Space: toggle proxied | Esc: cancel")

	return lipgloss.JoinVertical(lipgloss.Left,
		subtitle,
		"",
		typeRow,
		nameRow,
		contentRow,
		ttlRow,
		proxiedRow,
		help,
	)
}

// Proxied returns the current proxied toggle value.
func (m EditModel) Proxied() bool {
	return m.proxied
}

// Focused returns the currently focused field.
func (m EditModel) Focused() editField {
	return m.focused
}

// NameValue returns the current value of the name input.
func (m EditModel) NameValue() string {
	return m.nameInput.Value()
}

// ContentValue returns the current value of the content input.
func (m EditModel) ContentValue() string {
	return m.contentInput.Value()
}

// TTLValue returns the current value of the TTL input.
func (m EditModel) TTLValue() string {
	return m.ttlInput.Value()
}
