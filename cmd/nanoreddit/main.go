package main

import (
	"context"
	"io"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/joeshaw/envdecode"
	"github.com/oklog/run"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"nanoreddit/internal/handler"
	"nanoreddit/internal/server"
	"nanoreddit/internal/signal"
	"nanoreddit/internal/storage"
)

type config struct {
	Server       server.Config
	Storage      storage.Config
	RedisURL     string `env:"REDIS_URL,default=redis://localhost:6379/0"`
	Logger       struct {
		Level     string `env:"LOGGER_LEVEL,default=info"`
		Timestamp bool   `env:"LOGGER_TIMESTAMP,default=true"`
		Caller    bool   `env:"LOGGER_CALLER,default=true"`
		Pretty    bool   `env:"LOGGER_PRETTY,default=true"`
	}
}

func newLogger(cfg *config) *zerolog.Logger {
	var output io.Writer = os.Stdout
	if cfg.Logger.Pretty {
		output = zerolog.ConsoleWriter{Out: output}
	}
	logger := zerolog.New(output).With().Logger()

	if cfg.Logger.Timestamp {
		logger = logger.With().Timestamp().Logger()
	}
	if cfg.Logger.Caller {
		logger = logger.With().Caller().Logger()
	}

	level, err := zerolog.ParseLevel(cfg.Logger.Level)
	if err != nil {
		logger.Warn().Err(err).Str("level", cfg.Logger.Level).Msg("Cannot parse a logging level")
	} else {
		logger = logger.Level(level)
	}

	return &logger
}

func main() {
	var cfg config
	if err := envdecode.StrictDecode(&cfg); err != nil {
		log.Fatal().Err(err).Msg("Cannot decode config envs")
		return
	}

	l := newLogger(&cfg)
	ctx, cancel := context.WithCancel(l.WithContext(context.Background()))
	zerolog.Ctx(ctx).Info().Interface("config", &cfg).Msg("The gathered config")

	redisOpt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		zerolog.Ctx(ctx).Fatal().Err(err).Send()
		return
	}
	redisClient := redis.NewClient(redisOpt)
	defer func() {
		if err := redisClient.Close(); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Send()
		}
	}()
	storage := storage.NewStorage(&cfg.Storage, redisClient)

	g := &run.Group{}
	{
		srv := signal.NewService(cancel)
		g.Add(srv.Execute, srv.Interrupt)
	}
	{
		handler, err := handler.NewHandler(storage)
		if err != nil {
			zerolog.Ctx(ctx).Fatal().Err(err).Msg("Couldn't initialize an endpoints handler")
			return
		}
		//TODO use dependency injection github.com/google/wire
		srv := server.NewService(ctx, &cfg.Server, handler)
		g.Add(srv.Execute, srv.Interrupt)
	}

	zerolog.Ctx(ctx).Info().Msg("Running the service...")
	if err := g.Run(); err != nil {
		zerolog.Ctx(ctx).Fatal().Err(err).Msg("The service has been stopped with an error")
		return
	}
	zerolog.Ctx(ctx).Info().Msg("The service is stopped")
}
