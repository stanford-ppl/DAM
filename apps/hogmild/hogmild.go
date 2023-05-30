package main

import (
	"flag"
	"fmt"
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
	done          bool
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

func hogmild(conf *config) ([]*sample, core.Time) {
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

	return paramsState.updateLog, *paramsServerNode.TickLowerBound()
}

func main() {
	conf := new(config)
	flag.UintVar(&conf.sendingTime, "sendingTime", 8,
		"How long it takes to serialize a packet onto the network")
	flag.UintVar(&conf.networkDelay, "networkDelay", 16,
		"How long does it take a network to traverse the network")
	flag.UintVar(&conf.foldLatency, "foldLatency", 32,
		"Latency of folding in one gradient")
	flag.UintVar(&conf.gradientLatency, "gradientLatency", 64,
		"Latency of computing one gradient")
	flag.UintVar(&conf.fifo_depth, "fifo_depth", 8,
		"Depth of channel buffers")
	flag.UintVar(&conf.nSamples, "nSamples", 512,
		"Number of data samples there is")
	flag.UintVar(&conf.nWorkers, "nWorkers", 16,
		"Number of workers available to compute the gradient")
	flag.UintVar(&conf.nWeightBanks, "nWeightBanks", 8,
		"Number of banked storage units for weights")
	flag.Parse()

	updateLogs, finalTick := hogmild(conf)
	finalTime := finalTick.GetTime()
	fmt.Println(finalTime.String())
	for _, s := range updateLogs {
		fmt.Printf("%d, %d\n", s.sampleId, s.weightVersion)
	}
}
