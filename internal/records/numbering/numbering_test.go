package numbering_test

import (
	"errors"
	"regexp"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
)

// Pinned test 1 (ADR-005 §4.2, P6-02): the default format is epoch 0 with
// scope key ”.
func TestCompileDefaultIsEpochZeroWithEmptyScope(t *testing.T) {
	c, err := numbering.Compile("{prefix}{seq:7}", numbering.Grow)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.ScopeKey(); got != "" {
		t.Errorf("ScopeKey() = %q, want \"\"", got)
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

func TestCompileRefusesEveryOtherTemplate(t *testing.T) {
	for _, tc := range []struct {
		template string
		overflow numbering.Overflow
	}{
		{"{prefix}{seq:7}", numbering.Refuse},
		{"{prefix}{seq:7}", ""},
		{"{prefix}{seq:6}", numbering.Grow},
		{"{prefix}-{yyyy}{MM}{seq:4}", numbering.Grow},
		{"{yyyy}{MM}{seq:4}", numbering.Grow},
		{"", numbering.Grow},
	} {
		if _, err := numbering.Compile(tc.template, tc.overflow); !errors.Is(err, numbering.ErrTemplatesUnavailable) {
			t.Errorf("Compile(%q, %q) = %v, want ErrTemplatesUnavailable", tc.template, tc.overflow, err)
		}
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
	c := numbering.Default()
	c.Overflow = numbering.Refuse
	if got, err := c.Render("INC", 9999999); err != nil || got != "INC9999999" {
		t.Errorf("Render(9999999) = %q, %v", got, err)
	}
	if _, err := c.Render("INC", 10000000); !errors.Is(err, numbering.ErrNumberSpaceExhausted) {
		t.Errorf("Render(10000000) = %v, want ErrNumberSpaceExhausted", err)
	}
	if _, err := c.SQL("$1", "v"); !errors.Is(err, numbering.ErrTemplatesUnavailable) {
		t.Errorf("SQL with refuse = %v, want ErrTemplatesUnavailable", err)
	}
}

func TestRenderRefusesUnsupportedShapes(t *testing.T) {
	for name, mutate := range map[string]func(*numbering.Compiled){
		"tail":        func(c *numbering.Compiled) { c.Tail = "/X" },
		"head":        func(c *numbering.Compiled) { c.Head = "{prefix}-" },
		"scope parts": func(c *numbering.Compiled) { c.ScopeParts = []string{"{yyyy}"} },
		"width 0":     func(c *numbering.Compiled) { c.Width = 0 },
		"width 13":    func(c *numbering.Compiled) { c.Width = 13 },
		"overflow":    func(c *numbering.Compiled) { c.Overflow = "wrap" },
	} {
		c := numbering.Default()
		mutate(&c)
		if _, err := c.Render("INC", 1); !errors.Is(err, numbering.ErrTemplatesUnavailable) {
			t.Errorf("%s: Render = %v, want ErrTemplatesUnavailable", name, err)
		}
		if _, err := c.SQL("$1", "v"); !errors.Is(err, numbering.ErrTemplatesUnavailable) {
			t.Errorf("%s: SQL = %v, want ErrTemplatesUnavailable", name, err)
		}
	}
}

func TestSQLDefault(t *testing.T) {
	got, err := numbering.Default().SQL("$15::text", "(c.next - 1)")
	if err != nil {
		t.Fatal(err)
	}
	const want = "($15::text || CASE WHEN (c.next - 1) < 10000000 THEN lpad((c.next - 1)::text, 7, '0') ELSE (c.next - 1)::text END)"
	if got != want {
		t.Errorf("SQL =\n%s\nwant\n%s", got, want)
	}
}
