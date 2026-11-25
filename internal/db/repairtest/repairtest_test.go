package repairtest

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

const doc = "text\n```sql tick-1125-paste-1\nBEGIN;\nDO $nava_repair$\nBEGIN\n  a;\nEND\n$nava_repair$;\n```\nmore\n"

func TestBlock(t *testing.T) {
	got, err := Block(doc, PasteOne)
	if err != nil || !strings.HasPrefix(got, "BEGIN;\nDO ") || !strings.HasSuffix(got, "$nava_repair$;\n") {
		t.Errorf("Block = %q, %v", got, err)
	}
	if _, err := Block(doc, PasteTwo); !errors.Is(err, errNoBlock) {
		t.Errorf("a missing block: %v", err)
	}
	if _, err := Block("```sql tick-1125-check\nSELECT 1;\n", Check); !errors.Is(err, errNoBlock) {
		t.Errorf("a block with no end: %v", err)
	}
}

func TestWithInputs(t *testing.T) {
	var sql strings.Builder
	for _, k := range []string{"tenant_id", "type_id", "new_name", "source_ref", "record_limit"} {
		sql.WriteString("SELECT f('nava_repair." + k + "', '<" + k + ">', true);\n")
	}
	got, err := WithInputs(sql.String(), Inputs{"a", "b", "c$1", "d", "e"})
	if err != nil || strings.Contains(got, "<") || !strings.Contains(got, "'nava_repair.new_name', 'c$1', true") {
		t.Errorf("WithInputs = %q, %v", got, err)
	}
	if _, err := WithInputs("SELECT 1;\n", Inputs{}); !errors.Is(err, errNoInput) {
		t.Errorf("no inputs: %v", err)
	}
}

func TestStatements(t *testing.T) {
	body, err := Block(doc, PasteOne)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Statements(body)
	if want := []string{"BEGIN;", "DO $nava_repair$\nBEGIN\n  a;\nEND\n$nava_repair$;"}; err != nil || !slices.Equal(got, want) {
		t.Errorf("Statements = %q, %v", got, err)
	}
	if _, err := Statements("SELECT 1\n"); !errors.Is(err, errNotPaste) {
		t.Errorf("no end: %v", err)
	}
}
