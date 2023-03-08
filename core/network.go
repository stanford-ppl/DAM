package core

type Port struct {
	ID     int
	Target *Node
}

type Network interface {
	Tick(channels []CommunicationChannel[interface{}])
}

type DAMChannel[T any] struct {
	channel chan T
	head    *T
}

func (channel *DAMChannel[T]) Peek() T {
	if channel.channel != nil {
		return *channel.head
	}
	tmp := <-channel.channel
	channel.head = &tmp
	return tmp
}

func (channel *DAMChannel[T]) Dequeue() T {
	if channel.head != nil {
		tmp := *channel.head
		channel.head = nil
		return tmp
	}
	return <-channel.channel
}

func (channel *DAMChannel[T]) Enqueue(data T) {
	channel.channel <- data
}

func (channel *DAMChannel[T]) Full() bool {
	return len(channel.channel) == cap(channel.channel)
}

func (channel *DAMChannel[T]) Capacity() int {
	return len(channel.channel)
}

func (channel *DAMChannel[T]) Empty() bool {
	return len(channel.channel) == 0
}

func (channel *DAMChannel[T]) NumElements() int {
	return len(channel.channel)
}

type CommunicationChannel[T any] struct {
	InputPort  Port
	OutputPort Port

	// Serves as the input 'buffer' of a DAM Node
	InputChannel DAMChannel[T]

	// Serves as the output 'buffer' of a DAM Node
	OutputChannel DAMChannel[T]
}
