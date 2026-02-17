package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Azahorscak/cloudflare-tui/internal/api"
)

func TestNew_StartsAtZonesView(t *testing.T) {
	m := New(nil)
	if m.currentView != ViewZones {
		t.Errorf("expected initial view to be ViewZones, got %d", m.currentView)
	}
}

func TestModel_ViewTransitionToRecords(t *testing.T) {
	m := New(nil)

	zone := api.Zone{ID: "zone-1", Name: "example.com"}
	updated, _ := m.Update(selectZoneMsg{zone: zone})
	model := updated.(Model)

	if model.currentView != ViewRecords {
		t.Errorf("expected ViewRecords after selectZoneMsg, got %d", model.currentView)
	}
}

func TestModel_ViewTransitionBackToZones(t *testing.T) {
	m := New(nil)

	// Transition to records first.
	updated, _ := m.Update(selectZoneMsg{zone: api.Zone{ID: "z1", Name: "example.com"}})
	model := updated.(Model)

	// Now go back.
	updated, _ = model.Update(backToZonesMsg{})
	model = updated.(Model)

	if model.currentView != ViewZones {
		t.Errorf("expected ViewZones after backToZonesMsg, got %d", model.currentView)
	}
}

func TestModel_CtrlCReturnsQuit(t *testing.T) {
	m := New(nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected quit command from ctrl+c, got nil")
	}
	// Execute the command and check it produces a QuitMsg.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestModel_WindowSizeMsg(t *testing.T) {
	m := New(nil)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(Model)

	if model.width != 120 || model.height != 40 {
		t.Errorf("expected size 120x40, got %dx%d", model.width, model.height)
	}
}

func TestZonesModel_LoadingView(t *testing.T) {
	m := NewZonesModel(nil)
	view := m.View()
	if !strings.Contains(view, "Loading zones") {
		t.Errorf("expected loading view to contain 'Loading zones', got: %s", view)
	}
}

func TestZonesModel_ErrorView(t *testing.T) {
	m := NewZonesModel(nil)
	m.loading = false
	m.err = &testError{msg: "connection refused"}

	view := m.View()
	if !strings.Contains(view, "Error loading zones") {
		t.Errorf("expected error view to contain 'Error loading zones', got: %s", view)
	}
	if !strings.Contains(view, "connection refused") {
		t.Errorf("expected error view to contain the error message, got: %s", view)
	}
}

func TestRecordsModel_LoadingView(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24)
	view := m.View()
	if !strings.Contains(view, "Loading DNS records") {
		t.Errorf("expected loading view to contain 'Loading DNS records', got: %s", view)
	}
	if !strings.Contains(view, "example.com") {
		t.Errorf("expected loading view to contain zone name, got: %s", view)
	}
}

func TestRecordsModel_ErrorView(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24)
	m.loading = false
	m.err = &testError{msg: "invalid token"}

	view := m.View()
	if !strings.Contains(view, "Error loading records") {
		t.Errorf("expected error view to contain 'Error loading records', got: %s", view)
	}
	if !strings.Contains(view, "invalid token") {
		t.Errorf("expected error view to contain the error message, got: %s", view)
	}
}

func TestRecordsModel_BackKeyReturnsBackMsg(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24)
	m.loading = false

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected command from q key, got nil")
	}
	msg := cmd()
	if _, ok := msg.(backToZonesMsg); !ok {
		t.Errorf("expected backToZonesMsg, got %T", msg)
	}
}

func TestRecordsModel_EscKeyReturnsBackMsg(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24)
	m.loading = false

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected command from esc key, got nil")
	}
	msg := cmd()
	if _, ok := msg.(backToZonesMsg); !ok {
		t.Errorf("expected backToZonesMsg, got %T", msg)
	}
}

func TestRecordsModel_BuildTable(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24)

	records := []api.DNSRecord{
		{Type: "A", Name: "example.com", Content: "192.0.2.1", TTL: 300, Proxied: true},
		{Type: "CNAME", Name: "www.example.com", Content: "example.com", TTL: 1, Proxied: false},
	}

	tbl := m.buildTable(records)
	rows := tbl.Rows()
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	// Check first row values.
	if rows[0][0] != "A" || rows[0][1] != "example.com" || rows[0][2] != "192.0.2.1" {
		t.Errorf("unexpected first row: %v", rows[0])
	}
	if rows[0][3] != "300" {
		t.Errorf("expected TTL '300', got %q", rows[0][3])
	}
	if rows[0][4] != "Yes" {
		t.Errorf("expected Proxied 'Yes', got %q", rows[0][4])
	}

	// Check TTL=1 renders as "Auto".
	if rows[1][3] != "Auto" {
		t.Errorf("expected TTL 'Auto' for TTL=1, got %q", rows[1][3])
	}
	if rows[1][4] != "No" {
		t.Errorf("expected Proxied 'No', got %q", rows[1][4])
	}
}

