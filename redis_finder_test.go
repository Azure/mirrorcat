package mirrorcat_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Azure/mirrorcat"
	"github.com/go-redis/redis"
	"github.com/spf13/viper"
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

func TestRedisFinder_FindMirrors(t *testing.T) {
	viper.BindEnv("redis-connection", "MIRRORCAT_REDIS_CONNECTION")
	viper.SetDefault("redis-connection", "redis://localhost:6379")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Log("using redis connection:", viper.GetString("redis-connection"))
	connectionOptions, err := redis.ParseURL(viper.GetString("redis-connection"))
	if err != nil {
		t.Fatal(err)
	}

	const testKey = "master:testRepo"
	expected := map[mirrorcat.RemoteRef]struct{}{
		mirrorcat.RemoteRef{Repository: "testRepo", Ref: "dev"}:  struct{}{},
		mirrorcat.RemoteRef{Repository: "otherRepo", Ref: "dev"}: struct{}{},
	}

	expectedArr := make([]interface{}, 0, len(expected))
	for item := range expected {
		expectedArr = append(expectedArr, mirrorcat.RedisRemoteRef(item).String())
	}

	client := redis.NewClient(connectionOptions)
	_, err = client.SAdd(testKey, expectedArr...).Result()
	defer func() {
		_, err = client.Del(testKey).Result()
		if err != nil {
			t.Log("Unable to cleanup Redis Instance: ", err)
		} else {
			t.Log("Redis Instance cleaned up.")
		}
	}()

	if err != nil {
		t.Log("Unable to connect to Redis instance: ", err)
		t.SkipNow()
	}

	subject := mirrorcat.RedisFinder(*client)

	testRepo, err := mirrorcat.ParseRedisRemoteRef(testKey)
	if err != nil {
		t.Fatal(err)
	}

	results, errs := make(chan mirrorcat.RemoteRef), make(chan error, 1)

	go func() {
		select {
		case errs <- subject.FindMirrors(ctx, mirrorcat.RemoteRef(testRepo), results):
		case <-ctx.Done():
			errs <- ctx.Err()
		}
	}()

loop:
	for {
		select {
		case err = <-errs:
			if err != nil {
				t.Fatal(err)
			}
		case seen, ok := <-results:
			if !ok {
				break loop
			}

			_, ok = expected[seen]
			if ok {
				delete(expected, seen)
			} else {
				t.Log("unexpected result: ", seen)
				t.Fail()
			}
		}
	}

	for unseen := range expected {
		t.Log("didn't see expected value: ", unseen)
		t.Fail()
	}
}
