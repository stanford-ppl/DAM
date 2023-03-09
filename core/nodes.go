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

	InputChannels  []NodeInputChannel[any]
	OutputChannels []NodeOutputChannel[any]

	InputTags  []InputTag[any, any]
	OutputTags []OutputTag[any, any]

	Worker NodeBody
}

func (node *Node) CanRun() bool {
	inputChannelMap := map[int]any{}
	for _, inputChannel := range node.InputChannels {
		targetID := inputChannel.ID.ID
		head := inputChannel.Channel.Peek()
		inputChannelMap[targetID] = head
	}

	for _, inputTag := range node.InputTags {
		head := inputChannelMap[inputTag.InputPort.ID]
		if !inputTag.Updater.CanRun(head) {
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
