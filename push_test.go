package mirrorcat_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/Azure/mirrorcat"
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

func TestPushEvent_UnmarshalJSON(t *testing.T) {
	fileContent, err := os.Open(path.Join(".", "testdata", "examplePush.json"))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	limited := &io.LimitedReader{
		R: fileContent,
		N: 5 * 1024 * 1024,
	}

	limitedContent, err := ioutil.ReadAll(limited)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	var subject mirrorcat.PushEvent
	err = json.Unmarshal(limitedContent, &subject)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if want := "refs/heads/current"; subject.Ref != want {
		t.Logf("\ngot:  %q\nwant: %q", subject.Ref, want)
		t.Fail()
	}

	if want := "0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c"; subject.Head.ID != want {
		t.Logf("\ngot:  %q\nwant: %q", subject.Head.ID, want)
		t.Fail()
	}
}

//TestNormalizeRef exists to test edge cases that would just be confusing in a Example block.
func TestNormalizeRef(t *testing.T) {
	testCases := []struct {
		string
		want string
	}{
		{"remotes/foo/myBranch", "myBranch"},
		{"remotes/bar/a/b/c", "a/b/c"},
		{"refs/heads/a/b/c", "a/b/c"},
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
	locPrefix, err := ioutil.TempDir("", "mirrorcat_test")
	if err != nil {
		t.Error()
		t.FailNow()
	}

	originalLoc, mirrorLoc := path.Join(locPrefix, "leader"), path.Join(locPrefix, "follower")

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

	original := mirrorcat.RemoteRef{
		Repository: originalLoc,
		Ref:        "master",
	}

	mirror := mirrorcat.RemoteRef{
		Repository: mirrorLoc,
		Ref:        "master",
	}

	err = mirrorcat.Push(context.Background(), original, mirror, -1)
	if err != nil {
		t.Error(err)
		return
	}
}
