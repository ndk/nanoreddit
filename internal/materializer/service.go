package materializer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog"

	"nanoreddit/pkg/protocol"
)

type service struct {
	ctx    context.Context
	cancel context.CancelFunc
	cfg    *Config
	client redis.Cmdable
	decode func(data []byte, v interface{}) error
}

func (s *service) Execute() error {
	ctx := s.ctx

	// https://github.com/go-redis/redis/pull/924#issuecomment-446267518
	if err := s.client.XGroupCreateMkStream(ctx, s.cfg.Stream, s.cfg.Group, "0").Err(); err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("Couldn't create a stream: %w", err)
	}

	readArg := &redis.XReadGroupArgs{
		Group:    s.cfg.Group,
		Consumer: s.cfg.Consumer,
		Streams:  []string{s.cfg.Stream, ">"},
	}
	for {
		streams, err := s.client.XReadGroup(ctx, readArg).Result()
		if err != nil {
			return fmt.Errorf("Couldn't read a group: %w", err)
		}
		if len(streams) != 1 {
			return fmt.Errorf("Unexpected number of streams: %d", len(streams))
		}

		for _, message := range streams[0].Messages {
			//TODO get rid a magic constant
			blob, ok := message.Values["event"].(string)
			if !ok {
				//TODO What will we do with the other messages?
				return fmt.Errorf("Couldn't find an event in a message: %v", message)
			}
			var post protocol.Post
			if err := json.Unmarshal([]byte(blob), &post); err != nil {
				return fmt.Errorf("Couldn't unmarshal a saved post: %w", err)
			}

			if post.Promoted {
				if err := s.client.LPush(ctx, s.cfg.Promotion, blob).Err(); err != nil {
					return fmt.Errorf("Couldn't put a promoted post into a ring: %w", err)
				}
				continue
			}
			if err := s.client.ZAdd(ctx, s.cfg.Feed, &redis.Z{
				Score:  float64(post.Score),
				Member: blob,
			}).Err(); err != nil {
				return fmt.Errorf("Couldn't put a post into the feed: %w", err)
			}
		}
	}
}

func (s *service) Interrupt(err error) {
	s.cancel()
}

func NewService(ctx context.Context, cancel context.CancelFunc, client redis.Cmdable, cfg *Config) *service {
	l := zerolog.Ctx(ctx).With().Str("service", "materializer").Logger()
	ctx = l.WithContext(ctx)

	return &service{
		ctx:    ctx,
		cancel: cancel,
		client: client,
		cfg:    cfg,
		decode: json.Unmarshal,
	}
}
