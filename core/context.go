package core

import (
	"math/big"
	"sync"

	"github.com/stanford-ppl/DAM/utils"
)

// A Context is a "program" subcomponent
type Context interface {
	// Creates a new ID, only needs to be unique within this context
	GetNewNodeID() int

	// Lower bound on progress on this context
	GetTickLowerBound() *big.Int

	Init()
	Run()

	ParentContext() Context

	// SetParent isn't intended to be called by anything other than AddChild.
	SetParent(parent Context)
	GetID() int
	AddChild(child Context)
}

type primitiveContext struct {
	nextID       int
	localTimings sync.Map
	children     []Context
	parent       Context
	ID           int
}

func (prim *primitiveContext) GetNewNodeID() (result int) {
	result = prim.nextID
	prim.nextID++
	return
}

func (prim *primitiveContext) GetTickLowerBound() (result *big.Int) {
	isFirst := true
	prim.localTimings.Range(func(key, value any) bool {
		if isFirst {
			isFirst = false
			result.Set(value.(*big.Int))
		} else {
			utils.Min[*big.Int](result, value.(*big.Int), result)
		}
		return true
	})
	return
}

func (prim *primitiveContext) AddChild(child Context) {
	prim.children = append(prim.children, child)
	child.SetParent(prim)
}

func (prim *primitiveContext) Init() {
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

func (prim *primitiveContext) ParentContext() Context {
	return prim.parent
}

func (prim *primitiveContext) Run() {
	var wg sync.WaitGroup
	wg.Add(len(prim.children))
	for _, ctx := range prim.children {
		go (func(c Context) {
			c.Run()
			wg.Done()
		})(ctx)
	}
	wg.Wait()
}

func (prim *primitiveContext) GetID() int {
	return prim.ID
}

func (prim *primitiveContext) SetParent(parent Context) {
	if prim.parent != nil {
		panic("Already had a parent set! Can't set another parent.")
	}
	prim.parent = parent
	prim.ID = parent.GetNewNodeID()
}

func MakePrimitiveContext(parent Context) Context {
	ctx := new(primitiveContext)
	if parent != nil {
		ctx.parent = parent
	}
	return ctx
}
