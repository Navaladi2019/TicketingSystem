package temporaltls_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/temporaltls"
)

// ca is a test certificate authority that signs leaf certificates.
type ca struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	pem  []byte
}

func newCA(t *testing.T) ca {
	t.Helper()
	key := newKey(t)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return ca{cert: cert, key: key, pem: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})}
}

// leaf signs a certificate for name with the given usage and returns it as a
// tls.Certificate and as PEM cert and key.
func (c ca) leaf(t *testing.T, name string, usage x509.ExtKeyUsage) (tls.Certificate, []byte, []byte) {
	t.Helper()
	key := newKey(t)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: name},
		DNSNames:     []string{name},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{usage},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, c.cert, &key.PublicKey, c.key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return pair, certPEM, keyPEM
}

func newKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func write(t *testing.T, dir, name string, b []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// files writes a nava-worker client certificate signed by c, and c itself as
// the CA, and returns the config that points at them.
func files(t *testing.T, c ca) config.TemporalTLS {
	t.Helper()
	dir := t.TempDir()
	_, certPEM, keyPEM := c.leaf(t, "nava-worker", x509.ExtKeyUsageClientAuth)
	return config.TemporalTLS{
		CertFile:   write(t, dir, "client.pem", certPEM),
		KeyFile:    write(t, dir, "client.key", keyPEM),
		CAFile:     write(t, dir, "ca.pem", c.pem),
		ServerName: "temporal",
	}
}

func TestUnconfiguredIsPlaintext(t *testing.T) {
	t.Parallel()
	got, err := temporaltls.Load(config.TemporalTLS{})
	if got != nil || err != nil {
		t.Fatalf("got %v, %v; want nil, nil", got, err)
	}
}

// The config trusts the configured CA alone, through RootCAs, pins the server
// name, needs TLS 1.2 or later and never skips verification.
func TestConfigTrustsOnlyTheCAAndPinsTheServer(t *testing.T) {
	t.Parallel()
	c := newCA(t)
	got, err := temporaltls.Load(files(t, c))
	if err != nil {
		t.Fatal(err)
	}
	want := x509.NewCertPool()
	want.AddCert(c.cert)
	if got.RootCAs == nil || !got.RootCAs.Equal(want) {
		t.Errorf("RootCAs is not exactly the configured CA")
	}
	if got.ClientCAs != nil {
		t.Errorf("ClientCAs is set on a client config")
	}
	if got.ServerName != "temporal" {
		t.Errorf("ServerName = %q, want temporal", got.ServerName)
	}
	if got.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %#x, want TLS 1.2 (%#x)", got.MinVersion, tls.VersionTLS12)
	}
	if got.InsecureSkipVerify {
		t.Errorf("InsecureSkipVerify is set")
	}
	if got.VerifyPeerCertificate != nil || got.VerifyConnection != nil {
		t.Errorf("a custom verifier replaces or weakens the standard chain check")
	}
	if len(got.Certificates) != 1 {
		t.Errorf("Certificates has %d entries, want the one client certificate", len(got.Certificates))
	}
}

// A real handshake succeeds only against a frontend certificate for the
// pinned name signed by the configured CA.
func TestHandshakeVerifiesTheFrontend(t *testing.T) {
	t.Parallel()
	c := newCA(t)
	cfg, err := temporaltls.Load(files(t, c))
	if err != nil {
		t.Fatal(err)
	}
	good, _, _ := c.leaf(t, "temporal", x509.ExtKeyUsageServerAuth)
	wrongName, _, _ := c.leaf(t, "localhost", x509.ExtKeyUsageServerAuth)
	otherCA, _, _ := newCA(t).leaf(t, "temporal", x509.ExtKeyUsageServerAuth)
	for name, tc := range map[string]struct {
		server tls.Certificate
		ok     bool
	}{
		"signed by the CA":     {good, true},
		"wrong name":           {wrongName, false},
		"signed by another CA": {otherCA, false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := handshake(t, cfg, tc.server)
			if (err == nil) != tc.ok {
				t.Fatalf("handshake error = %v, want success %v", err, tc.ok)
			}
		})
	}
}

// handshake runs a client handshake with cfg against a loopback server
// presenting server, and returns the client's error.
func handshake(t *testing.T, cfg *tls.Config, server tls.Certificate) error {
	t.Helper()
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{server}, MinVersion: tls.VersionTLS12})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
		_ = conn.(*tls.Conn).HandshakeContext(t.Context())
	}()
	d := net.Dialer{Timeout: 10 * time.Second}
	conn, err := d.DialContext(t.Context(), "tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	return tls.Client(conn, cfg.Clone()).HandshakeContext(t.Context())
}

// Every failure names the variable to fix and never the path or what the
// file holds.
func TestErrorsNameTheVariableNotTheFile(t *testing.T) {
	t.Parallel()
	const secret = "secret-name"
	good := files(t, newCA(t))
	dir := t.TempDir()
	missing := filepath.Join(dir, secret+".pem")
	garbage := write(t, dir, secret+".garbage", []byte(secret+" not PEM"))
	for name, tc := range map[string]struct {
		cfg  config.TemporalTLS
		want string
	}{
		"missing cert and key": {config.TemporalTLS{CertFile: missing, KeyFile: missing, CAFile: good.CAFile}, "NAVA_TEMPORAL_TLS_CERT_FILE"},
		"unparsable key":       {config.TemporalTLS{CertFile: good.CertFile, KeyFile: garbage, CAFile: good.CAFile}, "NAVA_TEMPORAL_TLS_KEY_FILE"},
		"missing CA":           {config.TemporalTLS{CertFile: good.CertFile, KeyFile: good.KeyFile, CAFile: missing}, "NAVA_TEMPORAL_TLS_CA_FILE"},
		"CA holds no cert":     {config.TemporalTLS{CertFile: good.CertFile, KeyFile: good.KeyFile, CAFile: garbage}, "NAVA_TEMPORAL_TLS_CA_FILE holds no certificate"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := temporaltls.Load(tc.cfg)
			if got != nil || err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, %v; want an error naming %s", got, err, tc.want)
			}
			if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), dir) {
				t.Errorf("error %q quotes a path or file contents", err)
			}
		})
	}
}
