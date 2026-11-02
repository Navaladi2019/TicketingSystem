package def

import (
	"errors"
	"regexp"
	"testing"
)

func TestParseDurationValid(t *testing.T) {
	for in, want := range map[string]int32{
		"PT1S":           1,
		"PT5S":           5,
		"PT1M":           60,
		"PT1H":           3600,
		"P1D":            86400,
		"P1DT1H1M1S":     90061,
		"PT90M":          5400,
		"PT3600S":        3600,
		"P30D":           MaxWaitSeconds,
		"P29DT23H59M60S": MaxWaitSeconds,
		"PT2592000S":     MaxWaitSeconds,
		"P0DT1S":         1,
		"PT0H0M1S":       1,
		"PT001S":         1,
	} {
		got, err := ParseDuration(in)
		if err != nil || got != want {
			t.Errorf("ParseDuration(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
}

func TestParseDurationRefused(t *testing.T) {
	for _, in := range []string{
		"",
		"P",
		"PT",
		"P1Y",
		"P1M", // months
		"P1W",
		"P1Y2D",
		"P1DT",
		"PT0S", // 0 s
		"P0D",
		"P30DT1S", // 30d + 1s
		"PT2592001S",
		"P31D",
		"P99999999D",
		"PT1.5S",
		"PT1,5S",
		"-PT1S",
		"P-1D",
		"PT+1S",
		"pt1s",
		"1S",
		"PT1S1M", // out of order
		"PT1M1H",
		"PT1S1S", // repeated
		"P1D1D",
		"P1DT1D",
		"PT1H T1M",
		"PTS",
		"PT1",
		"P1H",  // H before T
		"PT1D", // D after T
		"PT1STT",
		" PT1S",
		"PT1S ",
		"/wait_for",
		"PT１S", // full-width digit
	} {
		if got, err := ParseDuration(in); !errors.Is(err, ErrDuration) {
			t.Errorf("ParseDuration(%q) = %d, %v; want ErrDuration", in, got, err)
		}
	}
}

// durationShape is schema.json's duration pattern; every duration
// ParseDuration accepts must pass it, so the schema and the parser agree.
var durationShape = regexp.MustCompile(`^P(?:[0-9]{1,7}D)?(?:T(?:[0-9]{1,7}H)?(?:[0-9]{1,7}M)?(?:[0-9]{1,7}S)?)?$`)

func FuzzParseDuration(f *testing.F) {
	for _, s := range []string{"PT5S", "P1DT1H1M1S", "P30D", "P30DT1S", "P1Y", "P1M", "P1W", "PT0S", "PT", "P"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		n, err := ParseDuration(s)
		if err != nil {
			if !errors.Is(err, ErrDuration) || n != 0 {
				t.Fatalf("ParseDuration(%q) = %d, %v", s, n, err)
			}
			return
		}
		if n < MinWaitSeconds || n > MaxWaitSeconds {
			t.Fatalf("ParseDuration(%q) = %d, out of range", s, n)
		}
		if !durationShape.MatchString(s) {
			t.Fatalf("ParseDuration accepted %q, which schema.json refuses", s)
		}
	})
}
