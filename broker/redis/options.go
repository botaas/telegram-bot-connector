package redis

type redisOptions struct {
	addr     string
	password string
}

type Option func(o *redisOptions)

func WithAddr(addr string) Option {
	return func(o *redisOptions) {
		o.addr = addr
	}
}

func WithPassword(password string) Option {
	return func(o *redisOptions) {
		o.password = password
	}
}
