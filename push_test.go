package mirrorcat_test

import (
	"fmt"

	"github.com/marstr/mirrorer"
)

func ExampleNormalizeRef() {
	fmt.Println(mirrorer.NormalizeRef("myBranch"))
	fmt.Println(mirrorer.NormalizeRef("remotes/origin/myBranch"))
	fmt.Println(mirrorer.NormalizeRef("refs/heads/myBranch"))
	// Output:
	// myBranch
	// myBranch
	// myBranch
}
