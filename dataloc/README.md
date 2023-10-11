# dataloc

[![PkgGoDev](https://pkg.go.dev/badge/github.com/motemen/go-testutil/dataloc)](https://pkg.go.dev/github.com/motemen/go-testutil/dataloc)

Package dataloc provides functionality to find the source code location of table-driven test cases.

## Example

~~~go
import (
	"fmt"

	"github.com/motemen/go-testutil/dataloc"
)

func Example() {
	testcases := []struct {
		name string
		a, b int
		sum  int
	}{
		{
			name: "100+200",
			a:    100,
			b:    200,
			sum:  -1,
		},
		{
			name: "1+1",
			a:    1,
			b:    1,
			sum:  99,
		},
	}

	for _, testcase := range testcases {
		if expected, got := testcase.sum, testcase.a+testcase.b; got != expected {
			fmt.Printf("expected %d but got %d, test case at %s\n", expected, got, dataloc.L(testcase.name))
		}
	}

	// Output:
	// expected -1 but got 300, test case at example_test.go:15
	// expected 99 but got 2, test case at example_test.go:21
}
~~~

