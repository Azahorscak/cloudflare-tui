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
	kubeconfig := flag.String("kubeconfig", "", "path to kubeconfig file (optional, uses default context if omitted)")
	flag.Parse()

	if *secret == "" {
		fmt.Fprintln(os.Stderr, "error: --secret flag is required (format: namespace/secret-name)")
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	cfg, err := config.Load(ctx, *secret, *kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	client := api.NewClient(cfg)
	model := tui.New(client)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
