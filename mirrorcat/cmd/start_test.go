package cmd_test

import (
	"context"
	"os"
	"testing"

	"github.com/Azure/mirrorcat/mirrorcat/cmd"
)

func TestFetchIdentity(t *testing.T) {
	// Personal Access Token (pat)
	var pat string

	tokenEnvs := []string{"MIRRORCAT_GITHUB_AUTH_TOKEN", "GITHUB_AUTH_TOKEN"}

	for _, envVar := range tokenEnvs {
		if val := os.Getenv(envVar); val != "" {
			pat = val
			break
		}
	}

	if "" == pat {
		t.Log("Unable to find environment variable defining Auth Token")
		t.SkipNow()
	}

	result, err := cmd.FetchGitHubIdentity(context.Background(), pat)
	if err != nil {
		t.Error(err)
	} else if result == "" {
		t.Log("Personal Access Token not associated with any user.")
		t.Fail()
	} else {
		t.Logf("Personal Access Token associated with user %q", result)
	}

}
