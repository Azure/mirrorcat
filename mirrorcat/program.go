package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/Azure/mirrorcat"
	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

// MaxPayloadSize is the largest payload that GitHub would transmit. It is defined by GitHub
// and was written here on 12/6/2017 after reading this page: https://developer.github.com/webhooks/#payloads
const MaxPayloadSize = 5 * 1024 * 1024 // 1024 * 1024 = 1 MB

func init() {
	if _, err := exec.LookPath("git"); err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("MIRRORCAT")

	viper.SetConfigName(".mirrorcat")
	viper.AddConfigPath(".")
	if home, err := homedir.Dir(); err == nil {
		viper.AddConfigPath(home)
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		viper.AddConfigPath(path.Join(gopath, "src", "github.com", "Azure", "mirrorcat", "testdata"))
	}

	viper.SetDefault("port", 8080)

	viper.AutomaticEnv()
	viper.ReadInConfig()
	viper.WatchConfig()
	populateStaticMirrors()
	viper.OnConfigChange(func(in fsnotify.Event) {
		populateStaticMirrors()
	})

	log.Println("Used Config File: ", viper.ConfigFileUsed())
}

var staticMirrors = mirrorcat.NewDefaultMirrorFinder()

func populateStaticMirrors() {
	log.Println("Removing all Static Mirrors")
	staticMirrors.ClearAll()

	for origRepo, refs := range viper.Get("mirrors").(map[string]interface{}) {
		for origRef, mirrors := range refs.(map[string]interface{}) {
			original := mirrorcat.RemoteRef{
				Repository: origRepo,
				Ref:        origRef,
			}

			for remote, branches := range mirrors.(map[string]interface{}) {
				for _, remoteRef := range branches.([]interface{}) {
					mirror := mirrorcat.RemoteRef{
						Repository: remote,
						Ref:        remoteRef.(string),
					}

					staticMirrors.AddMirrors(original, mirror)
					log.Println("Adding Static Mirror:\n\t", original, "\n\t", mirror)
				}
			}
		}
	}
	//fmt.Fprintln(os.Stderr, viper.Get("mirrors"))
}

func handlePushEvent(output http.ResponseWriter, req *http.Request) {
	// After spinning for 10 minutes, give up
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*10)

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

	original := mirrorcat.RemoteRef{
		Repository: pushed.Repository.CloneURL,
		Ref:        mirrorcat.NormalizeRef(pushed.Ref),
	}
	mirrors := make(chan mirrorcat.RemoteRef)

	go staticMirrors.FindMirrors(ctx, original, mirrors)

	any := false
loop:
	for {
		select {
		case entry, ok := <-mirrors:
			if !ok {
				break loop
			}
			any = true

			err = mirrorcat.Push(ctx, original, entry)
			if err == nil {
				log.Println("Pushed", pushed.Ref, "at", pushed.Head.ID, "from ", original, " to ", entry)
			} else {
				output.WriteHeader(http.StatusInternalServerError)
				log.Println("Unable to complete push:\n ", err.Error())
			}
		case <-ctx.Done():
			output.WriteHeader(http.StatusRequestTimeout)
			log.Println(ctx.Err())
			return
		}
	}

	if any {
		output.WriteHeader(http.StatusAccepted)
	} else {
		output.WriteHeader(http.StatusOK)
	}

	log.Println("Request Completed.")
}

func main() {
	http.HandleFunc("/push", handlePushEvent)

	log.Printf("Listening on port %d\n", viper.GetInt("port"))

	if http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt("port")), nil) != nil {
		return
	}
}
