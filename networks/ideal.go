package networks

import (
	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/datatypes"
)

// Stateless ideal network, which has a 1-cycle latency between enqueue and dequeue.
type IdealNetwork[T datatypes.DAMType] struct {
	Channels []core.CommunicationChannel[T]
}

func (ideal *IdealNetwork[T]) TickChannels() {
	for _, channel := range ideal.Channels {

		if channel.InputChannel.Empty() || channel.OutputChannel.Full() {
			return
		}
		value := channel.InputChannel.Dequeue()
		channel.OutputChannel.Enqueue(value)

	}
}
