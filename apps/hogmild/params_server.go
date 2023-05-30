package main

import (
	"fmt"

	"github.com/stanford-ppl/DAM/core"
)

type weightVersionUpdate struct {
	time    core.Time
	version uint
}

type paramsServerState struct {
	conf *config

	nextSample        uint
	currWeightVersion uint

	// When the bank will be ready to send another sample
	bankStates []*core.Time

	// At new weight version to be used at each time
	weightVersionQueue []*weightVersionUpdate
	// When the folding unit will be ready again
	foldReadyAt core.Time

	// This represent the sequence of updates to the weights.
	// Each sample represent a gradient that is folded into the weights,
	// the sampleId represent the data used to calculate the gradient,
	// and the weightVersion is the iteration of the weights used.
	// weightVersion starts at 0,
	// and each update increment the weightVersion by 1.
	updateLog []*sample
}

func newParamsState(conf *config) *paramsServerState {
	return &paramsServerState{
		conf:               conf,
		nextSample:         0,
		currWeightVersion:  0,
		foldReadyAt:        *core.NewTime(0),
		weightVersionQueue: make([]*weightVersionUpdate, 0),
		bankStates:         make([]*core.Time, 0),
		updateLog:          make([]*sample, 0),
	}
}

func clearFreeBanks(node *core.SimpleNode[paramsServerState]) {
	newBankStates := make([]*core.Time, 0)
	currTick := node.TickLowerBound()
	for _, bankState := range node.State.bankStates {
		if bankState.Cmp(currTick) >= 0 {
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
	elem := core.MakeChannelElement(arrivalTime, s)
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

	for i := 0; i < int(node.State.conf.nWorkers); i++ {
		if node.OutputChannel(i).IsFull() {
			continue
		}

		sendSample(node, i, false)

		if doneSending(node) {
			return
		}
	}
}

func spearHeadWeightVersion(node *core.SimpleNode[paramsServerState]) uint {
	queueSize := len(node.State.weightVersionQueue)
	if queueSize == 0 {
		return node.State.currWeightVersion
	} else {
		return node.State.weightVersionQueue[queueSize-1].version
	}
}

func updateWeightVersion(node *core.SimpleNode[paramsServerState]) {
	currTick := node.TickLowerBound()
	for len(node.State.weightVersionQueue) > 0 {
		head := node.State.weightVersionQueue[0]
		if currTick.Cmp(&head.time) >= 0 {
			node.State.currWeightVersion = head.version
			node.State.weightVersionQueue = node.State.weightVersionQueue[1:]
		} else {
			return
		}
	}
}

func newWeightVersion(node *core.SimpleNode[paramsServerState],
	nUpdates uint,
) {
	update := new(weightVersionUpdate)

	foldII := node.TickLowerBound()
	foldII.Add(foldII, core.NewTime(int64(node.State.conf.foldLatency)))
	update.time = *foldII

	update.version = node.State.currWeightVersion + spearHeadWeightVersion(node)

	node.State.weightVersionQueue = append(
		node.State.weightVersionQueue, update)
}

func foldReady(node *core.SimpleNode[paramsServerState]) bool {
	currTick := node.TickLowerBound()
	return currTick.Cmp(&node.State.foldReadyAt) >= 0
}

func foldGradient(
	node *core.SimpleNode[paramsServerState],
	updates []*core.ChannelElement,
) {
	for _, update := range updates {
		s := update.Data.(sample)
		node.State.updateLog = append(node.State.updateLog, &s)
	}

	newWeightVersion(node, uint(len(updates)))

	foldReadyAt := node.TickLowerBound()
	foldReadyAt.Add(foldReadyAt, core.NewTime(int64(node.State.conf.foldII)))
	node.State.foldReadyAt = *foldReadyAt
}

func tryReceiveSamples(node *core.SimpleNode[paramsServerState]) {
	if !foldReady(node) {
		return
	}

	updates := make([]*core.ChannelElement, 0)

	for i := uint(0); i < node.State.conf.nWorkers; i++ {
		ce, status := node.InputChannel(int(i)).Dequeue()
		if status == core.Ok {
			updates = append(updates, &ce)
			if len(updates) == int(node.State.conf.nFolders) {
				break
			}
		}
	}

	foldGradient(node, updates)
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

func receiveAllSamples(node *core.SimpleNode[paramsServerState]) {
	channelBundles := makeBundles(int(node.State.conf.nWorkers))
	for len(node.State.updateLog) < int(node.State.conf.nSamples) {
		fmt.Printf("Param servers got %d updates\n", len(node.State.updateLog))

		if !foldReady(node) {
			node.AdvanceToTime(&node.State.foldReadyAt)
		}

		fmt.Println("Param server waiting for updates")
		err, ces := core.DequeueInputBundles(node, channelBundles...)
		fmt.Printf("Param server got %d updates\n", len(ces))
		if err < 0 {
			panic("Did not receive all of the gradients")
		}

		updates := make([]*core.ChannelElement, 0)
		for _, ce_with_status := range ces {
			if ce_with_status.Status == core.Ok {
				updates = append(updates, &ce_with_status.ChannelElement)
			} else {
				panic("Received unexpected gradient status")
			}
		}
		foldGradient(node, updates)

	}
}

func shutdown(node *core.SimpleNode[paramsServerState]) {
	for i := 0; i < int(node.State.conf.nWorkers); i++ {
		sendSample(node, i, true)
	}
}

func runParamsServer(node *core.SimpleNode[paramsServerState]) {
	for hasMoreSamples(node) {
		updateWeightVersion(node)
		clearFreeBanks(node)
		sendSamples(node)
		tryReceiveSamples(node)
		node.IncrCycles(core.NewTime(1))
	}

	fmt.Printf("params server sent %d samples\n", node.State.nextSample)
	receiveAllSamples(node)

	print("params server shutting down\n")
	shutdown(node)
}
