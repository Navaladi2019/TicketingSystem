package main

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"

	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/dynamicconfig"
)

// disableStreamingAuthorizerKey turns off the Authorizer on streaming calls.
const disableStreamingAuthorizerKey dynamicconfig.Key = "system.disableStreamingAuthorizer"

// guardedClient hides disableStreamingAuthorizerKey, so that editing the
// dynamic config file after the startup check cannot switch it on.
type guardedClient struct{ dynamicconfig.Client }

func (g guardedClient) GetValue(key dynamicconfig.Key) []dynamicconfig.ConstrainedValue {
	if strings.EqualFold(string(key), string(disableStreamingAuthorizerKey)) {
		return nil
	}
	return g.Client.GetValue(key)
}

// checkConfig refuses to start on a config that weakens the Authorizer: no
// frontend mTLS, a frontend client CA that is not a dedicated path-length-0
// root, the same CA on frontend and internode, the frontend HTTP API, or the
// streaming Authorizer switched off.
func checkConfig(cfg *config.Config, dc dynamicconfig.Client) error {
	errs := []error{tlsErrors(cfg)}
	if cfg.Services["frontend"].RPC.HTTPPort != 0 {
		errs = append(errs, errors.New("services.frontend.rpc.httpPort must not be set: the HTTP API bypasses the gRPC Authorizer setup"))
	}
	if dc != nil {
		for _, v := range dc.GetValue(disableStreamingAuthorizerKey) {
			if b, _ := v.Value.(bool); b {
				errs = append(errs, errors.New("dynamic config system.disableStreamingAuthorizer must not be true"))
			}
		}
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("refusing to start: %w", err)
	}
	return nil
}

// checkTLS is the TLS part of checkConfig on its own.
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
			if bytes.Equal(f.Raw, i.Raw) {
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
