package numbering_test

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
)

// nov11 is 2026-11-11, the instant of the ADR-005 §3.1 examples.
func nov11() time.Time { return time.Date(2026, 11, 11, 9, 30, 0, 0, time.UTC) }

func mustCompile(t *testing.T, template string, overflow numbering.Overflow) numbering.Compiled {
	t.Helper()
	c, err := numbering.Compile(template, overflow)
	if err != nil {
		t.Fatalf("Compile(%q, %s): %v", template, overflow, err)
	}
	return c
}

func wantFormatError(t *testing.T, err error, code, field string, pos int) {
	t.Helper()
	var fe *numbering.FormatError
	if !errors.As(err, &fe) {
		t.Fatalf("err = %v, want a *FormatError %s", err, code)
	}
	if fe.Code != code || fe.Field != field || fe.Position != pos || fe.Status() != 422 {
		t.Errorf("err = %+v (status %d), want code %s, field %s, position %d, status 422", fe, fe.Status(), code, field, pos)
	}
	sentinel := numbering.ErrFormatInvalid
	if code == numbering.CodePrefixRequired {
		sentinel = numbering.ErrPrefixRequired
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(%v, %v) = false", err, sentinel)
	}
}

// The ADR-005 §3.1 examples, prefix VISA, counter value 1. VISA-2026111001
// under {yyyy}{MM}{dd}{seq:2} is 2026-11-10, value 01.
func TestCompileADRExamples(t *testing.T) {
	for _, tc := range []struct {
		template string
		at       time.Time
		n        int64
		want     string
		parts    []string
		datePart string
	}{
		{"{prefix}-{yyyy}{MM}{seq:4}", nov11(), 1, "VISA-2026110001", []string{"{yyyy}", "{MM}"}, "202611"},
		{"{prefix}-{yyyy}{MM}{dd}{seq:2}", time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC), 1, "VISA-2026111001", []string{"{yyyy}", "{MM}", "{dd}"}, "20261110"},
	} {
		c := mustCompile(t, tc.template, numbering.Refuse)
		if got, err := c.RenderAt("VISA", tc.at, tc.n); err != nil || got != tc.want {
			t.Errorf("%s: RenderAt = %q, %v; want %q", tc.template, got, err, tc.want)
		}
		if !slices.Equal(c.ScopeParts, tc.parts) {
			t.Errorf("%s: ScopeParts = %v, want %v", tc.template, c.ScopeParts, tc.parts)
		}
		if got, err := c.DatePart(tc.at); err != nil || got != tc.datePart {
			t.Errorf("%s: DatePart = %q, %v; want %q", tc.template, got, err, tc.datePart)
		}
		if c.Epoch != numbering.EpochUnassigned || c.TimeZone != "" {
			t.Errorf("%s: Epoch %d, TimeZone %q; want unassigned and the tenant's zone", tc.template, c.Epoch, c.TimeZone)
		}
		// Under grow a date part touches the growing {seq:N}: ambiguous.
		if _, err := numbering.Compile(tc.template, numbering.Grow); !errors.Is(err, numbering.ErrFormatInvalid) {
			t.Errorf("%s with grow: %v, want number_format_invalid", tc.template, err)
		}
	}

	// The field example is S19: refused at the {field:…} token.
	_, err := numbering.Compile("{prefix}/{field:branch}/{yy}/{seq:2}/{MM}", numbering.Refuse)
	wantFormatError(t, err, numbering.CodeInvalid, "template", 9)

	// The three refused examples.
	for template, pos := range map[string]int{
		"{yyyy}{MM}{seq:4}":             0,
		"{prefix}{field:branch}{seq:3}": 8,
		"{prefix}X{seq:4}":              8,
	} {
		_, err := numbering.Compile(template, numbering.Refuse)
		wantFormatError(t, err, numbering.CodePrefixRequired, "template", pos)
	}
}

