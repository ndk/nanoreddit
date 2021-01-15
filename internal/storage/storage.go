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

func NewStorage(cfg *Config, client redis.Cmdable) *storage {
	return &storage{
		cfg:    cfg,
		client: client,
		encode: json.Marshal,
	}
}
