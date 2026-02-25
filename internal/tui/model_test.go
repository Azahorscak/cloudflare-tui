package tui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Azahorscak/cloudflare-tui/internal/api"
	"github.com/Azahorscak/cloudflare-tui/internal/config"
)

func TestNew_StartsAtZonesView(t *testing.T) {
	m := New(nil, false)
	if m.currentView != ViewZones {
		t.Errorf("expected initial view to be ViewZones, got %d", m.currentView)
	}
}

func TestModel_ViewTransitionToRecords(t *testing.T) {
	m := New(nil, false)

	zone := api.Zone{ID: "zone-1", Name: "example.com"}
	updated, _ := m.Update(selectZoneMsg{zone: zone})
	model := updated.(Model)

	if model.currentView != ViewRecords {
		t.Errorf("expected ViewRecords after selectZoneMsg, got %d", model.currentView)
	}
}

func TestModel_ViewTransitionBackToZones(t *testing.T) {
	m := New(nil, false)

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
	m := New(nil, false)
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
	m := New(nil, false)
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
	m := NewRecordsModel(nil, zone, 80, 24, false)
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
	m := NewRecordsModel(nil, zone, 80, 24, false)
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
	m := NewRecordsModel(nil, zone, 80, 24, false)
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
	m := NewRecordsModel(nil, zone, 80, 24, false)
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
	m := NewRecordsModel(nil, zone, 80, 24, false)

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
	if !strings.Contains(view, "Space: toggle proxied") {
		t.Error("expected help text to contain 'Space: toggle proxied'")
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

// --- Step 13 tests: edit view integration with root model and records table ---

func TestModel_EditRecordMsgTransitionsToEdit(t *testing.T) {
	m := New(nil, false)

	// Transition to records first.
	zone := api.Zone{ID: "zone-1", Name: "example.com"}
	updated, _ := m.Update(selectZoneMsg{zone: zone})
	model := updated.(Model)

	// Send editRecordMsg.
	rec := newTestRecord()
	updated, cmd := model.Update(editRecordMsg{record: rec})
	model = updated.(Model)

	if model.currentView != ViewEdit {
		t.Errorf("expected ViewEdit after editRecordMsg, got %d", model.currentView)
	}
	if cmd == nil {
		t.Error("expected Init command from EditModel")
	}
}

func TestModel_CancelEditMsgTransitionsBackToRecords(t *testing.T) {
	m := New(nil, false)

	// Transition to records, then edit.
	zone := api.Zone{ID: "zone-1", Name: "example.com"}
	updated, _ := m.Update(selectZoneMsg{zone: zone})
	model := updated.(Model)

	rec := newTestRecord()
	updated, _ = model.Update(editRecordMsg{record: rec})
	model = updated.(Model)

	if model.currentView != ViewEdit {
		t.Fatalf("expected ViewEdit, got %d", model.currentView)
	}

	// Cancel the edit.
	updated, cmd := model.Update(cancelEditMsg{})
	model = updated.(Model)

	if model.currentView != ViewRecords {
		t.Errorf("expected ViewRecords after cancelEditMsg, got %d", model.currentView)
	}
	if cmd != nil {
		t.Error("expected no command after cancel")
	}
}

func TestModel_EditDoneMsgTransitionsBackToRecordsWithStatus(t *testing.T) {
	m := New(nil, false)

	// Transition to records first.
	zone := api.Zone{ID: "zone-1", Name: "example.com"}
	updated, _ := m.Update(selectZoneMsg{zone: zone})
	model := updated.(Model)

	rec := newTestRecord()
	updated, _ = model.Update(editRecordMsg{record: rec})
	model = updated.(Model)

	// Simulate successful save.
	savedRecord := api.DNSRecord{
		ID:      "rec-1",
		Type:    "A",
		Name:    "updated.example.com",
		Content: "192.0.2.99",
		TTL:     300,
		Proxied: true,
	}
	updated, cmd := model.Update(editDoneMsg{record: savedRecord})
	model = updated.(Model)

	if model.currentView != ViewRecords {
		t.Errorf("expected ViewRecords after editDoneMsg, got %d", model.currentView)
	}
	if model.records.statusMsg == "" {
		t.Error("expected status message to be set after successful save")
	}
	if !strings.Contains(model.records.statusMsg, "updated.example.com") {
		t.Errorf("expected status message to contain record name, got %q", model.records.statusMsg)
	}
	if !strings.Contains(model.records.statusMsg, "saved successfully") {
		t.Errorf("expected status message to contain 'saved successfully', got %q", model.records.statusMsg)
	}
	if cmd == nil {
		t.Error("expected batch command (fetchRecords + clearStatus) after editDoneMsg")
	}
}

func TestRecordsModel_EnterEmitsEditRecordMsg(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24, false)
	m.loading = false

	// Populate records and build table.
	records := []api.DNSRecord{
		{ID: "rec-1", Type: "A", Name: "example.com", Content: "192.0.2.1", TTL: 300, Proxied: true},
		{ID: "rec-2", Type: "CNAME", Name: "www.example.com", Content: "example.com", TTL: 1, Proxied: false},
	}
	m.records = records
	m.table = m.buildTable(records)

	// Press Enter on the first row (cursor defaults to 0).
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command from enter on record, got nil")
	}
	msg := cmd()
	edit, ok := msg.(editRecordMsg)
	if !ok {
		t.Fatalf("expected editRecordMsg, got %T", msg)
	}
	if edit.record.ID != "rec-1" {
		t.Errorf("expected record ID 'rec-1', got %q", edit.record.ID)
	}
	if edit.record.Name != "example.com" {
		t.Errorf("expected record name 'example.com', got %q", edit.record.Name)
	}
}

