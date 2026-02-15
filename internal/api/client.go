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
	Type    string
	Name    string
	Content string
	TTL     int
	Proxied bool
}

// NewClient creates an authenticated Cloudflare API client from the given config.
func NewClient(cfg *config.Config) *Client {
	return newClient(cfg)
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
