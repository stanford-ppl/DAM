package core

type CommunicationChannel[T any] struct {
	Name string
	Channel chan T
}

func NewCommunicationChannel[T any](name string, capacity int) *CommunicationChannel[T] {
	var newChannel = new(CommunicationChannel[T])
	newChannel.Name = name
	newChannel.Channel = make(chan T, capacity)
	return newChannel
}
