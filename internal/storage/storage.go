package storage

import (
	"context"
	"encoding/json"

	"github.com/go-redis/redis/v8"

	"nanoreddit/pkg/protocol"
)

//TODO using such constants crosspackagely isn't a good idea. it would be better to extract it into an abstration
const StreamValueField = "event"

type storage struct {
	cfg    *Config
	client redis.Cmdable
	encode func(v interface{}) ([]byte, error)
	decode func(data []byte, v interface{}) error
}

func (s *storage) AddPost(ctx context.Context, post *protocol.Post) error {
	blob, err := s.encode(post)
	if err != nil {
		return err
	}

	a := redis.XAddArgs{
		Stream: s.cfg.Stream,
		Values: map[string]interface{}{StreamValueField: blob},
	}
	if err := s.client.XAdd(ctx, &a).Err(); err != nil {
		return err
	}

	return nil
}

func (s *storage) GetFeed(ctx context.Context, page int) ([]protocol.Post, error) {
	// Posts on Redis are already sorted by score.
	blobs, err := s.client.ZRevRangeByScore(ctx, s.cfg.Feed, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    "+inf",
		Offset: int64(page * s.cfg.PageSize),
		Count:  int64(s.cfg.PageSize),
	}).Result()
	if err != nil {
		return nil, err
	}

	// A result can have up to two additional promoted posts.
	feed := make([]protocol.Post, 0, len(blobs)+2)
	for _, blob := range blobs {
		var post protocol.Post
		if err := json.Unmarshal([]byte(blob), &post); err != nil {
			return nil, err
		}
		feed = append(feed, post)

		// TODO get rid a magic number
		// TODO check the case when we got 16 posts
		// We have to add a promoted post only if we've just reached 3 or 17 posts but not yet exceeded it.
		if len(feed) != 3 && len(feed) != 17 {
			continue
		}
		// Unless there are some NSFW posts.
		if feed[len(feed)-3].NSFW || feed[len(feed)-2].NSFW {
			continue
		}

		// RPOPLPUSH lets a list to act as a circular one. Hence we can show promoted posts evenly.
		blob, err := s.client.RPopLPush(ctx, s.cfg.Promotion, s.cfg.Promotion).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			return nil, err
		}

		var promotedPost protocol.Post
		if err := json.Unmarshal([]byte(blob), &promotedPost); err != nil {
			return nil, err
		}
		// Insert a promoted post into feed.
		//TODO improve
		prev := len(feed)
		feed = append(feed, promotedPost)
		feed[prev-2], feed[prev-1], feed[prev] = feed[prev], feed[prev-2], feed[prev-1]
	}
	return feed, nil
}

func NewStorage(cfg *Config, client redis.Cmdable) *storage {
	return &storage{
		cfg:    cfg,
		client: client,
		encode: json.Marshal,
		decode: json.Unmarshal,
	}
}
