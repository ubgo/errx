package errx

import (
	"fmt"
	"io"
	"strconv"
)

var _ fmt.Formatter = (*Error)(nil)

// Format implements fmt.Formatter.
//
//	%s / %v  operator message (chain)
//	%q       quoted operator message
//	%+v      message + identity + fields + origin stack + suppressed
func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			e.writeVerbose(s)
			return
		}
		io.WriteString(s, e.Error())
	case 's':
		io.WriteString(s, e.Error())
	case 'q':
		io.WriteString(s, strconv.Quote(e.Error()))
	default:
		io.WriteString(s, e.Error())
	}
}

func (e *Error) writeVerbose(w io.Writer) {
	io.WriteString(w, e.Error())
	if e.domain != "" || e.code != "" {
		fmt.Fprintf(w, "\nidentity: %s/%s (%s, %s)", e.domain, e.code, e.class, e.severity)
	}
	if e.publicMsg != "" {
		fmt.Fprintf(w, "\npublic:   %s", e.publicMsg)
	}
	if e.hint != "" {
		fmt.Fprintf(w, "\nhint:     %s", e.hint)
	}
	for _, f := range e.fields {
		v := f.Value
		if !f.Safe {
			v = RedactedMarker
		}
		fmt.Fprintf(w, "\nfield:    %s=%v", f.Key, v)
	}
	if tr := e.Trace(); len(tr) > 1 {
		io.WriteString(w, "\ntrace:")
		for _, fr := range tr {
			fmt.Fprintf(w, "\n\t%s (%s:%d)", fr.Function, fr.File, fr.Line)
		}
	}
	for _, fr := range e.Frames() {
		fmt.Fprintf(w, "\n\t%s\n\t\t%s:%d", fr.Function, fr.File, fr.Line)
	}
	for _, sp := range e.suppressed {
		fmt.Fprintf(w, "\nsuppressed: %s", sp.Error())
	}
}
