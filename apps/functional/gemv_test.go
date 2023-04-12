package apps

import (
	"math"
	"math/big"
	"sync"
	"testing"

	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/datatypes"
	"github.com/stanford-ppl/DAM/networks"
	"github.com/stanford-ppl/DAM/utils"
)

// This test runs a Matrix-Vector (M x N) x (N) product
// This assumes a black-box dot product operation capable of doing a N-element dot product.
func TestNetworkWithBigStep(t *testing.T) {
	M := 1024
	N := 16
	timePerVecInMatrix := 32
	// Assume that it takes log2(vecSize) + 1 time to run a dot product
	dotTime := int(math.Log2(float64(N))) + 1

	fpt := datatypes.FixedPointType{Signed: true, Integer: 32, Fraction: 0}

	channelSize := uint(8)

	// We have three nodes -- a matrix producer, a vector producer, and a dot product unit.

	matToDot := core.MakeCommunicationChannel[datatypes.Vector[datatypes.FixedPoint]](channelSize)
	vecToDot := core.MakeCommunicationChannel[datatypes.Vector[datatypes.FixedPoint]](channelSize)
	dotOutput := core.MakeCommunicationChannel[datatypes.FixedPoint](uint(M))

	network := networks.IdealNetwork{}

	vecProducer := core.NewNode()
	vecProducer.SetID(0)
	vecProducer.SetOutputChannel(0, vecToDot)

	matProducer := core.NewNode()
	matProducer.SetID(1)
	matProducer.SetOutputChannel(0, matToDot)

	vecProducer.Step = func(node *core.Node, _ *big.Int) *big.Int {
		result := datatypes.NewVector[datatypes.FixedPoint](N)
		for i := 0; i < N; i++ {
			val := datatypes.FixedPoint{Tp: fpt}
			val.SetInt(big.NewInt(int64(i)))
			result.Set(i, val)
		}
		node.OutputChannels[0].Enqueue(core.MakeElement(&node.TickCount, result))
		return big.NewInt(1)
	}

	matProducer.Step = func(node *core.Node, _ *big.Int) *big.Int {
		result := datatypes.NewVector[datatypes.FixedPoint](N)
		for i := 0; i < N; i++ {
			val := datatypes.FixedPoint{Tp: fpt}
			val.SetInt(big.NewInt(int64(i + node.State.(int))))
			t.Logf("Mat[%d, %d] = %d", node.State.(int), i, val.ToInt().Int64())
			result.Set(i, val)
		}
		node.OutputChannels[0].Enqueue(core.MakeElement(&node.TickCount, result))
		return big.NewInt(int64(timePerVecInMatrix))
	}

	dotProduct := core.NewNode()
	dotProduct.SetID(2)
	dotProduct.SetInputChannel(0, matToDot)
	dotProduct.SetInputChannel(1, vecToDot)
	dotProduct.SetOutputChannel(0, dotOutput)

	network.Initialize([]core.CommunicationChannel{matToDot, vecToDot, dotOutput})

	// The state is whether we've read yet.
	type MatVecState struct {
		Vector         datatypes.Vector[datatypes.FixedPoint]
		HasInitialized bool
	}
	dotProduct.State = new(MatVecState)
	dotProduct.State.(*MatVecState).HasInitialized = false

	dotProduct.Step = func(node *core.Node, _ *big.Int) (tick *big.Int) {
		// Wait to have inputs on both inputs
		// We read from the vector input once, and the matrix one every time.
		state := node.State.(*MatVecState)

		matChannel := node.InputChannels[0]
		vecChannel := node.InputChannels[1]

		tick = big.NewInt(0)
		if !state.HasInitialized {
			t.Log("Initializing Vector")
			state.HasInitialized = true
			timeDelta := new(big.Int)
			vec := vecChannel.Dequeue()
			chanTime := vec.Time
			state.Vector = vec.Data.(datatypes.Vector[datatypes.FixedPoint])
			timeDelta.Sub(&chanTime, &node.TickCount)
			utils.Max[*big.Int](timeDelta, tick, tick)
		}
		matInput := matChannel.Dequeue()
		timeDelta := new(big.Int)
		timeDelta.Sub(&matInput.Time, &node.TickCount)
		utils.Max[*big.Int](timeDelta, tick, tick)

		matVec := matInput.Data.(datatypes.Vector[datatypes.FixedPoint])

		// Now compute the dot product of matVec and state.Vector
		t.Logf("Computing Dot Product")
		sum := datatypes.FixedPoint{Tp: fpt}
		for i := 0; i < N; i++ {
			vA := matVec.Get(i)
			vB := state.Vector.Get(i)
			mul := datatypes.FixedMulFull(vA, vB).FixedToFixed(fpt)
			sum = datatypes.FixedAdd(sum, mul)
		}

		outputTime := big.NewInt(int64(dotTime))
		outputTime.Add(&node.TickCount, tick)
		t.Logf("Enqueuing result %d (simulated time %d)", sum.ToInt().Int64(), outputTime.Int64())
		node.OutputChannels[0].Enqueue(core.MakeElement(outputTime, sum))
		return
	}

	// This only ever ticks once
	vecProducer.Tick()

	var wg sync.WaitGroup
	wg.Add(4) // Matrix Producer, Dot Product, Checker, network
	// Ticks the matrix producer
	go (func() {
		for i := 0; i < M; i++ {
			matProducer.State = i
			matProducer.Tick()
		}
		wg.Done()
	})()

	// Ticks the dot product
	go (func() {
		for i := 0; i < M; i++ {
			dotProduct.Tick()
		}
		wg.Done()
	})()

	// checker
	go (func() {
		for i := 0; i < M; i++ {
			recv := dotOutput.InputChannel.Dequeue()
			recvVal := recv.Data.(datatypes.FixedPoint).ToInt().Int64()
			// The reference value for element i is Sum(a * (a + i) for a in range(N))
			var refVal int = 0
			for tmp := 0; tmp < N; tmp++ {
				refVal += tmp * (tmp + i)
			}
			if recvVal != int64(refVal) {
				t.Errorf("Error at element %d, %d != %d", i, recvVal, refVal)
			}
		}
		network.Kill()
		wg.Done()
	})()

	go (func() { network.Run(); wg.Done() })()

	wg.Wait()

	t.Logf("Matrix Producer finished at %d", matProducer.TickCount.Int64())
	t.Logf("Dot Product finished at %d", dotProduct.TickCount.Int64())
	t.Logf("Dot product delay is %d", dotTime)
}

