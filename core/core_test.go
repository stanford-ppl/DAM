package core

import (
	"math/big"
	"sync"
	"testing"

	"github.com/stanford-ppl/DAM/datatypes"
	"github.com/stanford-ppl/DAM/utils"
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
				elements := DequeueInputChansByID(node, 0, 1)
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

func TestDequeueChannels(t *testing.T) {
	numChannels := 3

	const (
		Past    int64 = 3
		Present int64 = 5
		Future  int64 = 8
	)

	timeDeltas := []int64{Past, Present, Future}

	triggers := make([]chan *Time, numChannels)
	utils.Fill(triggers, func() chan *Time { return make(chan *Time, 1) })

	resps := make([]chan struct{}, numChannels)
	utils.Fill(resps, func() chan struct{} { return make(chan struct{}, 1) })

	killers := make([]chan struct{}, numChannels)
	utils.Fill(killers, func() chan struct{} { return make(chan struct{}) })

	enqueueChannels := make([]*CommunicationChannel, numChannels)
	utils.Fill(enqueueChannels, func() *CommunicationChannel {
		return MakeCommunicationChannel[datatypes.Bit](4)
	})
	enqueuers := make([]Context, numChannels)
	utils.Tabulate(enqueuers, func(id int) Context {
		node := &SimpleNode[any]{
			RunFunc: func(node *SimpleNode[any]) {
				for {
					select {
					case <-killers[id]:
						return
					case time := <-triggers[id]:
						AdvanceUntilCanEnqueue(node, 0)
						newTime := Time{}
						utils.Max[*Time](time, node.TickLowerBound(), &newTime)
						node.outputChannels[0].Enqueue(MakeChannelElement(&newTime, datatypes.Bit{Value: false}))
						node.AdvanceToTime(&newTime)
						resps[id] <- struct{}{}
					}
				}
			},
		}
		node.AddOutputChannel(enqueueChannels[id])
		return node
	})

	killer := make(chan struct{})
	enable := make(chan *Time, 1)
	checker := &SimpleNode[any]{
		RunFunc: func(node *SimpleNode[any]) {
			inputChannels := make([]InputChannel, len(node.inputChannels))
			utils.Tabulate(inputChannels, node.InputChannel)
			for {
				select {
				case <-killer:
					return
				case time := <-enable:
					node.AdvanceToTime(time)
					dequeued := DequeueInputChannels(node, inputChannels...)
					t.Log(dequeued)
					t.Log(node.TickLowerBound())
				}
			}
		},
	}
	for _, cc := range enqueueChannels {
		checker.AddInputChannel(cc)
	}

	parentCtx := MakePrimitiveContext(nil)
	utils.Foreach(enqueuers, parentCtx.AddChild)
	parentCtx.AddChild(checker)

	parentCtx.Init()
	var wg sync.WaitGroup
	wg.Add(2)
	go (func() { parentCtx.Run(); wg.Done() })()
	go (func() {
		combs := make([][]int64, numChannels)
		for i := range combs {
			combs[i] = timeDeltas
		}

		curTime := int64(0)
		for timeDelta := range utils.CartesianProduct(combs...) {
			for enqId := 0; enqId < numChannels; enqId++ {
				delta := timeDelta[enqId]
				triggers[enqId] <- NewTime(curTime + delta)
			}
			enable <- NewTime(curTime + Present)

			for _, v := range resps {
				<-v
			}

			curTime = curTime + int64(10)
		}

		utils.Foreach(killers, func(ch chan struct{}) { close(ch) })
		close(killer)
		wg.Done()
	})()
	wg.Wait()
}

// // This test tests DequeueBundle, for reading one of multiple groups of nodes. As a result, it can be somewhat complicated.
// func TestDequeueUtils(t *testing.T) {
// 	// For this test, we have four enqueuers. They are skewed in real-time via the trigger channels.
// 	numBundles := 3
// 	bundleWidth := 3
// 	numEnqueuers := numBundles * bundleWidth

// 	signalDepth := 1

// 	triggers := make([]chan *Time, numEnqueuers)
// 	utils.Fill(triggers, func() chan *Time { return make(chan *Time, signalDepth) })

// 	resps := make([]chan struct{}, numEnqueuers)
// 	utils.Fill(resps, func() chan struct{} { return make(chan struct{}, signalDepth) })

// 	killers := make([]chan struct{}, numEnqueuers)
// 	utils.Fill(killers, func() chan struct{} { return make(chan struct{}) })

// 	enqueueChannels := make([]CommunicationChannel, numEnqueuers)

// 	fpt := datatypes.FixedPointType{true, 32, 0}

// 	enqueuers := make([]Context, numEnqueuers)
// 	utils.Tabulate(enqueuers, func(lid int) Context {
// 		id := lid
// 		sn := &SimpleNode[any]{
// 			RunFunc: func(node *SimpleNode[any]) {
// 				for i := 0; true; i++ {
// 					// Allows this thread to run
// 					select {
// 					case time := <-triggers[id]:
// 						data := datatypes.FixedPoint{Tp: fpt}
// 						data.SetInt64(int64(i))
// 						node.outputChannels[0].Enqueue(MakeChannelElement(time, data))
// 						node.AdvanceToTime(time)
// 						// Responds after this thread finishes
// 						resps[id] <- struct{}{}
// 					case <-killers[id]:
// 						return
// 					}
// 				}
// 			},
// 		}
// 		return sn
// 	})
// }
