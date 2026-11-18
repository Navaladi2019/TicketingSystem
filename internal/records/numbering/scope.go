package numbering

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Scope is what the allocation statement needs of a format to compute the
// counter's scope key and period_end in PostgreSQL, from now() in the
// numbering time zone (ADR-005 A7, §4.1). Go passes it; SQL derives every
// date from the transaction's own now(), so the date in the number, the
// scope key and created_at come from one instant, midnight included.
type Scope struct {
	// KeyPrefix is the scope key before its date part: "" for the default
	// (epoch 0), e<epoch> for a format with no date part, and e<epoch>|d:
	// for one with date parts, which SQL completes with the date digits.
	KeyPrefix string
	// Patterns are the to_char patterns of ScopeParts, in order. The date
	// part is their outputs concatenated. Empty for a format with no date
	// part. They are fixed constants (sqlPattern), never template text.
	Patterns []string
	// PeriodUnit is the date_trunc unit of the period the scope key names
	// ("day", "month" or "year"): period_end is the start of the next one,
	// in the numbering time zone. "" when the key names no closed period,
	// so period_end is NULL and the row is never purged: no date part, or
	// date parts that recur ({MM} with no year, {dd} with no month), whose
	// key a later period reuses and whose counter must therefore continue
	// (a purged and restarted counter would hand out a number again).
	PeriodUnit string
	// TimeZone is the format's own IANA zone; "" means the tenant's
	// time_zone setting (ADR-005 §3.6).
	TimeZone string
}

// Dated reports whether the scope key has a date part, so the statement
// needs the numbering time zone.
func (s Scope) Dated() bool { return len(s.Patterns) > 0 }

// Scope is c's allocation scope. It refuses a format with no epoch yet, or
// one that is not what Compile makes of its own template.
func (c Compiled) Scope() (Scope, error) {
	ct, err := c.check()
	if err != nil {
		return Scope{}, err
	}
	s := Scope{TimeZone: c.TimeZone}
	switch c.Epoch {
	case 0:
		// check guarantees epoch 0 is the default, which has no date part.
		return s, nil
	case EpochUnassigned:
		return Scope{}, errors.New("numbering: the format has no epoch yet")
	}
	s.KeyPrefix = "e" + strconv.Itoa(c.Epoch)
	var year, month, day bool
	for _, t := range ct.tokens {
		if t.kind != tokDate {
			continue
		}
		s.Patterns = append(s.Patterns, sqlPattern(t))
		switch t.text {
		case "{yyyy}", "{yy}":
			year = true
		case "{MM}":
			month = true
		case "{dd}":
			day = true
		}
	}
	if len(s.Patterns) > 0 {
		s.KeyPrefix += "|d:"
	}
	switch {
	case year && month && day:
		s.PeriodUnit = "day"
	case year && month && !day:
		s.PeriodUnit = "month"
	case year && !month && !day:
		s.PeriodUnit = "year"
	}
	return s, nil
}

// DatePartOf returns the date part of key, a scope key PostgreSQL built for
// c (Scope), after checking that key is exactly ScopeKeyAt of that date
// part: the allocation's result is checked against the create's own format
// before it becomes a number (ADR-005 A2).
func (c Compiled) DatePartOf(key string) (string, error) {
	s, err := c.Scope()
	if err != nil {
		return "", err
	}
	part, ok := strings.CutPrefix(key, s.KeyPrefix)
	if !ok {
		return "", fmt.Errorf("numbering: scope key %q is not one of epoch %d", key, c.Epoch)
	}
	want, err := c.ScopeKeyAt(part)
	if err != nil {
		return "", err
	}
	if want != key {
		return "", fmt.Errorf("numbering: scope key %q is not one of epoch %d", key, c.Epoch)
	}
	return part, nil
}

// checkDatePart reports whether datePart is the date part of ct: its date
// tokens' digits, concatenated.
func checkDatePart(ct compiled, datePart string) error {
	width := 0
	for _, t := range ct.tokens {
		if t.kind == tokDate {
			width += t.width
		}
	}
	if len(datePart) != width || strings.IndexFunc(datePart, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return fmt.Errorf("numbering: the date part must be %d digits, got %d characters", width, len(datePart))
	}
	return nil
}

// sqlPattern is a date token's PostgreSQL to_char pattern. Each token is
// formatted by a to_char call of its own, so adjacent tokens never merge
// into another pattern ("DD" "DD" would read as "DDD", the day of the year).
func sqlPattern(t token) string {
	switch t.text {
	case "{yyyy}":
		return "YYYY"
	case "{yy}":
		return "YY"
	case "{MM}":
		return "MM"
	}
	return "DD" // {dd}: datePart admits no other date token
}
