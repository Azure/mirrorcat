package mirrorcat

import (
	"context"
	"sync"
)

// RemoteRef combines a location with a branch name.
type RemoteRef struct {
	Repository string
	Ref        string
}

// MirrorFinder provides an abstraction for communication which branches
// on which repositories are mirrors of others.
type MirrorFinder interface {
	FindMirrors(context.Context, RemoteRef, chan<- RemoteRef) error
}

// DefaultMirrorFinder provides an in-memory location for storing information about
// which branches should mirror which others.
//
// Note: This is most powerful if combined with github.com/spf13/viper and its Watch
// 	functionality
type DefaultMirrorFinder struct {
	sync.RWMutex
	underlyer map[RemoteRef][]RemoteRef
}

// NewDefaultMirrorFinder creates an empty instance of a MirrorFinder
func NewDefaultMirrorFinder() *DefaultMirrorFinder {
	return &DefaultMirrorFinder{
		underlyer: make(map[RemoteRef][]RemoteRef),
	}
}

// AddMirrors registers a new remote and branch that should be cloned whenever
func (dmf *DefaultMirrorFinder) AddMirrors(original RemoteRef, branches ...RemoteRef) {
	dmf.Lock()
	defer dmf.Unlock()

	dmf.underlyer[original] = append(dmf.underlyer[original], branches...)
}

// ClearMirrors removes the association between a particular `RemoteRef` and all mirrored copies.
func (dmf *DefaultMirrorFinder) ClearMirrors(original RemoteRef) {
	dmf.Lock()
	defer dmf.Unlock()

	delete(dmf.underlyer, original)
}

// ClearAll removes all associations between References
func (dmf *DefaultMirrorFinder) ClearAll() {
	dmf.underlyer = make(map[RemoteRef][]RemoteRef)
}

// FindMirrors iterates through the entries that had been added and publishes them all to `results`
// See Also: AddMirror(url.URL, branches ...string)
func (dmf *DefaultMirrorFinder) FindMirrors(ctx context.Context, original RemoteRef, results chan<- RemoteRef) error {
	dmf.RLock()
	defer dmf.RUnlock()
	defer close(results)

	for _, m := range dmf.underlyer[original] {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case results <- m:
			// Intentionally Left Blank
		}
	}
	return nil
}
