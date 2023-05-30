package main

import (
	"fmt"

	"github.com/stanford-ppl/DAM/core"
)

type workerState struct {
	conf     *config
	sent     uint
	received uint
}

func newWorkerState(conf *config) *workerState {
	return &workerState{
		conf: conf,
	}
}

func computeGradient(
	node *core.SimpleNode[workerState],
	ce core.ChannelElement,
) bool {
	s := ce.Data.(sample)
	if s.done {
		return true
	}

	// TODO: if sendingTime is grater than gradientII, this can back-pressure
	totalLatency := node.State.conf.gradientLatency +
		node.State.conf.sendingTime + node.State.conf.networkDelay
	ce.Time.Add(&ce.Time,
		core.NewTime(int64(totalLatency)))
	node.OutputChannel(0).Enqueue(ce)
	node.State.sent += 1
	fmt.Printf("Worker_%d sent %d samples\n", node.ID(), node.State.sent)

	node.IncrCycles(core.NewTime(int64(node.State.conf.gradientII)))
	return false
}

func runWorker(node *core.SimpleNode[workerState]) {
	for {
		ces_with_statuses := core.DequeueInputChansByID(node, 0)
		node.State.received += uint(len(ces_with_statuses))
		fmt.Printf("Worker_%d got %d samples\n", node.ID(), node.State.received)
		for _, ce_with_status := range ces_with_statuses {
			switch ce_with_status.Status {
			case core.Ok:
				if computeGradient(node, ce_with_status.ChannelElement) {
					return
				}
			case core.Closed:
				return
			default:
				panic("Got nothing from sample input channel")
			}
		}
	}
}
