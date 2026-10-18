// Package apidiff is the additive-only check for the /api/v1 contract. Within
// v1 a change may add paths, operations, responses and properties; it may not
// remove or narrow what a client can already rely on. CI runs it against the
// copy of api/openapi.yaml on main (cmd/openapi-compat).
package apidiff

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/getkin/kin-openapi/openapi3"
)

// Check returns one line per breaking change from oldData to newData, sorted,
// and nil when the change is additive.
func Check(oldData, newData []byte) ([]string, error) {
	oldSpec, err := load(oldData)
	if err != nil {
		return nil, fmt.Errorf("base spec: %w", err)
	}
	newSpec, err := load(newData)
	if err != nil {
		return nil, fmt.Errorf("new spec: %w", err)
	}
	c := &checker{seen: map[[2]*openapi3.Schema]bool{}}
	c.paths(oldSpec, newSpec)
	sort.Strings(c.out)
	return c.out, nil
}

func load(data []byte) (*openapi3.T, error) {
	spec, err := openapi3.NewLoader().LoadFromData(data)
	if err != nil {
		return nil, err
	}
	return spec, spec.Validate(context.Background())
}

type checker struct {
	out  []string
	seen map[[2]*openapi3.Schema]bool
}

func (c *checker) fail(format string, args ...any) {
	c.out = append(c.out, fmt.Sprintf(format, args...))
}

type direction int

const (
	request  direction = iota // the client sends it: narrowing breaks callers
	response                  // the client reads it: removing breaks callers
)

func (c *checker) paths(oldSpec, newSpec *openapi3.T) {
	for path, oldItem := range oldSpec.Paths.Map() {
		newItem := newSpec.Paths.Value(path)
		if newItem == nil {
			c.fail("path %s removed", path)
			continue
		}
		for method, oldOp := range oldItem.Operations() {
			where := method + " " + path
			newOp := newItem.GetOperation(method)
			if newOp == nil {
				c.fail("%s removed", where)
				continue
			}
			c.operation(where, oldOp, newOp)
		}
	}
}

func (c *checker) operation(where string, oldOp, newOp *openapi3.Operation) {
	// Parameters: none may be removed, none may become required, none may be
	// added as required.
	for _, np := range newOp.Parameters {
		op := findParam(oldOp.Parameters, np.Value)
		switch {
		case op == nil && np.Value.Required:
			c.fail("%s: new required parameter %s", where, np.Value.Name)
		case op != nil && !op.Required && np.Value.Required:
			c.fail("%s: parameter %s became required", where, np.Value.Name)
		case op != nil:
			c.schema(fmt.Sprintf("%s parameter %s", where, np.Value.Name), op.Schema, np.Value.Schema, request)
		}
	}
	for _, op := range oldOp.Parameters {
		if findParam(newOp.Parameters, op.Value) == nil {
			c.fail("%s: parameter %s removed", where, op.Value.Name)
		}
	}

	switch oldBody, newBody := oldOp.RequestBody, newOp.RequestBody; {
	case newBody == nil && oldBody != nil:
		c.fail("%s: request body removed", where)
	case newBody != nil && oldBody == nil:
		if newBody.Value.Required {
			c.fail("%s: new required request body", where)
		}
	case newBody != nil:
		if !oldBody.Value.Required && newBody.Value.Required {
			c.fail("%s: request body became required", where)
		}
		c.content(where+" request", oldBody.Value.Content, newBody.Value.Content, request)
	}

	for code, oldResp := range oldOp.Responses.Map() {
		newResp := newOp.Responses.Value(code)
		if newResp == nil {
			c.fail("%s: response %s removed", where, code)
			continue
		}
		c.content(fmt.Sprintf("%s response %s", where, code), oldResp.Value.Content, newResp.Value.Content, response)
	}
}

func findParam(list openapi3.Parameters, p *openapi3.Parameter) *openapi3.Parameter {
	for _, q := range list {
		if q.Value.Name == p.Name && q.Value.In == p.In {
			return q.Value
		}
	}
	return nil
}

func (c *checker) content(where string, oldC, newC openapi3.Content, d direction) {
	for media, oldM := range oldC {
		newM, ok := newC[media]
		if !ok {
			c.fail("%s: media type %s removed", where, media)
			continue
		}
		c.schema(where+" "+media, oldM.Schema, newM.Schema, d)
	}
}

func (c *checker) schema(where string, oldRef, newRef *openapi3.SchemaRef, d direction) {
	if oldRef == nil || oldRef.Value == nil {
		return
	}
	if newRef == nil || newRef.Value == nil {
		c.fail("%s: schema removed", where)
		return
	}
	o, n := oldRef.Value, newRef.Value
	key := [2]*openapi3.Schema{o, n}
	if c.seen[key] {
		return
	}
	c.seen[key] = true

	if !sameSet(typeList(o), typeList(n)) {
		c.fail("%s: type changed from %v to %v", where, typeList(o), typeList(n))
	}
	if o.Format != n.Format {
		c.fail("%s: format changed from %q to %q", where, o.Format, n.Format)
	}
	for _, name := range sortedKeys(o.Properties) {
		np, ok := n.Properties[name]
		if !ok {
			c.fail("%s: property %s removed", where, name)
			continue
		}
		c.schema(where+"."+name, o.Properties[name], np, d)
	}
	switch d {
	case response:
		for _, r := range o.Required {
			if !slices.Contains(n.Required, r) {
				c.fail("%s: property %s is no longer always present", where, r)
			}
		}
	case request:
		for _, r := range n.Required {
			if !slices.Contains(o.Required, r) {
				c.fail("%s: property %s became required", where, r)
			}
		}
		for _, v := range o.Enum {
			if len(n.Enum) > 0 && !slices.Contains(n.Enum, v) {
				c.fail("%s: enum value %v removed", where, v)
			}
		}
		if len(o.Enum) == 0 && len(n.Enum) > 0 {
			c.fail("%s: enum added", where)
		}
	}
	if o.Items != nil {
		c.schema(where+"[]", o.Items, n.Items, d)
	}
}

func typeList(s *openapi3.Schema) []string {
	if s.Type == nil {
		return nil
	}
	return slices.Clone(*s.Type)
}

func sameSet(a, b []string) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

func sortedKeys(m openapi3.Schemas) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
