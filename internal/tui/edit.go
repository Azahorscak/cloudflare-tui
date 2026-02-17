package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
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
	fieldSubmit
)

const editFieldCount = 5

// cancelEditMsg signals that the user cancelled editing.
type cancelEditMsg struct{}

// submitEditMsg carries the validated edit data for saving.
type submitEditMsg struct {
	zoneID   string
	recordID string
	params   api.UpdateDNSRecordParams
}

// saveResultMsg carries the result of the API update call.
type saveResultMsg struct {
	record api.DNSRecord
	err    error
}

// editDoneMsg signals that a record was saved successfully.
type editDoneMsg struct {
	record api.DNSRecord
}

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
	errors  map[editField]string
	saving  bool
	saveErr error
	spinner spinner.Model
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

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

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
		errors:       make(map[editField]string),
		spinner:      sp,
		width:        width,
		height:       height,
	}
}

// Init returns the text input blink command.
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

	case submitEditMsg:
		m.saving = true
		m.saveErr = nil
		return m, tea.Batch(m.spinner.Tick, m.saveCmd(msg))

	case saveResultMsg:
		m.saving = false
		if msg.err != nil {
			m.saveErr = msg.err
			return m, nil
		}
		record := msg.record
		return m, func() tea.Msg { return editDoneMsg{record: record} }

	case spinner.TickMsg:
		if m.saving {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		// Block all key input while saving.
		if m.saving {
			return m, nil
		}

		switch msg.String() {
		case "tab":
			m.focused = (m.focused + 1) % editFieldCount
			m.updateFocus()
			return m, nil
		case "shift+tab":
			m.focused = (m.focused - 1 + editFieldCount) % editFieldCount
			m.updateFocus()
			return m, nil
		case "esc":
			return m, func() tea.Msg { return cancelEditMsg{} }
		case "enter":
			if m.focused == fieldSubmit {
				errs := m.validate()
				if len(errs) > 0 {
					m.errors = errs
					return m, nil
				}
				m.errors = make(map[editField]string)
				return m, m.submitCmd()
			}
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

// validate checks form values and returns a map of field errors.
func (m EditModel) validate() map[editField]string {
	errs := make(map[editField]string)
	if strings.TrimSpace(m.nameInput.Value()) == "" {
		errs[fieldName] = "Name must be non-empty"
	}
	if strings.TrimSpace(m.contentInput.Value()) == "" {
		errs[fieldContent] = "Content must be non-empty"
	}
	ttl := strings.TrimSpace(m.ttlInput.Value())
	if strings.EqualFold(ttl, "auto") {
		// valid — maps to TTL 1
	} else {
		n, err := strconv.Atoi(ttl)
		if err != nil || n <= 0 {
			errs[fieldTTL] = "TTL must be a positive integer or \"Auto\""
		}
	}
	return errs
}

// submitCmd builds a command that emits a submitEditMsg with the current form values.
func (m EditModel) submitCmd() tea.Cmd {
	ttl := 1
	ttlStr := strings.TrimSpace(m.ttlInput.Value())
	if !strings.EqualFold(ttlStr, "auto") {
		ttl, _ = strconv.Atoi(ttlStr) // already validated
	}
	return func() tea.Msg {
		return submitEditMsg{
			zoneID:   m.zoneID,
			recordID: m.record.ID,
			params: api.UpdateDNSRecordParams{
				Name:    strings.TrimSpace(m.nameInput.Value()),
				Type:    m.record.Type,
				Content: strings.TrimSpace(m.contentInput.Value()),
				TTL:     ttl,
				Proxied: m.proxied,
			},
		}
	}
}

// saveCmd fires the API update call and returns a saveResultMsg.
func (m EditModel) saveCmd(msg submitEditMsg) tea.Cmd {
	client := m.client
	zoneID := msg.zoneID
	recordID := msg.recordID
	params := msg.params
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		record, err := client.UpdateDNSRecord(ctx, zoneID, recordID, params)
		return saveResultMsg{record: record, err: err}
	}
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

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Padding(0, 0, 0, 12)

	submitStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 0, 0, 2)

	focusedSubmitStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
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

	apiErrorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Padding(0, 0, 0, 2)

	// Submit button / saving indicator
	var submitText string
	if m.saving {
		savingStyle := lipgloss.NewStyle().Padding(0, 0, 0, 2)
		submitText = savingStyle.Render(m.spinner.View() + " Saving…")
	} else if m.focused == fieldSubmit {
		submitText = focusedSubmitStyle.Render("[ Save ]")
	} else {
		submitText = submitStyle.Render("[ Save ]")
	}

	help := helpStyle.Render("Tab/Shift+Tab: navigate | Enter: save | Esc: cancel")

	// Build the view with inline validation errors
	sections := []string{subtitle, "", typeRow}

	sections = append(sections, nameRow)
	if err, ok := m.errors[fieldName]; ok {
		sections = append(sections, errorStyle.Render("! "+err))
	}

	sections = append(sections, contentRow)
	if err, ok := m.errors[fieldContent]; ok {
		sections = append(sections, errorStyle.Render("! "+err))
	}

	sections = append(sections, ttlRow)
	if err, ok := m.errors[fieldTTL]; ok {
		sections = append(sections, errorStyle.Render("! "+err))
	}

	sections = append(sections, proxiedRow, "", submitText)

	// Show API error prominently above help text
	if m.saveErr != nil {
		sections = append(sections, apiErrorStyle.Render("Error: "+m.saveErr.Error()))
	}

	sections = append(sections, help)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
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

// Errors returns the current validation errors.
func (m EditModel) Errors() map[editField]string {
	return m.errors
}

// Saving returns whether a save is in progress.
func (m EditModel) Saving() bool {
	return m.saving
}

// SaveErr returns the last API save error, if any.
func (m EditModel) SaveErr() error {
	return m.saveErr
}
