package basispursuit

import (
	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/libs/blas"
)

type XUpdater struct {
	core.SimpleNode[blas.Matrix]
}

func (xu *XUpdater) Init() {
	mat := blas.AllocMatrix(100, 100)
	xu.State = &mat
}

type ZUpdater struct {
	core.SimpleNode[blas.Matrix]
}

func (zu *ZUpdater) Init() {
	mat := blas.AllocMatrix(100, 100)
	zu.State = &mat
}

var (
	_ core.Context = (*XUpdater)(nil)
	_ core.Context = (*ZUpdater)(nil)
)
