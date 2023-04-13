package functional_test

import (
	"math/big"
	"sync"
	"testing"

	"github.com/stanford-ppl/DAM/networks"

	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/datatypes"
)

var fpt = datatypes.FixedPointType{Signed: true, Integer: 32, Fraction: 0}

func Test_ideal_network(t *testing.T) {
	net := networks.IdealNetwork{}

	var channelSize uint = 5
	fpt := datatypes.FixedPointType{Signed: true, Integer: 32, Fraction: 0}
	var wg sync.WaitGroup

	channelA := core.MakeCommunicationChannel[datatypes.FixedPoint](channelSize)
	channelB := core.MakeCommunicationChannel[datatypes.FixedPoint](channelSize)
	channelC := core.MakeCommunicationChannel[datatypes.FixedPoint](channelSize)
	channelD := core.MakeCommunicationChannel[datatypes.FixedPoint](channelSize)

	ctx := core.MakePrimitiveContext(nil)

	node0 := core.SimpleNode[any]{
		RunFunc: func(sn *core.SimpleNode[any]) {
			for i := 0; i < 10; i++ {
				a := sn.InputChannels[0].Dequeue().Data.(datatypes.FixedPoint)
				b := sn.InputChannels[1].Dequeue().Data.(datatypes.FixedPoint)
				c := datatypes.FixedAdd(a, b)
				sn.OutputChannels[0].Enqueue(core.MakeElement(sn.TickCount(), c))
				sn.TickCount().Add(sn.TickCount(), big.NewInt(1))
			}
		},
	}
	node0.AddInputChannel(channelA)
	node0.AddInputChannel(channelB)
	node0.AddOutputChannel(channelC)

	// ----------------------

	node1 := core.SimpleNode[any]{
		RunFunc: func(node *core.SimpleNode[any]) {
			for i := 0; i < 10; i++ {
				a := node.InputChannels[0].Dequeue().Data.(datatypes.FixedPoint)
				one := datatypes.FixedPoint{Tp: fpt}
				one.SetInt(big.NewInt(int64(1)))
				c := datatypes.FixedAdd(a, one)
				node.OutputChannels[0].Enqueue(core.MakeElement(node.TickCount(), c))
				node.TickCount().Add(node.TickCount(), big.NewInt(1))
			}
		},
	}
	node1.AddInputChannel(channelC)
	node1.AddOutputChannel(channelD)

	ctx.AddChild(&node0)
	ctx.AddChild(&node1)

	net.Initialize([]core.CommunicationChannel{channelA, channelB, channelC, channelD})
	t.Logf("Network Initialized")

	ctx.Init()
	t.Logf("Context Initialized")

	genA := func() {
		for i := 0; i < 10; i++ {
			aVal := datatypes.FixedPoint{Tp: fpt}
			aVal.SetInt(big.NewInt(int64(i)))
			channelA.OutputChannel.Enqueue(core.MakeElement(big.NewInt(int64(i)), aVal))
		}
		wg.Done()
	}

	genB := func() {
		for i := 0; i < 10; i++ {
			bVal := datatypes.FixedPoint{Tp: fpt}
			bVal.SetInt(big.NewInt(int64(2 * i)))
			channelB.OutputChannel.Enqueue(core.MakeElement(big.NewInt(int64(i)), bVal))
		}
		wg.Done()
	}
	checker := func() {
		for i := 0; i < 10; i++ {
			recv := channelD.InputChannel.Dequeue().Data.(datatypes.FixedPoint)
			t.Logf("Checking %d\n", recv.ToInt())
			if recv.ToInt().Int64() != int64(3*i+1) {
				t.Errorf("Expected: %d, received: %d", 3*i+1, recv.ToInt().Int64())
			}
		}
		net.Kill()
		wg.Done()
	}

	wg.Add(5)

	go genA()
	go genB()
	go (func() { ctx.Run(); wg.Done() })()
	go checker()
	go (func() { net.Run(); wg.Done() })()

	wg.Wait()
}

