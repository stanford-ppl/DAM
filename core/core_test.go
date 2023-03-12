package core

import (
	"math/big"
	"sync"
	"testing"

	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/datatypes/fixed"
)

func TestSimpleNodeIO(t *testing.T) {
	var channelSize uint = 10
	inputA := core.MakeChannel[datatypes.FixedPoint](channelSize)
	inputB := core.MakeChannel[datatypes.FixedPoint](channelSize)
	output := core.MakeChannel[datatypes.FixedPoint](channelSize)
	inputChannelA := core.NodeInputChannel{
		Channel: &inputA,
	}
	inputChannelB := core.NodeInputChannel{
		Channel: &inputB,
	}
	inputChannels := map[int]core.NodeInputChannel{
		0: inputChannelA,
		1: inputChannelB,
	}
	outputChannel := core.NodeOutputChannel{
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

	var wg sync.WaitGroup
	stuffA := func() {
		for i := 0; i < 10; i++ {
			aVal := datatypes.FixedPoint{Tp: fpt}
			aVal.SetInt(big.NewInt(int64(i)))
			inputChannelA.Channel.Enqueue(aVal)
		}
		wg.Done()
	}

	stuffB := func() {
		for i := 0; i < 10; i++ {
			bVal := datatypes.FixedPoint{Tp: fpt}
			bVal.SetInt(big.NewInt(int64(2 * i)))
			inputChannelB.Channel.Enqueue(bVal)
		}
		wg.Done()
	}

	node.Step = func(node *core.Node) {
		a := node.InputChannels[0].Channel.Dequeue().(datatypes.FixedPoint)
		b := node.InputChannels[1].Channel.Dequeue().(datatypes.FixedPoint)
		node.OutputChannels[0].Channel.Enqueue(datatypes.FixedAdd(a, b))
	}

	wg.Add(4)

	go stuffA()
	go stuffB()
	go (func() {
		for i := 0; i < 10; i++ {
			node.Tick()
		}
		wg.Done()
	})()

	go (func() {
		for i := 0; i < 10; i++ {
			recv := outputChannel.Channel.Dequeue().(datatypes.FixedPoint)
			t.Logf("Output %d\n", recv.ToInt())
			if recv.ToInt().Int64() != int64(3*i) {
				t.Errorf("Expected: %d, received: %d", recv.ToInt().Int64(), 3*i)
			}
		}
		wg.Done()
	})()
	wg.Wait()
}
