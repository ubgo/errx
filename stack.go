package errx

import "runtime"

const maxStackDepth = 32

// callers captures the program counters of the stack at the construction
// site, skipping `skip` extra frames plus callers itself and the
// constructor. Symbolization is deferred to Frames() — capture is a single
// runtime.Callers, the cheap path criticized libraries lack.
func callers(skip int) []uintptr {
	var pcs [maxStackDepth]uintptr
	// runtime.Callers(2) starts at the constructor (New/Wrap/...); +skip=1
	// from the constructor lands the first frame on its caller — the user
	// site we want as the origin (also keeps the fingerprint stable).
	n := runtime.Callers(2+skip, pcs[:])
	if n == 0 {
		return nil
	}
	out := make([]uintptr, n)
	copy(out, pcs[:n])
	return out
}

// wrapCaller captures a single PC at the Wrap site for the return-trace.
func wrapCaller(skip int) uintptr {
	var pcs [1]uintptr
	if runtime.Callers(2+skip, pcs[:]) == 0 {
		return 0
	}
	return pcs[0]
}

// Frame is a symbolized stack frame.
type Frame struct {
	Function string
	File     string
	Line     int
}

// Trace returns the return-trace: one frame per errx layer that wrapped the
// error, innermost-first — the path the error traveled up the call stack,
// distinct from the origin Stack. Cheap (one PC per Wrap) and works across
// goroutine boundaries since it is recorded on the value, not sampled.
func (e *Error) Trace() []Frame {
	var pcs []uintptr
	for cur := e; cur != nil; {
		if cur.wrapPC != 0 {
			pcs = append(pcs, cur.wrapPC)
		}
		next, ok := cur.cause.(*Error)
		if !ok {
			break
		}
		cur = next
	}
	if len(pcs) == 0 {
		return nil
	}
	return symbolize(pcs)
}

func symbolize(pcs []uintptr) []Frame {
	cf := runtime.CallersFrames(pcs)
	out := make([]Frame, 0, len(pcs))
	for {
		fr, more := cf.Next()
		if fr.PC != 0 {
			out = append(out, Frame{Function: fr.Function, File: fr.File, Line: fr.Line})
		}
		if !more {
			break
		}
	}
	return out
}

// Frames symbolizes the captured origin stack lazily. Returns nil if the
// error carries no stack.
func (e *Error) Frames() []Frame {
	if e == nil {
		return nil
	}
	if len(e.pcs) == 0 {
		// Reconstructed from the wire: return the frames carried across.
		return e.decoded
	}
	return symbolize(e.pcs)
}

// originFrame returns the topmost user frame, used by the fingerprint.
func (e *Error) originFrame() Frame {
	fs := e.Frames()
	if len(fs) == 0 {
		return Frame{}
	}
	return fs[0]
}
