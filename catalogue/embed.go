// Package catalogue embeds foxtrainer's catalogue data: catalogue.toml and groups/*.toml.
package catalogue

import "embed"

// Files holds the catalogue data compiled into the binary.
//
//go:embed catalogue.toml groups/*.toml
var Files embed.FS
