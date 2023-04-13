package core

import (
	"math/big"
)

type PrimitiveNode[T any] struct {
	id             int
	TickCount      big.Int
	State          *T
	InputChannels  []*DAMChannel
	OutputChannels []*DAMChannel
	// TODO: Tags

	parent Context
}

type EmptyInit struct{}

func (node *EmptyInit) Init() {}

func (prim *PrimitiveNode[T]) IncrCycles(step int64) {
	prim.IncrCyclesBigInt(big.NewInt(step))
}

func (prim *PrimitiveNode[T]) IncrCyclesBigInt(step *big.Int) {
	prim.TickCount.Add(&prim.TickCount, step)
}

func (prim *PrimitiveNode[T]) AddChild(child Context) {
	panic("AddChild Not Implemented on Primitive Nodes")
}

func (prim *PrimitiveNode[T]) GetNewNodeID() int {
	panic("GetNewID Not Implemented on Primitive Nodes")
}

func (prim *PrimitiveNode[T]) SetParent(parent Context) {
	prim.parent = parent
	prim.id = parent.GetNewNodeID()
}

func (prim *PrimitiveNode[T]) GetID() int {
	return prim.id
}

func (prim *PrimitiveNode[T]) GetTickLowerBound() (result *big.Int) {
	result.Set(&prim.TickCount)
	return
}

func (prim *PrimitiveNode[T]) ParentContext() Context { return prim.parent }

func (prim *PrimitiveNode[T]) AddInputChannel(channel CommunicationChannel) (result int) {
	result = len(prim.InputChannels)
	prim.InputChannels = append(prim.InputChannels, channel.InputChannel)
	return
}

func (prim *PrimitiveNode[T]) AddOutputChannel(channel CommunicationChannel) (result int) {
	result = len(prim.OutputChannels)
	prim.OutputChannels = append(prim.OutputChannels, channel.OutputChannel)
	return
}

type SimpleNode[T any] struct {
	PrimitiveNode[T]
	RunFunc func(node *SimpleNode[T])
}

func (sn *SimpleNode[T]) Run() {
	sn.RunFunc(sn)
}

func (sn *SimpleNode[T]) AddChild(child Context) {
	panic("Don't know how to add a child to a SimpleNode")
}

func (sn *SimpleNode[T]) Init() {
	sn.State = new(T)
}

// Compile-time assertion that SimpleNode[any] is a Context
var _ Context = (*SimpleNode[any])(nil)