func TestRecordsModel_EnterNoOpWhileLoading(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24, false)
	// m.loading is true by default

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected no command from enter while loading")
	}
}

func TestRecordsModel_EnterNoOpWithNoRecords(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24, false)
	m.loading = false
	m.records = []api.DNSRecord{}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected no command from enter with no records")
	}
}

func TestRecordsModel_StatusClearMsgClearsStatus(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24, false)
	m.loading = false
	m.statusMsg = "Record saved successfully"

	m, _ = m.Update(statusClearMsg{})
	if m.statusMsg != "" {
		t.Errorf("expected statusMsg to be cleared, got %q", m.statusMsg)
	}
}

func TestRecordsModel_ViewRendersStatusMsg(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24, false)
	m.loading = false
	m.records = []api.DNSRecord{
		{ID: "rec-1", Type: "A", Name: "example.com", Content: "192.0.2.1", TTL: 300, Proxied: true},
	}
	m.table = m.buildTable(m.records)
	m.statusMsg = "Record \"example.com\" saved successfully"

	view := m.View()
	if !strings.Contains(view, "saved successfully") {
		t.Error("expected view to contain status message 'saved successfully'")
	}
}

func TestRecordsModel_ViewNoStatusMsgWhenEmpty(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24, false)
	m.loading = false
	m.records = []api.DNSRecord{
		{ID: "rec-1", Type: "A", Name: "example.com", Content: "192.0.2.1", TTL: 300, Proxied: true},
	}
	m.table = m.buildTable(m.records)

	view := m.View()
	if strings.Contains(view, "saved successfully") {
		t.Error("expected no status message when statusMsg is empty")
	}
}

func TestRecordsModel_HelpBarShowsEditHint(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24, false)
	m.loading = false
	m.records = []api.DNSRecord{
		{ID: "rec-1", Type: "A", Name: "example.com", Content: "192.0.2.1", TTL: 300, Proxied: true},
	}
	m.table = m.buildTable(m.records)

	view := m.View()
	if !strings.Contains(view, "Enter: edit record") {
		t.Error("expected help bar to contain 'Enter: edit record'")
	}
	if !strings.Contains(view, "↑/↓: navigate") {
		t.Error("expected help bar to contain '↑/↓: navigate'")
	}
}

func TestRecordsModel_ReadOnlyEnterNoOp(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24, true) // readOnly=true
	m.loading = false

	records := []api.DNSRecord{
		{ID: "rec-1", Type: "A", Name: "example.com", Content: "192.0.2.1", TTL: 300, Proxied: true},
	}
	m.records = records
	m.table = m.buildTable(records)

	// Press Enter — should NOT emit editRecordMsg in read-only mode.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected no command from enter in read-only mode, but got one")
	}
}

