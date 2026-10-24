// Package db embeds the SQL the way the real db/ package does.
package db

import "embed"

//go:embed *.sql
var files embed.FS

// Files returns the embedded SQL.
func Files() embed.FS { return files }
