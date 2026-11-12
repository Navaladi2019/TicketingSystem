//go:build ignore

// Spike for ADR-005-E2 (TICK-667, Technical Architect, comment 01a1d155-4d94):
// is overflow "grow" injective over (scope, counter) when a digit touches {seq:N}?
// THROWAWAY. Not production code. Never import. Run: go run grow-injectivity-main.go
package main

import (
	"fmt"
	"strings"
	"time"
)

func render(tpl string, d time.Time, n int64, w int) string {
	r := strings.NewReplacer("{prefix}", "INC", "{yyyy}", d.Format("2006"), "{yy}", d.Format("06"), "{MM}", d.Format("01"), "{dd}", d.Format("02"))
	s := r.Replace(tpl)
	i := strings.Index(s, "{seq:")
	j := strings.Index(s[i:], "}") + i
	return s[:i] + fmt.Sprintf("%0*d", w, n) + s[j+1:]
}

func scope(tpl string, d time.Time) string {
	var b strings.Builder
	for _, p := range []struct{ t, l string }{{"{yyyy}", "2006"}, {"{yy}", "06"}, {"{MM}", "01"}, {"{dd}", "02"}} {
		if strings.Contains(tpl, p.t) {
			b.WriteString(d.Format(p.l))
		}
	}
	return b.String()
}

func main() {
	cases := []struct {
		tpl string
		w   int
	}{{"{prefix}{yy}{seq:3}", 3}, {"{prefix}2{seq:1}", 1}, {"{prefix}-{yyyy}{MM}{seq:4}", 4}, {"{prefix}{dd}{seq:1}{MM}", 1}, {"{prefix}-{seq:2}{yy}", 2}}
	ks := []int64{}
	for k := int64(1); k <= 110000; k++ {
		if k < 1200 || k%4999 == 0 || k == 100000 || k == 100001 {
			ks = append(ks, k)
		}
	}
	for _, c := range cases {
		seen := map[string]string{}
		total, clashes := 0, 0
		for d := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC); d.Year() < 2031; d = d.AddDate(0, 0, 11) {
			for _, k := range ks {
				r := render(c.tpl, d, k, c.w)
				id := scope(c.tpl, d) + "#" + fmt.Sprint(k)
				if p, ok := seen[r]; ok && p != id {
					clashes++
				}
				seen[r] = id
				total++
			}
		}
		fmt.Printf("%-28s grow: %d allocations, %d distinct numbers, %d clashes\n", c.tpl, total, len(seen), clashes)
	}
}
