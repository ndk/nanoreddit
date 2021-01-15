package materializer

type Config struct {
	Group     string `env:"ES_GROUP,default=materializer"`
	Consumer  string `env:"ES_CONSUMER,default=nanoreddit"`
	Stream    string `env:"ES_STREAM,default=posts"`
	Feed      string `env:"ES_FEED,default=feed"`
	Promotion string `env:"ES_PROMOTION,default=promotion"`
}
