package mirrorcat_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/marstr/mirrorcat"
	"github.com/marstr/randname"
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

func TestPush(t *testing.T) {
	var err error
	locPrefix := path.Join(os.TempDir(), "test_push")
	originalLoc, mirrorLoc := path.Join(locPrefix, randname.Generate()), path.Join(locPrefix, randname.Generate())

	t.Log("Original Repo Location: \t", originalLoc)
	t.Log("Mirror Repo Location:   \t", mirrorLoc)

	runCmd := func(cmd *exec.Cmd) {
		err = cmd.Run()
		if err != nil {
			output, _ := cmd.CombinedOutput()
			t.Log(output)
			t.Error(err)
			t.FailNow()
		}
	}

	runCmd(exec.Command("git", "init", originalLoc))
	defer os.RemoveAll(originalLoc)

	runCmd(exec.Command("git", "init", "--bare", mirrorLoc))
	defer os.RemoveAll(mirrorLoc)

	err = ioutil.WriteFile(path.Join(originalLoc, "content.txt"), []byte("Hello World!!!"), os.ModePerm)
	if err != nil {
		t.Error(err)
		return
	}

	adder := exec.Command("git", "add", "--all")
	adder.Dir = originalLoc
	runCmd(adder)

	commiter := exec.Command("git", "commit", "-m", `"This is only a test."`)
	commiter.Dir = originalLoc
	runCmd(commiter)

	err = mirrorcat.Push(context.Background(), originalLoc, mirrorLoc, "master")
	if err != nil {
		t.Error(err)
		return
	}
}
