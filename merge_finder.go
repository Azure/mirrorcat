package mirrorcat

import "context"

// MergeFinder allows a mechanism to find mirror mappings from multiple underlying MirrorFinders.
type MergeFinder []MirrorFinder

// FindMirrors enumerates each MirrorFinder, and
func (haystack MergeFinder) FindMirrors(ctx context.Context, needle RemoteRef, results chan<- RemoteRef) (err error) {
	defer close(results)

	for _, finder := range haystack {
		// Due to the fact that all FindMirrors implementations must close the results channel to communicate
		// that no more matches have been found, we must create a layer of separation between the merged results
		// and the results from each child MirrorFinder.
		intermediate := make(chan RemoteRef)

		errs := make(chan error, 1)

		// Kick-off a goroutine to fetch the child's matching mirrors.
		go func() {
			select {
			case errs <- finder.FindMirrors(ctx, needle, intermediate):
				// Intentionally Left Blank
			case <-ctx.Done():
				// This case prevents leaking this goroutine in the case that the underlying type
				// of `finder` does not respect cancellation tokens appropriately.
				errs <- ctx.Err()
			}
		}()

		for mirror := range intermediate {
			select {
			case results <- mirror:
				// Intentionally Left Blank
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err = <-errs
		if err != nil {
			return
		}
	}
	return
}
