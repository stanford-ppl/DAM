package networks

import (
	"github.com/stanford-ppl/DAM/core"
	datatypes "github.com/stanford-ppl/DAM/datatypes/base"
)

// Stateless ideal network, which has a 1-cycle latency between enqueue and dequeue.
type IdealNetwork struct{}

func (ideal *IdealNetwork) Tick(channels []core.CommunicationChannel[datatypes.DAMType]) {
	for _, channel := range channels {
		ideal.tickChannel(channel)
	}
}

func (ideal *IdealNetwork) tickChannel(channel core.CommunicationChannel[datatypes.DAMType]) {
	if channel.InputChannel.Empty() || channel.OutputChannel.Full() {
		return
	}
	value := channel.InputChannel.Dequeue()
	channel.OutputChannel.Enqueue(value)
}