func TestNetworkWithAbstractValues(t *testing.T) {
	M := 1024
	N := 16
	timePerVecInMatrix := 32
	// Assume that it takes log2(vecSize) + 1 time to run a dot product
	dotTime := int(math.Log2(float64(N))) + 1

	fpt := datatypes.FixedPointType{Signed: true, Integer: 32, Fraction: 0}

	channelSize := uint(8)

	// We have three nodes -- a matrix producer, a vector producer, and a dot product unit.

	matToDot := core.MakeCommunicationChannel[datatypes.AbstractValue](channelSize)
	vecToDot := core.MakeCommunicationChannel[datatypes.AbstractValue](channelSize)
	dotOutput := core.MakeCommunicationChannel[datatypes.AbstractValue](uint(M))

	network := networks.IdealNetwork{}

	vecProducer := core.NewNode()
	vecProducer.SetID(0)
	vecProducer.SetOutputChannel(0, vecToDot)

	matProducer := core.NewNode()
	matProducer.SetID(1)
	matProducer.SetOutputChannel(0, matToDot)

	vecProducer.Step = func(node *core.Node, _ *big.Int) *big.Int {
		result := datatypes.AbstractValue{
			FakeSize: big.NewInt(int64(N * int(fpt.NBits()))),
		}
		node.OutputChannels[0].Enqueue(core.MakeElement(&node.TickCount, result))
		return big.NewInt(1)
	}

	matProducer.Step = func(node *core.Node, _ *big.Int) *big.Int {
		result := datatypes.AbstractValue{
			FakeSize: big.NewInt(int64(N * int(fpt.NBits()))),
		}
		node.OutputChannels[0].Enqueue(core.MakeElement(&node.TickCount, result))
		return big.NewInt(int64(timePerVecInMatrix))
	}

	dotProduct := core.NewNode()
	dotProduct.SetID(2)
	dotProduct.SetInputChannel(0, matToDot)
	dotProduct.SetInputChannel(1, vecToDot)
	dotProduct.SetOutputChannel(0, dotOutput)

	network.Initialize([]core.CommunicationChannel{matToDot, vecToDot, dotOutput})

	// The state is whether we've read yet.
	type MatVecState struct {
		Vector         datatypes.AbstractValue
		HasInitialized bool
	}
	dotProduct.State = new(MatVecState)
	dotProduct.State.(*MatVecState).HasInitialized = false

	dotProduct.Step = func(node *core.Node, _ *big.Int) (tick *big.Int) {
		// Wait to have inputs on both inputs
		// We read from the vector input once, and the matrix one every time.
		state := node.State.(*MatVecState)

		matChannel := node.InputChannels[0]
		vecChannel := node.InputChannels[1]

		tick = big.NewInt(0)
		if !state.HasInitialized {
			t.Log("Initializing Vector")
			state.HasInitialized = true
			timeDelta := new(big.Int)
			vec := vecChannel.Dequeue()
			chanTime := vec.Time
			state.Vector = vec.Data.(datatypes.AbstractValue)
			timeDelta.Sub(&chanTime, &node.TickCount)
			utils.Max[*big.Int](timeDelta, tick, tick)
		}
		matInput := matChannel.Dequeue()
		timeDelta := new(big.Int)
		timeDelta.Sub(&matInput.Time, &node.TickCount)
		utils.Max[*big.Int](timeDelta, tick, tick)

		// Now compute the dot product of matVec and state.Vector
		t.Logf("Computing Dot Product")

		outputTime := big.NewInt(int64(dotTime))
		outputTime.Add(&node.TickCount, tick)
		t.Logf("Enqueuing result (simulated time %d)", outputTime.Int64())
		node.OutputChannels[0].Enqueue(core.MakeElement(outputTime, datatypes.AbstractValue{FakeSize: big.NewInt(int64(fpt.NBits()))}))
		return
	}

	// This only ever ticks once
	vecProducer.Tick()

	var wg sync.WaitGroup
	wg.Add(4) // Matrix Producer, Dot Product, Checker, network
	// Ticks the matrix producer
	go (func() {
		for i := 0; i < M; i++ {
			matProducer.State = i
			matProducer.Tick()
		}
		wg.Done()
	})()

	// Ticks the dot product
	go (func() {
		for i := 0; i < M; i++ {
			dotProduct.Tick()
		}
		wg.Done()
	})()

	// checker
	go (func() {
		for i := 0; i < M; i++ {
			dotOutput.InputChannel.Dequeue()
		}
		network.Kill()
		wg.Done()
	})()

	go (func() { network.Run(); wg.Done() })()

	wg.Wait()

	t.Logf("Matrix Producer finished at %d", matProducer.TickCount.Int64())
	t.Logf("Dot Product finished at %d", dotProduct.TickCount.Int64())
	t.Logf("Dot product delay is %d", dotTime)
}