func TestCompileAccepts(t *testing.T) {
	for _, tc := range []struct {
		template string
		overflow numbering.Overflow
		n        int64
		want     string
		head     string
		width    int
		tail     string
	}{
		{"{prefix}{seq:7}", numbering.Grow, 12345678, "INC12345678", "{prefix}", 7, ""},
		{"{prefix}{seq:7}", numbering.Refuse, 42, "INC0000042", "{prefix}", 7, ""},
		{"{prefix}{seq:1}", numbering.Grow, 7, "INC7", "{prefix}", 1, ""},
		{"{prefix}{seq:12}", numbering.Refuse, 1, "INC000000000001", "{prefix}", 12, ""},
		{"{prefix}-{seq:5}", numbering.Grow, 1234567, "INC-1234567", "{prefix}-", 5, ""},
		{"{prefix}{seq:4}/X", numbering.Grow, 12345, "INC12345/X", "{prefix}", 4, "/X"},
		{"{prefix}_{yy}.{seq:3}-{MM}", numbering.Grow, 1000, "INC_26.1000-11", "{prefix}_{yy}.", 3, "-{MM}"},
		{"{prefix}2{seq:4}", numbering.Refuse, 5, "INC20005", "{prefix}2", 4, ""},
		{"{prefix}{yy}{seq:3}", numbering.Refuse, 5, "INC26005", "{prefix}{yy}", 3, ""},
		{"{prefix}-ABCDEFGHIJKLMNO{seq:3}", numbering.Refuse, 5, "INC-ABCDEFGHIJKLMNO005", "{prefix}-ABCDEFGHIJKLMNO", 3, ""},
	} {
		c := mustCompile(t, tc.template, tc.overflow)
		if c.Head != tc.head || c.Width != tc.width || c.Tail != tc.tail || c.Overflow != tc.overflow {
			t.Errorf("%s: compiled %+v, want head %q width %d tail %q", tc.template, c, tc.head, tc.width, tc.tail)
		}
		if got, err := c.RenderAt("INC", nov11(), tc.n); err != nil || got != tc.want {
			t.Errorf("%s: RenderAt(%d) = %q, %v; want %q", tc.template, tc.n, got, err, tc.want)
		}
	}
}

func TestCompileRefuses(t *testing.T) {
	// 6 + 1 + 2 + 16 + 2 + 16 + 2 + 16 + 4 = 65 characters.
	long := "{prefix}-{yy}" + strings.Repeat("A", 16) + "{MM}" + strings.Repeat("B", 16) + "{dd}" + strings.Repeat("C", 16) + "{seq:4}"
	for _, tc := range []struct {
		template string
		overflow numbering.Overflow
		code     string
		field    string
		pos      int
	}{
		{"", numbering.Grow, numbering.CodePrefixRequired, "template", 0},
		{"{seq:4}", numbering.Grow, numbering.CodePrefixRequired, "template", 0},
		{"-{prefix}{seq:4}", numbering.Grow, numbering.CodePrefixRequired, "template", 0},
		{"{prefix}{prefix}{seq:4}", numbering.Grow, numbering.CodePrefixRequired, "template", 8},
		{"{prefix}AB{seq:4}", numbering.Grow, numbering.CodePrefixRequired, "template", 8},
		{"{prefix}{seq:7}", "", numbering.CodeInvalid, "overflow", -1},
		{"{prefix}{seq:7}", "wrap", numbering.CodeInvalid, "overflow", -1},
		{"{prefix}", numbering.Grow, numbering.CodeInvalid, "template", 8},
		{"{prefix}-{seq:3}-{seq:3}", numbering.Refuse, numbering.CodeInvalid, "template", 17},
		{"{prefix}{seq:0}", numbering.Grow, numbering.CodeInvalid, "template", 8},
		{"{prefix}{seq:13}", numbering.Grow, numbering.CodeInvalid, "template", 8},
		{"{prefix}{seq:07}", numbering.Grow, numbering.CodeInvalid, "template", 8},
		{"{prefix}{seq:+7}", numbering.Grow, numbering.CodeInvalid, "template", 8},
		{"{prefix}{seq:}", numbering.Grow, numbering.CodeInvalid, "template", 8},
		{"{prefix}{seq:4", numbering.Grow, numbering.CodeInvalid, "template", 8},
		{"{prefix}-{seq:4}}", numbering.Grow, numbering.CodeInvalid, "template", 16},
		{"{prefix}-{hh}{seq:4}", numbering.Grow, numbering.CodeInvalid, "template", 9},
		{"{prefix}-{YYYY}{seq:4}", numbering.Grow, numbering.CodeInvalid, "template", 9},
		{"{prefix}-{}{seq:4}", numbering.Grow, numbering.CodeInvalid, "template", 9},
		{"{prefix}-{field:x}{seq:4}", numbering.Refuse, numbering.CodeInvalid, "template", 9},
		{"{prefix}-{seq:4}{prefix}", numbering.Grow, numbering.CodeInvalid, "template", 16},
		{"{prefix}-x{seq:4}", numbering.Grow, numbering.CodeInvalid, "template", 9},
		{"{prefix}- {seq:4}", numbering.Grow, numbering.CodeInvalid, "template", 9},
		{"{prefix}-É{seq:4}", numbering.Grow, numbering.CodeInvalid, "template", 9},
		{"{prefix}-" + strings.Repeat("A", 17) + "{seq:4}", numbering.Grow, numbering.CodeInvalid, "template", 8 + 16}, // the run starts at the -
		{long, numbering.Refuse, numbering.CodeInvalid, "template", -1},
		// Grow: {seq:N} touches a digit (ADR-005 §3.5).
		{"{prefix}{yy}{seq:3}", numbering.Grow, numbering.CodeInvalid, "overflow", 12},
		{"{prefix}2{seq:4}", numbering.Grow, numbering.CodeInvalid, "overflow", 9},
		{"{prefix}-{seq:4}{MM}", numbering.Grow, numbering.CodeInvalid, "overflow", 9},
		{"{prefix}-{seq:4}1", numbering.Grow, numbering.CodeInvalid, "overflow", 9},
	} {
		_, err := numbering.Compile(tc.template, tc.overflow)
		t.Run(tc.template+"/"+string(tc.overflow), func(t *testing.T) {
			wantFormatError(t, err, tc.code, tc.field, tc.pos)
		})
	}
}

