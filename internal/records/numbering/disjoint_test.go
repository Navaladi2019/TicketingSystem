package numbering_test

import (
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
)

// The ADR-005 §3.3 examples and the edges of the rule, prefix INC.
func TestDisjointTable(t *testing.T) {
	epoch0 := numbering.Default()
	for _, tc := range []struct {
		a, b string // "template/overflow"; "" is epoch 0
		want bool
	}{
		// The three ADR examples.
		{"", "{prefix}-{yyyy}{MM}{seq:4}/refuse", true},
		{"", "{prefix}{yy}{seq:3}/refuse", false},
		{"{prefix}{yy}{seq:3}/refuse", "{prefix}2{seq:4}/refuse", false},
		// Heads that differ before either ends.
		{"{prefix}-{seq:4}/grow", "{prefix}/{seq:4}/grow", true},
		{"{prefix}-A{seq:4}/grow", "{prefix}-B{seq:4}/grow", true},
		// One head a prefix of the other, continuing with a non-digit.
		{"{prefix}-{seq:4}/grow", "{prefix}-X{seq:4}/grow", true},
		{"", "{prefix}_{seq:7}/grow", true},
		{"", "{prefix}.{yy}{seq:5}/refuse", true},
		// … or with a digit.
		{"", "{prefix}0{seq:6}/refuse", false},
		{"{prefix}-{seq:4}/grow", "{prefix}-2{seq:4}/refuse", false},
		// Equal heads, even with different tails or widths.
		{"{prefix}-{seq:4}/X/grow", "{prefix}-{seq:4}/Y/grow", false},
		{"", "{prefix}{seq:5}/grow", false},
		{"", "{prefix}{seq:7}/refuse", false},
		{"", "", false},
		{"{prefix}-{yyyy}{seq:4}/refuse", "{prefix}-{yy}{MM}{seq:6}/refuse", false},
	} {
		a, b := epoch0, epoch0
		if tc.a != "" {
			a = compileSpec(t, tc.a)
		}
		if tc.b != "" {
			b = compileSpec(t, tc.b)
		}
		if got := numbering.Disjoint(a, b, "INC"); got != tc.want {
			t.Errorf("Disjoint(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
		if got := numbering.Disjoint(b, a, "INC"); got != tc.want {
			t.Errorf("Disjoint(%q, %q) = %v, want %v (symmetry)", tc.b, tc.a, got, tc.want)
		}
	}
}

func TestLiteralHead(t *testing.T) {
	for spec, want := range map[string]string{
		"{prefix}{seq:7}/grow":              "VISA",
		"{prefix}-{yyyy}{MM}{seq:4}/refuse": "VISA-",
		"{prefix}-X/{seq:4}/grow":           "VISA-X/",
		"{prefix}2{yy}-{seq:4}/grow":        "VISA2",
	} {
		if got, err := compileSpec(t, spec).LiteralHead("VISA"); err != nil || got != want {
			t.Errorf("LiteralHead(%s) = %q, %v; want %q", spec, got, err, want)
		}
	}
}

// compileSpec compiles "template/overflow".
func compileSpec(t testing.TB, spec string) numbering.Compiled {
	t.Helper()
	i := strings.LastIndex(spec, "/")
	c, err := numbering.Compile(spec[:i], numbering.Overflow(spec[i+1:]))
	if err != nil {
		t.Fatalf("Compile(%s): %v", spec, err)
	}
	return c
}

// propertyPrefix is the type prefix of the property test.
const propertyPrefix = "INC"

// instants are the allocation instants renderings are sampled at.
func instants() []time.Time {
	return []time.Time{
		time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2020, 2, 26, 0, 0, 0, 0, time.UTC),
		time.Date(2002, 6, 10, 0, 0, 0, 0, time.UTC),
	}
}

// counters are the counter values renderings are sampled at: the first
// values, the widths' edges and values past them (grow).
func counters() []int64 {
	return []int64{1, 2, 5, 9, 10, 11, 26, 99, 100, 101, 260, 999, 1000, 1001, 2026, 6001, 9999, 10000, 26001, 123456}
}

// item is one piece of a pattern: literal text, or a run of digits of
// length min to max (max < 0: no bound).
type item struct {
	lit      string
	min, max int
}

// pattern is every string a template could render, with every date part
// and {seq:N} free over its digits: a superset of its numbers, so "no
// rendering of a matches b" is at least as strict as the property. (A
// hand-written matcher, not a regexp: the property test runs under -race.)
type pattern []item

func newPattern(c numbering.Compiled) pattern {
	var p pattern
	lit := func(s string) {
		if n := len(p); n > 0 && p[n-1].max == 0 && p[n-1].lit != "" {
			p[n-1].lit += s
			return
		}
		p = append(p, item{lit: s})
	}
	write := func(text string) {
		for text != "" {
			switch {
			case strings.HasPrefix(text, "{prefix}"):
				lit(propertyPrefix)
				text = text[len("{prefix}"):]
			case strings.HasPrefix(text, "{yyyy}"):
				p = append(p, item{min: 4, max: 4})
				text = text[len("{yyyy}"):]
			case strings.HasPrefix(text, "{yy}"), strings.HasPrefix(text, "{MM}"), strings.HasPrefix(text, "{dd}"):
				p = append(p, item{min: 2, max: 2})
				text = text[4:]
			default:
				lit(text[:1])
				text = text[1:]
			}
		}
	}
	write(c.Head)
	if c.Overflow == numbering.Grow {
		p = append(p, item{min: c.Width, max: -1})
	} else {
		p = append(p, item{min: c.Width, max: c.Width})
	}
	write(c.Tail)
	return p
}

func (p pattern) match(s string) bool {
	if len(p) == 0 {
		return s == ""
	}
	it := p[0]
	if it.lit != "" {
		return strings.HasPrefix(s, it.lit) && p[1:].match(s[len(it.lit):])
	}
	n := 0
	for n < len(s) && s[n] >= '0' && s[n] <= '9' {
		n++
	}
	if it.max >= 0 && n > it.max {
		n = it.max
	}
	for k := it.min; k <= n; k++ {
		if p[1:].match(s[k:]) {
			return true
		}
	}
	return false
}

// sampled is a compiled template with its renderings and matcher.
type sampled struct {
	spec    string
	c       numbering.Compiled
	renders []string
	re      pattern
}

func sample(t testing.TB, spec string, c numbering.Compiled) sampled {
	t.Helper()
	s := sampled{spec: spec, c: c, re: newPattern(c)}
	for _, at := range instants() {
		for _, n := range counters() {
			r, err := c.RenderAt(propertyPrefix, at, n)
			if err != nil {
				continue // past a refuse width
			}
			if !s.re.match(r) {
				t.Fatalf("%s: rendering %q does not match its own pattern %v", spec, r, s.re)
			}
			s.renders = append(s.renders, r)
		}
	}
	return s
}

// clash returns a rendering of one that the other could render, or "".
func clash(a, b sampled) string {
	for _, r := range a.renders {
		if b.re.match(r) {
			return r
		}
	}
	for _, r := range b.renders {
		if a.re.match(r) {
			return r
		}
	}
	return ""
}

// literals are the literal runs the generator draws from: separators,
// digits that collide with date parts and counters, and letters.
func literals() []string {
	return []string{"-", "/", "_", ".", "0", "1", "2", "26", "20", "-2", "X", "A1", "-X", "2X"}
}

func variables() []string { return []string{"{yyyy}", "{yy}", "{MM}", "{dd}"} }

// genTemplate draws a template of up to three pieces before {seq:N} and up
// to two after it, and an overflow. Many draws do not compile; the caller
// skips those.
func genTemplate(r *rand.Rand) (string, numbering.Overflow) {
	lits, vars := literals(), variables()
	piece := func() string {
		if r.IntN(2) == 0 {
			return lits[r.IntN(len(lits))]
		}
		return vars[r.IntN(len(vars))]
	}
	var b strings.Builder
	b.WriteString("{prefix}")
	for range r.IntN(4) {
		b.WriteString(piece())
	}
	b.WriteString("{seq:" + strconv.Itoa(1+r.IntN(5)) + "}")
	for range r.IntN(3) {
		b.WriteString(piece())
	}
	if r.IntN(2) == 0 {
		return b.String(), numbering.Grow
	}
	return b.String(), numbering.Refuse
}

// TestDisjointProperty is the TA-1 property test: over generated template
// pairs (epoch 0 included), whenever Disjoint says two epochs are disjoint,
// no sampled rendering of one is a string the other could render. The seed
// is fixed, so CI runs the same corpus every time; FuzzDisjoint explores
// beyond it.
func TestDisjointProperty(t *testing.T) {
	r := rand.New(rand.NewPCG(667, 3))
	pool := []sampled{sample(t, "epoch 0", numbering.Default())}
	seen := map[string]bool{}
	for len(pool) < 300 {
		template, overflow := genTemplate(r)
		spec := template + "/" + string(overflow)
		c, err := numbering.Compile(template, overflow)
		if err != nil || seen[spec] {
			continue
		}
		seen[spec] = true
		pool = append(pool, sample(t, spec, c))
	}
	var disjoint, clashing int
	for i := range pool {
		for j := i + 1; j < len(pool); j++ {
			a, b := pool[i], pool[j]
			hit := clash(a, b)
			if numbering.Disjoint(a.c, b.c, propertyPrefix) {
				disjoint++
				if hit != "" {
					t.Errorf("Disjoint(%s, %s) = true, but both can render %q", a.spec, b.spec, hit)
				}
			} else if hit != "" {
				clashing++
			}
		}
	}
	// The corpus must exercise both answers, or the test proves nothing.
	if disjoint < 1000 || clashing < 1000 {
		t.Errorf("corpus too weak: %d disjoint pairs, %d refused pairs with a real clash; want at least 1000 of each", disjoint, clashing)
	}
	t.Logf("%d templates, %d disjoint pairs, %d refused pairs with a clash", len(pool), disjoint, clashing)
}

// FuzzDisjoint is the same property on fuzzed template pairs. `go test` runs
// the seed corpus below (CI's fixed corpus); `go test -fuzz FuzzDisjoint`
// explores further.
func FuzzDisjoint(f *testing.F) {
	for _, seed := range [][4]string{
		{numbering.DefaultTemplate, "grow", "{prefix}-{yyyy}{MM}{seq:4}", "refuse"},
		{numbering.DefaultTemplate, "grow", "{prefix}{yy}{seq:3}", "refuse"},
		{"{prefix}{yy}{seq:3}", "refuse", "{prefix}2{seq:4}", "refuse"},
		{"{prefix}-{seq:4}", "grow", "{prefix}-X{seq:4}", "grow"},
		{"{prefix}_{seq:2}/2", "grow", "{prefix}_2{seq:3}", "refuse"},
		{"{prefix}.{dd}{seq:1}", "refuse", "{prefix}.{MM}{dd}{seq:2}", "refuse"},
		{"{prefix}0{seq:3}", "refuse", "{prefix}{seq:4}", "grow"},
		{"{prefix}/{yyyy}/{seq:5}", "grow", "{prefix}/{yy}/{seq:5}", "grow"},
	} {
		f.Add(seed[0], seed[1], seed[2], seed[3])
	}
	f.Fuzz(func(t *testing.T, ta, oa, tb, ob string) {
		a, err := numbering.Compile(ta, numbering.Overflow(oa))
		if err != nil {
			return
		}
		b, err := numbering.Compile(tb, numbering.Overflow(ob))
		if err != nil {
			return
		}
		if !numbering.Disjoint(a, b, propertyPrefix) {
			return
		}
		if hit := clash(sample(t, ta, a), sample(t, tb, b)); hit != "" {
			t.Errorf("Disjoint(%s/%s, %s/%s) = true, but both can render %q", ta, oa, tb, ob, hit)
		}
	})
}
