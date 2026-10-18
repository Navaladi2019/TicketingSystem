package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

func TestTemporalTLSUnconfiguredIsPlaintext(t *testing.T) {
	got, err := temporalTLS(config.TemporalTLS{})
	if got != nil || err != nil {
		t.Fatalf("got %v, %v; want nil, nil", got, err)
	}
}

func TestTemporalTLSMissingFilesFailClosedWithoutPaths(t *testing.T) {
	dir := t.TempDir()
	_, err := temporalTLS(config.TemporalTLS{
		CertFile: filepath.Join(dir, "secret-name.pem"), KeyFile: filepath.Join(dir, "secret-name.key"), CAFile: filepath.Join(dir, "ca.pem"),
	})
	if err == nil || !strings.Contains(err.Error(), "NAVA_TEMPORAL_TLS_CERT_FILE") || strings.Contains(err.Error(), "secret-name") {
		t.Fatalf("got %v, want an error naming the variable, not the path", err)
	}
}
