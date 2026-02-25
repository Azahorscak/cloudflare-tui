package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Azahorscak/cloudflare-tui/internal/api"
	"github.com/Azahorscak/cloudflare-tui/internal/config"
	"github.com/Azahorscak/cloudflare-tui/internal/tui"
)

func main() {
	secret := flag.String("secret", "", "Kubernetes secret in namespace/secret-name format (required)")
	secretKey := flag.String("secret-key", "cloudflare_api_token", "key within the Kubernetes secret that holds the Cloudflare API token")
	kubeconfig := flag.String("kubeconfig", "", "path to kubeconfig file (optional, uses default context if omitted)")
	readOnly := flag.Bool("readonly", false, "launch in read-only mode (no changes can be made)")
	flag.Parse()

	if *secret == "" {
		fmt.Fprintln(os.Stderr, "error: --secret flag is required (format: namespace/secret-name)")
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	cfg, err := config.Load(ctx, *secret, *kubeconfig, *secretKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	client := api.NewClient(cfg)
	model := tui.New(client, *readOnly)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
