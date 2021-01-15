package server

import "time"

type Config struct {
	Address         string        `env:"SERVICE_LISTEN_ADDRESS,default=:8080"`
	ShutdownTimeout time.Duration `env:"SERVICE_SHUTDOWN_TIMEOUT,default=30s"`
	LogRequests     bool          `env:"SERVICE_LOGREQUESTS,default=true"`
}
