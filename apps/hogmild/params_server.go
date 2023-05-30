package main

import (
	"github.com/stanford-ppl/DAM/core"
)

type paramsServerState struct {
	conf *config

	nextSample        uint
	currWeightVersion uint

	// When the bank will be ready to send another sample
	bankStates []*core.Time

	// This represent the sequence of updates to the weights. Each sample represent a gradient that is folded into the weights,
	// the sampleId represent the data used to calculate the gradient,
	// and the weightVersion is the iteration of the weights used.
	// weightVersion starts at 0,
	// and each update increment the weightVersion by 1.
	updateLog []*sample
}

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

func foldGradient(node *core.SimpleNode[paramsServerState], ce *core.ChannelElement) {
	s := ce.Data.(sample)
	node.State.updateLog = append(node.State.updateLog, &s)
	node.State.currWeightVersion += 1

	// TODO: Pipeline this,
	// and think about whether to block sending or not
	node.IncrCycles(core.NewTime(int64(node.State.conf.foldLatency)))
}

func tryReceiveSamples(node *core.SimpleNode[paramsServerState]) {
	for i := uint(0); i < node.State.conf.nWorkers; i++ {
		ce, status := node.InputChannel(int(i)).Dequeue()

		switch status {
		case core.Ok:
			foldGradient(node, &ce)
		default:
			break
		}
	}
}

func makeBundles(max int) [][]int {
	a := make([][]int, max)
	for i := 0; i < max; i++ {
		bundle := make([]int, 1)
		bundle[0] = i
		a[i] = bundle
	}
	return a
}

func runParamsServer(node *core.SimpleNode[paramsServerState]) {
	for node.State.nextSample < node.State.conf.nSamples {
		clearFreeBanks(node)
		sendSamples(node)
		tryReceiveSamples(node)
		node.IncrCycles(core.NewTime(1))
	}

	channelBundles := makeBundles(int(node.State.conf.nWorkers))
	for len(node.State.updateLog) < int(node.State.conf.nSamples) {
		err, ces := core.DequeueInputBundles(node, channelBundles...)
		if err < 0 {
			panic("Did not receive all of the gradients")
		}
		for _, ce_with_status := range ces {
			switch ce_with_status.Status {
			case core.Ok:
				foldGradient(node, &ce_with_status.ChannelElement)
			default:
				panic("Received unexpected gradient status")
			}
		}
	}
}
