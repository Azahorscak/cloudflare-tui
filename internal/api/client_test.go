package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Azahorscak/cloudflare-tui/internal/config"
	"github.com/cloudflare/cloudflare-go/v4/option"
)

// newTestClient creates a Client pointed at the given test server with retries disabled.
func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	cfg := &config.Config{APIToken: "test-token"}
	return newClient(cfg,
		option.WithBaseURL(baseURL),
		option.WithMaxRetries(0),
	)
}

func TestListZones(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/zones", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("unexpected Authorization header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")

		// The auto-pager fetches page 2 to check for more results.
		// Return empty result for any page beyond 1.
		if r.URL.Query().Get("page") != "" && r.URL.Query().Get("page") != "1" {
			fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":[],"result_info":{"page":2,"per_page":20,"total_count":2,"total_pages":1}}`)
			return
		}

		fmt.Fprint(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": [
				{"id": "zone-1", "name": "example.com"},
				{"id": "zone-2", "name": "example.org"}
			],
			"result_info": {"page": 1, "per_page": 20, "total_count": 2, "total_pages": 1}
		}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	zones, err := client.ListZones(context.Background())
	if err != nil {
		t.Fatalf("ListZones returned error: %v", err)
	}

	if len(zones) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(zones))
	}

	want := []Zone{
		{ID: "zone-1", Name: "example.com"},
		{ID: "zone-2", Name: "example.org"},
	}
	for i, z := range zones {
		if z != want[i] {
			t.Errorf("zone[%d] = %+v, want %+v", i, z, want[i])
		}
	}
}

func TestListZonesEmpty(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/zones", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":[],"result_info":{"page":1,"per_page":20,"total_count":0,"total_pages":1}}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	zones, err := client.ListZones(context.Background())
	if err != nil {
		t.Fatalf("ListZones returned error: %v", err)
	}
	if len(zones) != 0 {
		t.Fatalf("expected 0 zones, got %d", len(zones))
	}
}

func TestListZonesError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/zones", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"success":false,"errors":[{"code":9109,"message":"Invalid access token"}],"messages":[],"result":null}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	_, err := client.ListZones(context.Background())
	if err == nil {
		t.Fatal("expected error from ListZones, got nil")
	}
}

func TestListDNSRecords(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/zones/zone-1/dns_records", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("unexpected Authorization header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Query().Get("page") != "" && r.URL.Query().Get("page") != "1" {
			fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":[],"result_info":{"page":2,"per_page":20,"total_count":3,"total_pages":1}}`)
			return
		}

		fmt.Fprint(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": [
				{
					"id": "rec-1",
					"type": "A",
					"name": "example.com",
					"content": "192.0.2.1",
					"ttl": 300,
					"proxied": true
				},
				{
					"id": "rec-2",
					"type": "CNAME",
					"name": "www.example.com",
					"content": "example.com",
					"ttl": 1,
					"proxied": false
				},
				{
					"id": "rec-3",
					"type": "MX",
					"name": "example.com",
					"content": "mail.example.com",
					"ttl": 3600,
					"proxied": false
				}
			],
			"result_info": {"page": 1, "per_page": 20, "total_count": 3, "total_pages": 1}
		}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	records, err := client.ListDNSRecords(context.Background(), "zone-1")
	if err != nil {
		t.Fatalf("ListDNSRecords returned error: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}

	want := []DNSRecord{
		{Type: "A", Name: "example.com", Content: "192.0.2.1", TTL: 300, Proxied: true},
		{Type: "CNAME", Name: "www.example.com", Content: "example.com", TTL: 1, Proxied: false},
		{Type: "MX", Name: "example.com", Content: "mail.example.com", TTL: 3600, Proxied: false},
	}
	for i, r := range records {
		if r != want[i] {
			t.Errorf("record[%d] = %+v, want %+v", i, r, want[i])
		}
	}
}

func TestListDNSRecordsEmpty(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/zones/zone-1/dns_records", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":[],"result_info":{"page":1,"per_page":20,"total_count":0,"total_pages":1}}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	records, err := client.ListDNSRecords(context.Background(), "zone-1")
	if err != nil {
		t.Fatalf("ListDNSRecords returned error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}
}

func TestListDNSRecordsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/zones/zone-1/dns_records", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"success":false,"errors":[{"code":7003,"message":"Could not route to /zones/zone-1/dns_records, perhaps your object identifier is invalid?"}],"messages":[],"result":null}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	_, err := client.ListDNSRecords(context.Background(), "zone-1")
	if err == nil {
		t.Fatal("expected error from ListDNSRecords, got nil")
	}
}

func TestNewClient(t *testing.T) {
	cfg := &config.Config{APIToken: "my-token"}
	client := NewClient(cfg)
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.cf == nil {
		t.Fatal("NewClient returned Client with nil cloudflare client")
	}
}
