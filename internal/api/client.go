// Package api wraps the Cloudflare SDK for use by the TUI layer.
package api

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/zones"

	"github.com/Azahorscak/cloudflare-tui/internal/config"
)

// Client is a thin wrapper around the Cloudflare API.
type Client struct {
	cf *cloudflare.Client
}

// Zone represents a Cloudflare zone.
type Zone struct {
	ID   string
	Name string
}

// DNSRecord represents a single DNS record.
type DNSRecord struct {
	ID      string
	Type    string
	Name    string
	Content string
	TTL     int
	Proxied bool
}

// UpdateDNSRecordParams contains the editable fields for updating a DNS record.
type UpdateDNSRecordParams struct {
	Name    string
	Type    string
	Content string
	TTL     int
	Proxied bool
}

// NewClient creates an authenticated Cloudflare API client from the given config.
func NewClient(cfg *config.Config) *Client {
	return newClient(cfg)
}

// NewClientWithBaseURL creates a Client that targets a custom base URL.
// Intended for integration tests using a mock HTTP server.
func NewClientWithBaseURL(cfg *config.Config, baseURL string) *Client {
	return newClient(cfg, option.WithBaseURL(baseURL), option.WithMaxRetries(0))
}

// newClient creates a Client with optional extra request options (used for testing).
func newClient(cfg *config.Config, extra ...option.RequestOption) *Client {
	opts := append([]option.RequestOption{option.WithAPIToken(cfg.APIToken)}, extra...)
	cf := cloudflare.NewClient(opts...)
	return &Client{cf: cf}
}

// ListZones returns all zones visible to the configured API token.
func (c *Client) ListZones(ctx context.Context) ([]Zone, error) {
	var result []Zone

	pager := c.cf.Zones.ListAutoPaging(ctx, zones.ZoneListParams{})
	for pager.Next() {
		z := pager.Current()
		result = append(result, Zone{
			ID:   z.ID,
			Name: z.Name,
		})
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("listing zones: %w", err)
	}

	return result, nil
}

// ListDNSRecords returns all DNS records for the given zone.
func (c *Client) ListDNSRecords(ctx context.Context, zoneID string) ([]DNSRecord, error) {
	var result []DNSRecord

	pager := c.cf.DNS.Records.ListAutoPaging(ctx, dns.RecordListParams{
		ZoneID: cloudflare.F(zoneID),
	})
	for pager.Next() {
		r := pager.Current()
		result = append(result, DNSRecord{
			ID:      r.ID,
			Type:    string(r.Type),
			Name:    r.Name,
			Content: r.Content,
			TTL:     int(r.TTL),
			Proxied: r.Proxied,
		})
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("listing DNS records for zone %s: %w", zoneID, err)
	}

	return result, nil
}

// GetDNSRecord fetches a single DNS record by ID.
func (c *Client) GetDNSRecord(ctx context.Context, zoneID, recordID string) (DNSRecord, error) {
	resp, err := c.cf.DNS.Records.Get(ctx, recordID, dns.RecordGetParams{
		ZoneID: cloudflare.F(zoneID),
	})
	if err != nil {
		return DNSRecord{}, fmt.Errorf("getting DNS record %s in zone %s: %w", recordID, zoneID, err)
	}

	return DNSRecord{
		ID:      resp.ID,
		Type:    string(resp.Type),
		Name:    resp.Name,
		Content: resp.Content,
		TTL:     int(resp.TTL),
		Proxied: resp.Proxied,
	}, nil
}

// UpdateDNSRecord updates a DNS record and returns the updated record.
func (c *Client) UpdateDNSRecord(ctx context.Context, zoneID, recordID string, params UpdateDNSRecordParams) (DNSRecord, error) {
	resp, err := c.cf.DNS.Records.Update(ctx, recordID, dns.RecordUpdateParams{
		ZoneID: cloudflare.F(zoneID),
		Body: dns.RecordUpdateParamsBody{
			Name:    cloudflare.F(params.Name),
			Type:    cloudflare.F(dns.RecordUpdateParamsBodyType(params.Type)),
			Content: cloudflare.F(params.Content),
			TTL:     cloudflare.F(dns.TTL(params.TTL)),
			Proxied: cloudflare.F(params.Proxied),
		},
	})
	if err != nil {
		return DNSRecord{}, fmt.Errorf("updating DNS record %s in zone %s: %w", recordID, zoneID, err)
	}

	return DNSRecord{
		ID:      resp.ID,
		Type:    string(resp.Type),
		Name:    resp.Name,
		Content: resp.Content,
		TTL:     int(resp.TTL),
		Proxied: resp.Proxied,
	}, nil
}
