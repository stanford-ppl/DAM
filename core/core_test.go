package core

import (
	"math/big"
	"sync"
	"testing"

	"golang.org/x/exp/maps"

	"github.com/stanford-ppl/DAM/datatypes"
	"github.com/stanford-ppl/DAM/networks/ideal_network"
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

	fpt := datatypes.FixedPointType{true, 32, 0}

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

	node.Step = func(node *Node) {
		// Check if both channels are in the present
		if !node.IsPresent(maps.Values(node.InputChannels)) {
			return
		}

		a := node.InputChannels[0].Channel.Dequeue().Data.(datatypes.FixedPoint)
		b := node.InputChannels[1].Channel.Dequeue().Data.(datatypes.FixedPoint)
		c := datatypes.FixedAdd(a, b)
		node.OutputChannels[0].Channel.Enqueue(MakeElement(&node.tickCount, c))
		t.Logf("%d + %d = %d", a.ToInt().Int64(), b.ToInt().Int64(), c.ToInt().Int64())
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
	t.Logf("Total cycles elapsed: %s", node.tickCount.String())
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

	fpt := datatypes.FixedPointType{true, 32, 0}

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

	node.Step = func(node *Node) {
		a := node.InputChannels[0].Channel.Dequeue().Data.(datatypes.Vector[datatypes.FixedPoint])

		one := datatypes.FixedPoint{Tp: fpt}
		one.SetInt(big.NewInt(int64(1)))

		for i := 0; i < vecWidth; i++ {
			a.Set(i, datatypes.FixedAdd(a.Get(i), one))
		}
		node.OutputChannels[0].Channel.Enqueue(MakeElement(&node.tickCount, a))
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

func TestSimpleNodeIO_Ideal_Network(t *testing.T) {

	net := networks.IdealNetwork[datatypes.FixedPoint]{}

	var channelSize uint = 10
	fpt := datatypes.FixedPointType{true, 32, 0}
	var wg sync.WaitGroup

	inputChannelA := core.NodeInputChannel{ Channel: core.MakeChannel[datatypes.FixedPoint](channelSize), }
	inputChannelB := core.NodeInputChannel{ Channel: core.MakeChannel[datatypes.FixedPoint](channelSize), }
	outputChannel0 := core.NodeOutputChannel{ Channel: core.MakeChannel[datatypes.FixedPoint](channelSize), }
	
	node0 := core.NewNode()
	node0.SetID(0)
	node0.SetInputChannel(0 , inputChannelA)
	node0.SetInputChannel(1 , inputChannelB)
	node0.SetOutputChannel(0 , outputChannel0)

	if !node0.Validate() {
		t.Errorf("Node %d failed validation", node0.ID)	
	}

	node0.Step = func(node *core.Node) {
		a := node.InputChannels[0].Channel.Dequeue().(datatypes.FixedPoint)
		b := node.InputChannels[1].Channel.Dequeue().(datatypes.FixedPoint)
		node.OutputChannels[0].Channel.Enqueue(datatypes.FixedAdd(a, b))
		t.Logf("Node 0: %v and %v --> %v\n" , a.ToInt().Int64() , b.ToInt().Int64() , datatypes.FixedAdd(a, b).ToInt().Int64())
	}

	// ----------------------
	
	inputChannelC := core.NodeInputChannel{ Channel: core.MakeChannel[datatypes.FixedPoint](channelSize), }
	outputChannel2 := core.NodeOutputChannel{ Channel: core.MakeChannel[datatypes.FixedPoint](channelSize), }
	
	node1 := core.NewNode()
	node1.SetID(1)
	node1.SetInputChannel(0 , inputChannelC)
	node1.SetOutputChannel(0 , outputChannel2)

	if !node1.Validate() {
		t.Errorf("Node %d failed validation", node1.ID)	
	}

	node1.Step = func(node *core.Node) {
		a := node.InputChannels[0].Channel.Dequeue().(datatypes.FixedPoint)
		one := datatypes.FixedPoint{Tp: fpt}
		one.SetInt(big.NewInt(int64(1)))
		node.OutputChannels[0].Channel.Enqueue(datatypes.FixedAdd(a, one))
		t.Logf("Node 1: %v --> %v\n" , a.ToInt().Int64() , datatypes.FixedAdd(a, one).ToInt().Int64())
	}

	comchan1 := core.CommunicationChannel[datatypes.FixedPoint]{InputChannel: *(outputChannel0.Channel) , OutputChannel: *(inputChannelC.Channel)}
	networkChannels := []core.CommunicationChannel[datatypes.FixedPoint]{comchan1}
	net.Channels = networkChannels

	quit := make(chan int)

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

	networkTicker := func(quit chan int) {
		for {
			net.TickChannels()
			select {
				case <-quit:
					return 
				default:
			}
		}
	}

	main := func() {
		for i := 0; i < 10; i++ {
			node0.Tick()
			node1.Tick()
		}
		wg.Done()
	}

	checker := func() {
		for i := 0; i < 10; i++ {
			recv := outputChannel2.Channel.Dequeue().(datatypes.FixedPoint)
			t.Logf("Checking %d\n", recv.ToInt())
			if recv.ToInt().Int64() != int64(3*i+1) {
				t.Errorf("Expected: %d, received: %d", 3*i+1 , recv.ToInt().Int64())
			}
		}
		wg.Done()
	}

	wg.Add(4)

	go genA()
	go genB()
	go main()
	go networkTicker(quit)
	go checker()
	
	wg.Wait()
	close(quit)

}
