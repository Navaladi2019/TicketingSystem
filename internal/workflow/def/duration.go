package def

import "errors"

// Wait bounds in seconds, the workerguard.Directive WaitSeconds range (D-3-15).
const (
	MinWaitSeconds = 1
	MaxWaitSeconds = 30 * 24 * 60 * 60 // 2592000
)

// ErrDuration is returned by ParseDuration for any refused duration. It never
// echoes the input.
var ErrDuration = errors.New("workflow definition: duration must be PnDTnHnMnS within 1s..30d")

// ParseDuration parses an ISO 8601 duration restricted to PnDTnHnMnS: the
// designators D, H, M (minutes, after T) and S, each at most once, in that
// order, with non-negative integer values. Y, months, W, fractions and signs
// are refused. The total must be within [MinWaitSeconds, MaxWaitSeconds].
func ParseDuration(s string) (int32, error) {
	if len(s) < 3 || s[0] != 'P' {
		return 0, ErrDuration
	}
	const (
		inDate = iota
		inTime
	)
	part := inDate
	// order of the designator last seen: D=1, H=2, M=3, S=4
	last := 0
	var total int64
	units := 0
	i := 1
	for i < len(s) {
		if s[i] == 'T' {
			if part == inTime {
				return 0, ErrDuration
			}
			part = inTime
			i++
			if i == len(s) { // "PT" or "P1DT" with no time part
				return 0, ErrDuration
			}
			continue
		}
		start := i
		var n int64
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			if i-start == 7 { // schema.json allows at most 7 digits per designator
				return 0, ErrDuration
			}
			n = n*10 + int64(s[i]-'0')
			if n > MaxWaitSeconds { // early stop keeps the arithmetic small
				return 0, ErrDuration
			}
			i++
		}
		if i == start || i == len(s) {
			return 0, ErrDuration
		}
		var order int
		var mult int64
		switch {
		case s[i] == 'D' && part == inDate:
			order, mult = 1, 86400
		case s[i] == 'H' && part == inTime:
			order, mult = 2, 3600
		case s[i] == 'M' && part == inTime:
			order, mult = 3, 60
		case s[i] == 'S' && part == inTime:
			order, mult = 4, 1
		default:
			return 0, ErrDuration
		}
		if order <= last {
			return 0, ErrDuration
		}
		last = order
		total += n * mult
		if total > MaxWaitSeconds {
			return 0, ErrDuration
		}
		units++
		i++
	}
	if units == 0 || total < MinWaitSeconds {
		return 0, ErrDuration
	}
	return int32(total), nil
}
