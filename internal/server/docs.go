package server

import (
	"html/template"
	"net/http"
	"sort"

	"github.com/getkin/kin-openapi/openapi3"
)

// docsTemplate renders the operations of the contract. It loads nothing from
// outside, so the page works on a closed network and the CSP can be strict.
const docsPage = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>{{.Title}}</title></head>
<body>
<h1>{{.Title}} {{.Version}}</h1>
<p>The contract is <a href="/api/v1/openapi.yaml">openapi.yaml</a> (OpenAPI 3.1).
Errors are <code>application/problem+json</code> with a registered <code>code</code>.</p>
<table border="1" cellpadding="4">
<tr><th>Method</th><th>Path</th><th>Operation</th><th>Summary</th></tr>
{{range .Ops}}<tr><td>{{.Method}}</td><td><code>{{.Path}}</code></td><td>{{.ID}}</td><td>{{.Summary}}</td></tr>
{{end}}</table>
</body></html>
`

type docsOp struct{ Method, Path, ID, Summary string }

func specHandler(data []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(data)
	})
}

func docsHandler(spec *openapi3.T) http.Handler {
	docsTemplate := template.Must(template.New("docs").Parse(docsPage))
	var ops []docsOp
	for path, item := range spec.Paths.Map() {
		for method, op := range item.Operations() {
			ops = append(ops, docsOp{method, path, op.OperationID, op.Summary})
		}
	}
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Path != ops[j].Path {
			return ops[i].Path < ops[j].Path
		}
		return ops[i].Method < ops[j].Method
	})
	data := struct {
		Title, Version string
		Ops            []docsOp
	}{spec.Info.Title, spec.Info.Version, ops}
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_ = docsTemplate.Execute(w, data)
	})
}
