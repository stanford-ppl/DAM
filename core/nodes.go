package core

import (
	"fmt"
	"math/big"

	"github.com/stanford-ppl/DAM/datatypes"
)

type NodeInputChannel struct {
	Port    Port
	Channel *DAMChannel
}

type NodeOutputChannel struct {
	Port    Port
	Channel *DAMChannel
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

	Step func(node *Node) *big.Int
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

func (node *Node) CanRun() bool {
	for id := range node.InputTags {
		inputChannel := node.InputChannels[id]
		peeked := inputChannel.Channel.Peek()
		if peeked.Time.Cmp(&node.TickCount) > 0 {
			return false
		}
		if !node.InputTags[id].Updater.CanRun(peeked.Data) {
			return false
		}
	}
	return true
}

func (node *Node) UpdateTagData(enabled bool) {
	for id, inTag := range node.InputTags {
		inputChannel := node.InputChannels[id]
		dequeued := inputChannel.Channel.Dequeue()
		if node.TickCount.Cmp(&dequeued.Time) < 0 {
			panic(fmt.Sprintf("Ended up in future! Reading data at %s when we're at %s", dequeued.Time.String(), node.TickCount.String()))
		}
		inTag.State = inTag.Updater.Update(inTag.State, dequeued.Data, enabled)
	}
}

func (node *Node) Tick() {
	// Check to see if we can run
	canRun := node.CanRun()
	// Update all the tags for the node
	node.UpdateTagData(canRun)

	if !canRun {
		return
	}
	ticks := node.Step(node)

	// Add ticks before publishing
	node.TickCount.Add(&node.TickCount, ticks)

	// Publish new data out
	for id, outputTag := range node.OutputTags {
		// Check if we want to publish
		willPublish := outputTag.Publisher.HasPublish(node.State)
		if willPublish {
			publishData := outputTag.Publisher.Publish(node.State)
			targetChannel := node.OutputChannels[id].Channel
			targetChannel.Enqueue(MakeElement(&node.TickCount, publishData))
		}
	}
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
