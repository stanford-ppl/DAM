package networks

import (
	"math/big"

	"github.com/stanford-ppl/DAM/core"
)

// Stateless ideal network, which has a 1-cycle latency between enqueue and dequeue.
type IdealNetwork struct {
	channels    []core.CommunicationChannel
	initialized bool

	terminators []chan bool
}

func (ideal *IdealNetwork) Initialize(channels []core.CommunicationChannel) {
	if ideal.initialized {
		panic("Cannot re-initialize network! Make a new one if you would like to reset the network.")
	}
	ideal.initialized = true
	ideal.channels = channels
	ideal.terminators = make([]chan bool, len(ideal.channels))
	for i := range channels {
		ideal.terminators[i] = make(chan bool)
	}
}

func (ideal *IdealNetwork) Run() {
	if !ideal.initialized {
		panic("Cannot run on uninitialized network")
	}
	runChannel := func(killChan chan bool, channel core.CommunicationChannel) {
		inputChan := channel.InputChannel.Underlying()
		outputChan := channel.OutputChannel.Underlying()
		for {
			select {
			case <-killChan:
				return
			case input := <-outputChan:
				// increment the cycle count
				newTime := big.NewInt(1)
				newTime.Add(newTime, &input.Time)
				newInput := core.MakeElement(newTime, input.Data)
				select {
				case <-killChan:
					return
				case inputChan <- newInput:
				}
			}
		}
	}
	for i, channel := range ideal.channels {
		go runChannel(ideal.terminators[i], channel)
	}
}

func (ideal *IdealNetwork) Kill() {
	for _, v := range ideal.terminators {
		close(v)
	}
}
