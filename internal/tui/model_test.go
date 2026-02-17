package tui

import (
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

	// Tab wraps to name
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.Focused() != fieldName {
		t.Errorf("expected focus to wrap to fieldName, got %d", m.Focused())
	}
}

func TestEditModel_ShiftTabReversesFocus(t *testing.T) {
	rec := newTestRecord()
	m := NewEditModel(nil, "zone-1", "example.com", rec, 80, 24)

	// Shift+Tab from name wraps to proxied
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
