package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

func groupsyncEnv(extra map[string]string) config.LookupFunc {
	m := map[string]string{
		"NAVA_DATABASE_URL":            "postgres://nava_groupsync:pw@pg/nava",
		"NAVA_KEYCLOAK_URL":            "http://keycloak:8080",
		"NAVA_GROUPSYNC_CLIENT_ID":     "nava-groupsync",
		"NAVA_GROUPSYNC_CLIENT_SECRET": "s3cret",
	}
	for k, v := range extra {
		m[k] = v
	}
	return env(m)
}

func TestLoadGroupsyncDefaults(t *testing.T) {
	cfg, err := config.LoadGroupsync(groupsyncEnv(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Interval != 5*time.Minute || cfg.KeycloakRealm != "nava" || cfg.AdminAddr != ":9090" {
		t.Errorf("defaults = %+v", cfg)
	}
}

func TestLoadGroupsyncRejects(t *testing.T) {
	for name, tc := range map[string]struct {
		env  map[string]string
		want string
	}{
		"no database":   {map[string]string{"NAVA_DATABASE_URL": ""}, "NAVA_DATABASE_URL: required"},
		"no keycloak":   {map[string]string{"NAVA_KEYCLOAK_URL": ""}, "NAVA_KEYCLOAK_URL: required"},
		"no client id":  {map[string]string{"NAVA_GROUPSYNC_CLIENT_ID": ""}, "NAVA_GROUPSYNC_CLIENT_ID: required"},
		"no secret":     {map[string]string{"NAVA_GROUPSYNC_CLIENT_SECRET": ""}, "NAVA_GROUPSYNC_CLIENT_SECRET: required"},
		"bad interval":  {map[string]string{"NAVA_GROUPSYNC_INTERVAL": "soon"}, "NAVA_GROUPSYNC_INTERVAL"},
		"over 5 min":    {map[string]string{"NAVA_GROUPSYNC_INTERVAL": "6m"}, "NAVA_GROUPSYNC_INTERVAL: must be at most"},
		"bad keycloak":  {map[string]string{"NAVA_KEYCLOAK_URL": "http://u:p@keycloak"}, "NAVA_KEYCLOAK_URL"},
		"bad db scheme": {map[string]string{"NAVA_DATABASE_URL": "mysql://x"}, "NAVA_DATABASE_URL"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := config.LoadGroupsync(groupsyncEnv(tc.env))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.want)
			}
			if strings.Contains(err.Error(), "s3cret") || strings.Contains(err.Error(), ":pw@") {
				t.Errorf("error echoes a secret: %v", err)
			}
		})
	}
}

func TestLoadGroupsyncShorterInterval(t *testing.T) {
	cfg, err := config.LoadGroupsync(groupsyncEnv(map[string]string{"NAVA_GROUPSYNC_INTERVAL": "30s"}))
	if err != nil || cfg.Interval != 30*time.Second {
		t.Fatalf("cfg = %+v, err = %v", cfg, err)
	}
}
