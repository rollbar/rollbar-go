package rollbar

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

func stderr(format string, args ...interface{}) {
	format = "Rollbar error: " + format + "\n"
	fmt.Fprintf(os.Stderr, format, args)
}

var (
	knownFilePathPatterns []string = []string{
		"github.com/",
		"code.google.com/",
		"bitbucket.org/",
		"launchpad.net/",
	}
)

// Remove un-needed information from the source file path. This makes them
// shorter in Rollbar UI as well as making them the same, regardless of the
// machine the code was compiled on. That way they can be used to calculate
// fingerprint for intelligent grouping of messages.
//
// 1. for Go standard library, paths look like:
// "/usr/local/go/src/pkg/runtime/proc.c", we leave:
// "pkg/runtime/proc.c"
// 2. for other code file paths look like:
// "/home/vagrant/GoPath/src/github.com/rollbar/rollbar.go", we leave:
// "github.com/rollbar/rollbar.go"
func shortenFilePath(s string) string {
	idx := strings.Index(s, "/src/pkg/")
	if idx != -1 {
		return s[idx+len("/src/"):]
	}
	for _, pattern := range knownFilePathPatterns {
		idx = strings.Index(s, pattern)
		if idx != -1 {
			return s[idx:]
		}
	}
	return s
}

func stacktraceFrames(skip int) []map[string]interface{} {
	frames := []map[string]interface{}{}

	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		file = shortenFilePath(file)
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
