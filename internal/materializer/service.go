package materializer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog"

	"nanoreddit/internal/storage"
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

	readArg := &redis.XReadGroupArgs{
		Group:    s.cfg.Group,
		Consumer: s.cfg.Consumer,
		Streams:  []string{s.cfg.Stream, ">"},
	}
	// The worker's purpose is fetching messages from the stream and put them into a corresponding place in Redis.
	for {
		streams, err := s.client.XReadGroup(ctx, readArg).Result()
		if err != nil {
			return fmt.Errorf("couldn't read a group: %w", err)
		}
		// It mustn't happen because we're waiting for data only from a single source.
		if len(streams) != 1 {
			return fmt.Errorf("unexpected number of streams: %d", len(streams))
		}

		// We've got a bunch of messages, so the next step is extracting original posts.
		for _, message := range streams[0].Messages {
			blob, ok := message.Values[storage.StreamValueField].(string)
			if !ok {
				//TODO What will we do with the other messages? I believe it's a place for variate decisions.
				return fmt.Errorf("couldn't find an event in a message: %v", message)
			}
			var post protocol.Post
			if err := json.Unmarshal([]byte(blob), &post); err != nil {
				return fmt.Errorf("couldn't unmarshal a saved post: %w", err)
			}

			if post.Promoted {
				// Promoted posts go to a circular list.
				if err := s.client.LPush(ctx, s.cfg.Promotion, blob).Err(); err != nil {
					return fmt.Errorf("couldn't put a promoted post into a ring: %w", err)
				}
				continue
			}
			// Ordinary posts should be kept in a sorted set.
			if err := s.client.ZAdd(ctx, s.cfg.Feed, &redis.Z{
				Score:  float64(post.Score),
				Member: blob,
			}).Err(); err != nil {
				return fmt.Errorf("couldn't put a post into the feed: %w", err)
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
