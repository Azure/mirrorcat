package mirrorcat_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Azure/mirrorcat"
)

func ExampleDefaultMirrorFinder() {
	original := mirrorcat.RemoteRef{
		Repository: "https://github.com/Azure/mirrorcat",
		Ref:        "master",
	}

	mirrors := []mirrorcat.RemoteRef{
		{
			Repository: "https://github.com/marstr/mirrorcat",
			Ref:        "master",
		},
		{
			Repository: "https://github.com/haydenmc/mirrorcat",
			Ref:        "master",
		},
	}

	subject := mirrorcat.NewDefaultMirrorFinder()
	subject.AddMirrors(original, mirrors...)

	results := make(chan mirrorcat.RemoteRef)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go subject.FindMirrors(ctx, original, results)

loop:
	for {
		select {
		case entry, ok := <-results:
			if ok {
				fmt.Println(entry.Repository, entry.Ref)
			} else {
				break loop
			}
		case <-ctx.Done():
			break loop
		}
	}

	// Output:
	// https://github.com/marstr/mirrorcat master
	// https://github.com/haydenmc/mirrorcat master
}

func TestDefaultMirrorFinder_FindMirrors_RespectsCancellation(t *testing.T) {
	original := mirrorcat.RemoteRef{
		Repository: "https://github.com/Azure/mirrorcat",
		Ref:        "master",
	}

	mirrors := []mirrorcat.RemoteRef{
		{
			Repository: "https://github.com/marstr/mirrorcat",
			Ref:        "master",
		},
		{
			Repository: "https://github.com/haydenmc/mirrorcat",
			Ref:        "master",
		},
	}

	outer, cancelOuter := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelOuter()

	subject := mirrorcat.NewDefaultMirrorFinder()
	subject.AddMirrors(original, mirrors...)

	inner, cancelInner := context.WithCancel(outer)

	results := make(chan mirrorcat.RemoteRef)
	errs := make(chan error, 1)
	go func() {
		errs <- subject.FindMirrors(inner, original, results)
	}()

	cancelInner()

	select {
	case err := <-errs:
		t.Log("error received: ", err)
		if err == nil || !strings.Contains(err.Error(), "cancel") {
			t.Log("expected error to be a cancellation message")
			t.Fail()
		}
	case <-outer.Done():
		t.Error("timed out")
	}
}