// The 64-character bound counts the longest prefix and, under grow, a
// counter of 19 digits.
func TestCompileRenderedLengthBound(t *testing.T) {
	// 6 + 1 + 2 + 16 + 2 + 15 + 1 + 12 = 55 under refuse; 62 under grow.
	ok := "{prefix}-{yy}" + strings.Repeat("A", 16) + "{MM}" + strings.Repeat("B", 15) + "-{seq:12}"
	mustCompile(t, ok, numbering.Refuse)
	mustCompile(t, ok, numbering.Grow)
	// 6 + 1 + 2 + 16 + 2 + 15 + 1 + 4 + 1 + 15 = 63 under refuse; 78 under grow.
	edge := "{prefix}-{yy}" + strings.Repeat("A", 16) + "{MM}" + strings.Repeat("B", 15) + "-{seq:4}-" + strings.Repeat("C", 15)
	mustCompile(t, edge, numbering.Refuse)
	if _, err := numbering.Compile(edge, numbering.Grow); !errors.Is(err, numbering.ErrFormatInvalid) {
		t.Errorf("Compile(%q, grow) = %v, want number_format_invalid", edge, err)
	}
}

// Security (TICK-666 review, requirement 1): template text with a quote or
// a semicolon never compiles, and a snapshot edited to hold one renders
// neither in Go nor in SQL.
func TestCompileRefusesSQLMetacharacters(t *testing.T) {
	for _, template := range []string{
		"{prefix}-'{seq:4}",
		"{prefix}-;{seq:4}",
		"{prefix}-{seq:4}';DROP TABLE records;--",
		`{prefix}-\{seq:4}`,
		"{prefix}-{seq:4}$1",
	} {
		_, err := numbering.Compile(template, numbering.Refuse)
		wantFormatError(t, err, numbering.CodeInvalid, "template", strings.IndexAny(template, `';\$`))
	}
	edited := mustCompile(t, "{prefix}-{seq:5}/X", numbering.Grow)
	edited.Tail = "';DROP TABLE records;--"
	if _, _, err := edited.SQL("$15::text", "v", "", nil); !errors.Is(err, numbering.ErrInvalidCompiled) {
		t.Errorf("SQL of an edited tail = %v, want ErrInvalidCompiled", err)
	}
	if _, err := edited.Render("INC", 1); !errors.Is(err, numbering.ErrInvalidCompiled) {
		t.Errorf("Render of an edited tail = %v, want ErrInvalidCompiled", err)
	}
}