// testError is a simple error for testing view rendering.
type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }

// --- EditModel tests ---

func newTestRecord() api.DNSRecord {
	return api.DNSRecord{
		ID:      "rec-1",
		Type:    "A",
		Name:    "example.com",
		Content: "192.0.2.1",
		TTL:     300,
		Proxied: true,
	}
}

func TestEditModel_InitialFieldPopulation(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	if m.NameValue() != "example.com" {
		t.Errorf("expected name 'example.com', got %q", m.NameValue())
	}
	if m.ContentValue() != "192.0.2.1" {
		t.Errorf("expected content '192.0.2.1', got %q", m.ContentValue())
	}
	if m.TTLValue() != "300" {
		t.Errorf("expected TTL '300', got %q", m.TTLValue())
	}
	if m.Proxied() != true {
		t.Error("expected proxied to be true")
	}
	if m.Focused() != fieldName {
		t.Errorf("expected initial focus on fieldName, got %d", m.Focused())
	}
}

func TestEditModel_TTLAutoDisplay(t *testing.T) {
	rec := newTestRecord()
	rec.TTL = 1
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	if m.TTLValue() != "Auto" {
		t.Errorf("expected TTL 'Auto' for TTL=1, got %q", m.TTLValue())
	}
}

func TestEditModel_TabCyclesFocus(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	if m.Focused() != fieldName {
		t.Fatalf("expected initial focus on fieldName, got %d", m.Focused())
	}

	// Tab to content
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.Focused() != fieldContent {
		t.Errorf("expected focus on fieldContent after tab, got %d", m.Focused())
	}

	// Tab to TTL
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.Focused() != fieldTTL {
		t.Errorf("expected focus on fieldTTL after tab, got %d", m.Focused())
	}

	// Tab to proxied
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.Focused() != fieldProxied {
		t.Errorf("expected focus on fieldProxied after tab, got %d", m.Focused())
	}

	// Tab to submit
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.Focused() != fieldSubmit {
		t.Errorf("expected focus on fieldSubmit after tab, got %d", m.Focused())
	}

	// Tab wraps to name
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.Focused() != fieldName {
		t.Errorf("expected focus to wrap to fieldName, got %d", m.Focused())
	}
}

func TestEditModel_ShiftTabReversesFocus(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	// Shift+Tab from name wraps to submit
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.Focused() != fieldSubmit {
		t.Errorf("expected focus on fieldSubmit after shift+tab from name, got %d", m.Focused())
	}

	// Shift+Tab to proxied
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.Focused() != fieldProxied {
		t.Errorf("expected focus on fieldProxied after shift+tab, got %d", m.Focused())
	}

	// Shift+Tab to TTL
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.Focused() != fieldTTL {
		t.Errorf("expected focus on fieldTTL after shift+tab, got %d", m.Focused())
	}
}

func TestEditModel_ProxiedToggle(t *testing.T) {
	rec := newTestRecord()
	rec.Proxied = false
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	if m.Proxied() {
		t.Fatal("expected proxied to start as false")
	}

	// Navigate to proxied field
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // -> content
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // -> TTL
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // -> proxied

	// Toggle with space
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !m.Proxied() {
		t.Error("expected proxied to be true after space toggle")
	}

	// Toggle back with space
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if m.Proxied() {
		t.Error("expected proxied to be false after second space toggle")
	}
}

func TestEditModel_ViewRendersHeader(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)
	view := m.View()

	if !strings.Contains(view, "Edit") {
		t.Error("expected view to contain 'Edit'")
	}
	if !strings.Contains(view, "A") {
		t.Error("expected view to contain record type 'A'")
	}
	if !strings.Contains(view, "example.com") {
		t.Error("expected view to contain zone name 'example.com'")
	}
}

