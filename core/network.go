package core

import (
	"fmt"
	"sync"

	"github.com/stanford-ppl/DAM/datatypes"
	"github.com/stanford-ppl/DAM/utils"
)

type ChannelElement struct {
	Time Time
	Data datatypes.DAMType
}

type Status uint8

const (
	Ok Status = iota
	Nothing
	Closed
)

func (stat Status) String() string {
	switch stat {
	case Ok:
		return "Ok"
	case Nothing:
		return "Nothing"
	case Closed:
		return "Closed"
	}
	return "X"
}

// Enqueue(v) -> time to stall (possibly 0)
// ReadVal:
//   Nothing (present)
//   Something (present)
//   Something (future)
//   Closed (present & future)
// Peek() -> ReadVal
// Dequeue() -> ReadVal

// Challenge: how do we work with time-travelling backpressure?
// I.E. Dst is running far in the future w.r.t. Src, so that
// the channel never fills?
// In order to handle this, the response channel writes back the time it received the value.
// This way, if we've ever sent more than we should have, then we can stall!

// This needs to be public because we need it for network sim.
type CommunicationChannel struct {
	underlying chan ChannelElement
	resp       chan *Time

	capacityMutex sync.RWMutex

	sendRecvDelta int
	capacity      int
	nextTime      *Time

	// To support peeking
	headStatus Status
	head       *ChannelElement

	srcCtx ContextView
	dstCtx ContextView
}

func (cchan *CommunicationChannel) String() string {
	return fmt.Sprintf("%+v --> %v --> %+v (capacity = %d)", cchan.srcCtx, cchan.underlying, cchan.dstCtx, cchan.capacity)
}

type OutputChannel interface {
	// Enqueue returns a (success, nextAvailable) pair
	// nextAvailable may be nil if it's not clear when it'll become available
	Enqueue(ChannelElement) (bool, *Time)

	// IsFull returns if the channel is full from the srcCtx perspective
	IsFull() bool

	// Returns the next time if it's available, otherwise nil
	//    This is purely for performance, in case we want to take multiple steps forward.
	NextTime() *Time
	CloseOutput()
}

func (cchan *CommunicationChannel) incrSRDelta(amt int) {
	cchan.capacityMutex.Lock()
	defer cchan.capacityMutex.Unlock()
	cchan.sendRecvDelta += amt
}

func (cchan *CommunicationChannel) CloseOutput() {
	close(cchan.underlying)
}

func (cchan *CommunicationChannel) updateLen() {
	cchan.capacityMutex.Lock()
	defer cchan.capacityMutex.Unlock()
	updateTime := cchan.srcCtx.TickLowerBound()
	if cchan.nextTime != nil {
		if cchan.nextTime.Cmp(updateTime) > 0 {
			// Next update is in the future
			return
		}
		cchan.sendRecvDelta--
		cchan.nextTime = nil
	}
	waitChan := cchan.dstCtx.BlockUntil(updateTime)
	for {
		select {
		case <-waitChan:
			// we're up to date, just need to flush the channel
			for {
				select {
				case time := <-cchan.resp:
					if updateTime.Cmp(time) < 0 {
						// we're done for now.
						cchan.nextTime = time
						return
					}
					cchan.sendRecvDelta--
				default:
					return
				}
			}
		case time := <-cchan.resp:
			// we've received a time on the response channel
			if updateTime.Cmp(time) < 0 {
				// we're done for now.
				cchan.nextTime = time
				return
			}
			cchan.sendRecvDelta--
		}
	}
}

// Note that IsFull also waits for the destination to catch up if it might be full!
// This means that if IsFull() is true, then the destination is also up to date with the source.
func (cchan *CommunicationChannel) IsFull() bool {
	cchan.capacityMutex.RLock()
	if cchan.sendRecvDelta < cchan.capacity {
		cchan.capacityMutex.RUnlock()
		return false
	}
	cchan.capacityMutex.RUnlock()
	cchan.updateLen()
	// Now that we've updated capacity, re-check
	cchan.capacityMutex.RLock()
	defer cchan.capacityMutex.RUnlock()
	return cchan.sendRecvDelta == cchan.capacity
}

