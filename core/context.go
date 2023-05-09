package core

import (
	"sync"

	"github.com/stanford-ppl/DAM/utils"
)

// A ContextView is the "external" view of another context
type ContextView interface {
	TickLowerBound() *Time

	// Blocks until at least the input time, returns the new time.
	BlockUntil(*Time) <-chan *Time
}

// A Context is a "program" subcomponent
type Context interface {
	ContextView

	Init()
	Run()
	Cleanup()

	ParentContext() ParentContext

	// SetParent isn't intended to be called by anything other than AddChild.
	SetParent(parent ParentContext)
}

// A ParentContext is a context that can have children
type ParentContext interface {
	Context
	GetNewChildID() int
	AddChild(Context)
}

// This is intended to be a basic utilities mixin to parent contexts
type ChildIDManager struct {
	nextID int
}

func (hc *ChildIDManager) GetNewChildID() (result int) {
	result = hc.nextID
	hc.nextID++
	return
}

type HasParent struct {
	parentCtx ParentContext
	id        int
}

func (hp *HasParent) SetParent(parent ParentContext) {
	hp.parentCtx = parent
	hp.id = parent.GetNewChildID()
}

func (hp *HasParent) ParentContext() ParentContext {
	return hp.parentCtx
}

func (hp *HasParent) ID() int {
	return hp.id
}

type EmptyInit struct{}

func (node *EmptyInit) Init() {}

type NoCleanup struct{}

func (node *NoCleanup) Cleanup() {}

type basicContext struct {
	NoCleanup // We clean up our children after they finish running
	ChildIDManager
	localTimings sync.Map
	children     []Context
	parent       ParentContext
	ID           int
}

var (
	_ ParentContext = (*basicContext)(nil)
	_ Context       = (*basicContext)(nil)
)

func (prim *basicContext) TickLowerBound() (result *Time) {
	result = InfiniteTime()
	for i, ctx := range prim.children {
		if i == 0 {
			result.Set(ctx.TickLowerBound())
		} else {
			utils.Min[*Time](result, ctx.TickLowerBound(), result)
		}
	}
	return
}

func (prim *basicContext) AddChild(child Context) {
	prim.children = append(prim.children, child)
	child.SetParent(prim)
}

func (prim *basicContext) BlockUntil(time *Time) <-chan *Time {
	signalChan := make(chan *Time, 1)
	go (func() {
		res := InfiniteTime()
		for _, ctx := range prim.children {
			v := <-ctx.BlockUntil(time)
			utils.Min[*Time](res, v, res)
		}
		signalChan <- res
	})()
	return signalChan
}

func (prim *basicContext) Init() {
	var wg sync.WaitGroup
	wg.Add(len(prim.children))
	for _, ctx := range prim.children {
		go (func(c Context) {
			c.Init()
			wg.Done()
		})(ctx)
	}
	wg.Wait()
}

func (prim *basicContext) ParentContext() ParentContext {
	return prim.parent
}

func (prim *basicContext) Run() {
	var wg sync.WaitGroup
	wg.Add(len(prim.children))
	for _, ctx := range prim.children {
		go (func(c Context) {
			c.Run()
			c.Cleanup()
			wg.Done()
		})(ctx)
	}
	wg.Wait()
}

func (prim *basicContext) GetID() int {
	return prim.ID
}

func (prim *basicContext) SetParent(parent ParentContext) {
	if prim.parent != nil {
		panic("Already had a parent set! Can't set another parent.")
	}
	prim.parent = parent
	prim.ID = parent.GetNewChildID()
}

func MakePrimitiveContext(parent ParentContext) ParentContext {
	ctx := new(basicContext)
	if parent != nil {
		parent.AddChild(ctx)
	}
	return ctx
}
