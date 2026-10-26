// Package temporaltls builds the mTLS client config every platform binary
// dials the Temporal frontend with. nava-core, nava-worker and nava-relay all
// load their certificate here, so the three cannot drift apart.
package temporaltls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

// Load builds the client TLS config from the binary's certificate, or returns
// nil (plaintext) when none is configured. The frontend is trusted only
// through the configured CA, never the system pool. File errors name the
// variable, not the path or the file contents.
func Load(t config.TemporalTLS) (*tls.Config, error) {
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
