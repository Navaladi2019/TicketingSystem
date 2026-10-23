package main

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/dynamicconfig"
)

// disableStreamingAuthorizerKey turns off the Authorizer on streaming calls.
var disableStreamingAuthorizerKey = dynamicconfig.DisableStreamingAuthorizer.Key()

// guardedClient hides disableStreamingAuthorizerKey, so that editing the
// dynamic config file after the startup check cannot switch it on.
type guardedClient struct{ dynamicconfig.Client }

func (g guardedClient) GetValue(key dynamicconfig.Key) []dynamicconfig.ConstrainedValue {
	if strings.EqualFold(string(key), string(disableStreamingAuthorizerKey)) {
		return nil
	}
	return g.Client.GetValue(key)
}

// guardedKey is one dynamic-config row of the D6-K table (ADR-001 A1). Every
// value of the key, per-namespace constrained values included, must be false.
// requireExplicit keys default to true in the server, so a missing
// unconstrained value refuses too.
type guardedKey struct {
	key             dynamicconfig.Key
	requireExplicit bool
}

// d6kKeys are the D6-K dynamic-config keys, by Go symbol so that a renamed or
// removed key fails to compile. services.frontend.rpc.httpPort is the static
// row; checkSecurityConfig checks it on the config.
var d6kKeys = []guardedKey{
	{dynamicconfig.DisableStreamingAuthorizer.Key(), false},
	{dynamicconfig.ExposeAuthorizerErrors.Key(), false},
	{dynamicconfig.EnableCrossNamespaceCommands.Key(), true},
	{dynamicconfig.EnableBatcherNamespace.Key(), true},
	{dynamicconfig.WorkerEnableScheduler.Key(), true},
	{dynamicconfig.FrontendEnableSchedules.Key(), true},
	{dynamicconfig.FrontendEnableBatcher.Key(), true},
}

// SecurityConfigError is the named error checkSecurityConfig returns for each
// setting that breaks its D6-K rule. Key is the dynamic-config key, or
// "services.frontend.rpc.httpPort".
type SecurityConfigError struct {
	Key    string
	Reason string
}

func (e *SecurityConfigError) Error() string { return e.Key + " " + e.Reason }

const httpPortKey = "services.frontend.rpc.httpPort"

// checkSecurityConfig refuses to start on a config that weakens the
// Authorizer: no frontend mTLS, a frontend client CA that is not a dedicated
// path-length-0 root, the same CA on frontend and internode, or any D6-K
// setting with a forbidden value. A nil dc has no values, so every
// requireExplicit key refuses.
func checkSecurityConfig(cfg *config.Config, dc dynamicconfig.Client) error {
	errs := []error{tlsErrors(cfg)}
	if cfg.Services["frontend"].RPC.HTTPPort != 0 {
		errs = append(errs, &SecurityConfigError{httpPortKey, "must not be set: the HTTP API bypasses the gRPC Authorizer setup"})
	}
	for _, g := range d6kKeys {
		errs = append(errs, g.check(dc))
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("refusing to start: %w", err)
	}
	return nil
}

func (g guardedKey) check(dc dynamicconfig.Client) error {
	var values []dynamicconfig.ConstrainedValue
	if dc != nil {
		values = dc.GetValue(g.key)
	}
	explicit := false
	for _, v := range values {
		if b, err := parseBool(v.Value); err != nil || b {
			return &SecurityConfigError{g.key.String(), fmt.Sprintf("must be false, has %v (constraints %+v)", v.Value, v.Constraints)}
		}
		explicit = explicit || v.Constraints == (dynamicconfig.Constraints{})
	}
	if g.requireExplicit && !explicit {
		return &SecurityConfigError{g.key.String(), "must be set to false explicitly: the server default is true"}
	}
	return nil
}

// parseBool reads a value the way the server's dynamic config does
// (dynamicconfig convertBool), so "true" as a string counts as true.
func parseBool(v any) (bool, error) {
	switch b := v.(type) {
	case bool:
		return b, nil
	case string:
		return strconv.ParseBool(b)
	}
	return false, fmt.Errorf("not a bool: %T", v)
}

// checkTLS is the TLS part of checkSecurityConfig on its own.
func checkTLS(cfg *config.Config) error {
	if err := tlsErrors(cfg); err != nil {
		return fmt.Errorf("refusing to start: %w", err)
	}
	return nil
}

