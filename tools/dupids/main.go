// Command dupids fails when an id appears more than once in a shared table:
// docs/delivery/todo.md (TODO-...) or docs/security/accepted-risks.md (AR-...).
//
// merge=union (TICK-295, merge-safety D6) is only safe for appends. When two
// branches edit the same row, git keeps both versions as two rows with one id
// and reports no conflict. Two tasks of one story that pick the same id (for
// example both TODO-S5-01) look the same. This check catches both.
//
// A first cell in an "ID" table that is not a valid id (for example `AR-033`
// in backticks or **AR-033** in bold) also fails: it would never be counted.
//
// Usage: dupids [-dir repo-root]. Exits 1 on a duplicate, which is what
// `task lint` runs.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

type table struct {
	path    string
	id      *regexp.Regexp
	example string
}

// tables lists each shared file with the pattern its row ids follow.
func tables() []table {
	return []table{
		{"docs/delivery/todo.md", regexp.MustCompile(`^TODO-[A-Za-z0-9-]+$`), "TODO-S5-01"},
		{"docs/security/accepted-risks.md", regexp.MustCompile(`^AR-[A-Za-z0-9-]+$`), "AR-S5-01"},
	}
}

func main() {
	dir := flag.String("dir", ".", "repository root")
	flag.Parse()

	problems, err := check(*dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dupids:", err)
		os.Exit(2)
	}
	for _, p := range problems {
		fmt.Fprintln(os.Stderr, "dupids:", p)
	}
	if len(problems) > 0 {
		os.Exit(1)
	}
}

func check(root string) ([]string, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var problems []string
	for _, tb := range tables() {
		f, err := r.Open(tb.path)
		if err != nil {
			return nil, err
		}
		lines := map[string][]int{}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		inIDTable, prevRow, sawIDTable := false, false, false
		for n := 1; sc.Scan(); n++ {
			line := sc.Text()
			isRow := strings.HasPrefix(line, "|")
			if isRow && !prevRow {
				// First line of a table: only tables headed "ID" hold row ids.
				inIDTable = firstCell(line) == "ID"
				sawIDTable = sawIDTable || inIDTable
			}
			prevRow = isRow
			if !isRow || !inIDTable {
				continue
			}
			cell := firstCell(line)
			if cell == "" || cell == "ID" || separatorCell.MatchString(cell) {
				continue
			}
			id := rowID(line)
			if !tb.id.MatchString(id) {
				problems = append(problems, fmt.Sprintf("%s: line %d: malformed id cell %q; write the bare id, such as %s",
					tb.path, n, cell, tb.example))
				continue
			}
			lines[id] = append(lines[id], n)
		}
		err = sc.Err()
		_ = f.Close() // read-only
		if err != nil {
			return nil, fmt.Errorf("%s: %w", tb.path, err)
		}
		if !sawIDTable {
			problems = append(problems, fmt.Sprintf("%s: no table headed \"ID\"; dupids checks nothing", tb.path))
		}
		ids := make([]string, 0, len(lines))
		for id, ns := range lines {
			if len(ns) > 1 {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
		for _, id := range ids {
			problems = append(problems, fmt.Sprintf("%s: id %s appears %d times (lines %s); give each row its own id",
				tb.path, id, len(lines[id]), joinInts(lines[id])))
		}
	}
	return problems, nil
}

// rowID returns the first word of the first cell of a table row
// ("| TODO-001 | ..." gives "TODO-001"), or "" for any other line. Text after
// the first space, such as the "(R8)" in "AR-001 (R8)", is dropped.
func rowID(line string) string {
	id, _, _ := strings.Cut(firstCell(line), " ")
	return id
}

// separatorCell matches the first cell of a table's delimiter row ("---").
var separatorCell = regexp.MustCompile(`^:?-+:?$`)

// firstCell returns the trimmed first cell of a table row, or "" for any
// other line.
func firstCell(line string) string {
	if !strings.HasPrefix(line, "|") {
		return ""
	}
	cell, _, _ := strings.Cut(line[1:], "|")
	return strings.TrimSpace(cell)
}

func joinInts(ns []int) string {
	s := make([]string, len(ns))
	for i, n := range ns {
		s[i] = fmt.Sprint(n)
	}
	return strings.Join(s, ", ")
}
