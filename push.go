package mirrorcat

import "strings"

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
