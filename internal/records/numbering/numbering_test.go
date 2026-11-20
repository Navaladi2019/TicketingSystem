package numbering_test

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
)

// Pinned test 1 (ADR-005 §4.2, P6-02): the default format is epoch 0 with
// scope key ”.
func TestCompileDefaultIsEpochZeroWithEmptyScope(t *testing.T) {
	c, err := numbering.Compile("{prefix}{seq:7}", numbering.Grow)
	if err != nil {
		t.Fatal(err)
	}
	if s, err := c.Scope(); err != nil || s.KeyPrefix != "" || s.Dated() || s.PeriodUnit != "" {
		t.Errorf("Scope() = %+v, %v; want key '', no date part, no period", s, err)
	}
	if c.Epoch != 0 {
		t.Errorf("Epoch = %d, want 0", c.Epoch)
	}
	if c.Head != "{prefix}" || c.Width != 7 || c.Tail != "" || c.Overflow != numbering.Grow || len(c.ScopeParts) != 0 || c.TimeZone != "UTC" {
		t.Errorf("Compile = %+v, want head {prefix}, width 7, no tail, grow, no scope parts, UTC", c)
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(c.CompiledSHA256) {
		t.Errorf("CompiledSHA256 = %q, want 64 hex digits", c.CompiledSHA256)
	}
	again, _ := numbering.Compile("{prefix}{seq:7}", numbering.Grow)
	if again.CompiledSHA256 != c.CompiledSHA256 {
		t.Errorf("CompiledSHA256 is not stable: %q, then %q", c.CompiledSHA256, again.CompiledSHA256)
	}
	if d := numbering.Default(); d.CompiledSHA256 != c.CompiledSHA256 || d.Epoch != 0 {
		t.Errorf("Default() = %+v, want Compile's %+v", d, c)
	}
}

func TestRenderDefault(t *testing.T) {
	c := numbering.Default()
	for n, want := range map[int64]string{
		1: "INC0000001", 43: "INC0000043", 9999999: "INC9999999", 10000000: "INC10000000", 12345678: "INC12345678",
	} {
		got, err := c.Render("INC", n)
		if err != nil || got != want {
			t.Errorf("Render(INC, %d) = %q, %v; want %q", n, got, err, want)
		}
	}
	if _, err := c.Render("INC", 0); err == nil {
		t.Error("Render(INC, 0): no error, want one (values start at 1)")
	}
}

func TestRenderRefuseOverflow(t *testing.T) {
	c, err := numbering.Compile("{prefix}{seq:7}", numbering.Refuse)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := c.Render("INC", 9999999); err != nil || got != "INC9999999" {
		t.Errorf("Render(9999999) = %q, %v", got, err)
	}
	if _, err := c.Render("INC", 10000000); !errors.Is(err, numbering.ErrNumberSpaceExhausted) {
		t.Errorf("Render(10000000) = %v, want ErrNumberSpaceExhausted", err)
	}
	if got, _, err := c.SQL("$1", "v", "", nil); err != nil || got != "($1 || CASE WHEN v < 10000000 THEN lpad(v::text, 7, '0') END)" {
		t.Errorf("SQL with refuse = %q, %v; want NULL past the width", got, err)
	}
	if !c.Fits(9999999) || c.Fits(10000000) || !numbering.Default().Fits(10000000) {
		t.Error("Fits: refuse fits 7 digits only; grow fits every value")
	}
}

// RenderParts takes the date digits PostgreSQL put in the scope key
// (TICK-674, A7) and refuses digits that are not the format's date part.
func TestRenderParts(t *testing.T) {
	c, err := numbering.Compile("{prefix}/{yy}/{seq:2}/{MM}", numbering.Refuse)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := c.RenderParts("VISA", "2611", 1); err != nil || got != "VISA/26/01/11" {
		t.Errorf("RenderParts = %q, %v; want VISA/26/01/11", got, err)
	}
	for _, bad := range []string{"", "261", "26111", "26x1"} {
		if _, err := c.RenderParts("VISA", bad, 1); err == nil {
			t.Errorf("RenderParts(%q): no error, want one", bad)
		}
	}
	if _, err := c.RenderParts("VISA", "2611", 100); !errors.Is(err, numbering.ErrNumberSpaceExhausted) {
		t.Errorf("RenderParts(100) = %v, want ErrNumberSpaceExhausted", err)
	}
	edited := c
	edited.Tail = "/X"
	if _, err := edited.RenderParts("VISA", "2611", 1); !errors.Is(err, numbering.ErrInvalidCompiled) {
		t.Errorf("RenderParts of an edited format = %v, want ErrInvalidCompiled", err)
	}
}

// A Compiled that is not what Compile makes of its own template (a corrupt
// or edited snapshot) renders nothing, in Go or in SQL.
func TestRenderRefusesEditedCompiled(t *testing.T) {
	for name, mutate := range map[string]func(*numbering.Compiled){
		"tail":          func(c *numbering.Compiled) { c.Tail = "/X" },
		"head":          func(c *numbering.Compiled) { c.Head = "{prefix}-" },
		"scope parts":   func(c *numbering.Compiled) { c.ScopeParts = []string{"{yyyy}"} },
		"width 0":       func(c *numbering.Compiled) { c.Width = 0 },
		"width 13":      func(c *numbering.Compiled) { c.Width = 13 },
		"overflow":      func(c *numbering.Compiled) { c.Overflow = "wrap" },
		"hash":          func(c *numbering.Compiled) { c.CompiledSHA256 = strings.Repeat("0", 64) },
		"default epoch": func(c *numbering.Compiled) { c.Epoch = 3 },
		"epoch -2":      func(c *numbering.Compiled) { c.Epoch = -2 },
	} {
		c := numbering.Default()
		mutate(&c)
		if _, err := c.RenderAt("INC", time.Now(), 1); !errors.Is(err, numbering.ErrInvalidCompiled) {
			t.Errorf("%s: RenderAt = %v, want ErrInvalidCompiled", name, err)
		}
		if _, _, err := c.SQL("$1", "v", "l", nil); !errors.Is(err, numbering.ErrInvalidCompiled) {
			t.Errorf("%s: SQL = %v, want ErrInvalidCompiled", name, err)
		}
		if _, err := c.ScopeKeyAt(""); !errors.Is(err, numbering.ErrInvalidCompiled) {
			t.Errorf("%s: ScopeKeyAt = %v, want ErrInvalidCompiled", name, err)
		}
		if _, err := c.LiteralHead("INC"); !errors.Is(err, numbering.ErrInvalidCompiled) {
			t.Errorf("%s: LiteralHead = %v, want ErrInvalidCompiled", name, err)
		}
		if _, err := c.DatePart(time.Now()); !errors.Is(err, numbering.ErrInvalidCompiled) {
			t.Errorf("%s: DatePart = %v, want ErrInvalidCompiled", name, err)
		}
		if _, err := c.Scope(); !errors.Is(err, numbering.ErrInvalidCompiled) {
			t.Errorf("%s: Scope = %v, want ErrInvalidCompiled", name, err)
		}
		if _, err := c.DatePartOf(""); !errors.Is(err, numbering.ErrInvalidCompiled) {
			t.Errorf("%s: DatePart = %v, want ErrInvalidCompiled", name, err)
		}
		if numbering.Disjoint(c, numbering.Default(), "INC") {
			t.Errorf("%s: Disjoint = true, want false", name)
		}
	}
}

func TestSQLDefault(t *testing.T) {
	got, args, err := numbering.Default().SQL("$15::text", "(c.next - 1)", "", []any{"a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 1 {
		t.Errorf("SQL args = %v, want the caller's one arg only (the default has no literal)", args)
	}
	const want = "($15::text || CASE WHEN (c.next - 1) < 10000000 THEN lpad((c.next - 1)::text, 7, '0') ELSE (c.next - 1)::text END)"
	if got != want {
		t.Errorf("SQL =\n%s\nwant\n%s", got, want)
	}
}
