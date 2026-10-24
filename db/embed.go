// Package db embeds the platform database's migrations and grant manifest, so
// the nava-db binary carries exactly the files reviewed in the repository.
package db

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var migrations embed.FS

//go:embed grants.yaml
var grants []byte

//go:embed global-tables.yaml
var globalTables []byte

// Migrations returns the goose migration files, rooted at the migrations directory.
func Migrations() fs.FS {
	sub, err := fs.Sub(migrations, "migrations")
	if err != nil {
		panic(err) // the directory is embedded at build time
	}
	return sub
}

// Grants returns db/grants.yaml.
func Grants() []byte { return grants }

// GlobalTables returns db/global-tables.yaml.
func GlobalTables() []byte { return globalTables }
