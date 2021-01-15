package storage

type Config struct {
	Stream    string `env:"ES_STREAM,default=posts"`
	Feed      string `env:"ES_FEED,default=feed"`
	PageSize  int    `env:"FEED_PAGE_SIZE,default=25"`
	Promotion string `env:"ES_PROMOTION,default=promotion"`
}
