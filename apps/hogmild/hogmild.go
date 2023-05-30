package hogmild

import (
	"math/big"

	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/datatypes"
)

type config struct {
	sendingTime  uint
	networkDelay uint

	foldLatency     uint
	gradientLatency uint

	fifo_depth uint

	nSamples     uint
	nWorkers     uint
	nWeightBanks uint
}

type sample struct {
	sampleId      uint
	weightVersion uint
}

func (s sample) Validate() bool {
	return true
}

func (s sample) Size() *big.Int {
	return big.NewInt(42)
}

func (s sample) Payload() any {
	return s
}

var _ datatypes.DAMType = (*sample)(nil)

func hogmild(conf *config) []*sample {
	ctx := core.MakePrimitiveContext(nil)

	paramsState := paramsServerState{
		conf:              conf,
		nextSample:        0,
		currWeightVersion: 0,
		bankStates:        make([]*core.Time, 0),
		updateLog:         make([]*sample, 0),
	}
	paramsServerNode := core.MakeSimpleNode(runParamsServer, &paramsState)
	ctx.AddChild(paramsServerNode)

	for i := 0; i < int(conf.nWorkers); i++ {
		workerState := new(workerState)
		workerState.conf = conf
		worker := core.MakeSimpleNode(runWorker, workerState)
		ctx.AddChild(worker)

		sampleChan := core.MakeCommunicationChannel[sample](
			int(conf.fifo_depth))
		paramsServerNode.AddOutputChannel(sampleChan)
		worker.AddInputChannel(sampleChan)

		updateChan := core.MakeCommunicationChannel[sample](
			int(conf.fifo_depth))
		worker.AddOutputChannel(updateChan)
		paramsServerNode.AddInputChannel(updateChan)
	}

	ctx.Init()
	ctx.Run()

	return paramsState.updateLog
}
