package core

import (
	"math/big"
	"sync"
	"testing"

	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/datatypes/fixed"
	"github.com/stanford-ppl/DAM/vector"
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
				t.Errorf("Expected: %d, received: %d", 3*i , recv.ToInt().Int64())
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

func TestSimpleNodeIO_Vector(t *testing.T) {

	var channelSize uint = 10
	var vecWidth int = 10
	var numVecs int = 3

	inputChannelA := core.NodeInputChannel{ Channel: core.MakeChannel[vector.Vector[datatypes.FixedPoint]](channelSize), }
	outputChannel := core.NodeOutputChannel{ Channel: core.MakeChannel[vector.Vector[datatypes.FixedPoint]](channelSize), }

	node := core.NewNode()
	node.SetID(0)
	node.SetInputChannel(0 , inputChannelA)
	node.SetOutputChannel(0 , outputChannel)

	if !node.Validate() {
		t.Errorf("Node %d failed validation", node.ID)	
	}

	fpt := datatypes.FixedPointType{true, 32, 0}

	var wg sync.WaitGroup

	genA := func() {
		for n := 0; n < numVecs; n++{
			v := vector.NewVector[datatypes.FixedPoint](10)
			for i := 0; i < vecWidth; i++ {
				aVal := datatypes.FixedPoint{Tp: fpt}
				aVal.SetInt(big.NewInt(int64(i)))
				v.Set(i , aVal)
			}
			inputChannelA.Channel.Enqueue(v)
	    }
		wg.Done()
	}

	node.Step = func(node *core.Node) {
		a := node.InputChannels[0].Channel.Dequeue().(vector.Vector[datatypes.FixedPoint])

		one := datatypes.FixedPoint{Tp: fpt}
		one.SetInt(big.NewInt(int64(1)))

		for i := 0 ; i < vecWidth ; i++ {
			a.Set(i , datatypes.FixedAdd(a.Get(i) , one))
		}
		node.OutputChannels[0].Channel.Enqueue(a)
	}

	main := func() {
		for n := 0; n < numVecs; n++{
			for i := 0; i < 1; i++ {
				node.Tick()
			}
	    }
		wg.Done()
	}

	checker := func() {
		for n := 0; n < numVecs; n++ {
			for i := 0; i < 1; i++ {
				recv := outputChannel.Channel.Dequeue().(vector.Vector[datatypes.FixedPoint])
				for j := 0 ; j < vecWidth ; j++ {
					t.Logf("Output for index: %d is %d" , j , recv.Get(j).ToInt())
					if recv.Get(j).ToInt().Int64() != int64(j+1) {
						t.Errorf("Expected: %d, received: %d", (j+1), recv.Get(j).ToInt().Int64())
					}
				
				}
			}
	    }
		wg.Done()
	}

	wg.Add(3)

	go genA()
	go main()
	go checker()

	wg.Wait()

	
}