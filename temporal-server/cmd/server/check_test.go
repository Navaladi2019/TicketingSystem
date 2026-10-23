package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
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

// d6kCases is the D6-K table of ADR-001 A1 and A2, each key by its Go symbol so
// that a renamed or removed key fails to compile (CI-E).
var d6kCases = []struct {
	key dynamicconfig.Key
	// want is the only value that starts, and the shipped one.
	want bool
	// missingRefuses marks the rows that must be set explicitly.
	missingRefuses bool
}{
	{dynamicconfig.DisableStreamingAuthorizer.Key(), false, false},
	{dynamicconfig.ExposeAuthorizerErrors.Key(), false, false},
	{dynamicconfig.EnableCrossNamespaceCommands.Key(), false, true},
	{dynamicconfig.EnableBatcherNamespace.Key(), false, true},
	{dynamicconfig.WorkerEnableScheduler.Key(), false, true},
	{dynamicconfig.FrontendEnableSchedules.Key(), false, true},
	{dynamicconfig.FrontendEnableBatcher.Key(), false, true},
}

// shippedDC has every D6-K key at its shipped value.
func shippedDC() staticDC {
	dc := staticDC{}
	for _, c := range d6kCases {
		dc[c.key] = c.want
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
			if g := d6kKeys[i]; g.key != tc.key || g.want != tc.want || g.requireExplicit != tc.missingRefuses {
				t.Fatalf("guard row %d is %+v, want %s want=%v requireExplicit=%v", i, g, key, tc.want, tc.missingRefuses)
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

			good, bad := tc.want, !tc.want
			t.Run(fmt.Sprintf("%v refuses", bad), func(t *testing.T) { requireRefused(t, checkSecurityConfig(c, with(bad)), key) })
			// The server's convertBool parses strings, so "TRUE" is true.
			t.Run(fmt.Sprintf("string %v refuses", bad), func(t *testing.T) {
				requireRefused(t, checkSecurityConfig(c, with(strings.ToUpper(strconv.FormatBool(bad)))), key)
			})
			t.Run("non-bool refuses", func(t *testing.T) { requireRefused(t, checkSecurityConfig(c, with(1)), key) })
			t.Run(fmt.Sprintf("%v starts", good), func(t *testing.T) { mustStart(t, with(good)) })
			t.Run("missing", func(t *testing.T) {
				dc := shippedDC()
				delete(dc, tc.key)
				if tc.missingRefuses {
					requireRefused(t, checkSecurityConfig(c, dc), key)
				} else {
					mustStart(t, dc)
				}
			})
			t.Run(fmt.Sprintf("per-namespace %v refuses", bad), func(t *testing.T) {
				requireRefused(t, checkSecurityConfig(c, with([]dynamicconfig.ConstrainedValue{
					{Value: good}, {Constraints: nava, Value: bad},
				})), key)
			})
			t.Run(fmt.Sprintf("per-namespace %v starts", good), func(t *testing.T) {
				mustStart(t, with([]dynamicconfig.ConstrainedValue{{Value: good}, {Constraints: nava, Value: good}}))
			})
			// With only a namespace value, every other namespace gets the server default.
			t.Run(fmt.Sprintf("per-namespace %v only", good), func(t *testing.T) {
				dc := with([]dynamicconfig.ConstrainedValue{{Constraints: nava, Value: good}})
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
// and set every D6-K key explicitly to its shipped value.
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
				if got := dc.GetValue(tc.key); len(got) != 1 || got[0].Value != tc.want {
					t.Errorf("%s = %+v, want exactly one explicit %v", tc.key, got, tc.want)
				}
			}
		})
	}
}

// RS2: after startup, a rewrite of the dynamic config file cannot change a
// D6-K key, while every other key is still picked up. Reads the shipped file
// with the server's file client and waits for a real poll.
func TestGuardedClientPinsD6KKeysAtRuntime(t *testing.T) {
	shipped, err := os.ReadFile(filepath.Join("..", "..", "config", "dynamicconfig", "development.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "dynamicconfig.yaml")
	if err := os.WriteFile(path, shipped, 0o600); err != nil {
		t.Fatal(err)
	}
	const poll = 5 * time.Second // the file client's minimum
	done := make(chan interface{})
	defer close(done)
	file, err := dynamicconfig.NewFileBasedClient(&dynamicconfig.FileBasedClientConfig{
		Filepath: path, PollInterval: poll,
	}, log.NewNoopLogger(), done)
	if err != nil {
		t.Fatal(err)
	}
	dc := newGuardedClient(file)
	if err := checkSecurityConfig(validConfig(t), dc); err != nil {
		t.Fatalf("shipped file refused: %v", err)
	}

	other := dynamicconfig.FrontendRPS.Key()
	var flipped strings.Builder
	for _, tc := range d6kCases {
		fmt.Fprintf(&flipped, "%s:\n  - value: %v\n", tc.key, !tc.want)
	}
	flipped.WriteString(other.String() + ":\n  - value: 1234\n")
	if err := os.WriteFile(path, []byte(flipped.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	// The file client reloads only on a newer modification time.
	future := time.Now().Add(time.Minute)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}

	// Wait until the file client has loaded the rewrite.
	for deadline := time.Now().Add(3 * poll); ; time.Sleep(100 * time.Millisecond) {
		if v := file.GetValue(other); len(v) == 1 && v[0].Value == 1234 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the file client did not reload %s within %v", path, 3*poll)
		}
	}

	t.Run("non-D6-K key passes through", func(t *testing.T) {
		if got := dc.GetValue(other); len(got) != 1 || got[0].Value != 1234 {
			t.Errorf("%s = %+v, want the new value 1234", other, got)
		}
	})
	for _, tc := range d6kCases {
		t.Run(tc.key.String(), func(t *testing.T) {
			if got := file.GetValue(tc.key); len(got) != 1 || got[0].Value != !tc.want {
				t.Fatalf("file client has %+v, want the rewritten %v", got, !tc.want)
			}
			for _, k := range []dynamicconfig.Key{tc.key, dynamicconfig.Key(strings.ToUpper(tc.key.String()))} {
				if got := dc.GetValue(k); len(got) != 1 || got[0].Value != tc.want {
					t.Errorf("GetValue(%s) = %+v, want the startup value %v", k, got, tc.want)
				}
			}
		})
	}
}

// The snapshot is taken once: a caller that changes the returned slice does not
// change what the server reads next, and a key missing at startup stays missing.
func TestGuardedClientSnapshotIsImmutable(t *testing.T) {
	src := shippedDC()
	delete(src, dynamicconfig.ExposeAuthorizerErrors.Key())
	dc := newGuardedClient(src)
	key := dynamicconfig.EnableCrossNamespaceCommands.Key()
	dc.GetValue(key)[0].Value = true
	src[key] = true
	src[dynamicconfig.ExposeAuthorizerErrors.Key()] = true
	if got := dc.GetValue(key); len(got) != 1 || got[0].Value != false {
		t.Errorf("%s = %+v, want false", key, got)
	}
	if got := dc.GetValue(dynamicconfig.ExposeAuthorizerErrors.Key()); len(got) != 0 {
		t.Errorf("a key missing at startup came back as %+v", got)
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
