// Package devcerts writes a throwaway installation CA and certificates for local
// development and tests. Never use them outside a developer machine: the CA key
// is discarded, the certificates are short-lived, and nothing here is a secret
// store.
package devcerts

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Files written into the output directory. Each client identity gets
// <cn>.pem and <cn>.key.
const (
	// CAFile is the installation CA: the frontend client CA (clientCaFiles) and
	// the root that clients trust for the frontend certificate. Path length 0,
	// so it cannot have intermediates (identity is the CN alone).
	CAFile         = "ca.pem"
	ServerCertFile = "server.pem"
	ServerKeyFile  = "server.key"
	// InternodeCAFile is a second CA, separate from CAFile, that signs only the
	// internode certificate. The internode ports have no Authorizer, so no
	// client identity may chain to it.
	InternodeCAFile   = "internode-ca.pem"
	InternodeCertFile = "internode.pem"
	InternodeKeyFile  = "internode.key"
	// RogueCN is a client certificate for nava-relay issued by a different CA;
	// the frontend must reject it at the handshake.
	RogueCN = "rogue-nava-relay"
	// UnlistedCN is a client certificate from the installation CA whose CN is
	// no identity: the handshake passes and the Authorizer denies every call.
	UnlistedCN = "nava-unknown"
)

// PlatformCNs are the five identities of the installation. WriteInstallation
// issues exactly these.
var PlatformCNs = []string{"nava-relay", "nava-core", "nava-worker", "nava-operator", "temporal-system"}

// ClientCNs are the client certificates Write issues for tests: the platform
// identities plus one CN that the ClaimMapper does not know.
var ClientCNs = append(append([]string(nil), PlatformCNs...), UnlistedCN)

// ServerNames are the DNS SANs on the frontend and internode certificates (also
// 127.0.0.1): the in-container name, the gateway name clients dial, and the
// server's compose service name.
var ServerNames = []string{"localhost", "temporal", "temporal-server"}

type keyPair struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

// Write generates, for tests, everything WriteInstallation does plus a
// certificate with an unknown CN and the rogue one.
func Write(dir string, validFor time.Duration) error {
	return write(dir, validFor, ClientCNs, true)
}

// WriteInstallation generates the installation CA, the frontend certificate,
// the separate internode CA and certificate, and one client certificate for
// each of PlatformCNs. It is what `make dev-ca` runs.
func WriteInstallation(dir string, validFor time.Duration) error {
	return write(dir, validFor, PlatformCNs, false)
}

// WriteInstallationWithUnlisted is WriteInstallation plus UnlistedCN, for the
// compose stack's CI-6 suite (T32a). Nothing in the stack mounts UnlistedCN.
func WriteInstallationWithUnlisted(dir string, validFor time.Duration) error {
	return write(dir, validFor, ClientCNs, false)
}

// WriteRogue writes RogueCN: a nava-relay certificate from an unrelated CA,
// for tests that need a handshake failure.
func WriteRogue(dir string, validFor time.Duration) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	rogueCA, err := issue("rogue-ca", nil, kindCA, validFor)
	if err != nil {
		return err
	}
	rogue, err := issue("nava-relay", rogueCA, kindClient, validFor)
	if err != nil {
		return err
	}
	return writePair(dir, RogueCN, rogue)
}

func write(dir string, validFor time.Duration, cns []string, rogue bool) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	ca, err := issue("nava-dev-installation-ca", nil, kindCA, validFor)
	if err != nil {
		return err
	}
	if err := writeCert(filepath.Join(dir, CAFile), ca.cert); err != nil {
		return err
	}
	srv, err := issue("temporal", ca, kindServer, validFor)
	if err != nil {
		return err
	}
	if err := writePair(dir, "server", srv); err != nil {
		return err
	}
	inCA, err := issue("nava-dev-internode-ca", nil, kindCA, validFor)
	if err != nil {
		return err
	}
	if err := writeCert(filepath.Join(dir, InternodeCAFile), inCA.cert); err != nil {
		return err
	}
	in, err := issue("temporal-internode", inCA, kindServer, validFor)
	if err != nil {
		return err
	}
	if err := writePair(dir, "internode", in); err != nil {
		return err
	}
	for _, cn := range cns {
		c, err := issue(cn, ca, kindClient, validFor)
		if err != nil {
			return err
		}
		if err := writePair(dir, cn, c); err != nil {
			return err
		}
	}
	if rogue {
		return WriteRogue(dir, validFor)
	}
	return nil
}

type kind int

const (
	kindCA kind = iota
	kindServer
	kindClient
)

func issue(cn string, parent *keyPair, k kind, validFor time.Duration) (*keyPair, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 62))
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: cn, Organization: []string{"nava-dev"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(validFor),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	switch k {
	case kindCA:
		// Path length 0: this CA signs leaves only, never an intermediate.
		tmpl.IsCA, tmpl.BasicConstraintsValid, tmpl.MaxPathLen, tmpl.MaxPathLenZero = true, true, 0, true
		tmpl.KeyUsage |= x509.KeyUsageCertSign
	case kindServer:
		// The internode certificate is both the server and the client certificate.
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
		tmpl.DNSNames = ServerNames
		tmpl.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	case kindClient:
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	signer, signerKey := tmpl, key
	if parent != nil {
		signer, signerKey = parent.cert, parent.key
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, signer, &key.PublicKey, signerKey)
	if err != nil {
		return nil, fmt.Errorf("issue %s: %w", cn, err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	return &keyPair{cert: cert, key: key}, nil
}

func writeCert(path string, cert *x509.Certificate) error {
	return os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}), 0o644)
}

func writePair(dir, name string, kp *keyPair) error {
	if err := writeCert(filepath.Join(dir, name+".pem"), kp.cert); err != nil {
		return err
	}
	der, err := x509.MarshalECPrivateKey(kp.key)
	if err != nil {
		return err
	}
	// 0644 so the non-root server user in the container can read a bind mount.
	// These are throwaway development keys.
	return os.WriteFile(filepath.Join(dir, name+".key"),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}), 0o644)
}
