// Command openapibundle bundles the OpenAPI source tree into one document:
// api/openapi.yaml is the root (info, servers, security, shared components)
// whose `paths` entries are $refs into api/paths/<group>.yaml, one file per
// path group. The bundle is what api/embed.go embeds and oapi-codegen reads.
//
// Usage: openapibundle -root api/openapi.yaml -out api/openapi.bundled.yaml
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// run returns the exit code: 0 bundled, 1 the tree is invalid, 2 usage.
func run(args []string) int {
	fs := flag.NewFlagSet("openapibundle", flag.ContinueOnError)
	root := fs.String("root", "openapi.yaml", "the root spec file")
	out := fs.String("out", "openapi.bundled.yaml", "the bundle to write")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	data, err := Bundle(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "openapibundle:", err)
		return 1
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil { //nolint:gosec // a generated, committed, world-readable spec
		fmt.Fprintln(os.Stderr, "openapibundle:", err)
		return 1
	}
	return 0
}
