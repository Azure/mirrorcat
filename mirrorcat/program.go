package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"

	"github.com/spf13/viper"
)

// MaxPayloadSize is the largest payload that GitHub would transmit. It is defined by GitHub
// and was written here on 12/6/2017 after reading this page: https://developer.github.com/webhooks/#payloads
const MaxPayloadSize = 5 * 1024 * 1024 // 1024 * 1024 = 1 MB

const originalRemoteHandle = "origin"
const mirrorRemoteHandle = "other"

var (
	original string
	mirror   string
	branches []string
)

func init() {
	_, err := exec.LookPath("git")
	if err != nil {
		panic(err)
	}

	original = viper.GetString("original")
	mirror = viper.GetString("mirror")
	branches = viper.GetStringSlice("branches")

	exec.Command("git", "clone", original)

	exec.Command("git", "remote", "add", mirrorRemoteHandle, mirror)
}

func handlePushEvent(output http.ResponseWriter, req *http.Request) {
	var pushed PushEvent

	// Limited reader decorates Body to prevent DOS attacks which open
	// a request which will never be closed, or be closed after transmitting
	// a huge amount of data.
	payloadReader := &io.LimitedReader{
		R: req.Body,
		N: MaxPayloadSize,
	}

	payload, err := ioutil.ReadAll(payloadReader)
	if err != nil {
		return
	}

	err = json.Unmarshal(payload, &pushed)
	if err != nil {
		return
	}

}

func main() {
	http.HandleFunc("/push", handlePushEvent)
	if http.ListenAndServe(":8080", nil) != nil {
		return
	}
}

// RefIsMirrored
func RefIsMirrored(ref string) bool {
	ref = NormalizeRef(ref)

	for _, mirrored := range branches {
		strings.TrimPrefix(mirrored, strings.Join([]string{"remotes"}, "/"))
	}
}
