// Package diag renders an errx error as a compiler-grade diagnostic:
// severity + code + message, a documentation URL, a hint, and — when the
// error carries source via WithSource/WithLabel — the offending source
// line(s) with caret underlines (miette's GraphicalReportHandler), plus a
// NarratableReportHandler that describes the diagnostic in prose for CI,
// pipes, and screen readers. Stdlib-only.
package diag

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ubgo/errx"
)

// Options controls rendering. The zero value is a deterministic,
// no-color graphical report.
type Options struct {
	Color      bool // emit ANSI color
	Narratable bool // prose form (CI / screen readers) instead of graphical
}

// Auto picks options from the environment: narratable + no color when
// NO_COLOR or CI is set or stdout is not a character device; colored
// graphical otherwise.
func Auto() Options {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("CI") != "" {
		return Options{Narratable: true}
	}
	if fi, err := os.Stdout.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
		return Options{Narratable: true}
	}
	return Options{Color: true}
}

// String renders err to a string.
func String(err error, opts ...Options) string {
	var b strings.Builder
	Fprint(&b, err, opts...)
	return b.String()
}

// Fprint writes the diagnostic for err to w. Non-errx errors render as a
// plain one-line message.
func Fprint(w io.Writer, err error, opts ...Options) {
	o := Options{}
	if len(opts) > 0 {
		o = opts[0]
	}
	e := errx.Get(err)
	if e == nil {
		if err != nil {
			fmt.Fprintln(w, err.Error())
		}
		return
	}
	if o.Narratable {
		narrate(w, e)
		return
	}
	graphical(w, e, o.Color)
}

type palette struct{ red, dim, bold, reset string }

func pal(color bool) palette {
	if !color {
		return palette{}
	}
	return palette{red: "\x1b[31m", dim: "\x1b[2m", bold: "\x1b[1m", reset: "\x1b[0m"}
}

func sevWord(s errx.Severity) string {
	switch s {
	case errx.SevWarn:
		return "warning"
	case errx.SevDebug, errx.SevInfo:
		return "note"
	case errx.SevFatal:
		return "fatal"
	default:
		return "error"
	}
}

func graphical(w io.Writer, e *errx.Error, color bool) {
	p := pal(color)
	code := e.Code()
	head := sevWord(e.SeverityOf())
	if code != "" {
		head += "[" + code + "]"
	}
	fmt.Fprintf(w, "%s%s%s: %s\n", p.bold+p.red, head, p.reset, e.Error())

	name, src := e.Source()
	labels := e.Labels()
	if src != "" && len(labels) > 0 {
		lines, starts := indexLines(src)
		fmt.Fprintf(w, "  %s-->%s %s\n", p.dim, p.reset, nonempty(name, "<source>"))
		for _, lb := range labels {
			ln, col := locate(starts, lb.Start)
			if ln < 0 || ln >= len(lines) {
				continue
			}
			gutter := fmt.Sprintf("%d", ln+1)
			pad := strings.Repeat(" ", len(gutter))
			fmt.Fprintf(w, "%s %s|%s\n", pad, p.dim, p.reset)
			fmt.Fprintf(w, "%s%s |%s %s\n", p.dim, gutter, p.reset, lines[ln])
			ul := lb.Len
			if ul < 1 {
				ul = 1
			}
			if col+ul > len(lines[ln]) {
				ul = max(1, len(lines[ln])-col)
			}
			fmt.Fprintf(w, "%s %s|%s %s%s%s%s %s\n",
				pad, p.dim, p.reset,
				strings.Repeat(" ", col), p.red, strings.Repeat("^", ul), p.reset, lb.Msg)
		}
		fmt.Fprintf(w, "%s %s|%s\n", strings.Repeat(" ", len(fmt.Sprintf("%d", maxLine(starts, labels)))), p.dim, p.reset)
	}
	if h := e.Remediation(); h != "" {
		fmt.Fprintf(w, "%shelp:%s %s\n", p.bold, p.reset, h)
	}
	if u := e.URL(); u != "" {
		fmt.Fprintf(w, "%sdocs:%s %s\n", p.dim, p.reset, u)
	}
}

func narrate(w io.Writer, e *errx.Error) {
	code := e.Code()
	if code != "" {
		fmt.Fprintf(w, "%s [%s]: %s\n", capFirst(sevWord(e.SeverityOf())), code, e.Error())
	} else {
		fmt.Fprintf(w, "%s: %s\n", capFirst(sevWord(e.SeverityOf())), e.Error())
	}
	name, src := e.Source()
	if src != "" {
		_, starts := indexLines(src)
		for _, lb := range e.Labels() {
			ln, col := locate(starts, lb.Start)
			fmt.Fprintf(w, "  at %s line %d, column %d: %s\n",
				nonempty(name, "source"), ln+1, col+1, lb.Msg)
		}
	}
	if h := e.Remediation(); h != "" {
		fmt.Fprintf(w, "  help: %s\n", h)
	}
	if u := e.URL(); u != "" {
		fmt.Fprintf(w, "  see: %s\n", u)
	}
}

func indexLines(s string) (lines []string, starts []int) {
	lines = strings.Split(s, "\n")
	starts = make([]int, len(lines))
	off := 0
	for i, ln := range lines {
		starts[i] = off
		off += len(ln) + 1 // +1 for the consumed '\n'
	}
	return
}

// locate maps a byte offset to (lineIndex, columnIndex), both 0-based.
func locate(starts []int, off int) (int, int) {
	ln := -1
	for i, s := range starts {
		if off >= s {
			ln = i
		} else {
			break
		}
	}
	if ln < 0 {
		return -1, 0
	}
	return ln, off - starts[ln]
}

func maxLine(starts []int, labels []errx.Label) int {
	m := 1
	for _, lb := range labels {
		if ln, _ := locate(starts, lb.Start); ln+1 > m {
			m = ln + 1
		}
	}
	return m
}

func nonempty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func capFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
