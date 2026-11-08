package config_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

func TestLoadAuditReader(t *testing.T) {
	const dsn = "postgres://nava_audit_reader:pw@pgbouncer:5432/nava?sslmode=disable"
	cfg, err := config.LoadAuditReader(env(map[string]string{"NAVA_AUDIT_READER_DATABASE_URL": dsn}))
	if err != nil || cfg.DatabaseURL != dsn {
		t.Fatalf("cfg = %+v, err = %v", cfg, err)
	}
}

// ADR-004-A1 A1-6: nava-core refuses to start without the reader's DSN.
func TestLoadAuditReaderRejects(t *testing.T) {
	for name, tc := range map[string]struct {
		env  map[string]string
		want string
	}{
		"missing":    {map[string]string{}, "NAVA_AUDIT_READER_DATABASE_URL: required"},
		"empty":      {map[string]string{"NAVA_AUDIT_READER_DATABASE_URL": ""}, "NAVA_AUDIT_READER_DATABASE_URL: required"},
		"bad scheme": {map[string]string{"NAVA_AUDIT_READER_DATABASE_URL": "mysql://nava_audit_reader:s3cret@db/nava"}, "NAVA_AUDIT_READER_DATABASE_URL: want a postgres"},
		// The app's DSN is not the reader's.
		"only the app's": {map[string]string{"NAVA_DATABASE_URL": "postgres://nava_app:pw@pg/nava"}, "NAVA_AUDIT_READER_DATABASE_URL: required"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := config.LoadAuditReader(env(tc.env))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.want)
			}
			if strings.Contains(err.Error(), "s3cret") {
				t.Errorf("error echoes the password: %v", err)
			}
		})
	}
}

// PLAN-005-A1 D-A1-6: config.Load, shared by nava-core, nava-worker and
// nava-relay, has no reader DSN, and setting the key changes nothing it
// returns.
func TestLoadHasNoAuditReaderDSN(t *testing.T) {
	var walk func(reflect.Type, string)
	walk = func(ty reflect.Type, path string) {
		for i := range ty.NumField() {
			f := ty.Field(i)
			if strings.Contains(strings.ToLower(f.Name), "auditreader") || strings.Contains(strings.ToLower(f.Name), "reader") {
				t.Errorf("config.Config has %s%s: the reader DSN belongs to LoadAuditReader only", path, f.Name)
			}
			if f.Type.Kind() == reflect.Struct {
				walk(f.Type, path+f.Name+".")
			}
		}
	}
	walk(reflect.TypeOf(config.Config{}), "")
	base := map[string]string{}
	a, errA := config.Load(env(base))
	b, errB := config.Load(env(map[string]string{"NAVA_AUDIT_READER_DATABASE_URL": "postgres://nava_audit_reader:pw@pg/nava"}))
	if errA != nil || errB != nil || !reflect.DeepEqual(a, b) {
		t.Errorf("config.Load reads NAVA_AUDIT_READER_DATABASE_URL: %v, %v", errA, errB)
	}
}
