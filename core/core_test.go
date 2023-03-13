package core

import (
	"math/big"
	"sync"
	"testing"

	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/datatypes/fixed"
	//"github.com/stanford-ppl/DAM/networks/ideal_network"
)

func TestSimpleNodeIO(t *testing.T) {

	var channelSize uint = 10

	inputChannelA := core.NodeInputChannel{ Channel: core.MakeChannel[datatypes.FixedPoint](channelSize), }
	inputChannelB := core.NodeInputChannel{ Channel: core.MakeChannel[datatypes.FixedPoint](channelSize), }
	outputChannel := core.NodeOutputChannel{ Channel: core.MakeChannel[datatypes.FixedPoint](channelSize), }
	
	node := core.NewNode()
	node.SetID(0)
	node.SetInputChannel(0 , inputChannelA)
	node.SetInputChannel(1 , inputChannelB)
	node.SetOutputChannel(0 , outputChannel)

	if !node.Validate() {
		t.Errorf("Node %d failed validation", node.ID)	
	}

	fpt := datatypes.FixedPointType{true, 32, 0}

	var wg sync.WaitGroup
	genA := func() {
		for i := 0; i < 10; i++ {
			aVal := datatypes.FixedPoint{Tp: fpt}
			aVal.SetInt(big.NewInt(int64(i)))
			inputChannelA.Channel.Enqueue(aVal)
		}
		wg.Done()
	}

	genB := func() {
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

	main := func() {
		for i := 0; i < 10; i++ {
			node.Tick()
		}
		wg.Done()
	}

	checker := func() {
		for i := 0; i < 10; i++ {
			recv := outputChannel.Channel.Dequeue().(datatypes.FixedPoint)
			t.Logf("Output %d\n", recv.ToInt())
			if recv.ToInt().Int64() != int64(3*i) {
				t.Errorf("Expected: %d, received: %d", recv.ToInt().Int64(), 3*i)
			}
		}
		wg.Done()
	}

	wg.Add(4)

	go genA()
	go genB()
	go main()
	go checker()
	
	wg.Wait()
}