package core

import (
	"math/big"
	"testing"

	"github.com/stanford-ppl/DAM/datatypes"
)

func TestSimpleNodeIO(t *testing.T) {
	var channelSize int = 4

	mkChan := func() *CommunicationChannel {
		return MakeCommunicationChannel[datatypes.FixedPoint](channelSize)
	}

	// This test directly writes into the InputChannels and reads from the OutputChannels to avoid using a network.
	channelA := mkChan()
	channelB := mkChan()
	channelC := mkChan()
	fpt := datatypes.FixedPointType{Signed: true, Integer: 32, Fraction: 0}

	ctx := MakePrimitiveContext(nil)

	node := SimpleNode[any]{
		RunFunc: func(node *SimpleNode[any]) {
			for i := 0; i < 10; i++ {
				elements := DequeueInputChannels(node, 0, 1)
				a := elements[0].Data.(datatypes.FixedPoint)
				b := elements[1].Data.(datatypes.FixedPoint)
				c := datatypes.FixedAdd(a, b)
				// node has now incremented to the time needed to read both elements.
				// Advance node until there's space in the output queue
				AdvanceUntilCanEnqueue(node, 0)
				enqTime := node.TickLowerBound()
				enqTime.Add(enqTime, OneTick)
				succ, _ := node.OutputChannel(0).Enqueue(MakeChannelElement(enqTime, c))
				if !succ {
					panic("We advanced until we could enqueue, so enqueue should always succeed")
				}
				node.IncrCycles(OneTick)
			}
		},
	}

	node.AddInputChannel(channelA)
	node.AddInputChannel(channelB)
	node.AddOutputChannel(channelC)
	ctx.AddChild(&node)

	genA := SimpleNode[any]{
		RunFunc: func(node *SimpleNode[any]) {
			for i := 0; i < 10; i++ {
				aVal := datatypes.FixedPoint{Tp: fpt}
				aVal.SetInt(big.NewInt(int64(i)))
				cE := MakeChannelElement(node.TickLowerBound(), aVal)

				t.Logf("genA pushing %d", i)
				AdvanceUntilCanEnqueue(node, 0)
				node.OutputChannel(0).Enqueue(cE)
				node.IncrCycles(OneTick)
			}
		},
	}
	ctx.AddChild(&genA)
	genA.AddOutputChannel(channelA)

	genB := SimpleNode[any]{
		RunFunc: func(node *SimpleNode[any]) {
			for i := 0; i < 10; i++ {
				bVal := datatypes.FixedPoint{Tp: fpt}
				bVal.SetInt(big.NewInt(int64(2 * i)))
				cE := MakeChannelElement(OneTick, bVal)

				t.Logf("genB pushing %d", i)
				AdvanceUntilCanEnqueue(node, 0)
				node.OutputChannel(0).Enqueue(cE)
				node.IncrCycles(OneTick)
			}
		},
	}

	ctx.AddChild(&genB)
	genB.AddOutputChannel(channelB)

	checker := SimpleNode[any]{
		RunFunc: func(node *SimpleNode[any]) {
			for i := 0; i < 10; i++ {
				t.Logf("Checking %d", i)
				var val ChannelElement
				var status Status
				for {
					val, status = node.InputChannel(0).Dequeue()
					if status != Nothing {
						break
					}
					node.IncrCycles(OneTick)
				}
				recv := val.Data.(datatypes.FixedPoint)
				t.Logf("Output %d\n", recv.ToInt())
				if recv.ToInt().Int64() != int64(3*i) {
					t.Errorf("Expected: %d, received: %d", 3*i, recv.ToInt().Int64())
				}
			}
		},
	}

	checker.AddInputChannel(channelC)
	ctx.AddChild(&checker)

	ctx.Init()
	ctx.Run()

	t.Logf("Finished after %s cycles", checker.TickLowerBound().String())
}
