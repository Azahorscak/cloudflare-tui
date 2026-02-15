package main

import (
	"fmt"
	"os"

	_ "github.com/Azahorscak/cloudflare-tui/internal/api"
	_ "github.com/Azahorscak/cloudflare-tui/internal/config"
	_ "github.com/Azahorscak/cloudflare-tui/internal/tui"
)

func main() {
	fmt.Fprintln(os.Stderr, "cloudflare-tui: not yet implemented")
	os.Exit(1)
}
