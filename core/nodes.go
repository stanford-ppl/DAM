package core

import (
	"fmt"
	"math/big"

	"github.com/stanford-ppl/DAM/datatypes"
	"github.com/stanford-ppl/DAM/utils"
)

type NodeInputChannel struct {
	Port    Port
	Channel *DAMChannel
}

type NodeOutputChannel struct {
	Port    Port
	Channel *DAMChannel
}

func MakeInputOutputChannelPair[T datatypes.DAMType](size uint) (input NodeInputChannel, output NodeOutputChannel) {
	input = NodeInputChannel{Channel: MakeChannel[T](size)}
	output = NodeOutputChannel{Channel: MakeChannel[T](size)}
	return
}

type Node struct {
	ID        int
	TickCount big.Int

	// maps port number to Input Channels
	InputChannels  map[int]NodeInputChannel
	OutputChannels map[int]NodeOutputChannel

	// Maps port number to the Tag
	InputTags  map[int]InputTag[datatypes.DAMType, datatypes.DAMType]
	OutputTags map[int]OutputTag[datatypes.DAMType, datatypes.DAMType]

	State interface{}

	Step func(node *Node, ffTime *big.Int) *big.Int
}

func NewNode() Node {
	n := Node{}
	n.ID = -1
	n.InputChannels = map[int]NodeInputChannel{}
	n.OutputChannels = map[int]NodeOutputChannel{}
	n.InputTags = map[int]InputTag[datatypes.DAMType, datatypes.DAMType]{}
	n.OutputTags = map[int]OutputTag[datatypes.DAMType, datatypes.DAMType]{}
	n.TickCount.SetInt64(0)
	return n
}

func (node *Node) SetID(id int) {
	node.ID = id
}

func (node *Node) SetInputChannel(portNum int, inputchan NodeInputChannel) {
	node.InputChannels[portNum] = inputchan
	inputchan.Port = Port{Target: node, ID: portNum}
}

func (node *Node) SetOutputChannel(portNum int, outputchan NodeOutputChannel) {
	node.OutputChannels[portNum] = outputchan
	outputchan.Port = Port{Target: node, ID: portNum}
}

func (node *Node) SetInputTag(portNum int, input InputTag[datatypes.DAMType, datatypes.DAMType]) {
	node.InputTags[portNum] = input
}

func (node *Node) SetOutputTag(portNum int, output OutputTag[datatypes.DAMType, datatypes.DAMType]) {
	node.OutputTags[portNum] = output
}

func (node *Node) Validate() bool {
	for port, tag := range node.InputTags {
		// Check that we're the port described
		if node != tag.InputPort.Target {
			return false
		}
		// Check if the reported port is the same as the tag's port
		if port != tag.InputPort.ID {
			return false
		}
		// Check if we have a channel on that port
		_, hasChannel := node.InputChannels[port]
		if !hasChannel {
			return false
		}
	}

	for port, tag := range node.OutputTags {
		if node != tag.OutputPort.Target {
			return false
		}
		if port != tag.OutputPort.ID {
			return false
		}

		_, hasChannel := node.OutputChannels[port]
		if !hasChannel {
			return false
		}
	}
	return true
}

func (node *Node) CanRun() *big.Int {
	// Number of cycles we need to skip forward before we can maybe run
	nextRun := big.NewInt(0)
	for id := range node.InputTags {
		inputChannel := node.InputChannels[id]
		updater := node.InputTags[id].Updater

		// If a tag can run without any new update, then we don't need to check for updates
		if updater.CanRun(nil) {
			continue
		}

		// If the tag can't run without a new update, then we have to
		// wait for an update.
		peeked := inputChannel.Channel.Peek()

		// If the update is in the future, then we need to wait for that update to come through.
		utils.Max[*big.Int](nextRun, &peeked.Time, nextRun)

		// If we can't run after the update, then we need to wait at least one iteration and try again.
		if !node.InputTags[id].Updater.CanRun(peeked.Data) {
			utils.Max[*big.Int](nextRun, big.NewInt(1), nextRun)
		}
	}
	return nextRun
}

func (node *Node) UpdateTagData(ffTime *big.Int) {
	// fast forward to ffTime
	newTime := new(big.Int)
	newTime.Add(ffTime, &node.TickCount)

	enabled := ffTime.Cmp(big.NewInt(0)) == 0

	for id, inTag := range node.InputTags {
		inputChannel := node.InputChannels[id]
		// to fast forward to FFTime:
		// First, we need to fetch all of the updates up to FFTime.
		// If we don't have a new update at/after FFTime, then we have to stall because we don't know if the other node just hasn't been scheduled.
		var data []datatypes.DAMType
		for {
			newData := inputChannel.Channel.Peek()
			if newData.Time.Cmp(newTime) > 0 {
				// If this update is in the future, then we know we've collected all relevant updates
				break
			}
			// otherwise, we pop the data
			inputChannel.Channel.Dequeue()
			data = append(data, newData.Data)
		}
		inTag.State = inTag.Updater.Update(inTag.State, data, enabled)
	}
}

func (node *Node) Tick() {
	// Check to see if/when we can check/step again.
	ffTime := node.CanRun()
	// ffTime dictates "when" our next run can be.
	// if ffTime > 0, that means we need to fast-forward first.

	// Update all the tags for the node for ffTime cycles
	node.UpdateTagData(ffTime)

	// Need to step even if "canRun" is false because we could have a pipeline.
	// ffTime = 0 means that we're not fast forwarding at all.
	// Step is now also responsible for publishing data to tags, so that it has finer grained control.
	ticks := node.Step(node, ffTime)

	// we need to tick at least ffTime steps forward
	if ticks.Cmp(ffTime) < 0 {
		panic(fmt.Sprintf("Needed to skip forward %s cycles, but only ticked %s cycles", ffTime.String(), ticks.String()))
	}
	node.TickCount.Add(&node.TickCount, ticks)
}

func (node Node) IsPresent(checkedChannels []NodeInputChannel) bool {
	for _, v := range checkedChannels {
		stamp := v.Channel.Peek().Time
		if node.TickCount.Cmp(&stamp) < 0 {
			return false
		}
	}
	return true
}
