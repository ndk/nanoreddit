package storage

import (
	"context"
	"encoding/json"

	"github.com/go-redis/redis/v8"

	"nanoreddit/pkg/protocol"
)

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
		Values: map[string]interface{}{"event": blob},
	}
	if err := s.client.XAdd(ctx, &a).Err(); err != nil {
		return err
	}

	return nil
}

func (s *storage) GetFeed(ctx context.Context, page int) ([]protocol.Post, error) {
	blobs, err := s.client.ZRevRangeByScore(ctx, s.cfg.Feed, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    "+inf",
		Offset: int64(page * s.cfg.PageSize),
		Count:  int64(s.cfg.PageSize),
	}).Result()
	if err != nil {
		return nil, err
	}

	feed := make([]protocol.Post, 0, len(blobs))
	for _, blob := range blobs {
		var post protocol.Post
		if err := json.Unmarshal([]byte(blob), &post); err != nil {
			return nil, err
		}
		feed = append(feed, post)

		//TODO get rid a magic number
		if (len(feed) != 3 && len(feed) != 17) || (feed[len(feed)-3].NSFW || feed[len(feed)-2].NSFW) {
			continue
		}

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