// Security (TICK-666 review, requirement 1): literal text reaches SQL as
// bound parameters, never as statement text.
func TestSQLBindsLiterals(t *testing.T) {
	c := mustCompile(t, "{prefix}-{seq:5}/X.Y", numbering.Grow)
	got, args, err := c.SQL("$15::text", "(c.next - 1)", "", []any{"keep"})
	if err != nil {
		t.Fatal(err)
	}
	const want = "($15::text || $2::text || CASE WHEN (c.next - 1) < 100000 THEN lpad((c.next - 1)::text, 5, '0') ELSE (c.next - 1)::text END || $3::text)"
	if got != want {
		t.Errorf("SQL =\n%s\nwant\n%s", got, want)
	}
	if !slices.Equal(args, []any{"keep", "-", "/X.Y"}) {
		t.Errorf("args = %v, want [keep - /X.Y]", args)
	}
	if strings.Contains(got, "X.Y") {
		t.Errorf("SQL %q contains the literal text", got)
	}
	dated := mustCompile(t, "{prefix}-{yy}-{seq:4}", numbering.Grow)
	if _, _, err := dated.SQL("$1", "v", "", nil); !errors.Is(err, numbering.ErrTemplatesUnavailable) {
		t.Errorf("SQL with a date part and no local time = %v, want ErrTemplatesUnavailable", err)
	}
}

// A7 (TICK-674): date parts render in SQL from the local-time expression,
// one to_char per token with its pattern bound; refuse renders NULL past the
// width, so the records.number NOT NULL refuses the INSERT.
func TestSQLRendersDatePartsAndRefuse(t *testing.T) {
	c := mustCompile(t, "{prefix}/{yy}/{seq:2}/{MM}", numbering.Refuse)
	got, args, err := c.SQL("p", "v", "l", nil)
	if err != nil {
		t.Fatal(err)
	}
	const want = "(p || $1::text || to_char(l, $2::text) || $3::text || CASE WHEN v < 100 THEN lpad(v::text, 2, '0') END || $4::text || to_char(l, $5::text))"
	if got != want {
		t.Errorf("SQL =\n%s\nwant\n%s", got, want)
	}
	if !slices.Equal(args, []any{"/", "YY", "/", "/", "MM"}) {
		t.Errorf("args = %v, want [/ YY / / MM]", args)
	}
}

