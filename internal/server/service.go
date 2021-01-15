package server

import (
	"context"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"

	"nanoreddit/internal/chi_utils"
	"nanoreddit/internal/middleware"
)

type service struct {
	logger  zerolog.Logger
	httpSrv http.Server
	cfg     *Config
}

func (s *service) Execute() error {
	if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *service) Interrupt(err error) {
	ctx, cancel := context.WithTimeout(s.logger.WithContext(context.Background()), s.cfg.ShutdownTimeout)
	defer cancel()

	s.httpSrv.SetKeepAlivesEnabled(false)
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("Cannot shut down the service properly")
	}
}

func NewService(ctx context.Context, cfg *Config,
	handler interface {
		Submit(w http.ResponseWriter, r *http.Request)
		Feed(w http.ResponseWriter, r *http.Request)
	},
) *service {
	l := zerolog.Ctx(ctx).With().Str("service", "server").Logger()

	r := chi.NewRouter()
	r.Use(hlog.NewHandler(l))
	//TODO put a recoverer here
	r.Use(hlog.RequestIDHandler("id_request", "X-Request-ID"))
	r.Use(hlog.RequestHandler("request"))
	if cfg.LogRequests {
		render := chi_utils.NewRender()
		r.Use(middleware.RequestBody(render.InvalidRequest))
	}
	r.Post("/submit", handler.Submit)
	r.Get("/feed", handler.Feed)

	return &service{
		logger: l,
		httpSrv: http.Server{
			Addr:    cfg.Address,
			Handler: r,
		},
		cfg: cfg,
	}
}
