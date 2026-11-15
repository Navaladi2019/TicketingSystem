//go:build interptest

package main

import (
	"errors"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp/interptest"
)

// FixtureDirEnv names the directory the interptest stack worker reads its
// definitions and records from (the layout in
// internal/workflow/interp/interptest); the stack mounts it read-only.
// WritesDirEnv names where the fake writer records its calls; unset, it is
// FixtureDirEnv. Only an interptest build reads them; the production image is
// built without the tag and never links the fakes
// (TestProductionBuildLinksNoFakes).
//
// PortsEnv=db swaps the fakes for the production ports (productionActivities:
// the database DefinitionStore and the real records pipeline through
// recordports), keeping the pause-after-commit hook: the P3-44 crash case
// (TICK-713) kills this worker after the real pipeline.Write committed. Any
// other value but empty is refused, so a typo cannot fall back to the fakes.
const (
	FixtureDirEnv = "NAVA_INTERPTEST_FIXTURE_DIR"
	WritesDirEnv  = "NAVA_INTERPTEST_WRITES_DIR"
	PortsEnv      = "NAVA_INTERPTEST_PORTS"
)

// interpreterActivities is the S11a fakes over FixtureDirEnv (P3-22a): the
// fixture store, the file records and the counting writer; with PortsEnv=db,
// the production ports instead. The pause-after-commit hook is on when
// interp.PauseDirEnv is set, with either.
func interpreterActivities(lookup config.LookupFunc, pool, pipePool *pgxpool.Pool, _ bool) (*interp.Activities, error) {
	ports, set := lookup(PortsEnv)
	switch {
	case !set || ports == "":
	case ports == "db":
		return productionActivities(pool, pipePool)
	default:
		return nil, errors.New("config: " + PortsEnv + " is empty (the fakes) or db (the production ports)")
	}
	dir, err := absDir(lookup, FixtureDirEnv, "")
	if err != nil {
		return nil, err
	}
	writes, err := absDir(lookup, WritesDirEnv, dir)
	if err != nil {
		return nil, err
	}
	return &interp.Activities{
		Definitions: interptest.FixtureStore{Root: dir},
		Records:     interptest.FileRecords{Root: dir},
		Writer:      &interptest.FileWriter{Root: writes},
	}, nil
}

// absDir reads name as an absolute path; unset or empty, it is def, and an
// empty def is an error.
func absDir(lookup config.LookupFunc, name, def string) (string, error) {
	dir, ok := lookup(name)
	if !ok || dir == "" {
		dir = def
	}
	if dir == "" || !filepath.IsAbs(dir) {
		return "", errors.New("config: an interptest nava-worker needs " + name + " as an absolute path")
	}
	return dir, nil
}
