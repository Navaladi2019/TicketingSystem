package config_test

import "testing"

func TestLoader(t *testing.T) {
	t.Setenv("NAVA_AUDIT_READER_DATABASE_URL", "postgres://x")
}
