// Command dupids fails when an id appears more than once in a shared table:
// docs/delivery/todo.md (TODO-...) or docs/security/accepted-risks.md (AR-...).
//
// merge=union (TICK-295, merge-safety D6) is only safe for appends. When two
// branches edit the same row, git keeps both versions as two rows with one id
// and reports no conflict. Two tasks of one story that pick the same id (for
// example both TODO-S5-01) look the same. This check catches both.
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
	path string
	id   *regexp.Regexp
}

// tables lists each shared file with the pattern its row ids follow.
func tables() []table {
	return []table{
		{"docs/delivery/todo.md", regexp.MustCompile(`^TODO-[A-Za-z0-9-]+$`)},
		{"docs/security/accepted-risks.md", regexp.MustCompile(`^AR-[A-Za-z0-9-]+$`)},
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
		for n := 1; sc.Scan(); n++ {
			if id := rowID(sc.Text()); tb.id.MatchString(id) {
				lines[id] = append(lines[id], n)
			}
		}
		err = sc.Err()
		_ = f.Close() // read-only
		if err != nil {
			return nil, fmt.Errorf("%s: %w", tb.path, err)
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

// rowID returns the first cell of a table row ("| TODO-001 | ..." gives
// "TODO-001"), or "" for any other line. A trailing parenthesis such as
// "AR-001 (R8)" is dropped.
func rowID(line string) string {
	if !strings.HasPrefix(line, "|") {
		return ""
	}
	cell, _, _ := strings.Cut(line[1:], "|")
	cell = strings.TrimSpace(cell)
	cell, _, _ = strings.Cut(cell, " ")
	return cell
}

func joinInts(ns []int) string {
	s := make([]string, len(ns))
	for i, n := range ns {
		s[i] = fmt.Sprint(n)
	}
	return strings.Join(s, ", ")
}
