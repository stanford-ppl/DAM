package core

import (
	"math/big"
	"sync"
	"testing"

	"golang.org/x/exp/maps"

	"github.com/stanford-ppl/DAM/datatypes"
)

func TestSimpleNodeIO(t *testing.T) {
	var channelSize uint = 1

	inputChannelA := NodeInputChannel{Channel: MakeChannel[datatypes.FixedPoint](channelSize)}
	inputChannelB := NodeInputChannel{Channel: MakeChannel[datatypes.FixedPoint](channelSize)}
	outputChannel := NodeOutputChannel{Channel: MakeChannel[datatypes.FixedPoint](channelSize)}

	node := NewNode()
	node.SetID(0)
	node.SetInputChannel(0, inputChannelA)
	node.SetInputChannel(1, inputChannelB)
	node.SetOutputChannel(0, outputChannel)

	if !node.Validate() {
		t.Errorf("Node %d failed validation", node.ID)
	}

	fpt := datatypes.FixedPointType{Signed: true, Integer: 32, Fraction: 0}

	var wg sync.WaitGroup
	genA := func() {
		for i := 0; i < 10; i++ {
			aVal := datatypes.FixedPoint{Tp: fpt}
			aVal.SetInt(big.NewInt(int64(i)))

			cE := ChannelElement{Data: aVal}
			cE.Time.Set(big.NewInt(int64(i)))

			inputChannelA.Channel.Enqueue(cE)
		}
		wg.Done()
	}

	genB := func() {
		for i := 0; i < 10; i++ {
			bVal := datatypes.FixedPoint{Tp: fpt}
			bVal.SetInt(big.NewInt(int64(2 * i)))

			cE := ChannelElement{Data: bVal}
			cE.Time.Set(big.NewInt(int64(i)))
			inputChannelB.Channel.Enqueue(cE)
		}
		wg.Done()
	}

	node.Step = func(node *Node, _ *big.Int) *big.Int {
		// Check if both channels are in the present
		if !node.IsPresent(maps.Values(node.InputChannels)) {
			return big.NewInt(1)
		}

		a := node.InputChannels[0].Channel.Dequeue().Data.(datatypes.FixedPoint)
		b := node.InputChannels[1].Channel.Dequeue().Data.(datatypes.FixedPoint)
		c := datatypes.FixedAdd(a, b)
		node.OutputChannels[0].Channel.Enqueue(MakeElement(&node.TickCount, c))
		t.Logf("%d + %d = %d", a.ToInt().Int64(), b.ToInt().Int64(), c.ToInt().Int64())
		return big.NewInt(1)
	}

	main := func() {
		for i := 0; i < 10; i++ {
			t.Logf("PreTicking:  %d %d %d", i, inputChannelA.Channel.Len(), inputChannelB.Channel.Len())
			node.Tick()
			t.Logf("PostTicking: %d %d %d", i, inputChannelA.Channel.Len(), inputChannelB.Channel.Len())
			t.Logf("Output Channel Size: %d", outputChannel.Channel.Len())
		}
		wg.Done()
	}

	checker := func() {
		for i := 0; i < 10; i++ {
			recv := outputChannel.Channel.Dequeue().Data.(datatypes.FixedPoint)
			t.Logf("Output %d\n", recv.ToInt())
			if recv.ToInt().Int64() != int64(3*i) {
				t.Errorf("Expected: %d, received: %d", 3*i, recv.ToInt().Int64())
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
	t.Logf("Total cycles elapsed: %s", node.TickCount.String())
}

func TestSimpleNodeIO_Vector(t *testing.T) {
	var channelSize uint = 10
	var vecWidth int = 10
	var numVecs int = 3

	inputChannelA := NodeInputChannel{Channel: MakeChannel[datatypes.Vector[datatypes.FixedPoint]](channelSize)}
	outputChannel := NodeOutputChannel{Channel: MakeChannel[datatypes.Vector[datatypes.FixedPoint]](channelSize)}

	node := NewNode()
	node.SetID(0)
	node.SetInputChannel(0, inputChannelA)
	node.SetOutputChannel(0, outputChannel)

	if !node.Validate() {
		t.Errorf("Node %d failed validation", node.ID)
	}

	fpt := datatypes.FixedPointType{Signed: true, Integer: 32, Fraction: 0}

	var wg sync.WaitGroup

	genA := func() {
		for n := 0; n < numVecs; n++ {
			v := datatypes.NewVector[datatypes.FixedPoint](10)
			for i := 0; i < vecWidth; i++ {
				aVal := datatypes.FixedPoint{Tp: fpt}
				aVal.SetInt(big.NewInt(int64(i)))
				v.Set(i, aVal)
			}
			inputChannelA.Channel.Enqueue(MakeElement(big.NewInt(int64(n)), v))
		}
		wg.Done()
	}

	node.Step = func(node *Node, _ *big.Int) *big.Int {
		a := node.InputChannels[0].Channel.Dequeue().Data.(datatypes.Vector[datatypes.FixedPoint])

		one := datatypes.FixedPoint{Tp: fpt}
		one.SetInt(big.NewInt(int64(1)))

		for i := 0; i < vecWidth; i++ {
			a.Set(i, datatypes.FixedAdd(a.Get(i), one))
		}
		node.OutputChannels[0].Channel.Enqueue(MakeElement(&node.TickCount, a))
		return big.NewInt(1)
	}

	main := func() {
		for n := 0; n < numVecs; n++ {
			for i := 0; i < 1; i++ {
				node.Tick()
			}
		}
		wg.Done()
	}

	checker := func() {
		for n := 0; n < numVecs; n++ {
			for i := 0; i < 1; i++ {
				recv := outputChannel.Channel.Dequeue().Data.(datatypes.Vector[datatypes.FixedPoint])
				for j := 0; j < vecWidth; j++ {
					t.Logf("Output for index: %d is %d", j, recv.Get(j).ToInt())
					if recv.Get(j).ToInt().Int64() != int64(j+1) {
						t.Errorf("Expected: %d, received: %d", (j + 1), recv.Get(j).ToInt().Int64())
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
