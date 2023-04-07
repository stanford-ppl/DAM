package networks

import (
	"fmt"

	"github.com/stanford-ppl/DAM/core"
)

// Stateless ideal network, which has a 1-cycle latency between enqueue and dequeue.
type IdealNetwork struct {
	Channels []core.CommunicationChannel
}

func (ideal *IdealNetwork) TickChannels() {
	for _, channel := range ideal.Channels {
		if channel.OutputChannel.Empty() || channel.InputChannel.Full() {
			continue
		}
		value := channel.OutputChannel.Dequeue()
		fmt.Printf("Passing on packet: %d\n", value.Time.Int64())
		channel.InputChannel.Enqueue(value)

	}
}
