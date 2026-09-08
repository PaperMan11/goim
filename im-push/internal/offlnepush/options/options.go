package options

type Options struct {
	Signal        *Signal
	IOSPushSound  string
	IOSBadgeCount int
	Ex            string
}

// Signal message id.
type Signal struct {
	ClientMsgID string
}

type Option func(*Options)

// WithSignal sets the signal message id.
func WithSignal(clientMsgID string) Option {
	return func(opts *Options) {
		opts.Signal = &Signal{
			ClientMsgID: clientMsgID,
		}
	}
}
func WithIOSPushSound(sound string) Option {
	return func(opts *Options) {
		opts.IOSPushSound = sound
	}
}

// WithIOSBadgeCount sets the ios badge count.
func WithIOSBadgeCount(count int) Option {
	return func(opts *Options) {
		opts.IOSBadgeCount = count
	}
}

// WithEx sets the extra data.
func WithEx(extra string) Option {
	return func(opts *Options) {
		opts.Ex = extra
	}
}
