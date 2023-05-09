package core

import (
	"fmt"
	"strings"
	"sync"

	"github.com/stanford-ppl/DAM/utils"
)

// Provides basic input/output functionality for a node.
type LowLevelIO struct {
	inputChannels  []*CommunicationChannel
	outputChannels []*CommunicationChannel
}

func (llio *LowLevelIO) AddInputChannel(channel *CommunicationChannel) (result int) {
	result = len(llio.inputChannels)
	llio.inputChannels = append(llio.inputChannels, channel)
	return
}

func (llio *LowLevelIO) InputChannel(i int) InputChannel {
	return llio.inputChannels[i]
}

func (llio *LowLevelIO) OutputChannel(i int) OutputChannel {
	return llio.outputChannels[i]
}

func (llio *LowLevelIO) InitWithCtx(ctx ContextView) {
	// fmt.Printf("Initializing with context! %#v <- %#v\n", llio, ctx)
	for _, cc := range llio.inputChannels {
		cc.dstCtx = ctx
	}
	for _, cc := range llio.outputChannels {
		cc.srcCtx = ctx
	}
}

func (llio *LowLevelIO) String() string {
	inputStrings := utils.Map(llio.inputChannels, func(chn *CommunicationChannel) string {
		return fmt.Sprintf("%#v", chn.srcCtx)
	})
	outputStrings := utils.Map(llio.outputChannels, func(chn *CommunicationChannel) string {
		return fmt.Sprintf("%#v", chn.dstCtx)
	})
	return fmt.Sprintf("LLIO{Inputs: %v, Outputs: %v}", strings.Join(inputStrings, ", "), strings.Join(outputStrings, ", "))
}

func (llio *LowLevelIO) AddOutputChannel(channel *CommunicationChannel) (result int) {
	result = len(llio.outputChannels)
	llio.outputChannels = append(llio.outputChannels, channel)
	return
}

func (prim *LowLevelIO) Cleanup() {
	utils.Foreach(prim.outputChannels, func(c *CommunicationChannel) { c.CloseOutput() })
}

// SignalElement and TickTime provide basic time functionality for a node
// Additionally, this provides the ability for nodes to wait (stall) until a different node has reached a tick count.
type signalElement struct {
	when Time
	how  chan<- *Time
}

type TickTime struct {
	tickCount Time
	tickMutex sync.RWMutex

	signalBuffer []signalElement
}

func (prim *TickTime) scanAndWriteSignals() {
	newSignals := []signalElement{}
	for _, se := range prim.signalBuffer {
		if se.when.Cmp(&prim.tickCount) <= 0 {
			cpy := new(Time)
			cpy.Set(&prim.tickCount)
			se.how <- cpy
			close(se.how)
		} else {
			newSignals = append(newSignals, se)
		}
	}
	prim.signalBuffer = newSignals
}

func (prim *TickTime) IncrCycles(step *Time) {
	prim.tickMutex.Lock()
	defer prim.tickMutex.Unlock()
	prim.tickCount.Add(&prim.tickCount, step)
	prim.scanAndWriteSignals()
}

func (prim *TickTime) AdvanceToTime(newTime *Time) {
	prim.tickMutex.Lock()
	defer prim.tickMutex.Unlock()
	if newTime.Cmp(&prim.tickCount) < 0 {
		return
	}
	prim.tickCount.Set(newTime)
	prim.scanAndWriteSignals()
}

func (prim *TickTime) TickLowerBound() (result *Time) {
	prim.tickMutex.RLock()
	result = new(Time)
	result.Set(&prim.tickCount)
	prim.tickMutex.RUnlock()
	return
}

func (prim *TickTime) BlockUntil(time *Time) <-chan *Time {
	ch := make(chan *Time, 1)
	prim.tickMutex.RLock()
	if prim.tickCount.Cmp(time) >= 0 {
		// We've already reached time, so we just return the current time.
		resp := new(Time)
		resp.Set(&prim.tickCount)
		ch <- resp
		close(ch)
		prim.tickMutex.RUnlock()
		return ch
	}
	prim.tickMutex.RUnlock()
	// In this case, we need to register a new callback
	prim.tickMutex.Lock()
	defer prim.tickMutex.Unlock()
	if prim.tickCount.Cmp(time) >= 0 {
		// After acquiring the lock, double-check that we haven't reached the time already.
		resp := new(Time)
		resp.Set(&prim.tickCount)
		ch <- resp
		close(ch)
		return ch
	}
	// We've confirmed that we haven't reached the new time.
	newElement := signalElement{how: ch}
	newElement.when.Set(time)
	prim.signalBuffer = append(prim.signalBuffer, newElement)
	return ch
}

func (prim *TickTime) Cleanup() {
	// This increments time to "done", and also notifies all listeners that the task is done.
	prim.IncrCycles(InfiniteTime())
}

type PrimitiveNode[T any] struct {
	id int

	State *T
	// TODO: Tags

	parent ParentContext
}

func (prim *PrimitiveNode[T]) SetParent(parent ParentContext) {
	prim.parent = parent
	prim.id = parent.GetNewChildID()
}

func (prim *PrimitiveNode[T]) GetID() int {
	return prim.id
}

type LLIOWithTime struct {
	LowLevelIO
	TickTime
}

type PrimitiveNodeWithIO[T any] struct {
	PrimitiveNode[T]
	LLIOWithTime
}

func (lliowt *LLIOWithTime) Cleanup() {
	lliowt.TickTime.Cleanup()
	lliowt.LowLevelIO.Cleanup()
}

func (prim *PrimitiveNode[T]) ParentContext() ParentContext { return prim.parent }

type SimpleNode[T any] struct {
	PrimitiveNodeWithIO[T]
	RunFunc func(node *SimpleNode[T])
}

// Compile-time assertion that SimpleNode[any] is a Context
var _ Context = (*SimpleNode[any])(nil)

func (sn *SimpleNode[T]) Run() {
	sn.RunFunc(sn)
}

func (sn *SimpleNode[T]) Init() {
	sn.State = new(T)
	sn.LowLevelIO.InitWithCtx(sn)
}
