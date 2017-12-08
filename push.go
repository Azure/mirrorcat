package mirrorcat

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

// PushEvent encapsulates all data that will be provided by a GitHub Webhook PushEvent.
// Read more at: https://developer.github.com/v3/activity/events/types/#pushevent
type PushEvent struct {
	Ref          string     `json:"ref"`
	Before       string     `json:"before"`
	Size         int        `json:"size"`
	DistinctSize int        `json:"distinct_size"`
	Commits      []Commit   `json:"commits"`
	Head         Commit     `json:"head_commit"`
	Repository   Repository `json:"repository"`
	Pusher       Identity   `json:"pusher"`
}

// Commit is an item detailed in the PushEvent page linked above, which contains metadata
// about commits that were pushed and that we're being informed of by a webhook.
type Commit struct {
	ID      string   `json:"id"`
	Message string   `json:"message"`
	Author  Identity `json:"author"`
	URL     string   `json:"url"`
}

// Identity holds metadata about a GitHub Author
type Identity struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// Repository holds metadat about a GitHub URL.
type Repository struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	SSHURL   string `json:"ssh_url"`
	GITURL   string `json:"git_url"`
	CloneURL string `json:"clone_url"`
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

// CmdErr allows the Output of a completed CMD to be included with the error itself.
// This is useful for capturing Failure messages communicated through /dev/stderr
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
func Push(ctx context.Context, original, mirror RemoteRef) (err error) {
	const mirrorRemoteHandle = "other"

	cloneLoc, err := ioutil.TempDir("", "mirrorcat")
	if err != nil {
		return
	}
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

	if err = runCmd(exec.CommandContext(ctx, "git", "clone", original.Repository, cloneLoc)); err != nil {
		return
	}

	remoteAdder := exec.CommandContext(ctx, "git", "remote", "add", mirrorRemoteHandle, mirror.Repository)
	remoteAdder.Dir = cloneLoc

	if err = runCmd(remoteAdder); err != nil {
		return
	}

	pusher := exec.CommandContext(ctx, "git", "push", mirrorRemoteHandle, fmt.Sprintf("%s:%s", NormalizeRef(original.Ref), NormalizeRef(mirror.Ref)))
	pusher.Dir = cloneLoc
	err = runCmd(pusher)
	return
}