func TestRecordsModel_ReadOnlyHelpBar(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24, true) // readOnly=true
	m.loading = false
	m.records = []api.DNSRecord{
		{ID: "rec-1", Type: "A", Name: "example.com", Content: "192.0.2.1", TTL: 300, Proxied: true},
	}
	m.table = m.buildTable(m.records)

	view := m.View()
	if strings.Contains(view, "Enter: edit record") {
		t.Error("read-only help bar should NOT contain 'Enter: edit record'")
	}
	if !strings.Contains(view, "READ-ONLY") {
		t.Error("read-only help bar should contain 'READ-ONLY' indicator")
	}
}

func TestModel_ReadOnlyBlocksEditRecordMsg(t *testing.T) {
	m := New(nil, true) // readOnly=true

	// Transition to records.
	zone := api.Zone{ID: "zone-1", Name: "example.com"}
	updated, _ := m.Update(selectZoneMsg{zone: zone})
	model := updated.(Model)

	if model.currentView != ViewRecords {
		t.Fatalf("expected ViewRecords, got %d", model.currentView)
	}

	// Try sending an editRecordMsg — should be ignored.
	record := api.DNSRecord{ID: "rec-1", Type: "A", Name: "example.com", Content: "192.0.2.1", TTL: 300, Proxied: true}
	updated, cmd := model.Update(editRecordMsg{record: record})
	model = updated.(Model)

	if model.currentView != ViewRecords {
		t.Errorf("expected to stay on ViewRecords in read-only mode, got %d", model.currentView)
	}
	if cmd != nil {
		t.Error("expected no command from editRecordMsg in read-only mode")
	}
}

func TestModel_FullNavigationLoop(t *testing.T) {
	// Test the full loop: zones -> records -> edit -> records (cancel)
	m := New(nil, false)

	// Start at zones.
	if m.currentView != ViewZones {
		t.Fatalf("expected ViewZones, got %d", m.currentView)
	}

	// Transition to records.
	zone := api.Zone{ID: "zone-1", Name: "example.com"}
	updated, _ := m.Update(selectZoneMsg{zone: zone})
	model := updated.(Model)
	if model.currentView != ViewRecords {
		t.Fatalf("expected ViewRecords, got %d", model.currentView)
	}

	// Transition to edit.
	rec := newTestRecord()
	updated, _ = model.Update(editRecordMsg{record: rec})
	model = updated.(Model)
	if model.currentView != ViewEdit {
		t.Fatalf("expected ViewEdit, got %d", model.currentView)
	}

	// Cancel back to records.
	updated, _ = model.Update(cancelEditMsg{})
	model = updated.(Model)
	if model.currentView != ViewRecords {
		t.Fatalf("expected ViewRecords after cancel, got %d", model.currentView)
	}

	// Back to zones.
	updated, _ = model.Update(backToZonesMsg{})
	model = updated.(Model)
	if model.currentView != ViewZones {
		t.Fatalf("expected ViewZones after back, got %d", model.currentView)
	}
}

func TestModel_FullEditSaveLoop(t *testing.T) {
	// Test: zones -> records -> edit -> save -> records (with status)
	m := New(nil, false)

	// Transition to records.
	zone := api.Zone{ID: "zone-1", Name: "example.com"}
	updated, _ := m.Update(selectZoneMsg{zone: zone})
	model := updated.(Model)

	// Transition to edit.
	rec := newTestRecord()
	updated, _ = model.Update(editRecordMsg{record: rec})
	model = updated.(Model)
	if model.currentView != ViewEdit {
		t.Fatalf("expected ViewEdit, got %d", model.currentView)
	}

	// Simulate successful save.
	savedRecord := api.DNSRecord{
		ID:      "rec-1",
		Type:    "A",
		Name:    "example.com",
		Content: "192.0.2.99",
		TTL:     300,
		Proxied: true,
	}
	updated, cmd := model.Update(editDoneMsg{record: savedRecord})
	model = updated.(Model)

	if model.currentView != ViewRecords {
		t.Fatalf("expected ViewRecords after editDoneMsg, got %d", model.currentView)
	}
	if model.records.statusMsg == "" {
		t.Error("expected status message after save")
	}
	if cmd == nil {
		t.Error("expected batch command (fetch + clearStatus) after save")
	}
}

// --- Step 14: edge case tests ---

func TestEditModel_LongContentValue(t *testing.T) {
	rec := newTestRecord()
	rec.Content = strings.Repeat("a", 2000)
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	if m.ContentValue() != rec.Content {
		t.Errorf("expected content length %d, got %d", len(rec.Content), len(m.ContentValue()))
	}

	// Should render without error.
	view := m.View()
	if !strings.Contains(view, "Content") {
		t.Error("expected view to render Content label for long content")
	}

	// Should validate and submit successfully.
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected valid submission with long content")
	}
}

