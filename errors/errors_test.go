package errors

import (
	"fmt"
	"strings"
	"testing"

	pkgerr "github.com/pkg/errors"
)

func TestStackTracerOfPkgErrorsWithoutParent(t *testing.T) {
	err := pkgerr.New("")
	frs, ok := StackTracer(err)
	if !ok {
		t.Errorf("got: unsupported type")
	}

	fr := frs[0]
	if !strings.HasSuffix(fr.File, "rollbar-go/errors/errors_test.go") {
		t.Errorf("got: %s", fr.File)
	}
	if fr.Function != "github.com/rollbar/rollbar-go/errors.TestStackTracerOfPkgErrorsWithoutParent" {
		t.Errorf("got: %s", fr.Function)
	}
	if fr.Line != 12 {
		t.Errorf("got: %d", fr.Line)
	}
}

func TestStackTracerOfPkgErrorsWithParent(t *testing.T) {
	cause := fmt.Errorf("cause")
	effect := pkgerr.Wrap(cause, "effect")
	effect2 := pkgerr.Wrap(effect, "effect2")
	frs, ok := StackTracer(effect2)
	if !ok {
		t.Errorf("got: unsupported type")
	}

	fr := frs[0]
	if !strings.HasSuffix(fr.File, "rollbar-go/errors/errors_test.go") {
		t.Errorf("got: %s", fr.File)
	}
	if fr.Function != "github.com/rollbar/rollbar-go/errors.TestStackTracerOfPkgErrorsWithParent" {
		t.Errorf("got: %s", fr.Function)
	}
	if fr.Line != 33 {
		t.Errorf("got: %d", fr.Line)
	}
}
