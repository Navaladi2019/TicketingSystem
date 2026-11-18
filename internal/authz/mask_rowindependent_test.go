package authz

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"slices"
	"testing"
)

// CI-2b (ADR-006 R9): D6 and D7 filter a data field after one masked check
// that sees the caller and the field, never the row. A mask that depended on
// the row would let the answer to a filter differ per row, and the side-table
// EXISTS would leak it. So a Mask stays {field, read, write}, a mask subject
// matches on Groups and Users only, and the readable check takes no row. A
// change to any of these fails here and needs the Security reviewer (R9).
func TestMaskStaysRowIndependent(t *testing.T) {
	t.Parallel()

	var fields []string
	for f := range reflect.TypeFor[Mask]().Fields() {
		fields = append(fields, f.Name)
	}
	if want := []string{"Field", "Read", "Write"}; !slices.Equal(fields, want) {
		t.Errorf("Mask fields = %v, want %v: a new mask member may be row-dependent (R9)", fields, want)
	}

	// Subject's row-dependent members are the ones a mask ignores; a new one
	// must be judged here before it can reach a mask.
	var subj []string
	for f := range reflect.TypeFor[Subject]().Fields() {
		subj = append(subj, f.Name)
	}
	if want := []string{"Groups", "Users", "Requester", "Assignee", "Shared"}; !slices.Equal(subj, want) {
		t.Errorf("Subject fields = %v, want %v: a new subject may be row-dependent (R9)", subj, want)
	}

	// The checks a filter relies on take (policy, principal, field): no row.
	for name, fn := range map[string]any{"Readable": Readable, "Writable": Writable} {
		if n := reflect.TypeOf(fn).NumIn(); n != 3 {
			t.Errorf("%s takes %d arguments, want 3 (policy, principal, field): it must not see a row", name, n)
		}
	}

	// maskSubjectNames reads Groups and Users of the subject and nothing else.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "mask.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var used []string
	for _, d := range file.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name.Name != "maskSubjectNames" {
			continue
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "s" && !slices.Contains(used, sel.Sel.Name) {
					used = append(used, sel.Sel.Name)
				}
			}
			return true
		})
	}
	slices.Sort(used)
	if want := []string{"Groups", "Users"}; !slices.Equal(used, want) {
		t.Errorf("maskSubjectNames reads %v of the subject, want %v only (R9)", used, want)
	}
}
