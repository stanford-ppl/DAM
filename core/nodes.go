package core

import (
	datatypes "github.com/stanford-ppl/DAM/datatypes/base"
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
	ID int

	// maps port number to Input Channels
	InputChannels  map[int]NodeInputChannel
	OutputChannels map[int]NodeOutputChannel

	// Maps port number to the Tag
	InputTags  map[int]InputTag[datatypes.DAMType, datatypes.DAMType]
	OutputTags map[int]OutputTag[datatypes.DAMType, datatypes.DAMType]

	State interface{}

	Step func(node *Node)
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
	// Cannot publish if any of the outputTags are full
	for _, outputChannel := range node.OutputChannels {
		if outputChannel.Channel.Full() {
			return false
		}
	}

	for id, inTag := range node.InputTags {
		inputChannel := node.InputChannels[id]
		var head datatypes.DAMType
		if inputChannel.Channel.Empty() {
			head = inTag.Null
		} else {
			head = inputChannel.Channel.Peek()
		}
		if !node.InputTags[id].Updater.CanRun(head) {
			return false
		}
	}
	return true
}

func (node *Node) UpdateTagData(enabled bool) {
	for id, inTag := range node.InputTags {
		inputChannel := node.InputChannels[id]
		var update datatypes.DAMType
		if inputChannel.Channel.Empty() {
			update = inTag.Null
		} else {
			update = inputChannel.Channel.Dequeue()
		}
		inTag.State = inTag.Updater.Update(inTag.State, update, enabled)
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
	node.Step(node)

	// Publish new data out
	for id, outputTag := range node.OutputTags {
		// Check if we want to publish
		willPublish := outputTag.Publisher.HasPublish(node.State)
		if willPublish {
			publishData := outputTag.Publisher.Publish(node.State)
			targetChannel := node.OutputChannels[id].Channel
			targetChannel.Enqueue(publishData)
		}
	}
}
