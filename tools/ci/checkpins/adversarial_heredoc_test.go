package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TICK-1165: a line inside a RUN/COPY heredoc body, or on a continuation of an
// earlier line, is not a FROM instruction, so `from os import path` in a
// Python heredoc must not be read as image os. The skip must never be wider
// than BuildKit's: every case where a real FROM line could be mistaken for a
// heredoc body or continuation stays red.
func TestHeredocBodyIsNotAnInstruction(t *testing.T) {
	const f = "deploy/compose/nava-relay/Dockerfile"
	for name, text := range map[string]string{
		"RUN heredoc":             "\nRUN <<EOF\nfrom os import path\nEOF\n",
		"command before heredoc":  "\nRUN python3 <<PY\nfrom os import path\nFROM golang:1.26.0\nPY\n",
		"dash heredoc, quoted":    "\nRUN python3 <<-'PY'\n\tfrom os import path\n\tPY\n",
		"two heredocs on COPY":    "\nCOPY <<A <<\"B\" /etc/x/\nfrom a import b\nA\nfrom c import d\nB\n",
		"heredoc after flags":     "\nRUN --mount=type=cache,target=/root/.cache 3<<EOF cat /dev/fd/3\nfrom os import path\nEOF\n",
		"heredoc on continuation": "\nRUN --network=none \\\n  python3 <<EOF\nfrom os import path\nEOF\n",
		"continuation line":       "\nRUN apk add \\\n  from \\\n  py3-pip\n",
		"comment in continuation": "\nRUN apk add \\\n# from here on\n  py3-pip\n",
		"CRLF line endings":       "\r\nRUN <<EOF\r\nfrom os import path\r\nEOF\r\n",
	} {
		t.Run(name, func(t *testing.T) {
			dir := goCopy(t)
			appendTo(t, dir, f, text)
			mustSameGo(t, dir, "")
		})
	}
	t.Run("FROMX is not FROM", func(t *testing.T) {
		if fromAnyRe.MatchString("FROMX golang:1.26.0") || fromRe.MatchString("FROMX golang:1.26.0") {
			t.Fatal("a word that only starts with FROM is read as FROM")
		}
	})
	t.Run("pins check skips heredoc bodies", func(t *testing.T) {
		got := checkDockerfile(t, "FROM postgres:16.10\nRUN <<EOF\nfrom os import path\nEOF\nRUN apk add \\\n  from\n")
		if got != "" {
			t.Fatalf("want no problems, got:\n%s", got)
		}
	})
}

// Each case hides a real FROM of a non-golang Go image behind something that
// looks like a heredoc body or continuation. BuildKit reads the FROM, so the
// check must too.
func TestHeredocSkipCannotHideAFrom(t *testing.T) {
	const (
		f    = "deploy/compose/nava-relay/Dockerfile"
		evil = "FROM cgr.dev/chainguard/go:latest AS real"
		want = "FROM cgr.dev/chainguard/go:latest is neither golang:<version>"
	)
	for name, text := range map[string]string{
		// BuildKit reads heredocs only on RUN, COPY and ADD.
		"heredoc on CMD":                 "\nCMD cat <<EOF\n" + evil + "\nEOF\n",
		"heredoc on ONBUILD RUN":         "\nONBUILD RUN <<EOF\n" + evil + "\nEOF\n",
		"<< inside single quotes":        "\nRUN echo 'x <<EOF y'\n" + evil + "\nEOF\n",
		"<< inside double quotes":        "\nRUN echo \"x <<EOF y\"\n" + evil + "\nEOF\n",
		"escaped <<":                     "\nRUN echo \\<<EOF\n" + evil + "\nEOF\n",
		"RUN on a CMD continuation":      "\nCMD echo \\\nRUN <<EOF\n" + evil + "\nRUN echo \\\nEOF\n",
		"heredoc never closed":           "\nRUN <<EOF\n" + evil + "\n",
		"delimiter with leading tab":     "\nRUN <<EOF\n" + evil + "\n\tEOF\n",
		"comment ends with backslash":    "\n# a comment \\\n" + evil + "\n",
		"backtick is not the escape":     "\nRUN echo x `\n" + evil + "\n",
		"continuation ends at a heredoc": "\nRUN <<EOF\nEOF\n" + evil + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			dir := goCopy(t)
			appendTo(t, dir, f, text)
			mustSameGo(t, dir, want)
		})
	}
	t.Run("escape directive makes backslash literal", func(t *testing.T) {
		dir := goCopy(t)
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f)))
		if err != nil {
			t.Fatal(err)
		}
		put(t, dir, f, "# escape=`\n"+string(data)+"\nRUN echo x \\\n"+evil+"\n")
		mustSameGo(t, dir, want)
	})
	// BuildKit reads no directive after an unknown one, so the backtick
	// below is not the escape and the FROM after it is real.
	t.Run("escape directive after an unknown directive", func(t *testing.T) {
		dir := goCopy(t)
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f)))
		if err != nil {
			t.Fatal(err)
		}
		put(t, dir, f, "# foo=bar\n# escape=`\n"+string(data)+"\nRUN echo x `\n"+evil+"\n")
		mustSameGo(t, dir, want)
	})
	// BuildKit strips a UTF-8 BOM before reading directives, so the escape
	// below is the backtick and the backslash line does not continue.
	t.Run("BOM before escape directive", func(t *testing.T) {
		dir := goCopy(t)
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f)))
		if err != nil {
			t.Fatal(err)
		}
		put(t, dir, f, "\ufeff# escape=`\n"+string(data)+"\nRUN echo x \\\n"+evil+"\n")
		mustSameGo(t, dir, want)
	})
	t.Run("pins check reads a FROM after a BOM", func(t *testing.T) {
		got := checkDockerfile(t, "\ufeffFROM postgres:15\n")
		if !strings.Contains(got, "Dockerfile:1: image postgres:15") {
			t.Fatalf("want the FROM on line 1 checked, got:\n%s", got)
		}
	})
	t.Run("unclosed heredoc is named", func(t *testing.T) {
		dir := goCopy(t)
		appendTo(t, dir, f, "\nRUN <<EOF\nfrom os import path\n")
		mustSameGo(t, dir, "heredoc <<EOF is never closed")
	})
	t.Run("pins check reads a FROM after a fake heredoc", func(t *testing.T) {
		got := checkDockerfile(t, "CMD echo \\\nRUN <<EOF\nFROM postgres:15\nRUN echo \\\nEOF\n")
		if !strings.Contains(got, "Dockerfile:3: image postgres:15") {
			t.Fatalf("want the FROM on line 3 checked, got:\n%s", got)
		}
	})
	t.Run("pins check names an unclosed heredoc", func(t *testing.T) {
		got := checkDockerfile(t, "FROM postgres:16.10\nRUN <<EOF\nfrom os import path\n")
		if !strings.Contains(got, "Dockerfile:2: heredoc <<EOF is never closed") {
			t.Fatalf("want the unclosed heredoc named, got:\n%s", got)
		}
	})
}

// checkDockerfile runs the pins image check on a Dockerfile holding data.
func checkDockerfile(t *testing.T, data string) string {
	t.Helper()
	p, err := parsePins(fixturePins(t))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "Dockerfile")
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := checkImages(path, "Dockerfile", p, fromRe, false)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(got, "\n")
}
