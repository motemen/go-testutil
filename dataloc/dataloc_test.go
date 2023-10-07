package dataloc_test

import (
	"fmt"
	"runtime"
	"testing"

	// calling by dataloc.L() is important; L() without package name won't work
	"github.com/motemen/go-testutil/dataloc"
)

var file = "dataloc_test.go"

func __line__() int {
	_, _, line, _ := runtime.Caller(1)
	return line
}

func TestL_caseTypeInsideFunc(t *testing.T) {
	type testcaseInsideFunc struct {
		name string
		line int
	}

	tests := []testcaseInsideFunc{
		{name: "keyed", line: __line__()},
		{"unkeyed", __line__()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got, expected := dataloc.L(test.name), fmt.Sprintf("%s:%d", file, test.line); got != expected {
				t.Errorf("expected %q, got %q", expected, got)
			}
		})
	}
}

type testcaseOutsideFunc struct {
	line        int
	description string
}

func TestL_caseTypeOutsideFunc(t *testing.T) {
	tests := []testcaseOutsideFunc{
		{description: "keyed", line: __line__()},
		{__line__(), "unkeyed"},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if got, expected := dataloc.L(test.description), fmt.Sprintf("%s:%d", file, test.line); got != expected {
				t.Errorf("expected %q, got %q", expected, got)
			}
		})
	}
}

func TestL_caseTypeInline(t *testing.T) {
	tests := []struct {
		name string
		line int
	}{
		{name: "keyed", line: __line__()},
		{"unkeyed", __line__()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got, expected := dataloc.L(test.name), fmt.Sprintf("%s:%d", file, test.line); got != expected {
				t.Errorf("expected %q, got %q", expected, got)
			}
		})
	}
}
