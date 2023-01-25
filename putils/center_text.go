package putils

import "github.com/solo-io/pterm/internal"

// CenterText returns a centered string with each line centered in respect to the longest line.
func CenterText(text string) string {
	return internal.CenterText(text, 0)
}
