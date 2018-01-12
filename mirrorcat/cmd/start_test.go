package cmd_test

import (
	"context"
	"testing"

	"github.com/spf13/viper"

	"github.com/Azure/mirrorcat/mirrorcat/cmd"
)

func TestFetchIdentity(t *testing.T) {
	if !viper.IsSet("github-auth-token") || viper.GetString("github-auth-token") == "" {
		t.Log("Unable to find environment variable defining Auth Token")
		t.SkipNow()
	}

	result, err := cmd.FetchGitHubIdentity(context.Background(), viper.GetString("github-auth-token"))
	if err != nil {
		t.Error(err)
	} else if result == "" {
		t.Log("Personal Access Token not associated with any user.")
		t.Fail()
	} else {
		t.Logf("Personal Access Token associated with user %q", result)
	}

}
