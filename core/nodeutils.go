package core

import (
	"fmt"

	"github.com/stanford-ppl/DAM/utils"
)

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

func DequeueInputChansByID(node DeqInputChans, channelIndices ...int) (ret []CEWithStatus) {
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

func DequeueInputChannels(node DeqInputChans, chans ...InputChannel) (ret []CEWithStatus) {
	ret = make([]CEWithStatus, len(chans))
	for _, cc := range chans {
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
	for i, cc := range chans {
		cE, status := cc.Dequeue()
		if status == Nothing {
			panic("We peeked just to make sure that status wouldn't be Nothing!")
		}
		ret[i] = CEWithStatus{cE, status}
	}
	return
}

// Advances the node to time when at least one bundle is available, and dequeues from it.
// If all of the channels are closed, then we return (-1, nil)
func DequeueInputBundles(node DeqInputChans, channelBundles ...[]int) (int, []CEWithStatus) {
	for {
		curTime := node.TickLowerBound()
		nextTime := InfiniteTime()
		for i, bundle := range channelBundles {
			bundleNextTime := NewTime(0)
			var ready bool = true
		L:
			for _, chanInd := range bundle {
				cc := node.InputChannel(chanInd)
				cE, status := cc.Peek()
				switch status {
				case Nothing:
					tmp := Time{}
					tmp.Add(&cE.Time, OneTick)
					utils.Max[*Time](bundleNextTime, &tmp, bundleNextTime)
					ready = false
				case Closed:
					// If the channel is closed, then there's no more data coming through.
					ready = false
					bundleNextTime = InfiniteTime()
					break L
				default:
					if cE.Time.Cmp(curTime) > 0 {
						ready = false
					}
					utils.Max[*Time](bundleNextTime, &cE.Time, bundleNextTime)
				}
			}
			if ready {
				return i, DequeueInputChansByID(node, bundle...)
			} else {
				utils.Min[*Time](nextTime, bundleNextTime, nextTime)
			}
		}
		if nextTime.IsInf() {
			return -1, nil
		}
		// Otherwise, advance to nextTime, try again
		node.AdvanceToTime(nextTime)
	}
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

type HasID interface {
	GetID() int
}

func CtxToString(ctx Context) string {
	var parentString string
	if ctx.ParentContext() != nil {
		parentString = CtxToString(ctx.ParentContext()) + "."
	}
	switch context := ctx.(type) {
	case fmt.Stringer:
		return fmt.Sprintf("%s%s", parentString, context.String())
	case HasID:
		return fmt.Sprintf("%s%T(id=%d)", parentString, ctx, context.GetID())
	default:
		return fmt.Sprintf("%s%T(%p)", parentString, ctx, &ctx)
	}
}