func TestEditModel_ViewRendersTypeReadOnly(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)
	view := m.View()

	if !strings.Contains(view, "read-only") {
		t.Error("expected view to indicate type is read-only")
	}
}

func TestEditModel_ViewRendersAllLabels(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)
	view := m.View()

	for _, label := range []string{"Type", "Name", "Content", "TTL", "Proxied"} {
		if !strings.Contains(view, label) {
			t.Errorf("expected view to contain label %q", label)
		}
	}
}

func TestEditModel_WindowSizeMsg(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if m.width != 120 || m.height != 40 {
		t.Errorf("expected size 120x40, got %dx%d", m.width, m.height)
	}
}

func TestEditModel_EscEmitsCancelMsg(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected command from esc key, got nil")
	}
	msg := cmd()
	if _, ok := msg.(cancelEditMsg); !ok {
		t.Errorf("expected cancelEditMsg, got %T", msg)
	}
}

func TestEditModel_EscWorksFromAnyField(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	// Tab to content, then Esc
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.Focused() != fieldContent {
		t.Fatalf("expected fieldContent, got %d", m.Focused())
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected command from esc key on content field, got nil")
	}
	msg := cmd()
	if _, ok := msg.(cancelEditMsg); !ok {
		t.Errorf("expected cancelEditMsg, got %T", msg)
	}
}

func TestEditModel_SubmitWithValidData(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	// Navigate to submit button
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	if m.Focused() != fieldSubmit {
		t.Fatalf("expected fieldSubmit, got %d", m.Focused())
	}

	// Press Enter to submit
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command from enter on submit, got nil")
	}
	msg := cmd()
	sub, ok := msg.(submitEditMsg)
	if !ok {
		t.Fatalf("expected submitEditMsg, got %T", msg)
	}
	if sub.zoneID != "zone-1" {
		t.Errorf("expected zoneID 'zone-1', got %q", sub.zoneID)
	}
	if sub.recordID != "rec-1" {
		t.Errorf("expected recordID 'rec-1', got %q", sub.recordID)
	}
	if sub.params.Name != "example.com" {
		t.Errorf("expected name 'example.com', got %q", sub.params.Name)
	}
	if sub.params.Content != "192.0.2.1" {
		t.Errorf("expected content '192.0.2.1', got %q", sub.params.Content)
	}
	if sub.params.TTL != 300 {
		t.Errorf("expected TTL 300, got %d", sub.params.TTL)
	}
	if sub.params.Proxied != true {
		t.Error("expected proxied true")
	}
	if sub.params.Type != "A" {
		t.Errorf("expected type 'A', got %q", sub.params.Type)
	}
}

func TestEditModel_SubmitAutoTTL(t *testing.T) {
	rec := newTestRecord()
	rec.TTL = 1
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	// Navigate to submit
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command from enter on submit, got nil")
	}
	msg := cmd()
	sub, ok := msg.(submitEditMsg)
	if !ok {
		t.Fatalf("expected submitEditMsg, got %T", msg)
	}
	if sub.params.TTL != 1 {
		t.Errorf("expected TTL 1 for Auto, got %d", sub.params.TTL)
	}
}

func TestEditModel_ValidationEmptyName(t *testing.T) {
	rec := newTestRecord()
	rec.Name = ""
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	// Navigate to submit
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}

	// Press Enter to submit
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected no command when validation fails")
	}
	errs := m.Errors()
	if _, ok := errs[fieldName]; !ok {
		t.Error("expected validation error for empty name")
	}
}

func TestEditModel_ValidationEmptyContent(t *testing.T) {
	rec := newTestRecord()
	rec.Content = ""
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	// Navigate to submit
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected no command when validation fails")
	}
	errs := m.Errors()
	if _, ok := errs[fieldContent]; !ok {
		t.Error("expected validation error for empty content")
	}
}

func TestEditModel_ValidationInvalidTTL(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	// Navigate to TTL field and clear it, type "abc"
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // content
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // TTL
	// Clear the TTL field by selecting all and typing over
	m.ttlInput.SetValue("abc")

	// Navigate to submit
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // proxied
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // submit

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected no command when validation fails")
	}
	errs := m.Errors()
	if _, ok := errs[fieldTTL]; !ok {
		t.Error("expected validation error for invalid TTL")
	}
}

