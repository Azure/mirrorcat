package mirrorcat

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-redis/redis"
)

// RedisFinder implementes the MirrorFinder interface against a Redis Cache.
type RedisFinder redis.Client

// RedisRemoteRef allows easy conversion to `string` from a `mirrorcat.RemoteRef`.
type RedisRemoteRef RemoteRef

// ParseRedisRemoteRef reads a string formatted as a RedisRemoteRef and reads it into
// a `mirrorcat.RemoteRef`.
func ParseRedisRemoteRef(input string) (RedisRemoteRef, error) {
	splitPoint := strings.IndexRune(input, ':')
	if splitPoint < 0 {
		return RedisRemoteRef{}, fmt.Errorf("%q does not resemble a RedisRemoteRef", input)
	}

	return RedisRemoteRef{
		Ref:        input[:splitPoint],
		Repository: input[splitPoint+1:],
	}, nil
}

func (rrr RedisRemoteRef) String() string {
	return fmt.Sprintf("%s:%s", rrr.Ref, rrr.Repository)
}

// FindMirrors scrapes a Redis Cache, looking for any mirror entries.
//
// It is expected that the Redis Cache will contain a key which is the result of
// `mirrorcat.RedisRemoteRef(original).String()`. At that key, MirrorCat expects to
// find a Set of strings matching the format of the key, but targeting other repositories
// and refs.
func (rf RedisFinder) FindMirrors(ctx context.Context, original RemoteRef, results chan<- RemoteRef) error {
	defer close(results)

	base := redis.Client(rf)

	memberCmd := base.SMembers(RedisRemoteRef(original).String())

	mirrors, err := memberCmd.Result()
	if err != nil {
		return err
	}

	log.Printf("Found %d Redis entries for mirror %q", len(mirrors), RedisRemoteRef(original).String())

	for _, item := range mirrors {
		parsed, err := ParseRedisRemoteRef(item)
		if err != nil {
			return err
		}
		select {
		case results <- RemoteRef(parsed):
			// Intentionally Left Blank
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}
