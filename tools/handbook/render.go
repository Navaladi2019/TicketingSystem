package main

import (
	"bytes"
	"fmt"
	"html"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// pageRenderer turns one page's Markdown into HTML. Raw HTML in the source is
// never passed through (goldmark's default; WithUnsafe is not set): the only
// embeds are a mermaid fence and an Excalidraw image, both handled here.
type pageRenderer struct {
	facts    *Facts
	page     *Page
	byRel    map[string]*Page // by path relative to docs/handbook/src
	prefix   string           // "" for index.html, "../" for a section page
	problems []string
	diagrams map[string][]byte // output name -> svg
	mermaid  bool
	src      []byte
}

func (r *pageRenderer) bad(format string, a ...any) {
	r.problems = append(r.problems, r.page.Src+": "+fmt.Sprintf(format, a...))
}

// render returns the body HTML and the plain text for the search index.
func (r *pageRenderer) render(md string) (string, string) {
	r.src = []byte(md)
	gm := goldmark.New(
		goldmark.WithExtensions(extension.Table, extension.Strikethrough),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(
			gmhtml.WithHardWraps(),
			renderer.WithNodeRenderers(util.Prioritized(&diagramRenderer{r: r}, 500)),
		),
	)
	doc := gm.Parser().Parse(text.NewReader(r.src))
	r.walk(doc)
	var buf bytes.Buffer
	if err := gm.Renderer().Render(&buf, r.src, doc); err != nil {
		r.bad("render: %v", err)
	}
	return buf.String(), plainText(doc, r.src)
}

// walk validates and rewrites links and checks images.
func (r *pageRenderer) walk(doc ast.Node) {
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Link:
			v.Destination = []byte(r.link(string(v.Destination)))
		case *ast.AutoLink:
			if _, err := checkExternal(string(v.URL(r.src))); err != nil {
				r.bad("link: %v", err)
			}
		case *ast.Image:
			r.image(v)
		case *ast.FencedCodeBlock:
			if string(v.Language(r.src)) == "mermaid" {
				r.mermaid = true
			}
		}
		return ast.WalkContinue, nil
	})
}

var schemeRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

// checkExternal accepts only http(s) URLs as plain anchors; javascript:, data:
// and every other scheme are rejected.
func checkExternal(dest string) (bool, error) {
	if !schemeRe.MatchString(dest) {
		return false, nil
	}
	u, err := url.Parse(dest)
	if err != nil {
		return true, fmt.Errorf("%q is not a valid URL", dest)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return true, nil
	}
	return true, fmt.Errorf("%q: the %s: scheme is not allowed, use a relative link or an https URL", dest, strings.ToLower(u.Scheme))
}

func (r *pageRenderer) link(dest string) string {
	if dest == "" {
		r.bad("link with an empty target")
		return "#"
	}
	if strings.HasPrefix(dest, "#") {
		return dest
	}
	ext, err := checkExternal(dest)
	if err != nil {
		r.bad("link: %v", err)
		return "#"
	}
	if ext {
		return dest
	}
	file, frag, _ := strings.Cut(dest, "#")
	if frag != "" {
		frag = "#" + frag
	}
	repo := r.repoPath(file)
	if repo == "" {
		r.bad("link %q leaves the repository", dest)
		return "#"
	}
	if rel, ok := strings.CutPrefix(repo, "docs/handbook/src/"); ok && strings.HasSuffix(rel, ".md") {
		target, found := r.byRel[rel]
		if !found {
			r.bad("link %q names a handbook page that does not exist", dest)
			return "#"
		}
		return r.prefix + target.Out() + frag
	}
	if _, err := os.Stat(filepath.Join(r.facts.Root, filepath.FromSlash(repo))); err != nil {
		r.bad("link %q: %s does not exist in the repository", dest, repo)
		return "#"
	}
	return path.Join(r.upToRoot(), repo) + frag
}

// upToRoot is the relative path from the page's output directory to the repo root.
func (r *pageRenderer) upToRoot() string {
	if r.page.Section == homeSection {
		return "../.."
	}
	return "../../.."
}

// repoPath resolves a link written relative to the page's source directory to
// a repository path, or "" when it escapes the repository.
func (r *pageRenderer) repoPath(rel string) string {
	dir := path.Dir(r.page.Src)
	p := path.Clean(path.Join(dir, rel))
	if p == ".." || strings.HasPrefix(p, "../") || path.IsAbs(p) {
		return ""
	}
	return p
}

