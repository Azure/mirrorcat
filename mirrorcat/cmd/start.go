package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Azure/mirrorcat"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the MirrorCat server on localhost",
	// 	Long: `A longer description that spans multiple lines and likely contains examples
	// and usage of using your command. For example:

	// Cobra is a CLI library for Go that empowers applications.
	// This application is a tool to generate the needed files
	// to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		var host string
		if reportedHost, err := os.Hostname(); err == nil {
			host = reportedHost
		} else {
			host = "Unknown Host"
		}

		log.SetPrefix(fmt.Sprintf("[MirrorCat on %s]", host))

		http.HandleFunc("/push/github", handleGitHubPushEvent)

		port := viper.GetInt("port")
		log.Printf("Listening on port %d\n", port)

		if http.ListenAndServe(fmt.Sprintf(":%d", port), nil) != nil {
			return
		}
	},
}

// DefaultPort is the port that will be used by default as MirrorCat is started.
const DefaultPort uint = 8080

// DefaultCloneDepth is the number of commits that will be checked out if one is
// not specified by the invoker of MirrorCat.
// Most MirrorCat functions will treat negative values as indicative of needing to
// clone the entire repository. However, at their discretion, they may try to use
// information local to it to optimize its performance.
// However, if this value is greater than zero, MirrorCat function implementers are
// advised to respect it.
const DefaultCloneDepth = -1

func init() {
	RootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	startCmd.Flags().UintP("port", "p", 0, "The port that should be used to serve the MirrorCat service on.")
	viper.BindPFlag("port", startCmd.Flags().Lookup("port"))

	startCmd.Flags().UintP("clone-depth", "c", 0, "The number of commits to checkout while cloning the original repository. (The default behavior is to clone all of the commits in the original repository.)")
	viper.BindPFlag("clone-depth", startCmd.Flags().Lookup("clone-depth"))

	viper.SetDefault("port", DefaultPort)
	viper.SetDefault("clone-depth", DefaultCloneDepth)
}

func handleGitHubPushEvent(output http.ResponseWriter, req *http.Request) {
	// MaxPayloadSize is the largest payload that GitHub would transmit. It is defined by GitHub
	// and was written here on 12/6/2017 after reading this page: https://developer.github.com/webhooks/#payloads
	const MaxPayloadSize = 5 * 1024 * 1024 // 1024 * 1024 = 1 MB

	// After spinning for 10 minutes, give up
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()

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

			err = mirrorcat.Push(ctx, original, entry, viper.GetInt("clone-depth"))
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
}
