package storage

type Config struct {
	Stream    string `env:"ES_STREAM,default=posts"`
}
