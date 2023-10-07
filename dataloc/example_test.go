package dataloc_test

import (
	"fmt"

	"github.com/motemen/go-testutil/dataloc"
)

func Example() {
	testcases := []struct {
		name string
		a    int
		b    int
		sum  int
	}{
		{
			name: "100+200",
			a:    100, b: 200, sum: -1,
		},
		{"1+1", 1, 1, 99},
	}

	for _, testcase := range testcases {
		if expected, got := testcase.sum, testcase.a+testcase.b; got != expected {
			fmt.Printf("expected %d but got %d, case at %s\n", expected, got, dataloc.L(testcase.name))
		}
	}

	// Output:
	// expected -1 but got 300, case at example_test.go:16
	// expected 99 but got 2, case at example_test.go:20
}
