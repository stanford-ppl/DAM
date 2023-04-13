package utils

type Option[T any] struct {
	isSet      bool
	underlying T
}

func Some[T any](val T) Option[T] {
	return Option[T]{
		isSet:      true,
		underlying: val,
	}
}

func None[T any]() Option[T] {
	return Option[T]{
		isSet: false,
	}
}

func (opt *Option[T]) Get() T {
	if !opt.isSet {
		panic("Attempting to read from an unset option")
	}
	return opt.underlying
}

func (opt *Option[T]) IsSet() bool {
	return opt.isSet
}
