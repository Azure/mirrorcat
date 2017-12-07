package mirrorcat

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/marstr/randname"
)

// PushEvent encapsulates all data that will be provided by a GitHub Webhook PushEvent.
// Read more at: https://developer.github.com/v3/activity/events/types/#pushevent
type PushEvent struct {
	Ref          string   `json:"ref"`
	Before       string   `json:"before"`
	Size         int      `json:"size"`
	DistinctSize int      `json:"distinct_size"`
	Commits      []Commit `json:"commits"`
}

// Commit is an item detailed in the PushEvent page linked above, which contains metadata
// about commits that were pushed and that we're being informed of by a webhook.
type Commit struct {
	ID      string `json:"sha"`
	Message string `json:"message"`
	Author  `json:"author"`
	URL     string `json:"url"`
}

// Author holds the portion of a
type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// NormalizeRef removes metadata about the reference that was passed to us, and returns just it's name.
// Chiefly, this removes data about which repository the references belongs to, remote or local.
func NormalizeRef(ref string) string {
	ref = strings.TrimPrefix(ref, "refs/")
	if strings.HasPrefix(ref, "remotes/") {
		ref = strings.TrimPrefix(ref, "remotes/")
		ref = ref[strings.IndexRune(ref, '/')+1:]
	} else {
		ref = strings.TrimPrefix(ref, "heads/")
	}
	return ref
}

type CmdErr struct {
	error
	Output []byte
}

func (ce CmdErr) Error() string {
	builder := &bytes.Buffer{}

	fmt.Fprintln(builder, "Original Error: ", ce.error.Error())
	fmt.Fprintln(builder, "Command Output:\n", string(ce.Output))

	return builder.String()
}

// Push clones the original repository, then pushes the branch specified to another repository.
func Push(ctx context.Context, original, mirror, ref string) (err error) {
	const mirrorRemoteHandle = "other"

	cloneLoc := path.Join(os.TempDir(), randname.Generate())
	defer os.RemoveAll(cloneLoc)

	runCmd := func(cmd *exec.Cmd) (err error) {
		output, err := cmd.CombinedOutput()
		if err != nil {
			err = CmdErr{
				error:  err,
				Output: output,
			}
		}
		return
	}

	normalized := NormalizeRef(ref)

	if err = runCmd(exec.CommandContext(ctx, "git", "clone", original, cloneLoc)); err != nil {
		return
	}

	checkouter := exec.CommandContext(ctx, "git", "checkout", normalized)
	checkouter.Dir = cloneLoc

	if err = runCmd(checkouter); err != nil {
		return
	}

	remoteAdder := exec.CommandContext(ctx, "git", "remote", "add", mirrorRemoteHandle, mirror)
	remoteAdder.Dir = cloneLoc

	if err = runCmd(remoteAdder); err != nil {
		return
	}

	log.Println("Pushing ", mirrorRemoteHandle, normalized)
	pusher := exec.CommandContext(ctx, "git", "push", mirrorRemoteHandle, normalized)
	pusher.Dir = cloneLoc
	err = runCmd(pusher)
	return
}
