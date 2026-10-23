package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/log"

	"github.com/Navaladi2019/TicketingSystem/temporal-server/internal/devcerts"
)

// validConfig is a config that passes checkSecurityConfig, with real dev certificates.
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

// staticDC maps a key to one unconstrained value, or to the
// []dynamicconfig.ConstrainedValue given.
type staticDC map[dynamicconfig.Key]any

func (s staticDC) GetValue(k dynamicconfig.Key) []dynamicconfig.ConstrainedValue {
	switch v := s[k].(type) {
	case nil:
		return nil
	case []dynamicconfig.ConstrainedValue:
		return v
	default:
		return []dynamicconfig.ConstrainedValue{{Value: v}}
	}
}

// d6kCases is the D6-K table of ADR-001 A1, each key by its Go symbol so that a
// renamed or removed key fails to compile (CI-E).
var d6kCases = []struct {
	key dynamicconfig.Key
	// missingRefuses marks the "not false" rows, whose server default is true.
	missingRefuses bool
}{
	{dynamicconfig.DisableStreamingAuthorizer.Key(), false},
	{dynamicconfig.ExposeAuthorizerErrors.Key(), false},
	{dynamicconfig.EnableCrossNamespaceCommands.Key(), true},
	{dynamicconfig.EnableBatcherNamespace.Key(), true},
	{dynamicconfig.WorkerEnableScheduler.Key(), true},
	{dynamicconfig.FrontendEnableSchedules.Key(), true},
	{dynamicconfig.FrontendEnableBatcher.Key(), true},
}

// shippedDC has every D6-K key at its shipped value, false.
func shippedDC() staticDC {
	dc := staticDC{}
	for _, c := range d6kCases {
		dc[c.key] = false
	}
	return dc
}

// requireRefused fails unless err holds a *SecurityConfigError for key.
func requireRefused(t *testing.T, err error, key string) {
	t.Helper()
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		for _, e := range joined.Unwrap() {
			var sce *SecurityConfigError
			if errors.As(e, &sce) && sce.Key == key {
				return
			}
		}
	}
	t.Fatalf("got %v, want a SecurityConfigError for %s", err, key)
}

func TestCheckSecurityConfigAcceptsValidConfig(t *testing.T) {
	if err := checkSecurityConfig(validConfig(t), shippedDC()); err != nil {
		t.Fatalf("valid config refused: %v", err)
	}
}

func TestCheckSecurityConfigRefusesSameCAOnFrontendAndInternode(t *testing.T) {
	c := validConfig(t)
	c.Global.TLS.Internode.Server.ClientCAFiles = c.Global.TLS.Frontend.Server.ClientCAFiles
	err := checkSecurityConfig(c, shippedDC())
	if err == nil || !strings.Contains(err.Error(), "both frontend and internode") {
		t.Fatalf("shared CA: got %v", err)
	}
}