func TestCompileDefaultTimeZone(t *testing.T) {
	c, err := numbering.CompileFormat(numbering.Format{Template: numbering.DefaultTemplate, Overflow: numbering.Grow, TimeZone: "Asia/Kolkata"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Epoch != 0 || c.CompiledSHA256 != numbering.Default().CompiledSHA256 || c.TimeZone != "Asia/Kolkata" {
		t.Errorf("default with a zone = %+v, want epoch 0, the default's hash, Asia/Kolkata", c)
	}
	for _, zone := range []string{"UTC", "America/New_York", "Pacific/Kiritimati", "Etc/GMT+12"} {
		if _, err := numbering.CompileFormat(numbering.Format{Template: "{prefix}-{yy}-{seq:4}", Overflow: numbering.Grow, TimeZone: zone}); err != nil {
			t.Errorf("time zone %q: %v", zone, err)
		}
	}
	for _, zone := range []string{"Local", "Mars/Olympus", "utc ", "../../etc/passwd", "+05:30"} {
		_, err := numbering.CompileFormat(numbering.Format{Template: "{prefix}-{yy}-{seq:4}", Overflow: numbering.Grow, TimeZone: zone})
		wantFormatError(t, err, numbering.CodeInvalid, "time_zone", -1)
	}
}

// CompiledSHA256 is pinned: the epoch registry matches on it, so it must not
// move between runs, builds or Go versions.
func TestCompiledSHA256Golden(t *testing.T) {
	for _, tc := range []struct {
		template string
		overflow numbering.Overflow
		want     string
	}{
		{numbering.DefaultTemplate, numbering.Grow, "0682b49d0d760e30081173b7e23991af8a2442515f51f5ba3a401ec820cde398"},
		{"{prefix}-{yyyy}{MM}{seq:4}", numbering.Refuse, "8e9932d9c3c78ccd2cae0265544b772fbcca21b26f2a8aab72bc28f458272c4c"},
		{"{prefix}-{seq:5}/X", numbering.Grow, "f9874e9f878170cec06433d08b272bb451eeb447ecf63e82a0f29dc1146dde76"},
	} {
		c := mustCompile(t, tc.template, tc.overflow)
		if c.CompiledSHA256 != tc.want {
			t.Errorf("%s/%s: CompiledSHA256 = %s, want %s", tc.template, tc.overflow, c.CompiledSHA256, tc.want)
		}
	}
}

func TestScopeKeyAt(t *testing.T) {
	if k, err := numbering.Default().ScopeKeyAt(""); err != nil || k != "" {
		t.Errorf("default ScopeKeyAt(\"\") = %q, %v; want \"\"", k, err)
	}
	if _, err := numbering.Default().ScopeKeyAt("2026"); err == nil {
		t.Error("default ScopeKeyAt(2026): no error, want one (no date part)")
	}
	c := mustCompile(t, "{prefix}-{yyyy}{MM}{seq:4}", numbering.Refuse)
	if _, err := c.ScopeKeyAt("202611"); err == nil {
		t.Error("ScopeKeyAt before an epoch is taken: no error, want one")
	}
	c.Epoch = 0
	if _, err := c.ScopeKeyAt("202611"); !errors.Is(err, numbering.ErrInvalidCompiled) {
		t.Errorf("non-default at epoch 0: %v, want ErrInvalidCompiled", err)
	}
	c.Epoch = 1
	if k, err := c.ScopeKeyAt("202611"); err != nil || k != "e1|d:202611" {
		t.Errorf("ScopeKeyAt(202611) = %q, %v; want e1|d:202611", k, err)
	}
	for _, bad := range []string{"", "20261", "2026111", "2026-1", "20261'", "202611;"} {
		if _, err := c.ScopeKeyAt(bad); err == nil {
			t.Errorf("ScopeKeyAt(%q): no error, want one", bad)
		}
	}
	plain := mustCompile(t, "{prefix}-{seq:4}", numbering.Grow)
	plain.Epoch = 7
	if k, err := plain.ScopeKeyAt(""); err != nil || k != "e7" {
		t.Errorf("no-date ScopeKeyAt = %q, %v; want e7", k, err)
	}
}

// Security (TICK-666 review, requirement 2): a scope key is at most 128
// characters and holds only the epoch and date digits, never tenant text,
// whatever the template, epoch and instant.
func TestScopeKeyShapeAndBound(t *testing.T) {
	shape := regexp.MustCompile(`^e[1-9][0-9]*(\|d:[0-9]+)?$`)
	templates := []string{
		"{prefix}-{seq:4}",
		"{prefix}-{yyyy}{MM}{dd}{seq:2}",
		"{prefix}-{yyyy}-{yyyy}-{yyyy}-{yyyy}-{yyyy}-{seq:1}",
		"{prefix}-{yy}{MM}{dd}-{yy}{MM}{dd}-{yy}{MM}{dd}-{seq:1}",
	}
	for _, template := range templates {
		c := mustCompile(t, template, numbering.Refuse)
		for _, epoch := range []int{1, 42, 1<<31 - 1, int(^uint(0) >> 1)} {
			c.Epoch = epoch
			for _, at := range []time.Time{nov11(), time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)} {
				part, err := c.DatePart(at)
				if err != nil {
					t.Fatalf("%s: DatePart(%v): %v", template, at, err)
				}
				k, err := c.ScopeKeyAt(part)
				if err != nil {
					t.Fatalf("%s: ScopeKeyAt(%q): %v", template, part, err)
				}
				if len(k) > numbering.MaxScopeKey || !shape.MatchString(k) {
					t.Errorf("%s epoch %d: scope key %q (%d chars): want %s and at most %d", template, epoch, k, len(k), shape, numbering.MaxScopeKey)
				}
			}
		}
	}
}

func TestRenderDateParts(t *testing.T) {
	c := mustCompile(t, "{prefix}/{yy}/{MM}/{dd}/{seq:3}", numbering.Refuse)
	if _, err := c.Render("INC", 1); err == nil {
		t.Error("Render with date parts: no error, want one (RenderAt)")
	}
	kolkata, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		t.Fatal(err)
	}
	// 2026-12-31 20:00 UTC is 2027-01-01 01:30 in Kolkata: the caller's zone decides.
	at := time.Date(2026, 12, 31, 20, 0, 0, 0, time.UTC)
	if got, _ := c.RenderAt("INC", at, 9); got != "INC/26/12/31/009" {
		t.Errorf("RenderAt UTC = %q", got)
	}
	if got, _ := c.RenderAt("INC", at.In(kolkata), 9); got != "INC/27/01/01/009" {
		t.Errorf("RenderAt Kolkata = %q", got)
	}
	y4 := mustCompile(t, "{prefix}-{yyyy}-{seq:3}", numbering.Refuse)
	if _, err := y4.RenderAt("INC", time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC), 1); err == nil {
		t.Error("RenderAt year 10000: no error, want one")
	}
	if _, err := y4.DatePart(time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Error("DatePart year 10000: no error, want one")
	}
	if _, err := y4.RenderAt("INC", nov11(), 0); err == nil {
		t.Error("RenderAt(0): no error, want one")
	}
}
