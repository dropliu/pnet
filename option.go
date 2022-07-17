package pnet

type (
	Options struct {
		ID          string
		ListenAddrs []string
	}
	Option func(*Options)
)

// WithID 设置ID
func WithID(id string) Option {
	return func(o *Options) {
		o.ID = id
	}
}
