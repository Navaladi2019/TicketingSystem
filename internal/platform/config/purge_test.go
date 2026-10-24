package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

func TestLoadPurgeDefaults(t *testing.T) {
	cfg, err := config.LoadPurge(env(map[string]string{"NAVA_DATABASE_URL": "postgres://nava_purge:pw@pg/nava"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Interval != time.Hour || cfg.OutboxDispatched != 7*24*time.Hour ||
		cfg.OutboxDead != 30*24*time.Hour || cfg.WorkflowRef != 30*24*time.Hour || cfg.AdminAddr != ":9090" {
		t.Errorf("defaults = %+v", cfg)
	}
}

func TestLoadPurgeRejects(t *testing.T) {
	const url = "postgres://nava_purge:s3cret@pg/nava"
	for name, tc := range map[string]struct {
		env  map[string]string
		want string
	}{
		"no database":        {map[string]string{}, "NAVA_DATABASE_URL: required"},
		"bad database":       {map[string]string{"NAVA_DATABASE_URL": "mysql://x"}, "NAVA_DATABASE_URL"},
		"bad interval":       {map[string]string{"NAVA_DATABASE_URL": url, "NAVA_PURGE_INTERVAL": "soon"}, "NAVA_PURGE_INTERVAL"},
		"zero interval":      {map[string]string{"NAVA_DATABASE_URL": url, "NAVA_PURGE_INTERVAL": "0s"}, "NAVA_PURGE_INTERVAL"},
		"dispatched < 7d":    {map[string]string{"NAVA_DATABASE_URL": url, "NAVA_PURGE_OUTBOX_DISPATCHED_RETENTION": "24h"}, "NAVA_PURGE_OUTBOX_DISPATCHED_RETENTION: must be at least"},
		"dead < 30d":         {map[string]string{"NAVA_DATABASE_URL": url, "NAVA_PURGE_OUTBOX_DEAD_RETENTION": "48h"}, "NAVA_PURGE_OUTBOX_DEAD_RETENTION: must be at least"},
		"workflow_ref < 30d": {map[string]string{"NAVA_DATABASE_URL": url, "NAVA_PURGE_WORKFLOW_REF_RETENTION": "1h"}, "NAVA_PURGE_WORKFLOW_REF_RETENTION: must be at least"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := config.LoadPurge(env(tc.env))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.want)
			}
			if strings.Contains(err.Error(), "s3cret") {
				t.Errorf("error echoes the database password: %v", err)
			}
		})
	}
}

func TestLoadPurgeLongerWindows(t *testing.T) {
	cfg, err := config.LoadPurge(env(map[string]string{
		"NAVA_DATABASE_URL": "postgres://nava_purge:pw@pg/nava", "NAVA_PURGE_OUTBOX_DEAD_RETENTION": "1000h",
	}))
	if err != nil || cfg.OutboxDead != 1000*time.Hour {
		t.Fatalf("cfg.OutboxDead = %v, err = %v", cfg.OutboxDead, err)
	}
}
