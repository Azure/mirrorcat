package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"

	"github.com/Azure/mirrorcat"
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

	viper.SetConfigName(".mirrorcat")
	viper.SetConfigType("yaml")

	viper.SetEnvPrefix("MIRRORCAT")

	viper.SetDefault("branches", []string{"master"})
	viper.SetDefault("port", 8080)

	viper.AddConfigPath(".")
	if home, err := homedir.Dir(); err == nil {
		viper.AddConfigPath(home)
	}

	viper.AutomaticEnv()
	viper.ReadInConfig()
	log.Println("Used Config File: ", viper.ConfigFileUsed())
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

	original := viper.GetString("original")
	mirror := viper.GetString("mirror")

	err = mirrorcat.Push(context.Background(), original, mirror, pushed.Ref)
	if err == nil {
		log.Println("Pushed", pushed.Ref, "at", pushed.Head.ID, "from", original, "to", mirror)
	} else {
		output.WriteHeader(http.StatusInternalServerError)
		log.Println("Unable to complete push:\n ", err.Error())
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
