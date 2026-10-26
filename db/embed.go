// Package db embeds the platform database's migrations, grant manifest and
// pg_hba.conf, so the nava-db binary carries exactly the files reviewed in
// the repository.
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

//go:embed pg_hba.conf
var hbaConf []byte

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

// HBA returns db/pg_hba.conf, for internal/db/hba's checks.
func HBA() []byte { return hbaConf }
