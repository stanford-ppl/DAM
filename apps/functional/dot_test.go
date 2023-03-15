package apps

import (
	"testing"
	"sync"
	"math/big"

	"github.com/stanford-ppl/DAM/datatypes"
	"github.com/stanford-ppl/DAM/networks/ideal_network"
	"github.com/stanford-ppl/DAM/core"
)

func TestDotProduct(t *testing.T) {
	
	net := networks.IdealNetwork{}

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
		a := node.InputChannels[0].Channel.Dequeue().Data.(datatypes.FixedPoint)
		b := node.InputChannels[1].Channel.Dequeue().Data.(datatypes.FixedPoint)
		c := datatypes.FixedAdd(a, b)
		node.OutputChannels[0].Channel.Enqueue(core.MakeElement(&node.TickCount, c))
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
		a := node.InputChannels[0].Channel.Dequeue().Data.(datatypes.FixedPoint)
		one := datatypes.FixedPoint{Tp: fpt}
		one.SetInt(big.NewInt(int64(1)))
		c := datatypes.FixedAdd(a, one)
		node.OutputChannels[0].Channel.Enqueue(core.MakeElement(&node.TickCount, c))
		t.Logf("Node 1: %v --> %v\n" , a.ToInt().Int64() , datatypes.FixedAdd(a, one).ToInt().Int64())
	}

	comchan1 := core.CommunicationChannel{InputChannel: *(outputChannel0.Channel) , OutputChannel: *(inputChannelC.Channel)}
	networkChannels := []core.CommunicationChannel{comchan1}
	net.Channels = networkChannels

	quit := make(chan int)

	genA := func() {
		for i := 0; i < 10; i++ {
			aVal := datatypes.FixedPoint{Tp: fpt}
			aVal.SetInt(big.NewInt(int64(i)))

			cE := core.ChannelElement{Data: aVal}
			cE.Time.Set(big.NewInt(int64(i)))

			inputChannelA.Channel.Enqueue(cE)
		}
		wg.Done()
	}

	genB := func() {
		for i := 0; i < 10; i++ {
			bVal := datatypes.FixedPoint{Tp: fpt}
			bVal.SetInt(big.NewInt(int64(2 * i)))

			cE := core.ChannelElement{Data: bVal}
			cE.Time.Set(big.NewInt(int64(i)))
			inputChannelB.Channel.Enqueue(cE)
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
			recv := outputChannel2.Channel.Dequeue().Data.(datatypes.FixedPoint)
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