// TestCheckSecurityConfigD6K is CI-E: one sub-case per D6-K row, and per
// constrained (per-namespace) value.
func TestCheckSecurityConfigD6K(t *testing.T) {
	if len(d6kKeys) != len(d6kCases) {
		t.Fatalf("the guard checks %d keys, the D6-K table has %d", len(d6kKeys), len(d6kCases))
	}
	nava := dynamicconfig.Constraints{Namespace: "nava"}
	for i, tc := range d6kCases {
		key := tc.key.String()
		t.Run(key, func(t *testing.T) {
			if d6kKeys[i].key != tc.key || d6kKeys[i].requireExplicit != tc.missingRefuses {
				t.Fatalf("guard row %d is %+v, want %s requireExplicit=%v", i, d6kKeys[i], key, tc.missingRefuses)
			}
			c := validConfig(t)
			with := func(v any) staticDC {
				dc := shippedDC()
				dc[tc.key] = v
				return dc
			}
			mustStart := func(t *testing.T, dc staticDC) {
				t.Helper()
				if err := checkSecurityConfig(c, dc); err != nil {
					t.Fatalf("refused: %v", err)
				}
			}

			t.Run("true refuses", func(t *testing.T) { requireRefused(t, checkSecurityConfig(c, with(true)), key) })
			// The server's convertBool parses strings, so "true" is true.
			t.Run("string true refuses", func(t *testing.T) { requireRefused(t, checkSecurityConfig(c, with("TRUE")), key) })
			t.Run("non-bool refuses", func(t *testing.T) { requireRefused(t, checkSecurityConfig(c, with(1)), key) })
			t.Run("false starts", func(t *testing.T) { mustStart(t, with(false)) })
			t.Run("missing", func(t *testing.T) {
				dc := shippedDC()
				delete(dc, tc.key)
				if tc.missingRefuses {
					requireRefused(t, checkSecurityConfig(c, dc), key)
				} else {
					mustStart(t, dc)
				}
			})
			t.Run("per-namespace true refuses", func(t *testing.T) {
				requireRefused(t, checkSecurityConfig(c, with([]dynamicconfig.ConstrainedValue{
					{Value: false}, {Constraints: nava, Value: true},
				})), key)
			})
			t.Run("per-namespace false starts", func(t *testing.T) {
				mustStart(t, with([]dynamicconfig.ConstrainedValue{{Value: false}, {Constraints: nava, Value: false}}))
			})
			// With only a namespace value, every other namespace gets the server default.
			t.Run("per-namespace false only", func(t *testing.T) {
				dc := with([]dynamicconfig.ConstrainedValue{{Constraints: nava, Value: false}})
				if tc.missingRefuses {
					requireRefused(t, checkSecurityConfig(c, dc), key)
				} else {
					mustStart(t, dc)
				}
			})
		})
	}

	t.Run(httpPortKey, func(t *testing.T) {
		c := validConfig(t)
		c.Services = map[string]config.Service{"frontend": {RPC: config.RPC{GRPCPort: 7233, HTTPPort: 7243}}}
		requireRefused(t, checkSecurityConfig(c, shippedDC()), httpPortKey)
		c.Services["frontend"] = config.Service{RPC: config.RPC{GRPCPort: 7233}}
		if err := checkSecurityConfig(c, shippedDC()); err != nil {
			t.Fatalf("no httpPort refused: %v", err)
		}
	})

	t.Run("no dynamic config refuses", func(t *testing.T) {
		for _, dc := range []dynamicconfig.Client{nil, dynamicconfig.NewNoopClient()} {
			err := checkSecurityConfig(validConfig(t), dc)
			for _, tc := range d6kCases {
				if tc.missingRefuses {
					requireRefused(t, err, tc.key.String())
				}
			}
		}
	})
}

// The shipped dynamic config files, read the way the server reads them, start,
// and set every D6-K key explicitly to false.
func TestShippedDynamicConfigStarts(t *testing.T) {
	for _, path := range []string{
		filepath.Join("..", "..", "config", "dynamicconfig", "development.yaml"),
		filepath.Join("..", "..", "..", "deploy", "compose", "temporal", "config", "dynamicconfig", "compose.yaml"),
	} {
		t.Run(path, func(t *testing.T) {
			done := make(chan interface{})
			defer close(done)
			dc, err := dynamicconfig.NewFileBasedClient(&dynamicconfig.FileBasedClientConfig{
				Filepath: path, PollInterval: time.Minute,
			}, log.NewNoopLogger(), done)
			if err != nil {
				t.Fatal(err)
			}
			if err := checkSecurityConfig(validConfig(t), dc); err != nil {
				t.Fatalf("refused: %v", err)
			}
			for _, tc := range d6kCases {
				if got := dc.GetValue(tc.key); len(got) != 1 || got[0].Value != false {
					t.Errorf("%s = %+v, want exactly one explicit false", tc.key, got)
				}
			}
		})
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
		err := checkSecurityConfig(c, shippedDC())
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
	if err := checkSecurityConfig(c, shippedDC()); err == nil || !strings.Contains(err.Error(), "both frontend and internode") {
		t.Fatalf("same key, different certificate: got %v", err)
	}
}