func (r *pageRenderer) image(img *ast.Image) {
	dest := string(img.Destination)
	if ext, err := checkExternal(dest); ext || err != nil {
		r.bad("image %q: only images stored in the repository are allowed (nothing is loaded over the network)", dest)
		return
	}
	if !strings.HasSuffix(dest, ".excalidraw.svg") {
		r.bad("image %q: embed Excalidraw diagrams as <name>.excalidraw.svg under docs/handbook/src/diagrams; Mermaid goes in a mermaid fence", dest)
		return
	}
	repo := r.repoPath(dest)
	if !strings.HasPrefix(repo, "docs/handbook/src/diagrams/") || strings.Count(strings.TrimPrefix(repo, "docs/handbook/src/diagrams/"), "/") > 0 {
		r.bad("image %q must be a file directly under docs/handbook/src/diagrams", dest)
		return
	}
	data, err := os.ReadFile(filepath.Join(r.facts.Root, filepath.FromSlash(repo)))
	if err != nil {
		r.bad("image %q does not exist", dest)
		return
	}
	for _, p := range checkExcalidrawSVG(data) {
		r.bad("image %q: %s", dest, p)
	}
	name := path.Base(repo)
	r.diagrams[name] = data
	img.Destination = []byte(r.prefix + "diagrams/" + name)
}

var (
	svgActive = regexp.MustCompile(`(?is)<script|<foreignObject|\son[a-z]+\s*=|javascript:`)
	svgRemote = regexp.MustCompile(`(?is)(href|src)\s*=\s*["']\s*https?:|url\(\s*["']?\s*https?:|@import`)
)

// checkExcalidrawSVG requires the embedded scene (so the diagram stays
// editable) and refuses active content or remote references.
func checkExcalidrawSVG(data []byte) []string {
	var out []string
	if !bytes.Contains(data, []byte("payload-start")) {
		out = append(out, "has no embedded scene: export from Excalidraw with \"Embed scene\"")
	}
	if svgActive.Match(data) {
		out = append(out, "contains script, foreignObject or an event handler")
	}
	if svgRemote.Match(data) {
		out = append(out, "references a remote resource")
	}
	return out
}

// altText is the plain text of a node's children.
func altText(n ast.Node, src []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if t, ok := c.(*ast.Text); ok && entering {
			b.Write(t.Segment.Value(src))
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

// plainText flattens a document for the search index.
func plainText(doc ast.Node, src []byte) string {
	var b strings.Builder
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Text:
			b.Write(v.Segment.Value(src))
			b.WriteByte(' ')
		case *ast.CodeSpan:
			b.WriteString(altText(v, src))
			b.WriteByte(' ')
			return ast.WalkSkipChildren, nil
		case *ast.FencedCodeBlock:
			if string(v.Language(src)) == "mermaid" {
				return ast.WalkSkipChildren, nil
			}
			for i := 0; i < v.Lines().Len(); i++ {
				seg := v.Lines().At(i)
				b.Write(seg.Value(src))
			}
			b.WriteByte(' ')
		}
		return ast.WalkContinue, nil
	})
	return strings.Join(strings.Fields(b.String()), " ")
}

// diagramRenderer draws the two diagram embeds: a mermaid fence becomes a
// figure whose source the viewer renders lazily (the <pre> is also the print
// and no-script fallback), and an Excalidraw image becomes an <img> inside a
// figure, so the viewer transforms a wrapper and never inlines the SVG.
type diagramRenderer struct{ r *pageRenderer }

func (d *diagramRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, d.fenced)
	reg.Register(ast.KindImage, d.image)
}

func (d *diagramRenderer) fenced(w util.BufWriter, src []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	cb := n.(*ast.FencedCodeBlock)
	var code bytes.Buffer
	for i := 0; i < cb.Lines().Len(); i++ {
		seg := cb.Lines().At(i)
		code.Write(seg.Value(src))
	}
	lang := string(cb.Language(src))
	if lang == "mermaid" {
		fmt.Fprintf(w, `<figure class="diagram" data-kind="mermaid"><div class="dz-wrap"><pre class="mermaid-src">%s</pre></div></figure>`+"\n", html.EscapeString(code.String()))
		return ast.WalkSkipChildren, nil
	}
	cls := ""
	if lang != "" {
		cls = ` class="language-` + html.EscapeString(lang) + `"`
	}
	fmt.Fprintf(w, "<pre><code%s>%s</code></pre>\n", cls, html.EscapeString(code.String()))
	return ast.WalkSkipChildren, nil
}

func (d *diagramRenderer) image(w util.BufWriter, src []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	img := n.(*ast.Image)
	alt := altText(img, src)
	fmt.Fprintf(w, `<figure class="diagram" data-kind="excalidraw"><div class="dz-wrap"><img class="dz-img" src="%s" alt="%s"></div></figure>`,
		html.EscapeString(string(img.Destination)), html.EscapeString(alt))
	return ast.WalkSkipChildren, nil
}
