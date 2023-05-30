package hogmild

import (
	"math/big"

	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/datatypes"
)

type config struct {
	sendingTime  uint
	networkDelay uint

	nSamples     uint
	nWorkers     uint
	nWeightBanks uint
}

type sample struct {
	sampleId      uint
	weightVersion uint
}

type paramsServerState struct {
	conf *config

	nextSample        uint
	currWeightVersion uint

	// When the bank will be ready to send another sample
	bankStates []*core.Time

	// This represent the sequence of updates to the weights.
	// Each sample represent a gradient that is folded into the weights,
	// the sampleId represent the data used to calculate the gradient,
	// and the weightVersion is the iteration of the weights used.
	// weightVersion starts at 0,
	// and each update increment the weightVersion by 1.
	updateLog []*sample
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

func clearFreeBanks(node *core.SimpleNode[paramsServerState]) {
	newBankStates := make([]*core.Time, 0)
	currTick := node.TickLowerBound()
	for _, bankState := range node.State.bankStates {
		if bankState.Cmp(currTick) > 0 {
			newBankStates = append(newBankStates, bankState)
		}
	}
	node.State.bankStates = newBankStates
}

func sendSamples(node *core.SimpleNode[paramsServerState]) {
	if node.State.nextSample == node.State.conf.nSamples ||
		len(node.State.bankStates) == int(node.State.conf.nWeightBanks) {
		return
	}

	for i := uint(0); i < node.State.conf.nWorkers; i++ {
		if node.OutputChannel(int(i)).IsFull() {
			continue
		}

		s := sample{
			sampleId:      node.State.nextSample,
			weightVersion: node.State.currWeightVersion,
		}
		arrivalTime := node.TickLowerBound()
		arrivalTime.Add(arrivalTime, core.NewTime(
			int64(node.State.conf.sendingTime+node.State.conf.networkDelay)))
		elem := core.MakeChannelElement(node.TickLowerBound(), s)
		node.OutputChannel(int(i)).Enqueue(elem)

		node.State.nextSample += 1
		nextReadyTick := node.TickLowerBound()
		nextReadyTick.Add(nextReadyTick,
			core.NewTime(int64(node.State.conf.sendingTime)))
		node.State.bankStates = append(node.State.bankStates, nextReadyTick)

		if len(node.State.bankStates) == int(node.State.conf.nWeightBanks) ||
			node.State.nextSample == node.State.conf.nSamples {
			break
		}
	}
}

func receiveSamples(node *core.SimpleNode[paramsServerState]) {
	for i := uint(0); i < node.State.conf.nWorkers; i++ {
		ce, status := node.InputChannel(int(i)).Peek()

		switch status {
		case core.Ok:
			s := ce.Data.Payload().(sample)
			node.State.updateLog = append(node.State.updateLog, &s)
            node.State.currWeightVersion += 1
		default:
			break
		}
	}
}

func paramsServer(node *core.SimpleNode[paramsServerState]) {
	for {
		clearFreeBanks(node)
		sendSamples(node)
		receiveSamples(node)

		if node.State.nextSample == node.State.conf.nSamples {
			break
		}
	}
}

func hogmild() {
}
