package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/dynamicconfig"

	"github.com/Navaladi2019/TicketingSystem/temporal-server/internal/devcerts"
)

// validConfig is a config that passes checkConfig, with real dev certificates.
func validConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	if err := devcerts.WriteInstallation(dir, time.Hour); err != nil {
		t.Fatal(err)
	}
	c := &config.Config{}
	c.Global.TLS.Frontend.Server = config.ServerTLS{
		CertFile: filepath.Join(dir, devcerts.ServerCertFile), KeyFile: filepath.Join(dir, devcerts.ServerKeyFile),
		ClientCAFiles: []string{filepath.Join(dir, devcerts.CAFile)}, RequireClientAuth: true,
	}
	c.Global.TLS.Internode.Server = config.ServerTLS{
		CertFile: filepath.Join(dir, devcerts.InternodeCertFile), KeyFile: filepath.Join(dir, devcerts.InternodeKeyFile),
		ClientCAFiles: []string{filepath.Join(dir, devcerts.InternodeCAFile)}, RequireClientAuth: true,
	}
	c.Global.TLS.SystemWorker.CertFile = filepath.Join(dir, "temporal-system.pem")
	return c
}

type staticDC map[dynamicconfig.Key]any

func (s staticDC) GetValue(k dynamicconfig.Key) []dynamicconfig.ConstrainedValue {
	if v, ok := s[k]; ok {
		return []dynamicconfig.ConstrainedValue{{Value: v}}
	}
	return nil
}

func TestCheckConfigAcceptsValidConfig(t *testing.T) {
	if err := checkConfig(validConfig(t), dynamicconfig.NewNoopClient()); err != nil {
		t.Fatalf("valid config refused: %v", err)
	}
}

func TestCheckConfigRefusesSameCAOnFrontendAndInternode(t *testing.T) {
	c := validConfig(t)
	c.Global.TLS.Internode.Server.ClientCAFiles = c.Global.TLS.Frontend.Server.ClientCAFiles
	err := checkConfig(c, nil)
	if err == nil || !strings.Contains(err.Error(), "both frontend and internode") {
		t.Fatalf("shared CA: got %v", err)
	}
}

func TestCheckConfigRefusesUnsafeSettings(t *testing.T) {
	c := validConfig(t)
	err := checkConfig(c, staticDC{disableStreamingAuthorizerKey: true})
	if err == nil || !strings.Contains(err.Error(), "disableStreamingAuthorizer") {
		t.Errorf("system.disableStreamingAuthorizer=true: got %v", err)
	}
	if err := checkConfig(c, staticDC{disableStreamingAuthorizerKey: false}); err != nil {
		t.Errorf("system.disableStreamingAuthorizer=false refused: %v", err)
	}

	c = validConfig(t)
	c.Services = map[string]config.Service{"frontend": {RPC: config.RPC{GRPCPort: 7233, HTTPPort: 7243}}}
	err = checkConfig(c, nil)
	if err == nil || !strings.Contains(err.Error(), "httpPort") {
		t.Errorf("services.frontend.rpc.httpPort set: got %v", err)
	}
	c.Services["frontend"] = config.Service{RPC: config.RPC{GRPCPort: 7233}}
	if err := checkConfig(c, nil); err != nil {
		t.Errorf("no httpPort refused: %v", err)
	}
}

func TestGuardedClientIgnoresDisableStreamingAuthorizer(t *testing.T) {
	dc := guardedClient{staticDC{disableStreamingAuthorizerKey: true, "other.key": 1}}
	if got := dc.GetValue("System.DisableStreamingAuthorizer"); len(got) != 0 {
		t.Errorf("guarded client returned %v", got)
	}
	if got := dc.GetValue("other.key"); len(got) != 1 {
		t.Errorf("guarded client dropped an unrelated key: %v", got)
	}
}

func TestCheckConfigRefusesClientCAThatIsNotARootWithPathLenZero(t *testing.T) {
	dir := t.TempDir()
	root := writeCA(t, dir, "root.pem", nil, -1) // self-signed, unlimited path length
	writeCA(t, dir, "zero.pem", nil, 0)          // acceptable
	writeCA(t, dir, "inter.pem", root, 0)        // signed by root: an intermediate
	cases := []struct {
		name, file, want string
	}{
		{"unlimited path length", "root.pem", "path length 0"},
		{"intermediate", "inter.pem", "self-signed root"},
		{"missing file", "nope.pem", "read "},
		{"valid", "zero.pem", ""},
	}
	for _, tc := range cases {
		c := validConfig(t)
		c.Global.TLS.Frontend.Server.ClientCAFiles = []string{filepath.Join(dir, tc.file)}
		err := checkConfig(c, nil)
		switch {
		case tc.want == "" && err != nil:
			t.Errorf("%s: refused: %v", tc.name, err)
		case tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)):
			t.Errorf("%s: got %v, want %q", tc.name, err, tc.want)
		}
	}
}

type ca struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

// writeCA writes a CA certificate; parent nil means self-signed. maxPathLen -1
// leaves the path length unlimited.
func writeCA(t *testing.T, dir, name string, parent *ca, maxPathLen int) *ca {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	tmpl := &x509.Certificate{
		SerialNumber: serial, Subject: pkix.Name{CommonName: strings.TrimSuffix(name, ".pem")},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
		MaxPathLen: maxPathLen, MaxPathLenZero: maxPathLen == 0,
	}
	signer, signerKey := tmpl, key
	if parent != nil {
		signer, signerKey = parent.cert, parent.key
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, signer, &key.PublicKey, signerKey)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(der)
	if err := os.WriteFile(filepath.Join(dir, name), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	return &ca{cert, key}
}

// A CA reissued with the same key is the same trust anchor: refuse it on both sides.
func TestCheckConfigRefusesSameCAKeyReissuedForInternode(t *testing.T) {
	dir := t.TempDir()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.pem", "b.pem"} {
		serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
		tmpl := &x509.Certificate{
			SerialNumber: serial, Subject: pkix.Name{CommonName: "ca-" + name},
			NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
			IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign, MaxPathLenZero: true,
		}
		der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	c := validConfig(t)
	c.Global.TLS.Frontend.Server.ClientCAFiles = []string{filepath.Join(dir, "a.pem")}
	c.Global.TLS.Internode.Server.ClientCAFiles = []string{filepath.Join(dir, "b.pem")}
	if err := checkConfig(c, nil); err == nil || !strings.Contains(err.Error(), "both frontend and internode") {
		t.Fatalf("same key, different certificate: got %v", err)
	}
}
