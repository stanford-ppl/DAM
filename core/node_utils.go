package core

type CEWithStatus struct {
	ChannelElement
	Status
}

type DeqInputChans interface {
	InputChannel(int) InputChannel
	AdvanceToTime(*Time)
	IncrCycles(*Time)
	TickLowerBound() *Time
}

func DequeueInputChannels(node DeqInputChans, channelIndices ...int) (ret []CEWithStatus) {
	ret = make([]CEWithStatus, len(channelIndices))
	for _, i := range channelIndices {
		cc := node.InputChannel(i)
	L:
		for {
			cE, status := cc.Peek()
			node.AdvanceToTime(&cE.Time)
			switch status {
			case Nothing:
				node.IncrCycles(OneTick)
			default:
				break L
			}
		}
	}

	// At this point, we haven't actually done any work, but we've advanced to the point where all elements are visible.
	for sub, i := range channelIndices {
		cc := node.InputChannel(i)
		cE, status := cc.Dequeue()
		if status == Nothing {
			panic("We peeked just to make sure that status wouldn't be Nothing!")
		}
		ret[sub] = CEWithStatus{cE, status}
	}
	return
}

type EnqOutputChans interface {
	OutputChannel(int) OutputChannel
	AdvanceToTime(*Time)
	IncrCycles(*Time)
}

func AdvanceUntilCanEnqueue(node EnqOutputChans, chanIndices ...int) {
	for _, i := range chanIndices {
		cc := node.OutputChannel(i)
		for {
			if cc.IsFull() {
				nextTime := cc.NextTime()
				if nextTime != nil {
					node.AdvanceToTime(nextTime)
					break
				} else {
					node.IncrCycles(OneTick)
				}
			} else {
				break
			}
		}
	}
}
