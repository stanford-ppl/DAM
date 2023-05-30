package main

import "github.com/stanford-ppl/DAM/core"

type workerState struct {
	conf *config
}

func runWorker(node *core.SimpleNode[workerState]) {
	for {
		ces_with_statuses := core.DequeueInputChansByID(node, 0)
		for _, ce_with_status := range ces_with_statuses {
			switch ce_with_status.Status {
			case core.Ok:
				// TODO: Add pipeling
				ce := ce_with_status.ChannelElement
				s := ce.Data.(sample)
				if s.done {
					return
				}

				ce.Time.Add(&ce.Time,
					core.NewTime(int64(node.State.conf.gradientLatency)))
				node.OutputChannel(0).Enqueue(ce)
			case core.Closed:
				return
			default:
				panic("Got nothing from sample input channel")
			}
		}
	}
}
