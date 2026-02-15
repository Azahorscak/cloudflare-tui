package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Azahorscak/cloudflare-tui/internal/api"
)

func TestZoneItem(t *testing.T) {
	z := zoneItem{zone: api.Zone{ID: "z1", Name: "example.com"}}

	if z.Title() != "example.com" {
		t.Errorf("Title() = %q, want %q", z.Title(), "example.com")
	}
	if z.Description() != "z1" {
		t.Errorf("Description() = %q, want %q", z.Description(), "z1")
	}
	if z.FilterValue() != "example.com" {
		t.Errorf("FilterValue() = %q, want %q", z.FilterValue(), "example.com")
	}
}

func TestZonesModel_ViewLoading(t *testing.T) {
	m := NewZonesModel(nil)
	// Model starts in loading state.
	out := m.View()
	if !strings.Contains(out, "Loading zones") {
		t.Errorf("loading view missing expected text, got: %q", out)
	}
}

func TestZonesModel_ViewError(t *testing.T) {
	m := NewZonesModel(nil)
	m.loading = false
	m.err = fmt.Errorf("connection refused")

	out := m.View()
	if !strings.Contains(out, "connection refused") {
		t.Errorf("error view missing error message, got: %q", out)
	}
	if !strings.Contains(out, "Ctrl+C") {
		t.Errorf("error view missing quit hint, got: %q", out)
	}
}

func TestZonesModel_UpdateZonesLoaded(t *testing.T) {
	m := NewZonesModel(nil)

	zones := []api.Zone{
		{ID: "z1", Name: "example.com"},
		{ID: "z2", Name: "example.org"},
	}

	updated, _ := m.Update(zonesLoadedMsg{zones: zones})
	if updated.loading {
		t.Error("model should not be loading after zonesLoadedMsg")
	}
	if updated.err != nil {
		t.Errorf("unexpected error: %v", updated.err)
	}
}

func TestZonesModel_UpdateZonesLoadedError(t *testing.T) {
	m := NewZonesModel(nil)

	updated, _ := m.Update(zonesLoadedMsg{err: fmt.Errorf("API error")})
	if updated.loading {
		t.Error("model should not be loading after zonesLoadedMsg with error")
	}
	if updated.err == nil {
		t.Fatal("expected error to be set")
	}
	if updated.err.Error() != "API error" {
		t.Errorf("err = %q, want %q", updated.err.Error(), "API error")
	}
}
