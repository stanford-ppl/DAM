package core

import (
	"math/big"

	"github.com/stanford-ppl/DAM/datatypes"
)

type Network interface {
	Run()
	Kill()
	Initialize(channels []CommunicationChannel)
}

type DAMChannel struct {
	channel chan ChannelElement
	head    *ChannelElement
	headOk  bool
}

type ChannelElement struct {
	Time big.Int
	Data datatypes.DAMType
}

func MakeElement(time *big.Int, data datatypes.DAMType) ChannelElement {
	ce := ChannelElement{Data: data}
	ce.Time.Set(time)
	return ce
}

func MakeChannel[T datatypes.DAMType](channelSize uint) *DAMChannel {
	return &DAMChannel{make(chan ChannelElement, channelSize), nil, false}
}

func (channel *DAMChannel) Peek() ChannelElement {
	if channel.head == nil {
		tmp, ok := <-channel.channel
		channel.head = &tmp
		channel.headOk = ok
	}
	return *channel.head
}

func (channel *DAMChannel) Close() {
	close(channel.channel)
}

func (channel *DAMChannel) Dequeue() (ChannelElement, bool) {
	if channel.head != nil {
		tmp := *channel.head
		channel.head = nil
		ok := channel.headOk
		channel.headOk = false
		return tmp, ok
	}
	v, ok := <-channel.channel
	return v, ok
}

func (channel *DAMChannel) DequeueNoCheck() ChannelElement {
	if channel.head != nil {
		tmp := *channel.head
		channel.head = nil
		channel.headOk = false
		return tmp
	}
	v := <-channel.channel
	return v
}

func (channel *DAMChannel) Enqueue(element ChannelElement) {
	channel.channel <- element
}

func (channel DAMChannel) Full() bool {
	return len(channel.channel) == cap(channel.channel)
}

func (channel DAMChannel) Cap() int {
	return cap(channel.channel)
}

func (channel DAMChannel) Empty() bool {
	return (len(channel.channel) == 0 && channel.head == nil)
}

func (channel DAMChannel) Len() int {
	cur := len(channel.channel)
	if channel.head != nil {
		cur += 1
	}
	return cur
}

func (channel DAMChannel) Underlying() chan ChannelElement {
	return channel.channel
}

type CommunicationChannel struct {
	// Serves as the input 'buffer' of a DAM Node
	InputChannel *DAMChannel

	// Serves as the output 'buffer' of a DAM Node
	OutputChannel *DAMChannel
}

func MakeCommunicationChannel[T datatypes.DAMType](size uint) CommunicationChannel {
	return CommunicationChannel{
		InputChannel:  MakeChannel[T](size),
		OutputChannel: MakeChannel[T](size),
	}
}
