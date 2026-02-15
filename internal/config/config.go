// Package config handles loading Cloudflare credentials from Kubernetes secrets.
package config

import (
	_ "k8s.io/client-go/kubernetes"
)

// Config holds the Cloudflare API credentials.
type Config struct {
	APIToken string
}
