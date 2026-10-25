package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

// temporalTLS builds the client TLS config for the Temporal frontend from the
// nava-relay certificate, or returns nil (plaintext) when none is configured.
// File errors name the variable, not the path contents. It is nava-core's
// temporalTLS (cmd/nava-core/deps.go) and nava-worker's; keep the three the same.
func temporalTLS(t config.TemporalTLS) (*tls.Config, error) {
	if !t.Enabled() {
		return nil, nil
	}
	pair, err := tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
	if err != nil {
		return nil, errors.New("temporal TLS: cannot load NAVA_TEMPORAL_TLS_CERT_FILE and NAVA_TEMPORAL_TLS_KEY_FILE")
	}
	caPEM, err := os.ReadFile(t.CAFile)
	if err != nil {
		return nil, errors.New("temporal TLS: cannot read NAVA_TEMPORAL_TLS_CA_FILE")
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("temporal TLS: NAVA_TEMPORAL_TLS_CA_FILE holds no certificate")
	}
	return &tls.Config{
		Certificates: []tls.Certificate{pair},
		RootCAs:      pool,
		ServerName:   t.ServerName,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
