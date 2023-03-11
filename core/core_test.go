package core

import (
	"math/big"
	"testing"

	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/datatypes"
)

func TestSimpleNodeIO(t *testing.T) {
	inputA := core.MakeChannel[datatypes.FixedPoint](8)
	inputB := core.MakeChannel[datatypes.FixedPoint](8)
	output := core.MakeChannel[datatypes.FixedPoint](8)
	inputChannelA := core.NodeInputChannel{
		Port:    core.Port{},
		Channel: &inputA,
	}
	inputChannelB := core.NodeInputChannel{
		Port:    core.Port{},
		Channel: &inputB,
	}
	inputChannels := map[int]core.NodeInputChannel{
		0: inputChannelA,
		1: inputChannelB,
	}
	outputChannel := core.NodeOutputChannel{
		Port:    core.Port{},
		Channel: &output,
	}
	node := core.Node{
		ID:            0,
		InputChannels: inputChannels,
		OutputChannels: map[int]core.NodeOutputChannel{
			0: outputChannel,
		},
	}
	inputChannelA.Port.Target = &node
	inputChannelA.Port.ID = 0
	inputChannelB.Port.Target = &node
	inputChannelB.Port.ID = 1
	outputChannel.Port.Target = &node
	outputChannel.Port.ID = 0

	fpt := datatypes.FixPointType{true, 32, 0}
	aVal := datatypes.FixedPoint{Tp: fpt}
	aVal.SetInt(big.NewInt(3))

	bVal := datatypes.FixedPoint{Tp: fpt}
	bVal.SetInt(big.NewInt(5))

	inputChannelA.Channel.Enqueue(aVal)
	inputChannelB.Channel.Enqueue(bVal)

	node.Step = func(node *core.Node) {
		a := node.InputChannels[0].Channel.Dequeue().(datatypes.FixedPoint)
		b := node.InputChannels[1].Channel.Dequeue().(datatypes.FixedPoint)
		node.OutputChannels[0].Channel.Enqueue(datatypes.FixedAdd(a, b))
	}
	node.Tick()
	recv := outputChannel.Channel.Dequeue().(datatypes.FixedPoint)
	t.Logf("Output %d\n", recv.ToInt())
	if recv.ToInt().Int64() != 8 {
		t.Errorf("Expected 3+5=8, got %s", recv.ToInt())
	}
}
