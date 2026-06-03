// Package assets embeds every *.txt set decklist in the binary.
package assets

import "embed"

//go:embed *.txt
var FS embed.FS
