package mirrorcat_test

import (
	"context"
	"fmt"
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

	ctx, _ := context.WithTimeout(context.Background(), 500*time.Millisecond)
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

	subject := mirrorcat.NewDefaultMirrorFinder()
	subject.AddMirrors(original, mirrors...)

	ctx, cancel := context.WithCancel(context.Background())

	results := make(chan mirrorcat.RemoteRef)
	go subject.FindMirrors(ctx, original, results)

	cancel()

	// There's a race-condition imposed here, because FindMirrors will race the cancel()
	// function to see which gets to the read first. By waiting for 20 milliseconds, we can
	// pretty well ensure that cancel() wins the race. (cancel needs to complete before
	// the read/write handshake happens on results between this function and FindMirrors)
	<-time.After(20 * time.Millisecond)

	select {
	case _, ok := <-results:
		if ok {
			t.Logf("Able to read from results, and the channel was not closed.")
			t.Fail()
		}
	default:
		t.Logf("Unable to read from results, it was not closed.")
		t.Fail()
	}
}