func TestEditModel_TTLBoundaryMinNonAuto(t *testing.T) {
	rec := newTestRecord()
	rec.TTL = 2
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	if m.TTLValue() != "2" {
		t.Errorf("expected TTL '2', got %q", m.TTLValue())
	}

	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected valid submission with TTL=2")
	}
	msg := cmd()
	sub := msg.(submitEditMsg)
	if sub.params.TTL != 2 {
		t.Errorf("expected TTL 2, got %d", sub.params.TTL)
	}
}

func TestEditModel_TTLBoundaryLargeValue(t *testing.T) {
	rec := newTestRecord()
	rec.TTL = 86400
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	if m.TTLValue() != "86400" {
		t.Errorf("expected TTL '86400', got %q", m.TTLValue())
	}

	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected valid submission with TTL=86400")
	}
	msg := cmd()
	sub := msg.(submitEditMsg)
	if sub.params.TTL != 86400 {
		t.Errorf("expected TTL 86400, got %d", sub.params.TTL)
	}
}

func TestEditModel_TTLAutoLowerCase(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)
	m.ttlInput.SetValue("auto")

	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected valid submission with 'auto' TTL")
	}
	msg := cmd()
	sub := msg.(submitEditMsg)
	if sub.params.TTL != 1 {
		t.Errorf("expected TTL 1 for 'auto', got %d", sub.params.TTL)
	}
}

func TestEditModel_TTLAutoMixedCase(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)
	m.ttlInput.SetValue("AuTo")

	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected valid submission with 'AuTo' TTL")
	}
	msg := cmd()
	sub := msg.(submitEditMsg)
	if sub.params.TTL != 1 {
		t.Errorf("expected TTL 1 for 'AuTo', got %d", sub.params.TTL)
	}
}

func TestEditModel_ProxiedOnMXRecord(t *testing.T) {
	rec := api.DNSRecord{
		ID:      "rec-mx",
		Type:    "MX",
		Name:    "example.com",
		Content: "mail.example.com",
		TTL:     3600,
		Proxied: false,
	}
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	// Navigate to proxied field.
	for i := 0; i < 3; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	if m.Focused() != fieldProxied {
		t.Fatalf("expected focus on fieldProxied, got %d", m.Focused())
	}

	// Toggle proxied on.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !m.Proxied() {
		t.Error("expected proxied to toggle to true even for MX record")
	}

	// Submit should proceed (API would reject, but form allows it).
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected submission to proceed for MX record with proxied=true")
	}
	msg := cmd()
	sub := msg.(submitEditMsg)
	if sub.params.Type != "MX" {
		t.Errorf("expected type MX, got %q", sub.params.Type)
	}
	if !sub.params.Proxied {
		t.Error("expected proxied=true in submit params")
	}
}

func TestEditModel_ProxiedOnTXTRecord(t *testing.T) {
	rec := api.DNSRecord{
		ID:      "rec-txt",
		Type:    "TXT",
		Name:    "example.com",
		Content: "v=spf1 include:_spf.google.com ~all",
		TTL:     3600,
		Proxied: false,
	}
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	if m.Proxied() {
		t.Error("expected proxied to start as false for TXT record")
	}

	view := m.View()
	if !strings.Contains(view, "TXT") {
		t.Error("expected view to contain 'TXT' record type")
	}
}

func TestEditModel_ProxiedOnSRVRecord(t *testing.T) {
	rec := api.DNSRecord{
		ID:      "rec-srv",
		Type:    "SRV",
		Name:    "_sip._tcp.example.com",
		Content: "10 60 5060 sip.example.com",
		TTL:     3600,
		Proxied: false,
	}
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	// Navigate to proxied and toggle.
	for i := 0; i < 3; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !m.Proxied() {
		t.Error("expected proxied to toggle to true for SRV record")
	}

	// Submit includes the toggled value.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected submission to proceed for SRV record")
	}
	msg := cmd()
	sub := msg.(submitEditMsg)
	if sub.params.Type != "SRV" {
		t.Errorf("expected type SRV, got %q", sub.params.Type)
	}
}

