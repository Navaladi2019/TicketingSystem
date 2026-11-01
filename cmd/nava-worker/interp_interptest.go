//go:build interptest

package main

import (
	"errors"
	"path/filepath"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp/interptest"
)

// FixtureDirEnv names the mounted directory the interptest stack worker reads
// its definitions and records from and records its writes in (the layout in
// internal/workflow/interp/interptest). Only an interptest build reads it;
// the production image is built without the tag and never links the fakes
// (TestProductionBuildLinksNoFakes).
const FixtureDirEnv = "NAVA_INTERPTEST_FIXTURE_DIR"

// interpreterActivities is the S11a fakes over FixtureDirEnv (P3-22a): the
// fixture store, the file records and the counting writer. The
// pause-after-commit hook is on when interp.PauseDirEnv is set.
func interpreterActivities(lookup config.LookupFunc) (*interp.Activities, error) {
	dir, ok := lookup(FixtureDirEnv)
	if !ok || dir == "" || !filepath.IsAbs(dir) {
		return nil, errors.New("config: an interptest nava-worker needs " + FixtureDirEnv + " as an absolute path")
	}
	return &interp.Activities{
		Definitions: interptest.FixtureStore{Root: dir},
		Records:     interptest.FileRecords{Root: dir},
		Writer:      &interptest.FileWriter{Root: dir},
	}, nil
}