func Test_ideal_network_2(t *testing.T) {
	net := networks.IdealNetwork{}

	var channelSize uint = 10
	fpt := datatypes.FixedPointType{Signed: true, Integer: 32, Fraction: 0}
	var wg sync.WaitGroup

	mkChan := func() core.CommunicationChannel {
		return core.MakeCommunicationChannel[datatypes.FixedPoint](channelSize)
	}
	channelA := mkChan()
	channelB := mkChan()
	channelC := mkChan()
	channelD := mkChan()
	channelE := mkChan()
	net.Initialize([]core.CommunicationChannel{channelA, channelB, channelC, channelD, channelE})

	node0 := core.SimpleNode[any]{
		RunFunc: func(node *core.SimpleNode[any]) {
			for i := 0; i < 10; i++ {
				a := node.InputChannels[0].Dequeue().Data.(datatypes.FixedPoint)
				b := node.InputChannels[1].Dequeue().Data.(datatypes.FixedPoint)
				c := datatypes.FixedAdd(a, b)
				node.OutputChannels[0].Enqueue(core.MakeElement(node.TickCount(), c))
				node.OutputChannels[1].Enqueue(core.MakeElement(node.TickCount(), c))
				t.Logf("Node 0: %v and %v --> %v\n", a.ToInt().Int64(), b.ToInt().Int64(), datatypes.FixedAdd(a, b).ToInt().Int64())
				node.TickCount().Add(node.TickCount(), big.NewInt(1))
			}
		},
	}
	node0.AddInputChannel(channelA)
	node0.AddInputChannel(channelB)
	node0.AddOutputChannel(channelC)
	node0.AddOutputChannel(channelD)

	// ----------------------

	node1 := core.SimpleNode[any]{
		RunFunc: func(node *core.SimpleNode[any]) {
			for i := 0; i < 10; i++ {
				a := node.InputChannels[0].Dequeue().Data.(datatypes.FixedPoint)
				one := datatypes.FixedPoint{Tp: fpt}
				one.SetInt(big.NewInt(int64(1)))
				c := datatypes.FixedAdd(a, one)
				node.OutputChannels[0].Enqueue(core.MakeElement(node.TickCount(), c))
				t.Logf("Node 1: %v --> %v\n", a.ToInt().Int64(), datatypes.FixedAdd(a, one).ToInt().Int64())
				node.TickCount().Add(node.TickCount(), big.NewInt(1))
			}
		},
	}
	node1.AddInputChannel(channelC)
	node1.AddOutputChannel(channelE)

	ctx := core.MakePrimitiveContext(nil)
	ctx.AddChild(&node0)
	ctx.AddChild(&node1)
	ctx.Init()

	genA := func() {
		for i := 0; i < 10; i++ {
			aVal := datatypes.FixedPoint{Tp: fpt}
			aVal.SetInt(big.NewInt(int64(i)))

			cE := core.ChannelElement{Data: aVal}
			cE.Time.Set(big.NewInt(int64(i)))

			channelA.OutputChannel.Enqueue(cE)
		}
		wg.Done()
	}

	genB := func() {
		for i := 0; i < 10; i++ {
			bVal := datatypes.FixedPoint{Tp: fpt}
			bVal.SetInt(big.NewInt(int64(2 * i)))

			cE := core.ChannelElement{Data: bVal}
			cE.Time.Set(big.NewInt(int64(i)))
			channelB.OutputChannel.Enqueue(cE)
		}
		wg.Done()
	}

	checker := func() {
		for i := 0; i < 10; i++ {
			recv := channelE.InputChannel.Dequeue().Data.(datatypes.FixedPoint)
			t.Logf("Recv: %s", recv)
			recv_2 := channelD.InputChannel.Dequeue().Data.(datatypes.FixedPoint)
			t.Logf("Recv_2: %s", recv_2)
			res := datatypes.FixedAdd(recv, recv_2)
			t.Logf("Checking %d %d\n", i, res.ToInt())
			if res.ToInt().Int64() != int64(6*i+1) {
				t.Errorf("Expected: %d, received: %d", 6*i+1, res.ToInt().Int64())
			}
		}
		t.Logf("Killing the channel")
		net.Kill()
		wg.Done()
	}

	wg.Add(5)

	go genA()
	go genB()
	go (func() { ctx.Run(); wg.Done() })()
	go (func() { net.Run(); wg.Done() })()
	go checker()

	wg.Wait()
}
