// Package api wraps the Cloudflare SDK for use by the TUI layer.
package api

import (
	_ "github.com/cloudflare/cloudflare-go/v4"
)

// Client is a thin wrapper around the Cloudflare API.
type Client struct{}

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
