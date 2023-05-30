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
	foldII          uint
	gradientLatency uint
	gradientII      uint

	fifoDepth uint

	nSamples     uint
	nWorkers     uint
	nWeightBanks uint
	nFolders     uint
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

func makeParamsServer(
	ctx core.ParentContext,
	conf *config,
) *core.SimpleNode[paramsServerState] {
	paramsState := newParamsState(conf)
	paramsServerNode := core.MakeSimpleNode(runParamsServer, paramsState)
	ctx.AddChild(paramsServerNode)
	return paramsServerNode
}

func addWorker(
	ctx core.ParentContext,
	conf *config,
	paramsServerNode *core.SimpleNode[paramsServerState],
) {
	workerState := newWorkerState(conf)
	worker := core.MakeSimpleNode(runWorker, workerState)
	ctx.AddChild(worker)

	sampleChan := core.MakeCommunicationChannel[sample](
		int(conf.fifoDepth))
	paramsServerNode.AddOutputChannel(sampleChan)
	worker.AddInputChannel(sampleChan)

	updateChan := core.MakeCommunicationChannel[sample](
		int(conf.fifoDepth))
	worker.AddOutputChannel(updateChan)
	paramsServerNode.AddInputChannel(updateChan)
}

func hogmild(conf *config) ([]*sample, core.Time) {
	ctx := core.MakePrimitiveContext(nil)
	paramsServerNode := makeParamsServer(ctx, conf)
	for i := 0; i < int(conf.nWorkers); i++ {
		addWorker(ctx, conf, paramsServerNode)
	}

	ctx.Init()
	ctx.Run()

	return paramsServerNode.State.updateLog, *paramsServerNode.TickLowerBound()
}

func parseConfig() *config {
	conf := new(config)

	flag.UintVar(&conf.sendingTime, "sendingTime", 8,
		"How long it takes to serialize a packet onto the network")
	flag.UintVar(&conf.networkDelay, "networkDelay", 16,
		"How long does it take a network to traverse the network")
	flag.UintVar(&conf.foldLatency, "foldLatency", 32,
		"Latency of folding in one gradient")
	flag.UintVar(&conf.foldII, "foldII", 4,
		"Initiation interval of folding in one batch of  gradients")
	flag.UintVar(&conf.gradientII, "gradientII", 4,
		"Initiation interval of computing in one gradient")
	flag.UintVar(&conf.gradientLatency, "gradientLatency", 64,
		"Latency of computing one gradient")
	flag.UintVar(&conf.fifoDepth, "fifoDepth", 8,
		"Depth of channel buffers")
	flag.UintVar(&conf.nSamples, "nSamples", 128,
		"Number of data samples there is")
	flag.UintVar(&conf.nWorkers, "nWorkers", 1,
		"Number of workers available to compute the gradient")
	flag.UintVar(&conf.nWeightBanks, "nWeightBanks", 8,
		"Number of banked storage units for weights")
	flag.UintVar(&conf.nFolders, "nFolders", 8,
		"How many gradients can be folded in parallel")
	flag.Parse()

	return conf
}

func logResult(updateLogs []*sample, finalTick core.Time) {
	finalTime := finalTick.GetTime()
	fmt.Println(finalTime.String())
	for _, s := range updateLogs {
		fmt.Printf("%d, %d\n", s.sampleId, s.weightVersion)
	}
}

func main() {
	conf := parseConfig()
	updateLogs, finalTick := hogmild(conf)
	logResult(updateLogs, finalTick)
}
