package tui

import "regexp"

// reANSI matches ANSI/VT escape sequences that could be embedded in
// untrusted data (e.g. DNS record names or content fetched from the API).
//
// Pattern covers:
//   - CSI sequences:  ESC [ <params> <final>  (e.g. colour codes, cursor movement)
//   - Other Fe seqs:  ESC <byte in 0x40-0x5F range>  (e.g. ESC M, ESC 7)
var reANSI = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|[@-Z\\-_])`)

// sanitize strips ANSI escape sequences from s before it is passed to a
// lipgloss/bubbletea renderer.  Apply this to all string values that
// originate from external sources (Cloudflare API responses).
func sanitize(s string) string {
	return reANSI.ReplaceAllString(s, "")
}