func tlsErrors(cfg *config.Config) error {
	t := cfg.Global.TLS
	fe, in := t.Frontend.Server, t.Internode.Server
	var errs []error
	if !fe.RequireClientAuth {
		errs = append(errs, errors.New("global.tls.frontend.server.requireClientAuth must be true"))
	}
	if len(fe.ClientCAFiles) == 0 && len(fe.ClientCAData) == 0 {
		errs = append(errs, errors.New("global.tls.frontend.server needs the installation CA (clientCaFiles)"))
	}
	if fe.CertFile == "" && fe.CertData == "" {
		errs = append(errs, errors.New("global.tls.frontend.server needs a certificate"))
	}
	if !in.RequireClientAuth {
		errs = append(errs, errors.New("global.tls.internode.server.requireClientAuth must be true"))
	}
	if len(in.ClientCAFiles) == 0 && len(in.ClientCAData) == 0 {
		errs = append(errs, errors.New("global.tls.internode.server needs its own CA (clientCaFiles)"))
	}
	if t.SystemWorker.CertFile == "" && t.SystemWorker.CertData == "" {
		errs = append(errs, errors.New("global.tls.systemWorker needs the temporal-system certificate"))
	}
	if cfg.Global.Authorization.Authorizer != "" || cfg.Global.Authorization.ClaimMapper != "" {
		errs = append(errs, errors.New("global.authorization must be empty: the nava Authorizer and ClaimMapper are compiled in"))
	}
	feCAs, err := loadCAs(fe.ClientCAFiles, fe.ClientCAData)
	if err != nil {
		errs = append(errs, fmt.Errorf("global.tls.frontend.server client CA: %w", err))
	}
	inCAs, err := loadCAs(in.ClientCAFiles, in.ClientCAData)
	if err != nil {
		errs = append(errs, fmt.Errorf("global.tls.internode.server client CA: %w", err))
	}
	for _, f := range feCAs {
		for _, i := range inCAs {
			if bytes.Equal(f.Raw, i.Raw) || bytes.Equal(f.RawSubjectPublicKeyInfo, i.RawSubjectPublicKeyInfo) {
				errs = append(errs, fmt.Errorf("CA %q is trusted on both frontend and internode: internode needs its own CA", f.Subject.CommonName))
			}
		}
	}
	return errors.Join(errs...)
}

// loadCAs reads the PEM CA certificates and requires each to be a self-signed
// root with path length 0. Identity is the certificate CN alone, so a client CA
// that could have intermediates would let its holders mint any CN. Nothing is
// returned for an empty list (the caller reports that separately).
func loadCAs(files, data []string) ([]*x509.Certificate, error) {
	var pems [][]byte
	var errs []error
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			errs = append(errs, fmt.Errorf("read %s: %w", f, err))
			continue
		}
		pems = append(pems, b)
	}
	for _, d := range data {
		b, err := base64.StdEncoding.DecodeString(d)
		if err != nil {
			errs = append(errs, errors.New("clientCaData is not base64"))
			continue
		}
		pems = append(pems, b)
	}
	var out []*x509.Certificate
	for _, b := range pems {
		n := 0
		for len(b) > 0 {
			var blk *pem.Block
			blk, b = pem.Decode(b)
			if blk == nil {
				break
			}
			if blk.Type != "CERTIFICATE" {
				continue
			}
			c, err := x509.ParseCertificate(blk.Bytes)
			if err != nil {
				errs = append(errs, fmt.Errorf("parse certificate: %w", err))
				continue
			}
			n++
			switch {
			case !c.IsCA || !c.BasicConstraintsValid:
				errs = append(errs, fmt.Errorf("%q is not a CA certificate", c.Subject.CommonName))
			case c.MaxPathLen != 0 || !c.MaxPathLenZero:
				errs = append(errs, fmt.Errorf("CA %q must have path length 0 (no intermediates)", c.Subject.CommonName))
			case !bytes.Equal(c.RawSubject, c.RawIssuer) || c.CheckSignatureFrom(c) != nil:
				errs = append(errs, fmt.Errorf("CA %q must be a self-signed root", c.Subject.CommonName))
			}
			out = append(out, c)
		}
		if n == 0 {
			errs = append(errs, errors.New("no CA certificate in the PEM"))
		}
	}
	return out, errors.Join(errs...)
}
