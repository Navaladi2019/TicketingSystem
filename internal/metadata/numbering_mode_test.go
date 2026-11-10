package metadata_test

import (
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// P3-36f (TICK-659, D-3-9): a definition's numbering mode defaults to gaps,
// accepts gapless, refuses anything else, and is not a schema change.

func TestCompileNumberingMode(t *testing.T) {
	for in, want := range map[metadata.NumberingMode]metadata.NumberingMode{
		"":                        metadata.NumberingGaps,
		metadata.NumberingGaps:    metadata.NumberingGaps,
		metadata.NumberingGapless: metadata.NumberingGapless,
	} {
		d := baseDefinition(t, baseSchema())
		d.NumberingMode = in
		if got := mustCompile(t, d).NumberingMode(); got != want {
			t.Errorf("Compile(numbering_mode %q).NumberingMode() = %q, want %q", in, got, want)
		}
	}
	for _, bad := range []metadata.NumberingMode{"Gaps", "GAPLESS", "none", " gaps"} {
		d := baseDefinition(t, baseSchema())
		d.NumberingMode = bad
		_, err := metadata.Compile(d)
		wantPath(t, err, "/numbering_mode")
	}
}

func TestClassifyIgnoresNumberingMode(t *testing.T) {
	prev := baseDefinition(t, baseSchema())
	next := baseDefinition(t, baseSchema())
	next.NumberingMode = metadata.NumberingGapless
	c, paths, err := metadata.Classify(mustCompile(t, prev), mustCompile(t, next))
	if err != nil {
		t.Fatal(err)
	}
	if c == metadata.Breaking {
		t.Errorf("Classify(gaps -> gapless) = Breaking at %v; the mode lives on the type, not the version", paths)
	}
}
