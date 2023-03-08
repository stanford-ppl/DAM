package core

type NodeInputChannel[T any] struct {
	ID      Port
	Channel DAMChannel[T]
}

type NodeOutputChannel[T any] struct {
	ID      Port
	Channel DAMChannel[T]
}

type NodeBody interface {
	Tick(node *Node)
}

type Node struct {
	ID int

	InputChannels  []NodeInputChannel[interface{}]
	OutputChannels []NodeOutputChannel[interface{}]

	InputTags  []InputTag[interface{}, interface{}]
	OutputTags []OutputTag[interface{}, interface{}]

	Worker NodeBody
}

func (node *Node) CanRun() bool {
	inputChannelMap := map[int]interface{}{}
	for _, inputChannel := range node.InputChannels {
		targetID := inputChannel.ID.ID
		peeked := inputChannel.Channel.Peek()
		inputChannelMap[targetID] = peeked
	}

	for _, inputTag := range node.InputTags {
		peekValue := inputChannelMap[inputTag.InputPort.ID]
		if !inputTag.Updater.CanRun(peekValue) {
			return false
		}
	}
	return true
}

func (node *Node) Tick() {
	if !node.CanRun() {
		return
	}
	node.Worker.Tick(node)
}
