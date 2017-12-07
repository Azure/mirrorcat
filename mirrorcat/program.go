package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"

	"github.com/marstr/randname"

	"github.com/marstr/mirrorcat"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

// MaxPayloadSize is the largest payload that GitHub would transmit. It is defined by GitHub
// and was written here on 12/6/2017 after reading this page: https://developer.github.com/webhooks/#payloads
const MaxPayloadSize = 5 * 1024 * 1024 // 1024 * 1024 = 1 MB

const mirrorRemoteHandle = "other"

func init() {
	if _, err := exec.LookPath("git"); err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("MIRRORCAT")

	viper.SetDefault("branches", []string{"master"})
	viper.SetDefault("port", 8080)

	viper.AddConfigPath(".")
	if home, err := homedir.Dir(); err == nil {
		viper.AddConfigPath(home)
	}
	viper.AutomaticEnv()
}

func handlePushEvent(output http.ResponseWriter, req *http.Request) {
	log.Println("Request Received")

	var pushed mirrorcat.PushEvent

	// Limited reader decorates Body to prevent DOS attacks which open
	// a request which will never be closed, or be closed after transmitting
	// a huge amount of data.
	payloadReader := &io.LimitedReader{
		R: req.Body,
		N: MaxPayloadSize,
	}

	payload, err := ioutil.ReadAll(payloadReader)
	if err != nil {
		fmt.Fprintln(output, "Unable to read the request.")
		return
	}

	err = json.Unmarshal(payload, &pushed)
	if err != nil {
		log.Println("Bad Request:\n", err.Error())
		output.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(output, "Body of request didn't conform to expected pattern of GitHub v3 PushEvent. See https://developer.github.com/v3/activity/events/types/#pushevent for expected format.")
		return
	}

	if !refIsMirrored(pushed.Ref) {
		log.Println("No-Op\nNot Configrured to mirror ", pushed.Ref)
		fmt.Fprintf(output, "Not configured to mirror %q\n", mirrorcat.NormalizeRef(pushed.Ref))
		return
	}

	cloneLoc := path.Join(os.TempDir(), randname.Generate())
	defer os.RemoveAll(cloneLoc)

	log.Println("Clone Location: ", cloneLoc)

	if err = exec.Command("git", "clone", viper.GetString("original"), cloneLoc).Run(); err != nil {
		output.WriteHeader(http.StatusInternalServerError)
		fmt.Println(output, "Unable to clone original")
		log.Println(output, "failed to clone:\n", err.Error())
		return
	}

	remoteAdder := exec.Command("git", "remote", "add", mirrorRemoteHandle, viper.GetString("mirror"))
	remoteAdder.Dir = cloneLoc

	if err = remoteAdder.Run(); err != nil {
		output.WriteHeader(http.StatusInternalServerError)
		fmt.Println(output, "Unable to assign mirror remote")
		log.Println(output, "failed to assign mirror remote:\n", err.Error())
		return
	}

	pusher := exec.Command("git", "push", mirrorRemoteHandle, mirrorcat.NormalizeRef(pushed.Ref))
	pusher.Dir = cloneLoc
	if err = pusher.Run(); err != nil {
		output.WriteHeader(http.StatusInternalServerError)
		fmt.Println(output, "Unable to push")
		log.Println("Unable to push:\n", err.Error())
		return
	}

	output.WriteHeader(http.StatusAccepted)
	log.Println("Request Completed.")
}

func main() {
	http.HandleFunc("/push", handlePushEvent)

	log.Printf("Listening on port %d\n", viper.GetInt("port"))
	log.Println("Original: ", viper.GetString("original"))
	log.Println("Mirror: ", viper.GetString("mirror"))
	if http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt("port")), nil) != nil {
		return
	}
}

func refIsMirrored(ref string) bool {
	ref = mirrorcat.NormalizeRef(ref)

	for _, mirrored := range viper.GetStringSlice("branches") {
		if mirrored == ref {
			return true
		}
	}
	return false
}
