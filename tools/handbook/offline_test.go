package main

import (
	"path"
	"regexp"
	"strings"
	"testing"
)

// The handbook must open from file:// with no network request. These rules
// cover script/img/link and every other element that loads, @import, url(),
// fetch() and the other script-side network calls. Plain <a href> to GitHub
// is allowed: an anchor loads nothing.
var (
	loadingTag = regexp.MustCompile(`(?is)<(script|img|link|iframe|frame|embed|object|source|video|audio|track|input|image|use|meta)\b[^>]*>`)
	remoteAttr = regexp.MustCompile(`(?is)\b(?:src|href|data|srcset|poster|xlink:href|action|formaction)\s*=\s*["']?\s*(?:https?:)?//`)
	remoteMeta = regexp.MustCompile(`(?is)http-equiv\s*=\s*["']?refresh`)
	cssImport  = regexp.MustCompile(`(?is)@import\b`)
	cssURL     = regexp.MustCompile(`(?is)url\(\s*["']?\s*(?:https?:)?//`)
	jsNetwork  = regexp.MustCompile(`\b(?:fetch|importScripts|sendBeacon)\s*\(|XMLHttpRequest|new\s+(?:WebSocket|EventSource)\b`)
	jsRemote   = regexp.MustCompile(`(?i)["'\x60]\s*(?:https?:)?//[a-z0-9]`)
)

// offlineViolations returns one line per network load found in a generated file.
func offlineViolations(name string, data []byte) []string {
	var out []string
	s := string(data)
	switch path.Ext(name) {
	case ".html":
		for _, tag := range loadingTag.FindAllString(s, -1) {
			if remoteAttr.MatchString(tag) || remoteMeta.MatchString(tag) {
				out = append(out, name+": remote load in "+tag)
			}
		}
		for _, m := range cssImport.FindAllString(s, -1) {
			out = append(out, name+": "+m)
		}
		for _, m := range cssURL.FindAllString(s, -1) {
			out = append(out, name+": "+m)
		}
	case ".css":
		for _, m := range cssImport.FindAllString(s, -1) {
			out = append(out, name+": "+m)
		}
		for _, m := range cssURL.FindAllString(s, -1) {
			out = append(out, name+": "+m)
		}
	case ".js":
		for _, m := range jsNetwork.FindAllString(s, -1) {
			out = append(out, name+": "+m)
		}
		for _, m := range jsRemote.FindAllString(s, -1) {
			out = append(out, name+": remote string "+m)
		}
	}
	return out
}

func TestOfflineRulesCatchEveryKindOfNetworkLoad(t *testing.T) {
	bad := map[string]string{
		"a.html": `<script src="https://cdn.example/x.js"></script>`,
		"b.html": `<img src="http://example.com/a.png">`,
		"c.html": `<link rel="stylesheet" href="//fonts.example/x.css">`,
		"d.html": `<iframe src='https://example.com'></iframe>`,
		"e.html": `<style>@import "x.css";</style>`,
		"f.html": `<div style="background:url(https://example.com/a.png)"></div>`,
		"g.html": `<meta http-equiv="refresh" content="0;url=https://example.com">`,
		"h.html": `<script
 src = "https://cdn.example/x.js"></script>`,
		"a.css": `@import url("https://x.example/a.css");`,
		"b.css": `body{background:url(https://x.example/a.png)}`,
		"c.css": `body{background:url(//x.example/a.png)}`,
		"a.js":  `fetch("data.json")`,
		"b.js":  `var x = new XMLHttpRequest();`,
		"c.js":  `var u = "https://example.com/x";`,
		"d.js":  `navigator.sendBeacon("/x")`,
		"e.js":  `new WebSocket("wss://x")`,
	}
	for name, src := range bad {
		if len(offlineViolations(name, []byte(src))) == 0 {
			t.Errorf("%s not flagged: %s", name, src)
		}
	}
	good := map[string]string{
		"a.html": `<a href="https://github.com/Navaladi2019/TicketingSystem">repo</a><script src="../assets/x.js"></script><img src="../diagrams/a.svg"><link rel="stylesheet" href="assets/handbook.css">`,
		"a.css":  `body{background:url(img.png)}`,
		"a.js":   `var s = "search"; document.getElementById("x");`,
	}
	for name, src := range good {
		if v := offlineViolations(name, []byte(src)); len(v) != 0 {
			t.Errorf("%s flagged: %v", name, v)
		}
	}
}

// assertOffline scans every generated HTML, CSS and JS file of out. The pinned
// mermaid.min.js is a vendored third-party bundle: it holds http(s) strings
// (XML namespaces, upstream doc links) that are never fetched, so it is
// skipped; the page CSP (script-src 'self') is what stops it loading more.
func assertOffline(t *testing.T, label string, out Output) {
	t.Helper()
	scanned := 0
	for name, data := range out {
		if name == "assets/mermaid.min.js" {
			continue
		}
		scanned++
		for _, v := range offlineViolations(name, data) {
			t.Errorf("%s: %s", label, v)
		}
		if strings.HasSuffix(name, ".html") {
			for _, want := range []string{"default-src 'self'", "script-src 'self'", "object-src 'none'", "style-src 'self'"} {
				if !strings.Contains(string(data), want) {
					t.Errorf("%s: %s lacks CSP %q", label, name, want)
				}
			}
			if strings.Contains(string(data), "<script>") {
				t.Errorf("%s: %s has an inline script", label, name)
			}
		}
	}
	if scanned < 4 {
		t.Errorf("%s: only %d files scanned", label, scanned)
	}
}

// TestFixtureFilesLoadNothingOverTheNetwork covers a page with a Mermaid
// diagram, an Excalidraw image and a GitHub anchor. TestRealHandbookBuilds
// runs the same scan over the real handbook.
func TestFixtureFilesLoadNothingOverTheNetwork(t *testing.T) {
	root := newFixture(t)
	page(t, root, "system/a.md", "A", "system", "delivered", "", "```mermaid\nflowchart LR\n a-->b\n```\n![x](../diagrams/ok.excalidraw.svg)\n[gh](https://github.com/x)\n")
	assertOffline(t, "fixture", buildFixture(t, root))
}
