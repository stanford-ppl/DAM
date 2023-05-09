package plasticine

import (
	"math/big"
	"testing"

	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/datatypes"
	"github.com/stanford-ppl/DAM/utils"
)

func TestPMURW(t *testing.T) {
	ctx := core.MakePrimitiveContext(nil)
	comms := []*core.CommunicationChannel{}

	// This test has one writer, one write ack, and two readers to check the result.

	numIters := 32
	vecWidth := 16
	totalNums := numIters * vecWidth
	// Make chanSize large enough to not stall
	chanSize := totalNums

	fpt := datatypes.FixedPointType{Signed: true, Integer: 16, Fraction: 0}
	idxType := datatypes.FixedPointType{Signed: false, Integer: 16, Fraction: 0}

	pmu := MakePMU[datatypes.FixedPoint](int64(totalNums), 6, MakeBehavior())
	ctx.AddChild(pmu)

	// These go from the PMU to the read issuers
	wAck1 := core.MakeCommunicationChannel[datatypes.Bit](chanSize)
	wAck2 := core.MakeCommunicationChannel[datatypes.Bit](chanSize)

	{
		// Vector write to PMU
		wData := core.MakeCommunicationChannel[datatypes.Vector[datatypes.FixedPoint]](chanSize)
		wAddr := core.MakeCommunicationChannel[datatypes.FixedPoint](chanSize)
		comms = append(comms, wData, wAddr)

		writer := core.SimpleNode[any]{
			RunFunc: func(node *core.SimpleNode[any]) {
				for i := 0; i < numIters; i++ {
					curTime := node.TickLowerBound()
					data := datatypes.NewVector[datatypes.FixedPoint](vecWidth)
					for iV := 0; iV < vecWidth; iV++ {
						fp := datatypes.FixedPoint{Tp: fpt}
						fp.SetInt(big.NewInt(int64(iV + i*vecWidth)))
						data.Set(iV, fp)
					}
					curTime.Add(core.OneTick, curTime)
					t.Logf("Enqueuing write for %d at time %v", i*vecWidth, curTime)
					node.OutputChannel(0).Enqueue(core.MakeChannelElement(curTime, data))

					index := datatypes.FixedPoint{Tp: idxType}
					index.SetInt(big.NewInt(int64(i * vecWidth)))
					node.OutputChannel(1).Enqueue(core.MakeChannelElement(curTime, index))
					node.IncrCycles(core.OneTick)
				}
			},
		}
		writer.AddOutputChannel(wData)
		writer.AddOutputChannel(wAddr)
		ctx.AddChild(&writer)
		pmu.AddWriter(wAddr, wData, utils.None[*core.CommunicationChannel](), []*core.CommunicationChannel{wAck1, wAck2}, Vector{})
	}

	// Scalar read
	{
		readAddr := core.MakeCommunicationChannel[datatypes.FixedPoint](chanSize)
		readResult := core.MakeCommunicationChannel[datatypes.FixedPoint](chanSize)
		comms = append(comms, readAddr, readResult)
		readIssue := core.SimpleNode[any]{
			RunFunc: func(node *core.SimpleNode[any]) {
				for j := 0; j < numIters; j++ {
					_ = core.DequeueInputChannels(node, 0)
					t.Log("Current Time on ReadIssue:", node.TickLowerBound())
					for i := 0; i < vecWidth; i++ {
						node.IncrCycles(core.OneTick)
						ind := datatypes.FixedPoint{Tp: idxType}
						ind.SetInt64(int64(i + j*vecWidth))
						node.OutputChannel(0).Enqueue(core.MakeChannelElement(node.TickLowerBound(), ind))
					}
				}
				t.Log("ReadIssue Finished")
			},
		}
		readIssue.AddInputChannel(wAck1)
		readIssue.AddOutputChannel(readAddr)
		ctx.AddChild(&readIssue)

		readProcess := core.SimpleNode[any]{
			RunFunc: func(node *core.SimpleNode[any]) {
				for i := 0; i < totalNums; i++ {
					read := core.DequeueInputChannels(node, 0)[0]
					node.IncrCycles(core.OneTick)
					fetched := read.Data.(datatypes.FixedPoint)
					t.Logf("Fetched %d at iteration %d", fetched.ToInt().Int64(), i)
				}
			},
		}
		readProcess.AddInputChannel(readResult)
		ctx.AddChild(&readProcess)

		pmu.AddReader(readAddr, []*core.CommunicationChannel{readResult}, Scalar{})
	}

	// Vector read
	{
		readAddr := core.MakeCommunicationChannel[datatypes.FixedPoint](chanSize)
		readResult := core.MakeCommunicationChannel[datatypes.FixedPoint](chanSize)
		comms = append(comms, readAddr, readResult)
		readIssue2 := core.SimpleNode[any]{
			RunFunc: func(node *core.SimpleNode[any]) {
				_ = core.DequeueInputChannels(node, 0)
				t.Log("Current Time on ReadIssue2:", node.TickLowerBound())
				for i := 0; i < numIters; i++ {
					node.IncrCycles(core.OneTick)
					ind := datatypes.FixedPoint{Tp: idxType}
					ind.SetInt64(int64(i * vecWidth))
					node.OutputChannel(0).Enqueue(core.MakeChannelElement(node.TickLowerBound(), ind))
				}
				t.Log("ReadIssue 2 Finished")
			},
		}
		ctx.AddChild(&readIssue2)
		readIssue2.AddInputChannel(wAck2)
		readIssue2.AddOutputChannel(readAddr)
		readProcess2 := core.SimpleNode[any]{
			RunFunc: func(node *core.SimpleNode[any]) {
				for i := 0; i < numIters; i++ {
					read := core.DequeueInputChannels(node, 0)[0]
					t.Logf("Fetched Data: %v, %s", read, read.Status)
					node.AdvanceToTime(&read.Time)
					node.IncrCycles(core.OneTick)
					fetched := read.Data.(datatypes.Vector[datatypes.FixedPoint])
					for j := 0; j < vecWidth; j++ {
						t.Logf("Fetched %d at iteration %d [%d]", fetched.Get(j).ToInt().Int64(), i, j)
					}
				}
				t.Log("ReadProcess 2 Finished")
			},
		}
		readProcess2.AddInputChannel(readResult)
		ctx.AddChild(&readProcess2)
		pmu.AddReader(readAddr, []*core.CommunicationChannel{readResult}, Vector{Width: vecWidth})
	}

	ctx.Init()
	ctx.Run()
}
