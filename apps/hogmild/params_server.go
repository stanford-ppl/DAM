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

func sendSample(node *core.SimpleNode[paramsServerState], idx int, done bool) {
	s := sample{
		done:          done,
		sampleId:      node.State.nextSample,
		weightVersion: node.State.currWeightVersion,
	}

	arrivalTime := node.TickLowerBound()
	arrivalTime.Add(arrivalTime, core.NewTime(
		int64(node.State.conf.sendingTime+node.State.conf.networkDelay)))
	elem := core.MakeChannelElement(node.TickLowerBound(), s)
	node.OutputChannel(idx).Enqueue(elem)

	if !done {
		node.State.nextSample += 1
		nextReadyTick := node.TickLowerBound()
		nextReadyTick.Add(nextReadyTick,
			core.NewTime(int64(node.State.conf.sendingTime)))
		node.State.bankStates = append(node.State.bankStates, nextReadyTick)
	}
}

func hasMoreSamples(node *core.SimpleNode[paramsServerState]) bool {
	return node.State.nextSample < node.State.conf.nSamples
}

func hasFreeWeightBanks(node *core.SimpleNode[paramsServerState]) bool {
	return len(node.State.bankStates) < int(node.State.conf.nWeightBanks)
}

func doneSending(node *core.SimpleNode[paramsServerState]) bool {
	return !hasMoreSamples(node) || !hasFreeWeightBanks(node)
}

func sendSamples(node *core.SimpleNode[paramsServerState]) {
	if !hasFreeWeightBanks(node) {
		return
	}

	for i := uint(0); i < node.State.conf.nWorkers; i++ {
		if node.OutputChannel(int(i)).IsFull() {
			continue
		}

		sendSample(node, int(i), false)

		if doneSending(node) {
			return
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

func shutdown(node *core.SimpleNode[paramsServerState]) {
	for i := 0; i < int(node.State.conf.nWorkers); i++ {
		sendSample(node, i, true)
	}
}

func runParamsServer(node *core.SimpleNode[paramsServerState]) {
	for hasMoreSamples(node) {
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

	shutdown(node)
}
