package numbering_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
)

// withEpoch is c as the publish leaves it: with the type's epoch.
func withEpoch(c numbering.Compiled, epoch int) numbering.Compiled {
	c.Epoch = epoch
	return c
}

// TICK-674 (ADR-005 A7, §4.1, §4.3): what the allocation statement gets of
// a format. A period ends only where the key names a closed period; a key a
// later period reuses ({MM} with no year, {dd} with no month) is never
// purged, so its counter continues instead of restarting on a number
// already issued.
func TestScope(t *testing.T) {
	for _, tc := range []struct {
		template, tz string
		key          string
		patterns     []string
		unit         string
	}{
		{"{prefix}-{yyyy}{MM}{seq:4}", "", "e1|d:", []string{"YYYY", "MM"}, "month"},
		{"{prefix}-{yyyy}{MM}{dd}{seq:2}", "Asia/Kolkata", "e1|d:", []string{"YYYY", "MM", "DD"}, "day"},
		{"{prefix}/{yy}/{seq:2}/{MM}", "", "e1|d:", []string{"YY", "MM"}, "month"},
		{"{prefix}-{yy}-{seq:4}", "", "e1|d:", []string{"YY"}, "year"},
		{"{prefix}-{MM}-{seq:4}", "", "e1|d:", []string{"MM"}, ""},
		{"{prefix}-{dd}-{seq:4}", "", "e1|d:", []string{"DD"}, ""},
		{"{prefix}-{yyyy}{dd}-{seq:4}", "", "e1|d:", []string{"YYYY", "DD"}, ""},
		{"{prefix}-{dd}{dd}-{seq:4}", "", "e1|d:", []string{"DD", "DD"}, ""},
		{"{prefix}-{seq:4}", "", "e1", nil, ""},
	} {
		c, err := numbering.CompileFormat(numbering.Format{Template: tc.template, Overflow: numbering.Refuse, TimeZone: tc.tz})
		if err != nil {
			t.Fatalf("%s: %v", tc.template, err)
		}
		s, err := withEpoch(c, 1).Scope()
		if err != nil {
			t.Fatalf("%s: Scope: %v", tc.template, err)
		}
		if s.KeyPrefix != tc.key || !slices.Equal(s.Patterns, tc.patterns) || s.PeriodUnit != tc.unit || s.TimeZone != tc.tz ||
			s.Dated() != (len(tc.patterns) > 0) {
			t.Errorf("%s: Scope = %+v, want key %q, patterns %v, unit %q, zone %q", tc.template, s, tc.key, tc.patterns, tc.unit, tc.tz)
		}
	}
	c, err := numbering.Compile("{prefix}-{seq:4}", numbering.Grow)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Scope(); err == nil || !strings.Contains(err.Error(), "no epoch") {
		t.Errorf("Scope of an unassigned epoch = %v, want an error", err)
	}
}

// Security, P6-03 review (comment 01a1d153): a non-default, date-free format
// allocates on its own epoch's counter, never on epoch 0's ”.
func TestScopeOfADateFreeFormatIsItsEpochs(t *testing.T) {
	c, err := numbering.Compile("{prefix}-{seq:5}", numbering.Grow)
	if err != nil {
		t.Fatal(err)
	}
	s, err := withEpoch(c, 3).Scope()
	if err != nil || s.KeyPrefix != "e3" || s.Dated() {
		t.Errorf("Scope = %+v, %v; want key e3, no date part", s, err)
	}
}

func TestDatePartOf(t *testing.T) {
	c, err := numbering.Compile("{prefix}-{yyyy}{MM}{seq:4}", numbering.Refuse)
	if err != nil {
		t.Fatal(err)
	}
	c = withEpoch(c, 2)
	if part, err := c.DatePartOf("e2|d:202611"); err != nil || part != "202611" {
		t.Errorf("DatePartOf = %q, %v; want 202611", part, err)
	}
	for _, bad := range []string{"", "e1|d:202611", "e2|d:20261", "e2|d:2026111", "e2|d:2026x1", "e2", "e2|d:"} {
		if _, err := c.DatePartOf(bad); err == nil {
			t.Errorf("DatePartOf(%q): no error, want one", bad)
		}
	}
	if part, err := numbering.Default().DatePartOf(""); err != nil || part != "" {
		t.Errorf("default DatePartOf('') = %q, %v", part, err)
	}
	if _, err := numbering.Default().DatePartOf("e1"); err == nil {
		t.Error("default DatePartOf(e1): no error, want one")
	}
	plain, err := numbering.Compile("{prefix}-{seq:5}", numbering.Grow)
	if err != nil {
		t.Fatal(err)
	}
	if part, err := withEpoch(plain, 4).DatePartOf("e4"); err != nil || part != "" {
		t.Errorf("DatePartOf(e4) = %q, %v", part, err)
	}
	if _, err := withEpoch(plain, 4).DatePartOf("e4|d:"); err == nil {
		t.Error("DatePartOf(e4|d:) of a date-free format: no error, want one")
	}
	if _, err := plain.DatePartOf("e-1"); err == nil {
		t.Error("DatePartOf with no epoch: no error, want one")
	}
}
