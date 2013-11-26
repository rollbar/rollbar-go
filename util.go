package rollbar

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

func stderr(s string) {
	fmt.Fprintf(os.Stderr, "Rollbar error: %s\n", s)
}

func stacktraceFrames(skip int) []map[string]interface{} {
	frames := []map[string]interface{}{}

	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		frames = append(frames, map[string]interface{}{
			"filename": file,
			"lineno":   line,
			"method":   functionName(pc),
		})
	}

	return frames
}

func functionName(pc uintptr) string {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "???"
	}
	parts := strings.Split(fn.Name(), string(os.PathSeparator))
	return parts[len(parts)-1]
}