func (cchan *CommunicationChannel) NextTime() (ret *Time) {
	cchan.capacityMutex.RLock()
	defer cchan.capacityMutex.RUnlock()
	if cchan.nextTime == nil {
		return nil
	} else {
		ret = new(Time)
		ret.Set(cchan.nextTime)
		return
	}
}

func (cchan *CommunicationChannel) Enqueue(ce ChannelElement) (bool, *Time) {
	if cchan.IsFull() {
		// currently full!
		// In this case, consider one of the possibilities:
		// Since IsFull brings the target up to date w.r.t. us,
		// the destination is either current or in the future.
		// Since it's still full, that means that we can't enqueue
		cchan.capacityMutex.RLock()
		if cchan.nextTime != nil {
			defer cchan.capacityMutex.RUnlock()
			// There's a time in the future when it becomes available, so provide that info
			nT := new(Time)
			nT.Set(cchan.nextTime)
			return false, nT
		}
		cchan.capacityMutex.RUnlock()
		// check back again next cycle
		return false, nil
	}
	cchan.incrSRDelta(1)
	cchan.underlying <- ce
	return true, nil
}

var _ OutputChannel = (*CommunicationChannel)(nil)

// The key functionality here is to be able to maybe get
// depending on the current time.
// For any access, we can have:
// Some/None x Present/Future

// NOT threadsafe -- we assume that peek/dequeue is only ever called from one thread.
type InputChannel interface {
	Peek() (ChannelElement, Status)

	// This is a nonblocking dequeue
	Dequeue() (ChannelElement, Status)
}

func (cchan *CommunicationChannel) Peek() (ChannelElement, Status) {
	if cchan.head != nil {
		if cchan.headStatus != Nothing {
			return *cchan.head, cchan.headStatus
		} else {
			// The headstatus is nothing
			curTime := cchan.dstCtx.TickLowerBound()
			if cchan.head.Time.Cmp(curTime) >= 0 {
				// If the Nothing time is in the future, keep the nothing
				return *cchan.head, Nothing
			}
			// Nothing was in the past, continue ahead
		}
	}
	// Otherwise, we need to pop a value off of the channel
	select {
	case v, ok := <-cchan.underlying:
		cchan.head = &v
		if !ok {
			// Channel was closed
			cchan.headStatus = Closed
		} else {
			cchan.headStatus = Ok
		}
		return *cchan.head, cchan.headStatus
	default:
	}
	// there wasn't anything in the channel!
	// Case 1: reader in future
	// Case 2: reader in past/present

	curTime := cchan.dstCtx.TickLowerBound()
	// Wait until the writer is in the past/present
	srcTime := <-cchan.srcCtx.BlockUntil(curTime)
	select {
	case v, ok := <-cchan.underlying:
		cchan.head = &v
		if !ok {
			// Channel was closed
			cchan.headStatus = Closed
		} else {
			cchan.headStatus = Ok
		}
		return *cchan.head, cchan.headStatus
	default:
		// There wasn't anything in here, even after waiting.
		cchan.head = &ChannelElement{}
		cchan.headStatus = Nothing
		cchan.head.Time.Set(srcTime)
		return *cchan.head, cchan.headStatus
	}
}

func (cchan *CommunicationChannel) Dequeue() (ce ChannelElement, status Status) {
	ce, status = cchan.Peek()
	if status != Nothing {
		cchan.head = nil
		// The earliest we could have dequeued the result is either when the packet arrived
		// or the dequeuer's current time.
		utils.Max[*Time](&ce.Time, cchan.dstCtx.TickLowerBound(), &ce.Time)
		cchan.resp <- &ce.Time
	}
	return
}

func MakeChannelElement(time *Time, payload datatypes.DAMType) (ce ChannelElement) {
	ce.Time.Set(time)
	ce.Data = payload
	return
}

var _ InputChannel = (*CommunicationChannel)(nil)

func MakeCommunicationChannel[T datatypes.DAMType](size int) *CommunicationChannel {
	cchan := CommunicationChannel{
		underlying: make(chan ChannelElement, size),
		resp:       make(chan *Time, size),
		capacity:   size,
	}
	return &cchan
}
