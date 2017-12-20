package mirrorcat_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Azure/mirrorcat"
)

func ExampleMergeFinder_FindMirrors() {
	const mainRepo = "github.com/Azure/mirrorcat"
	const secondaryRepo = "github.com/marstr/mirrorcat"

	child1, child2 := mirrorcat.NewDefaultMirrorFinder(), mirrorcat.NewDefaultMirrorFinder()

	orig := mirrorcat.RemoteRef{Repository: mainRepo, Ref: "master"}

	child1.AddMirrors(orig, mirrorcat.RemoteRef{Repository: secondaryRepo, Ref: "master"})
	child2.AddMirrors(orig, mirrorcat.RemoteRef{Repository: mainRepo, Ref: "dev"})

	subject := mirrorcat.MergeFinder([]mirrorcat.MirrorFinder{child1, child2})

	results := make(chan mirrorcat.RemoteRef)

	var err error
	go func() {
		err = subject.FindMirrors(context.Background(), orig, results)
	}()

	for result := range results {
		fmt.Printf("%s:%s\n", result.Repository, result.Ref)
	}
	fmt.Println(err)

	// Output:
	// github.com/marstr/mirrorcat:master
	// github.com/Azure/mirrorcat:dev
	// <nil>
}

func TestMergeFinder_FindMirrors_RespectsCancel(t *testing.T) {
	const mainRepo = "github.com/Azure/mirrorcat"
	const secondaryRepo = "github.com/marstr/mirrorcat"

	child1, child2 := mirrorcat.NewDefaultMirrorFinder(), mirrorcat.NewDefaultMirrorFinder()

	orig := mirrorcat.RemoteRef{Repository: mainRepo, Ref: "master"}

	child1.AddMirrors(orig, mirrorcat.RemoteRef{Repository: secondaryRepo, Ref: "master"})
	child2.AddMirrors(orig, mirrorcat.RemoteRef{Repository: mainRepo, Ref: "dev"})

	subject := mirrorcat.MergeFinder([]mirrorcat.MirrorFinder{child1, child2})

	outside, cancelOutside := context.WithTimeout(context.Background(), time.Second*3)
	defer cancelOutside()

	inside, cancelInside := context.WithCancel(outside)

	results := make(chan mirrorcat.RemoteRef)

	errs := make(chan error, 1)

	go func() {
		errs <- subject.FindMirrors(inside, orig, results)
		t.Log("Finished FindMirrors routine")
	}()

	cancelInside()

	select {
	case err := <-errs:
		t.Log("error received: ", err)
		if err == nil || !strings.Contains(err.Error(), "cancel") {
			t.Log("expected error to be a cancellation message")
			t.Fail()
		}
	case <-outside.Done():
		t.Errorf("timed out")
	}
}
