package tcp

type Options struct {
	health bool
	logger bool
}

func Default() *Options {
	return &Options{
		health: true,
		logger: true,
	}
}

func None() *Options {
	return &Options{}
}

func (o *Options) WithoutHealth() *Options {
	clone := *o
	clone.health = false
	return &clone
}

func (o *Options) WithoutLogger() *Options {
	clone := *o
	clone.logger = false
	return &clone
}
