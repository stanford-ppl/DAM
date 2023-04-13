package functional_test

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

	ctx := core.MakePrimitiveContext(nil)

	vecProducer := core.SimpleNode[any]{
		RunFunc: func(node *core.SimpleNode[any]) {
			result := datatypes.NewVector[datatypes.FixedPoint](N)
			for i := 0; i < N; i++ {
				val := datatypes.FixedPoint{Tp: fpt}
				val.SetInt(big.NewInt(int64(i)))
				result.Set(i, val)
			}
			node.OutputChannels[0].Enqueue(core.MakeElement(node.TickCount(), result))
			node.IncrCycles(1)
		},
	}
	vecProducer.AddOutputChannel(vecToDot)
	ctx.AddChild(&vecProducer)

	matProducer := core.SimpleNode[any]{
		RunFunc: func(node *core.SimpleNode[any]) {
			for j := 0; j < M; j++ {
				result := datatypes.NewVector[datatypes.FixedPoint](N)
				for i := 0; i < N; i++ {
					val := datatypes.FixedPoint{Tp: fpt}
					val.SetInt(big.NewInt(int64(i + j)))
					t.Logf("Mat[%d, %d] = %d", j, i, val.ToInt().Int64())
					result.Set(i, val)
				}
				node.OutputChannels[0].Enqueue(core.MakeElement(node.TickCount(), result))
				node.IncrCycles(int64(timePerVecInMatrix))
			}
		},
	}
	matProducer.AddOutputChannel(matToDot)
	ctx.AddChild(&matProducer)

	type MatVecState struct {
		Vector         datatypes.Vector[datatypes.FixedPoint]
		HasInitialized bool
	}
	dotProduct := core.SimpleNode[MatVecState]{
		RunFunc: func(node *core.SimpleNode[MatVecState]) {
			for j := 0; j < M; j++ {
				matChannel := node.InputChannels[0]
				vecChannel := node.InputChannels[1]

				tick := big.NewInt(0)
				if !node.State.HasInitialized {
					t.Log("Initializing Vector")
					node.State.HasInitialized = true
					timeDelta := new(big.Int)
					vec := vecChannel.Dequeue()
					chanTime := vec.Time
					node.State.Vector = vec.Data.(datatypes.Vector[datatypes.FixedPoint])
					timeDelta.Sub(&chanTime, node.TickCount())
					utils.Max[*big.Int](timeDelta, tick, tick)
				}
				matInput := matChannel.Dequeue()
				timeDelta := new(big.Int)
				timeDelta.Sub(&matInput.Time, node.TickCount())
				utils.Max[*big.Int](timeDelta, tick, tick)

				matVec := matInput.Data.(datatypes.Vector[datatypes.FixedPoint])

				// Now compute the dot product of matVec and state.Vector
				t.Logf("Computing Dot Product")
				sum := datatypes.FixedPoint{Tp: fpt}
				for i := 0; i < N; i++ {
					vA := matVec.Get(i)
					vB := node.State.Vector.Get(i)
					mul := datatypes.FixedMulFull(vA, vB).FixedToFixed(fpt)
					sum = datatypes.FixedAdd(sum, mul)
				}

				outputTime := big.NewInt(int64(dotTime))
				outputTime.Add(node.TickCount(), tick)
				t.Logf("Enqueuing result %d (simulated time %d)", sum.ToInt().Int64(), outputTime.Int64())
				node.OutputChannels[0].Enqueue(core.MakeElement(outputTime, sum))
				node.IncrCyclesBigInt(tick)
			}
		},
	}
	dotProduct.AddInputChannel(matToDot)
	dotProduct.AddInputChannel(vecToDot)
	dotProduct.AddOutputChannel(dotOutput)
	ctx.AddChild(&dotProduct)

	network.Initialize([]core.CommunicationChannel{matToDot, vecToDot, dotOutput})

	var wg sync.WaitGroup
	wg.Add(3) // Checker, Network, Context

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

	go (func() { ctx.Init(); ctx.Run(); wg.Done() })()
	go (func() { network.Run(); wg.Done() })()

	wg.Wait()

	t.Logf("Matrix Producer finished at %d", matProducer.TickCount().Int64())
	t.Logf("Dot Product finished at %d", dotProduct.TickCount().Int64())
	t.Logf("Dot product delay is %d", dotTime)
}
