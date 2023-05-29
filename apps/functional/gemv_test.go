package functional_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/datatypes"
)

// This test runs a Matrix-Vector (M x N) x (N) product
// This assumes a black-box dot product operation
// capable of doing a N-element dot product.
func TestNetworkWithBigStep(t *testing.T) {
	M := 1024
	N := 16
	timePerVecInMatrix := 32
	// Assume that it takes log2(vecSize) + 1 time to run a dot product
	dotTime := int(math.Log2(float64(N))) + 1

	fpt := datatypes.FixedPointType{Signed: true, Integer: 32, Fraction: 0}

	channelSize := 8

	// We have three nodes --
	// a matrix producer, a vector producer, and a dot product unit.

	matToDot := core.MakeCommunicationChannel[datatypes.
		Vector[datatypes.FixedPoint]](timePerVecInMatrix + 1)
	vecToDot := core.MakeCommunicationChannel[datatypes.
		Vector[datatypes.FixedPoint]](channelSize)
	dotOutput := core.MakeCommunicationChannel[datatypes.FixedPoint](M)

	ctx := core.MakePrimitiveContext(nil)

	vecProducer := core.SimpleNode[any]{
		RunFunc: func(node *core.SimpleNode[any]) {
			result := datatypes.NewVector[datatypes.FixedPoint](N)
			for i := 0; i < N; i++ {
				val := datatypes.FixedPoint{Tp: fpt}
				val.SetInt(big.NewInt(int64(i)))
				result.Set(i, val)
			}
			node.OutputChannel(0).
				Enqueue(core.MakeChannelElement(node.TickLowerBound(), result))
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
					// t.Logf("Mat[%d, %d] = %d", j, i, val.ToInt().Int64())
					result.Set(i, val)
				}
				// Lower bound on when we can tick again
				nextTick := node.TickLowerBound()
				nextTick.Add(nextTick, core.NewTime(int64(timePerVecInMatrix)))
				core.AdvanceUntilCanEnqueue(node, 0)
				node.OutputChannel(0).Enqueue(core.
					MakeChannelElement(node.TickLowerBound(), result))
				node.AdvanceToTime(nextTick)
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

				tick := core.NewTime(0)
				if !node.State.HasInitialized {
					t.Log("Initializing Vector")
					node.State.HasInitialized = true
					tmp := core.DequeueInputChansByID(node, 0)
					vec := tmp[0]
					node.State.Vector = vec.Data.(datatypes.
						Vector[datatypes.FixedPoint])
				}
				tmp := core.DequeueInputChansByID(node, 1)
				matVec := tmp[0].Data.(datatypes.Vector[datatypes.FixedPoint])

				// Now compute the dot product of matVec and state.Vector
				t.Logf("Computing Dot Product")
				sum := datatypes.FixedPoint{Tp: fpt}
				for i := 0; i < N; i++ {
					vA := matVec.Get(i)
					vB := node.State.Vector.Get(i)
					mul := datatypes.FixedMulFull(vA, vB).FixedToFixed(fpt)
					sum = datatypes.FixedAdd(sum, mul)
				}

				outputTime := core.NewTime(int64(dotTime))
				outputTime.Add(node.TickLowerBound(), tick)
				core.AdvanceUntilCanEnqueue(node, 0)
				t.Logf("Enqueuing result %d (simulated time %v)", j, outputTime)
				node.OutputChannel(0).Enqueue(core.
					MakeChannelElement(outputTime, sum))
				node.IncrCycles(tick)
			}
			t.Logf("Dot Product Finished")
		},
	}
	dotProduct.AddInputChannel(vecToDot)
	dotProduct.AddInputChannel(matToDot)
	dotProduct.AddOutputChannel(dotOutput)
	ctx.AddChild(&dotProduct)

	checker := core.SimpleNode[any]{
		RunFunc: func(node *core.SimpleNode[any]) {
			for i := 0; i < M; i++ {
				recv := core.DequeueInputChansByID(node, 0)[0]
				t.Logf("Checking iteration %d", i)
				t.Logf("Received: %v", recv.Status)
				recvVal := recv.Data.(datatypes.FixedPoint).ToInt().Int64()
				// The reference value for element i is
				// Sum(a * (a + i) for a in range(N))
				var refVal int = 0
				for tmp := 0; tmp < N; tmp++ {
					refVal += tmp * (tmp + i)
				}
				if recvVal != int64(refVal) {
					t.Errorf("Error at element %d, %d != %d",
						i, recvVal, refVal)
				}
			}
		},
	}
	checker.AddInputChannel(dotOutput)
	ctx.AddChild(&checker)

	ctx.Init()
	ctx.Run()
	t.Logf("Matrix Producer finished at %v", matProducer.TickLowerBound())
	t.Logf("Dot Product finished at %v", dotProduct.TickLowerBound())
	t.Logf("Dot product delay is %d", dotTime)
}
