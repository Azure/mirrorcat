package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Azure/mirrorcat"
	"github.com/go-redis/redis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the MirrorCat server on localhost",
	Args:  cobra.NoArgs,

	// 	Long: `A longer description that spans multiple lines and likely contains examples
	// and usage of using your command. For example:

	// Cobra is a CLI library for Go that empowers applications.
	// This application is a tool to generate the needed files
	// to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		populateStaticMirrors()
		var host string
		if reportedHost, err := os.Hostname(); err == nil {
			host = reportedHost
		} else {
			host = "Unknown Host"
		}

		log.Printf("Starting MirrorCat\n\tBuilt using commit %q", commit)

		log.SetPrefix(fmt.Sprintf("[MirrorCat on %s]", host))

		http.HandleFunc("/push/github", handleGitHubPushEvent)

		port := viper.GetInt("port")
		log.Printf("Listening on port %d\n", port)

		if options, err := redis.ParseURL(viper.GetString("redis-connection")); err != nil {
			log.Println("Unable to connect to Redis Because: ", err)
		} else {
			client := redis.NewClient(options)

			go func() {
				log.Print("Connecting to Redis at ", options.Addr)

				allMirrors = append(allMirrors, mirrorcat.RedisFinder(*client))

				_, err := client.Keys("*").Result()
				if err != nil {
					log.Print("Unable to connect to Redis because: ", err)
				} else {
					log.Print("Successfully connected to Redis.")
				}
			}()
		}

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

type WrittenTuple struct {
	Original mirrorcat.RemoteRef `json:"original"`
	Mirror   mirrorcat.RemoteRef `json:"mirror"`
	CommitID string              `json:"commitID"`
}

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

	startCmd.Flags().StringP("redis-connection", "r", "", "The host to contact Redis with, if it's relevant.")
	viper.BindPFlag("redis-connection", startCmd.Flags().Lookup("redis-connection"))

	viper.SetDefault("port", DefaultPort)
	viper.SetDefault("clone-depth", DefaultCloneDepth)
}

func handleGitHubPushEvent(resp http.ResponseWriter, req *http.Request) {
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
		fmt.Fprintln(resp, "Unable to read the request.")
		return
	}

	err = json.Unmarshal(payload, &pushed)
	if err != nil {
		log.Println("Bad Request:\n", err.Error())
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(resp, "Body of request didn't conform to expected pattern of GitHub v3 PushEvent. See https://developer.github.com/v3/activity/events/types/#pushevent for expected format.")
		return
	}

	original := mirrorcat.RemoteRef{
		Repository: pushed.Repository.CloneURL,
		Ref:        mirrorcat.NormalizeRef(pushed.Ref),
	}
	mirrors := make(chan mirrorcat.RemoteRef)

	go allMirrors.FindMirrors(ctx, original, mirrors)

	bodyWriter := json.NewEncoder(resp)
loop:
	for {
		select {
		case entry, ok := <-mirrors:
			if !ok {
				break loop
			}

			err = mirrorcat.Push(ctx, original, entry, viper.GetInt("clone-depth"))
			if err == nil {
				bodyWriter.Encode(WrittenTuple{
					Original: original,
					Mirror:   entry,
					CommitID: pushed.Head.ID,
				})
				log.Println("Pushed", pushed.Ref, "at", pushed.Head.ID, "from ", original, " to ", entry)
			} else {
				resp.WriteHeader(http.StatusInternalServerError)
				log.Println("Unable to complete push:\n ", err.Error())
			}
		case <-ctx.Done():
			resp.WriteHeader(http.StatusRequestTimeout)
			log.Println(ctx.Err())
			return
		}
	}
	log.Println("Request Completed.")
}

var allMirrors = mirrorcat.MergeFinder{staticMirrors}
var staticMirrors = mirrorcat.NewDefaultMirrorFinder()

var populateStaticMirrors = func() func() error {
	var populating sync.Mutex

	return func() error {
		populating.Lock()
		defer populating.Unlock()

		if !viper.InConfig("mirrors") {
			return errors.New("no `mirrors` property found")
		}

		originalRepos, ok := viper.Get("mirrors").(map[string]interface{})
		if !ok {
			return errors.New("`mirrors` was in an unexpected format")
		}

		log.Println("Removing all Static Mirrors")
		staticMirrors.ClearAll()

		for origRepo, refs := range originalRepos {
			originalRefs, ok := refs.(map[string]interface{})
			if !ok {
				log.Printf("skipping because key %q was in an unexpected format.", origRepo)
				continue
			}

			for origRef, mirrors := range originalRefs {
				original := mirrorcat.RemoteRef{
					Repository: origRepo,
					Ref:        origRef,
				}

				remoteRepos, ok := mirrors.(map[string]interface{})
				if !ok {
					log.Printf("skipping because key %q was in an unexpected format.", origRef)
					continue
				}

				for remote, branches := range remoteRepos {

					remoteRefs, ok := branches.([]interface{})
					if !ok {
						log.Printf("skipping because key %q was in an unexpected format.", remote)
						continue
					}

					for _, remoteRef := range remoteRefs {
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
		return nil
	}
}()
