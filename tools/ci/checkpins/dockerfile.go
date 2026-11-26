package main

import (
	"regexp"
	"strings"
)

// TICK-1165: which lines of a Dockerfile start an instruction. A line inside a
// RUN/COPY/ADD heredoc body, or on a continuation of the line before it, is
// not an instruction, so `from os import path` in a Python heredoc is not a
// FROM. The skip must never be wider than BuildKit's, or a real FROM could
// hide in it: every guess that is unsure (a << in quotes or after an escape,
// a heredoc on any other instruction, one that is never closed) falls back to
// reading the line as an instruction, so the check fails closed.

var (
	// A parser directive line: # key=value, before any other line.
	directiveRe = regexp.MustCompile(`^#\s*([A-Za-z]+)\s*=\s*(\S*)\s*$`)
	// A heredoc word: [fd]<<[-]DELIM with DELIM bare, 'quoted' or "quoted".
	heredocRe = regexp.MustCompile(`^[0-9]*<<(-?)(?:([A-Za-z_][A-Za-z0-9_]*)|'([A-Za-z_][A-Za-z0-9_]*)'|"([A-Za-z_][A-Za-z0-9_]*)")$`)
)

// heredoc is one open heredoc: its delimiter, whether <<- strips leading tabs
// from its lines, and the line (0-based) that opened it.
type heredoc struct {
	delim string
	dash  bool
	at    int
}

// dockerLines splits a Dockerfile into lines (a trailing \r dropped) and
// reports for each whether it starts an instruction. unclosed lists the lines
// (0-based) that open a heredoc never closed; those heredocs are not skipped.
func dockerLines(data string) (lines []string, starts []bool, unclosed []int) {
	lines = strings.Split(data, "\n")
	for i := range lines {
		lines[i] = strings.TrimSuffix(lines[i], "\r")
	}
	// BuildKit drops a UTF-8 byte order mark before it reads directives, so
	// "\ufeff# escape=`" sets the escape and "\ufeffFROM x" is a FROM.
	lines[0] = strings.TrimPrefix(lines[0], "\ufeff")
	ignore := map[int]bool{} // heredoc openers read as plain lines
	for {
		var open int
		starts, open = scanLines(lines, ignore)
		if open < 0 {
			return lines, starts, unclosed
		}
		ignore[open] = true
		unclosed = append(unclosed, open)
	}
}

// scanLines marks instruction starts, skipping the bodies of the heredocs
// opened on lines not in ignore. It returns the opener of the first heredoc
// that is never closed, or -1.
func scanLines(lines []string, ignore map[int]bool) (starts []bool, open int) {
	starts = make([]bool, len(lines))
	escape := byte('\\')
	directives := true
	cont := false // the line before continues onto this one
	logical := "" // the instruction read so far, continuations joined
	var bodies []heredoc
	for i, l := range lines {
		if len(bodies) > 0 {
			end := l
			if bodies[0].dash {
				end = strings.TrimLeft(l, "\t")
			}
			if end == bodies[0].delim {
				bodies = bodies[1:]
			}
			continue
		}
		if directives {
			// BuildKit stops reading directives at the first other line,
			// an unknown directive included (it is a comment).
			if m := directiveRe.FindStringSubmatch(l); m != nil {
				switch strings.ToLower(m[1]) {
				case "escape":
					if len(m[2]) == 1 {
						escape = m[2][0]
					}
					continue
				case "syntax", "check":
					continue
				}
			}
			directives = false
		}
		t := strings.TrimSpace(l)
		if t == "" || t[0] == '#' {
			continue // a comment or blank line neither starts nor ends an instruction
		}
		starts[i] = !cont
		if !cont {
			logical = ""
		}
		body := strings.TrimRight(l, " \t")
		cont = strings.HasSuffix(body, string(escape))
		if cont {
			logical += strings.TrimSuffix(body, string(escape)) + " "
			continue
		}
		logical += l
		if ignore[i] {
			continue
		}
		for _, h := range heredocsOf(logical, escape) {
			h.at = i
			bodies = append(bodies, h)
		}
	}
	if len(bodies) > 0 {
		return starts, bodies[0].at
	}
	return starts, -1
}

// heredocsOf returns the heredocs a whole instruction opens: only RUN, COPY
// and ADD open any, and only a << word outside quotes counts.
func heredocsOf(instr string, escape byte) []heredoc {
	words := shellWords(instr, escape)
	if len(words) == 0 {
		return nil
	}
	switch strings.ToUpper(words[0]) {
	case "RUN", "COPY", "ADD":
	default:
		return nil
	}
	var hs []heredoc
	for _, w := range words[1:] {
		m := heredocRe.FindStringSubmatch(w)
		if m == nil {
			continue
		}
		hs = append(hs, heredoc{delim: m[2] + m[3] + m[4], dash: m[1] == "-"})
	}
	return hs
}

// heredocWord names the first heredoc a line opens (<<EOF), for messages.
func heredocWord(line string) string {
	for _, w := range strings.Fields(line) {
		if heredocRe.MatchString(w) {
			return w[strings.Index(w, "<<"):]
		}
	}
	return "<<"
}

// shellWords splits s at whitespace outside quotes, keeping each word's raw
// text (quotes and escapes included), so a << inside quotes or after an
// escape is part of a word that heredocRe does not match.
func shellWords(s string, escape byte) []string {
	var (
		words []string
		cur   strings.Builder
		quote byte
		in    bool
	)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote == 0 && (c == ' ' || c == '\t'):
			if in {
				words = append(words, cur.String())
				cur.Reset()
				in = false
			}
			continue
		case c == escape && quote != '\'' && i+1 < len(s):
			cur.WriteByte(c)
			i++
			c = s[i]
		case quote == 0 && (c == '\'' || c == '"'):
			quote = c
		case c == quote:
			quote = 0
		}
		cur.WriteByte(c)
		in = true
	}
	if in {
		words = append(words, cur.String())
	}
	return words
}
