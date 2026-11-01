//go:build interptest

package main

import (
	"errors"
	"path/filepath"

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
const (
	FixtureDirEnv = "NAVA_INTERPTEST_FIXTURE_DIR"
	WritesDirEnv  = "NAVA_INTERPTEST_WRITES_DIR"
)

// interpreterActivities is the S11a fakes over FixtureDirEnv (P3-22a): the
// fixture store, the file records and the counting writer. The
// pause-after-commit hook is on when interp.PauseDirEnv is set.
func interpreterActivities(lookup config.LookupFunc) (*interp.Activities, error) {
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
