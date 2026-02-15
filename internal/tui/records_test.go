package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Azahorscak/cloudflare-tui/internal/api"
)

func TestRecordsModel_ViewLoading(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24)

	out := m.View()
	if !strings.Contains(out, "Loading DNS records") {
		t.Errorf("loading view missing expected text, got: %q", out)
	}
	if !strings.Contains(out, "example.com") {
		t.Errorf("loading view missing zone name, got: %q", out)
	}
}

func TestRecordsModel_ViewError(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24)
	m.loading = false
	m.err = fmt.Errorf("zone not found")

	out := m.View()
	if !strings.Contains(out, "zone not found") {
		t.Errorf("error view missing error message, got: %q", out)
	}
	if !strings.Contains(out, "q") {
		t.Errorf("error view missing back hint, got: %q", out)
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

	// First row: TTL=300, Proxied=true
	if rows[0][3] != "300" {
		t.Errorf("row 0 TTL = %q, want %q", rows[0][3], "300")
	}
	if rows[0][4] != "Yes" {
		t.Errorf("row 0 Proxied = %q, want %q", rows[0][4], "Yes")
	}

	// Second row: TTL=1 -> "Auto", Proxied=false
	if rows[1][3] != "Auto" {
		t.Errorf("row 1 TTL = %q, want %q", rows[1][3], "Auto")
	}
	if rows[1][4] != "No" {
		t.Errorf("row 1 Proxied = %q, want %q", rows[1][4], "No")
	}
}

func TestRecordsModel_UpdateRecordsLoaded(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24)

	records := []api.DNSRecord{
		{Type: "A", Name: "example.com", Content: "192.0.2.1", TTL: 300, Proxied: true},
	}

	updated, _ := m.Update(recordsLoadedMsg{records: records})
	if updated.loading {
		t.Error("model should not be loading after recordsLoadedMsg")
	}
	if updated.err != nil {
		t.Errorf("unexpected error: %v", updated.err)
	}
}

func TestRecordsModel_UpdateRecordsLoadedError(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24)

	updated, _ := m.Update(recordsLoadedMsg{err: fmt.Errorf("forbidden")})
	if updated.loading {
		t.Error("model should not be loading after error recordsLoadedMsg")
	}
	if updated.err == nil {
		t.Fatal("expected error to be set")
	}
}

func TestRecordsModel_BackNavigation(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24)
	m.loading = false

	tests := []struct {
		name string
		key  string
	}{
		{"q key", "q"},
		{"esc key", "esc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var keyMsg tea.KeyMsg
			if tt.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEscape}
			} else {
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}

			_, cmd := m.Update(keyMsg)
			if cmd == nil {
				t.Fatal("expected a command from back navigation, got nil")
			}
			msg := cmd()
			if _, ok := msg.(backToZonesMsg); !ok {
				t.Errorf("expected backToZonesMsg, got %T", msg)
			}
		})
	}
}

func TestRecordsModel_ViewLoaded(t *testing.T) {
	zone := api.Zone{ID: "z1", Name: "example.com"}
	m := NewRecordsModel(nil, zone, 80, 24)

	records := []api.DNSRecord{
		{Type: "A", Name: "example.com", Content: "192.0.2.1", TTL: 300, Proxied: true},
	}

	m.loading = false
	m.table = m.buildTable(records)

	out := m.View()
	if !strings.Contains(out, "DNS Records") {
		t.Errorf("loaded view missing header, got: %q", out)
	}
	if !strings.Contains(out, "example.com") {
		t.Errorf("loaded view missing zone name, got: %q", out)
	}
	if !strings.Contains(out, "Ctrl+C") {
		t.Errorf("loaded view missing quit hint, got: %q", out)
	}
}
