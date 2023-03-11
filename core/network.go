package core

import (
	"github.com/stanford-ppl/DAM/datatypes"
)

type Port struct {
	ID     int
	Target *Node
}

type Network interface {
	Tick(channels []CommunicationChannel[datatypes.DAMType])
}

type DAMChannel struct {
	channel chan datatypes.DAMType
	head    *datatypes.DAMType
}

func MakeChannel[T datatypes.DAMType](channelSize uint) DAMChannel {
	return DAMChannel{make(chan datatypes.DAMType, channelSize), nil}
}

func (channel *DAMChannel) Peek() datatypes.DAMType {
	if channel.head != nil {
		return *channel.head
	}
	tmp := <-channel.channel
	channel.head = &tmp
	return *channel.head
}

func (channel *DAMChannel) Dequeue() datatypes.DAMType {
	if channel.head != nil {
		tmp := *channel.head
		channel.head = nil
		return tmp
	}
	v := <-channel.channel
	return v
}

func (channel *DAMChannel) Enqueue(data datatypes.DAMType) {
	channel.channel <- data
}

func (channel *DAMChannel) Full() bool {
	return len(channel.channel) == cap(channel.channel)
}

func (channel *DAMChannel) Cap() int {
	return cap(channel.channel)
}

func (channel *DAMChannel) Empty() bool {
	return len(channel.channel) == 0
}

func (channel *DAMChannel) Len() int {
	return len(channel.channel)
}

type CommunicationChannel[T datatypes.DAMType] struct {
	InputPort  Port
	OutputPort Port

	// Serves as the input 'buffer' of a DAM Node
	InputChannel DAMChannel

	// Serves as the output 'buffer' of a DAM Node
	OutputChannel DAMChannel
}