func TestEditModel_ValidationNegativeTTL(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	m.ttlInput.SetValue("-5")

	// Navigate to submit
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected no command when validation fails")
	}
	errs := m.Errors()
	if _, ok := errs[fieldTTL]; !ok {
		t.Error("expected validation error for negative TTL")
	}
}

func TestEditModel_ValidationZeroTTL(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	m.ttlInput.SetValue("0")

	// Navigate to submit
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected no command when validation fails")
	}
	errs := m.Errors()
	if _, ok := errs[fieldTTL]; !ok {
		t.Error("expected validation error for zero TTL")
	}
}

func TestEditModel_ValidationMultipleErrors(t *testing.T) {
	rec := newTestRecord()
	rec.Name = ""
	rec.Content = ""
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)
	m.ttlInput.SetValue("bad")

	// Navigate to submit
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	errs := m.Errors()
	if len(errs) != 3 {
		t.Errorf("expected 3 validation errors, got %d", len(errs))
	}
}

func TestEditModel_ValidationErrorsClearedOnSuccessfulSubmit(t *testing.T) {
	rec := newTestRecord()
	rec.Name = ""
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	// Navigate to submit and trigger validation error
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if len(m.Errors()) == 0 {
		t.Fatal("expected validation errors")
	}

	// Fix the name field
	m.nameInput.SetValue("fixed.example.com")

	// Submit again (still on submit button)
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command after fixing validation")
	}
	if len(m.Errors()) != 0 {
		t.Errorf("expected errors to be cleared after successful submit, got %d", len(m.Errors()))
	}
}

func TestEditModel_EnterOnNonSubmitFieldDoesNotSubmit(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	// Press Enter on the name field (should not submit)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// The command should be nil or a text input command, not a submit
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(submitEditMsg); ok {
			t.Error("enter on name field should not emit submitEditMsg")
		}
	}
}

func TestEditModel_ViewRendersSubmitButton(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)
	view := m.View()

	if !strings.Contains(view, "Save") {
		t.Error("expected view to contain 'Save' button")
	}
}

func TestEditModel_ViewRendersValidationErrors(t *testing.T) {
	rec := newTestRecord()
	rec.Name = ""
	rec.Content = ""
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)
	m.ttlInput.SetValue("bad")

	// Navigate to submit and trigger validation
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	view := m.View()
	if !strings.Contains(view, "Name must be non-empty") {
		t.Error("expected view to show name validation error")
	}
	if !strings.Contains(view, "Content must be non-empty") {
		t.Error("expected view to show content validation error")
	}
	if !strings.Contains(view, "TTL must be a positive integer") {
		t.Error("expected view to show TTL validation error")
	}
}

func TestEditModel_ViewRendersUpdatedHelpText(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)
	view := m.View()

	if !strings.Contains(view, "Enter: save") {
		t.Error("expected help text to contain 'Enter: save'")
	}
	if !strings.Contains(view, "Esc: cancel") {
		t.Error("expected help text to contain 'Esc: cancel'")
	}
}

// --- Step 12 tests: save action and success/error feedback ---

func TestEditModel_SubmitEditMsgSetsSavingState(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	// Directly send submitEditMsg to simulate the runtime delivering it
	m, cmd := m.Update(submitEditMsg{
		zoneID:   "zone-1",
		recordID: "rec-1",
		params: api.UpdateDNSRecordParams{
			Name:    "example.com",
			Type:    "A",
			Content: "192.0.2.1",
			TTL:     300,
			Proxied: true,
		},
	})

	if !m.Saving() {
		t.Error("expected saving to be true after submitEditMsg")
	}
	if cmd == nil {
		t.Error("expected command (spinner tick + save cmd) after submitEditMsg")
	}
}

func TestEditModel_SubmitEditMsgClearsPreviousSaveErr(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	// Simulate a previous save error
	m, _ = m.Update(saveResultMsg{err: fmt.Errorf("previous error")})
	if m.SaveErr() == nil {
		t.Fatal("expected saveErr to be set")
	}

	// Now submit again
	m, _ = m.Update(submitEditMsg{
		zoneID:   "zone-1",
		recordID: "rec-1",
		params:   api.UpdateDNSRecordParams{Name: "example.com", Type: "A", Content: "192.0.2.1", TTL: 300, Proxied: true},
	})

	if m.SaveErr() != nil {
		t.Error("expected saveErr to be cleared on new submit")
	}
}

