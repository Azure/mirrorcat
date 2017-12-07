package mirrorcat_test

import (
	"fmt"
	"testing"

	"github.com/marstr/mirrorcat"
)

func ExampleNormalizeRef() {
	fmt.Println(mirrorcat.NormalizeRef("myBranch"))
	fmt.Println(mirrorcat.NormalizeRef("remotes/origin/myBranch"))
	fmt.Println(mirrorcat.NormalizeRef("refs/heads/myBranch"))
	// Output:
	// myBranch
	// myBranch
	// myBranch
}

//TestNormalizeRef exists to test edge cases that would just be confusing in a Example block.
func TestNormalizeRef(t *testing.T) {
	testCases := []struct {
		string
		want string
	}{
		{"remotes/foo/myBranch", "myBranch"},
	}

	for _, tc := range testCases {
		t.Run(tc.string, func(t *testing.T) {
			if got := mirrorcat.NormalizeRef(tc.string); got != tc.want {
				t.Logf("got:  %q\nwant: %q", got, tc.want)
				t.Fail()
			}
		})
	}
}