func TestEditModel_TTLValidationBoundary(t *testing.T) {
	// TTL = 1 means Auto; values between 2 and the minimum Cloudflare
	// allows should still pass local validation.
	tests := []struct {
		input string
		valid bool
		ttl   int
	}{
		{"Auto", true, 1},
		{"auto", true, 1},
		{"1", true, 1},
		{"2", true, 2},
		{"60", true, 60},
		{"0", false, 0},
		{"-1", false, 0},
		{"abc", false, 0},
		{"", false, 0},
	}

	for _, tt := range tests {
		t.Run("ttl="+tt.input, func(t *testing.T) {
			rec := newTestRecord()
			m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)
			m.ttlInput.SetValue(tt.input)

			for i := 0; i < 4; i++ {
				m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
			}
			m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

			if tt.valid {
				if cmd == nil {
					t.Errorf("expected valid submission for TTL %q", tt.input)
					return
				}
				msg := cmd()
				sub := msg.(submitEditMsg)
				if sub.params.TTL != tt.ttl {
					t.Errorf("expected TTL %d, got %d", tt.ttl, sub.params.TTL)
				}
			} else {
				if cmd != nil {
					t.Errorf("expected validation failure for TTL %q", tt.input)
				}
				errs := m.Errors()
				if _, ok := errs[fieldTTL]; !ok {
					t.Errorf("expected TTL validation error for %q", tt.input)
				}
			}
		})
	}
}

// --- Step 14: integration test with mocked API ---

func TestEditFlow_IntegrationWithMockedAPI(t *testing.T) {
	updateCalled := false
	mux := http.NewServeMux()

	// Handle record update.
	mux.HandleFunc("/zones/zone-1/dns_records/rec-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			updateCalled = true
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"success": true,
				"errors": [],
				"messages": [],
				"result": {
					"id": "rec-1",
					"type": "A",
					"name": "example.com",
					"content": "203.0.113.50",
					"ttl": 300,
					"proxied": true
				}
			}`)
			return
		}
		http.NotFound(w, r)
	})

	// Handle record list (for the refresh after save).
	mux.HandleFunc("/zones/zone-1/dns_records", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") != "" && r.URL.Query().Get("page") != "1" {
			fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":[],"result_info":{"page":2,"per_page":20,"total_count":1,"total_pages":1}}`)
			return
		}
		fmt.Fprint(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": [{
				"id": "rec-1",
				"type": "A",
				"name": "example.com",
				"content": "203.0.113.50",
				"ttl": 300,
				"proxied": true
			}],
			"result_info": {"page": 1, "per_page": 20, "total_count": 1, "total_pages": 1}
		}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfg := &config.Config{APIToken: "test-token"}
	client := api.NewClientWithBaseURL(cfg, srv.URL)
	m := New(client, false)

	// Step 1: start at zones.
	if m.currentView != ViewZones {
		t.Fatalf("expected ViewZones, got %d", m.currentView)
	}

	// Step 2: select zone → records.
	zone := api.Zone{ID: "zone-1", Name: "example.com"}
	updated, _ := m.Update(selectZoneMsg{zone: zone})
	model := updated.(Model)
	if model.currentView != ViewRecords {
		t.Fatalf("expected ViewRecords, got %d", model.currentView)
	}

	// Step 3: select record → edit.
	rec := api.DNSRecord{
		ID: "rec-1", Type: "A", Name: "example.com",
		Content: "192.0.2.1", TTL: 300, Proxied: true,
	}
	updated, _ = model.Update(editRecordMsg{record: rec})
	model = updated.(Model)
	if model.currentView != ViewEdit {
		t.Fatalf("expected ViewEdit, got %d", model.currentView)
	}

	// Step 4: modify content.
	model.edit.contentInput.SetValue("203.0.113.50")

	// Step 5: navigate to submit and press enter.
	for i := 0; i < 4; i++ {
		model.edit, _ = model.edit.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	var cmd tea.Cmd
	model.edit, cmd = model.edit.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected submit command")
	}

	// Step 6: execute submit command → submitEditMsg.
	submitResult := cmd()
	sub, ok := submitResult.(submitEditMsg)
	if !ok {
		t.Fatalf("expected submitEditMsg, got %T", submitResult)
	}
	if sub.params.Content != "203.0.113.50" {
		t.Errorf("expected updated content '203.0.113.50', got %q", sub.params.Content)
	}

	// Step 7: deliver submitEditMsg → triggers API call.
	model.edit, cmd = model.edit.Update(sub)
	if !model.edit.Saving() {
		t.Error("expected saving state")
	}
	if cmd == nil {
		t.Fatal("expected batch command from submitEditMsg")
	}

	// Step 8: execute the batch to find the saveResultMsg.
	batchResult := cmd()
	batch, ok := batchResult.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg, got %T", batchResult)
	}

	var saveResult saveResultMsg
	foundSaveResult := false
	for _, c := range batch {
		msg := c()
		if sr, ok := msg.(saveResultMsg); ok {
			saveResult = sr
			foundSaveResult = true
		}
	}
	if !foundSaveResult {
		t.Fatal("expected saveResultMsg from batch commands")
	}
	if saveResult.err != nil {
		t.Fatalf("expected no save error, got %v", saveResult.err)
	}
	if saveResult.record.Content != "203.0.113.50" {
		t.Errorf("expected updated content '203.0.113.50', got %q", saveResult.record.Content)
	}

	// Step 9: deliver save result → editDoneMsg.
	model.edit, cmd = model.edit.Update(saveResult)
	if model.edit.Saving() {
		t.Error("expected saving to be false after successful save")
	}
	if cmd == nil {
		t.Fatal("expected editDoneMsg command")
	}

	doneResult := cmd()
	done, ok := doneResult.(editDoneMsg)
	if !ok {
		t.Fatalf("expected editDoneMsg, got %T", doneResult)
	}
	if done.record.Content != "203.0.113.50" {
		t.Errorf("expected updated content in editDoneMsg, got %q", done.record.Content)
	}

	// Step 10: deliver editDoneMsg → back to records with status.
	updated, cmd = model.Update(done)
	model = updated.(Model)
	if model.currentView != ViewRecords {
		t.Errorf("expected ViewRecords after editDoneMsg, got %d", model.currentView)
	}
	if !strings.Contains(model.records.statusMsg, "saved successfully") {
		t.Errorf("expected status message to contain 'saved successfully', got %q", model.records.statusMsg)
	}
	if cmd == nil {
		t.Error("expected batch command (fetchRecords + clearStatus)")
	}

	if !updateCalled {
		t.Error("expected API update endpoint to be called")
	}
}