func TestEditModel_SaveResultSuccess(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)
	m.saving = true

	updatedRecord := api.DNSRecord{
		ID:      "rec-1",
		Type:    "A",
		Name:    "example.com",
		Content: "192.0.2.99",
		TTL:     300,
		Proxied: true,
	}

	m, cmd := m.Update(saveResultMsg{record: updatedRecord})

	if m.Saving() {
		t.Error("expected saving to be false after successful save")
	}
	if m.SaveErr() != nil {
		t.Errorf("expected no save error, got %v", m.SaveErr())
	}
	if cmd == nil {
		t.Fatal("expected command (editDoneMsg) after successful save")
	}

	// Execute the command and check it produces editDoneMsg
	msg := cmd()
	done, ok := msg.(editDoneMsg)
	if !ok {
		t.Fatalf("expected editDoneMsg, got %T", msg)
	}
	if done.record.Content != "192.0.2.99" {
		t.Errorf("expected updated content '192.0.2.99', got %q", done.record.Content)
	}
}

func TestEditModel_SaveResultError(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)
	m.saving = true

	m, cmd := m.Update(saveResultMsg{err: fmt.Errorf("API rate limit exceeded")})

	if m.Saving() {
		t.Error("expected saving to be false after save error")
	}
	if m.SaveErr() == nil {
		t.Fatal("expected save error to be set")
	}
	if m.SaveErr().Error() != "API rate limit exceeded" {
		t.Errorf("expected error 'API rate limit exceeded', got %q", m.SaveErr().Error())
	}
	if cmd != nil {
		t.Error("expected no command after save error")
	}
}

func TestEditModel_SavingBlocksKeyInput(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)
	m.saving = true

	// Tab should be blocked
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.Focused() != fieldName {
		t.Errorf("expected focus to remain on fieldName while saving, got %d", m.Focused())
	}
	if cmd != nil {
		t.Error("expected no command from blocked key input")
	}

	// Esc should be blocked
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(cancelEditMsg); ok {
			t.Error("esc should not produce cancelEditMsg while saving")
		}
	}

	// Enter should be blocked
	m.focused = fieldSubmit
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected no command from enter while saving")
	}
	_ = m2
}

func TestEditModel_ViewRendersSavingIndicator(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)
	m.saving = true

	view := m.View()
	if !strings.Contains(view, "Saving") {
		t.Error("expected view to contain 'Saving' indicator while saving")
	}
}

func TestEditModel_ViewRendersAPIError(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)
	m.saveErr = fmt.Errorf("authentication failed")

	view := m.View()
	if !strings.Contains(view, "Error:") {
		t.Error("expected view to contain 'Error:' prefix for API error")
	}
	if !strings.Contains(view, "authentication failed") {
		t.Error("expected view to contain the API error message")
	}
}

func TestEditModel_ViewNoAPIErrorWhenNil(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	view := m.View()
	if strings.Contains(view, "Error:") {
		t.Error("expected no API error in view when saveErr is nil")
	}
}

func TestEditModel_SaveErrorRetryFlow(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	// Simulate save error
	m.saving = true
	m, _ = m.Update(saveResultMsg{err: fmt.Errorf("server error")})
	if m.SaveErr() == nil {
		t.Fatal("expected save error to be set")
	}

	// User should be able to navigate to submit and retry
	// (not saving, so keys work)
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	if m.Focused() != fieldSubmit {
		t.Fatalf("expected focus on fieldSubmit, got %d", m.Focused())
	}

	// Press Enter to retry submit
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command from retry submit")
	}
	msg := cmd()
	if _, ok := msg.(submitEditMsg); !ok {
		t.Errorf("expected submitEditMsg on retry, got %T", msg)
	}
}

func TestEditModel_SavingAccessorDefaultFalse(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	if m.Saving() {
		t.Error("expected saving to be false initially")
	}
}

func TestEditModel_SaveErrAccessorDefaultNil(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	if m.SaveErr() != nil {
		t.Error("expected saveErr to be nil initially")
	}
}

func TestEditModel_ViewSavingHidesSubmitButton(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	// When not saving, should show Save button
	view := m.View()
	if !strings.Contains(view, "Save") {
		t.Error("expected Save button in normal view")
	}

	// When saving, should show Saving indicator instead
	m.saving = true
	view = m.View()
	if !strings.Contains(view, "Saving") {
		t.Error("expected Saving indicator in saving view")
	}
}
