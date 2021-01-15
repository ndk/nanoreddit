package signal

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

type service struct {
	stop   chan os.Signal
	cancel context.CancelFunc
}

func (s *service) Execute() error {
	<-s.stop
	return nil
}

func (s *service) Interrupt(err error) {
	signal.Stop(s.stop)
	s.cancel()
	close(s.stop)
}

func NewService(cancel context.CancelFunc) *service {
	srv := service{
		stop:   make(chan os.Signal),
		cancel: cancel,
	}
	signal.Notify(srv.stop, syscall.SIGINT, syscall.SIGTERM)
	return &srv
}
