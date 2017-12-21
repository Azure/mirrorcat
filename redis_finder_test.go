package mirrorcat_test

import (
	"fmt"
	"testing"

	"github.com/Azure/mirrorcat"
)

func ExampleRedisRemoteRef_String() {
	subject := mirrorcat.RemoteRef{
		Repository: "https://github.com/Azure/mirrorcat",
		Ref:        "master",
	}

	marshaled := mirrorcat.RedisRemoteRef(subject).String()
	fmt.Println(marshaled)

	// Output: master:https://github.com/Azure/mirrorcat
}

func ExampleParseRedisRemoteRef() {
	unmarshaled, err := mirrorcat.ParseRedisRemoteRef("master:https://github.com/Azure/mirrorcat")
	if err != nil {
		return
	}

	fmt.Println("Repository:", unmarshaled.Repository)
	fmt.Println("Ref:", unmarshaled.Ref)

	// Output:
	// Repository: https://github.com/Azure/mirrorcat
	// Ref: master
}

func TestParseRedisRemoteRef(t *testing.T) {
	testCases := []struct {
		string
		want mirrorcat.RedisRemoteRef
	}{
		{":", mirrorcat.RedisRemoteRef{}},
		{"left:right", mirrorcat.RedisRemoteRef{Ref: "left", Repository: "right"}},
		{"branch:https://hostname:1234/folk?person=Pete%20Seeger", mirrorcat.RedisRemoteRef{Ref: "branch", Repository: "https://hostname:1234/folk?person=Pete%20Seeger"}},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			unmarshaled, err := mirrorcat.ParseRedisRemoteRef(tc.string)

			if err != nil {
				t.Log("unexpected err: ", err)
				t.Fail()
			}

			if unmarshaled.Repository != tc.want.Repository {
				t.Logf("got: %q want: %q", unmarshaled.Repository, tc.want.Repository)
				t.Fail()
			}

			if unmarshaled.Ref != tc.want.Ref {
				t.Logf("got: %q want: %q", unmarshaled.Ref, tc.want.Ref)
				t.Fail()
			}
		})
	}
}

func TestParseRedisRemoteRef_Invalid(t *testing.T) {
	testCases := []string{
		"",
		"github.com/Azure/mirrorcat",
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			unmarshaled, err := mirrorcat.ParseRedisRemoteRef(tc)
			if err == nil {
				t.Log("expected a non-nil error for:", tc)
				t.Log("got:", err)
				t.Fail()
			}
			if unmarshaled.Repository != "" || unmarshaled.Ref != "" {
				t.Log("expected the empty RedisRemoteRef for:", tc)
				t.Fail()
			}
		})
	}
}
