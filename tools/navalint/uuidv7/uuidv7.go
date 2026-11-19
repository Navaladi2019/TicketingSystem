// Package uuidv7 defines an analyzer that reports calls to github.com/google/uuid
// functions that mint a random v4 id (New, NewString, NewRandom,
// NewRandomFromReader) in non-test code under internal/ and cmd/ (TICK-963).
//
// Every id the platform persists or uses as a durable identifier is a UUIDv7
// from uuid.NewV7 (ADR-001, ADR-004 R1). Deterministic v5 ids (uuid.NewSHA1)
// are not reported.
//
// Exempt: every _test.go file, internal/**/contracttest and
// internal/tenancy/isolation (test helpers that live in non-test files).
package uuidv7

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const (
	modulePath = "github.com/Navaladi2019/TicketingSystem/"
	uuidPkg    = "github.com/google/uuid"
)

// banned are the v4 minting functions.
var banned = map[string]bool{"New": true, "NewString": true, "NewRandom": true, "NewRandomFromReader": true}

// Analyzer reports v4 uuid minting in internal/ and cmd/ non-test code.
var Analyzer = &analysis.Analyzer{
	Name: "uuidv7",
	Doc:  "reports uuid.New, uuid.NewString and uuid.NewRandom in non-test code under internal/ and cmd/: persisted ids are UUIDv7, use uuid.NewV7 (TICK-963)",
	Run:  run,
}

func inScope(pkgPath string) bool {
	p := strings.TrimSuffix(pkgPath, "_test")
	rest, ok := strings.CutPrefix(p, modulePath)
	if !ok {
		return false
	}
	if !strings.HasPrefix(rest, "internal/") && !strings.HasPrefix(rest, "cmd/") {
		return false
	}
	if strings.HasPrefix(rest, "internal/tenancy/isolation") {
		return false
	}
	for _, seg := range strings.Split(rest, "/") {
		if seg == "contracttest" {
			return false
		}
	}
	return true
}

func run(pass *analysis.Pass) (any, error) {
	if !inScope(pass.Pkg.Path()) {
		return nil, nil
	}
	for _, f := range pass.Files {
		if strings.HasSuffix(pass.Fset.File(f.Pos()).Name(), "_test.go") {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok || !banned[id.Name] {
				return true
			}
			fn, ok := pass.TypesInfo.Uses[id].(*types.Func)
			if !ok || fn.Pkg() == nil || fn.Pkg().Path() != uuidPkg {
				return true
			}
			pass.Reportf(id.Pos(), "uuid.%s mints a random v4 id: persisted ids are UUIDv7, use uuid.NewV7 and handle the error (TICK-963)", id.Name)
			return true
		})
	}
	return nil, nil
}
