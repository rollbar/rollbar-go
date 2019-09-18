package errors

import (
	"runtime"

	pkgerr "github.com/pkg/errors"
)

// StackTracer is able to extract stack traces from pkg/errors.
func StackTracer(err error) ([]runtime.Frame, bool) {
	type stackTracer interface {
		StackTrace() pkgerr.StackTrace
	}

	switch x := err.(type) {
	case stackTracer:
		st := x.StackTrace()
		pcs := make([]uintptr, len(st))
		for i, pc := range st {
			pcs[i] = uintptr(pc)
		}
		fr := runtime.CallersFrames(pcs)

		return framesToSlice(fr), true
	}

	return nil, false
}

// framesToSlice extracts all the runtime.Frame from runtime.Frames.
// This function has been copied from transform.go in rollbar-go.
func framesToSlice(fr *runtime.Frames) []runtime.Frame {
	frames := make([]runtime.Frame, 0)

	for frame, more := fr.Next(); frame != (runtime.Frame{}); frame, more = fr.Next() {
		frames = append(frames, frame)

		if !more {
			break
		}
	}

	return frames
}
