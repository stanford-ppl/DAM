package apps

import (
	"math/big"
	"sync"
	"testing"

	"github.com/stanford-ppl/DAM/networks"

	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/datatypes"
)

// TODO: This test should be in `core_test.go` but Bazel says it cannot find out where `ideal_network` package is.  Find out why this is the case.
func Test_ideal_network(t *testing.T) {
	net := networks.IdealNetwork{}

	var channelSize uint = 5
	fpt := datatypes.FixedPointType{Signed: true, Integer: 32, Fraction: 0}
	var wg sync.WaitGroup

	channelA := core.MakeCommunicationChannel[datatypes.FixedPoint](channelSize)
	channelB := core.MakeCommunicationChannel[datatypes.FixedPoint](channelSize)
	channelC := core.MakeCommunicationChannel[datatypes.FixedPoint](channelSize)

	node0 := core.NewNode()
	node0.SetID(0)
	node0.SetInputChannel(0, channelA)
	node0.SetInputChannel(1, channelB)
	node0.SetOutputChannel(0, channelC)

	if !node0.Validate() {
		t.Errorf("Node %d failed validation", node0.ID)
	}

	node0.Step = func(node *core.Node, _ *big.Int) *big.Int {
		a := node.InputChannels[0].Dequeue().Data.(datatypes.FixedPoint)
		b := node.InputChannels[1].Dequeue().Data.(datatypes.FixedPoint)
		c := datatypes.FixedAdd(a, b)
		node.OutputChannels[0].Enqueue(core.MakeElement(&node.TickCount, c))
		t.Logf("Node 0: %v and %v --> %v\n", a.ToInt().Int64(), b.ToInt().Int64(), datatypes.FixedAdd(a, b).ToInt().Int64())
		return big.NewInt(1)
	}

	// ----------------------

	channelD := core.MakeCommunicationChannel[datatypes.FixedPoint](channelSize)
	node1 := core.NewNode()
	node1.SetID(1)
	node1.SetInputChannel(0, channelC)
	node1.SetOutputChannel(0, channelD)

	if !node1.Validate() {
		t.Errorf("Node %d failed validation", node1.ID)
	}

	node1.Step = func(node *core.Node, _ *big.Int) *big.Int {
		a := node.InputChannels[0].Dequeue().Data.(datatypes.FixedPoint)
		one := datatypes.FixedPoint{Tp: fpt}
		one.SetInt(big.NewInt(int64(1)))
		c := datatypes.FixedAdd(a, one)
		node.OutputChannels[0].Enqueue(core.MakeElement(&node.TickCount, c))
		t.Logf("Node 1: %v --> %v\n", a.ToInt().Int64(), datatypes.FixedAdd(a, one).ToInt().Int64())
		return big.NewInt(1)
	}

	net.Initialize([]core.CommunicationChannel{channelA, channelB, channelC, channelD})
	t.Logf("Network Initialized")

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

	node0Ticker := func() {
		for i := 0; i < 10; i++ {
			node0.Tick()
		}
		wg.Done()
	}

	node1Ticker := func() {
		for i := 0; i < 10; i++ {
			node1.Tick()
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

	wg.Add(6)

	go genA()
	go genB()
	go node0Ticker()
	go node1Ticker()
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

	node0 := core.NewNode()
	node0.SetID(0)
	node0.SetInputChannel(0, channelA)
	node0.SetInputChannel(1, channelB)
	node0.SetOutputChannel(0, channelC)
	node0.SetOutputChannel(1, channelD)

	if !node0.Validate() {
		t.Errorf("Node %d failed validation", node0.ID)
	}

	node0.Step = func(node *core.Node, _ *big.Int) *big.Int {
		a := node.InputChannels[0].Dequeue().Data.(datatypes.FixedPoint)
		b := node.InputChannels[1].Dequeue().Data.(datatypes.FixedPoint)
		c := datatypes.FixedAdd(a, b)
		node.OutputChannels[0].Enqueue(core.MakeElement(&node.TickCount, c))
		node.OutputChannels[1].Enqueue(core.MakeElement(&node.TickCount, c))
		t.Logf("Node 0: %v and %v --> %v\n", a.ToInt().Int64(), b.ToInt().Int64(), datatypes.FixedAdd(a, b).ToInt().Int64())
		return big.NewInt(1)
	}

	// ----------------------

	node1 := core.NewNode()
	node1.SetID(1)
	node1.SetInputChannel(0, channelC)
	node1.SetOutputChannel(0, channelE)

	if !node1.Validate() {
		t.Errorf("Node %d failed validation", node1.ID)
	}

	node1.Step = func(node *core.Node, _ *big.Int) *big.Int {
		a := node.InputChannels[0].Dequeue().Data.(datatypes.FixedPoint)
		one := datatypes.FixedPoint{Tp: fpt}
		one.SetInt(big.NewInt(int64(1)))
		c := datatypes.FixedAdd(a, one)
		node.OutputChannels[0].Enqueue(core.MakeElement(&node.TickCount, c))
		t.Logf("Node 1: %v --> %v\n", a.ToInt().Int64(), datatypes.FixedAdd(a, one).ToInt().Int64())
		return big.NewInt(1)
	}

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

	node0Ticker := func() {
		for i := 0; i < 10; i++ {
			node0.Tick()
		}
		wg.Done()
	}

	node1Ticker := func() {
		for i := 0; i < 10; i++ {
			node1.Tick()
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

	wg.Add(6)

	go genA()
	go genB()
	go node0Ticker()
	go node1Ticker()
	go (func() { net.Run(); wg.Done() })()
	go checker()

	wg.Wait()
}