func TestEditFlow_IntegrationAPIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/zones/zone-1/dns_records/rec-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"success":false,"errors":[{"code":9109,"message":"Invalid access token"}],"messages":[],"result":null}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfg := &config.Config{APIToken: "bad-token"}
	client := api.NewClientWithBaseURL(cfg, srv.URL)
	m := New(client, false)

	// Navigate to edit.
	zone := api.Zone{ID: "zone-1", Name: "example.com"}
	updated, _ := m.Update(selectZoneMsg{zone: zone})
	model := updated.(Model)

	rec := api.DNSRecord{
		ID: "rec-1", Type: "A", Name: "example.com",
		Content: "192.0.2.1", TTL: 300, Proxied: true,
	}
	updated, _ = model.Update(editRecordMsg{record: rec})
	model = updated.(Model)

	// Submit the form.
	for i := 0; i < 4; i++ {
		model.edit, _ = model.edit.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	var cmd tea.Cmd
	model.edit, cmd = model.edit.Update(tea.KeyMsg{Type: tea.KeyEnter})
	submitResult := cmd()
	sub := submitResult.(submitEditMsg)

	// Deliver submitEditMsg → triggers API call.
	model.edit, cmd = model.edit.Update(sub)

	// Execute batch to find saveResultMsg.
	batchResult := cmd()
	batch := batchResult.(tea.BatchMsg)

	var saveResult saveResultMsg
	for _, c := range batch {
		msg := c()
		if sr, ok := msg.(saveResultMsg); ok {
			saveResult = sr
		}
	}

	if saveResult.err == nil {
		t.Fatal("expected API error in save result")
	}

	// Deliver error result → stays on edit form.
	model.edit, cmd = model.edit.Update(saveResult)
	if model.edit.Saving() {
		t.Error("expected saving to be false after error")
	}
	if model.edit.SaveErr() == nil {
		t.Error("expected saveErr to be set")
	}
	if cmd != nil {
		t.Error("expected no command after save error (should stay on form)")
	}

	// View should show the error.
	view := model.edit.View()
	if !strings.Contains(view, "Error:") {
		t.Error("expected view to show API error")
	}
}
