package core

import (
	"sync"

	"go.uber.org/zap"
)

var (
	smap       map[Context]*zap.Logger
	smapMutex  sync.RWMutex
	smapInit   sync.Once
	logOptions []zap.Option
)

func initSmap() {
	smapInit.Do(func() {
		smap = map[Context]*zap.Logger{}
		logOptions = append(logOptions, zap.AddCaller())
	})
}

func GetLogger(ctx Context) *zap.Logger {
	initSmap()
	smapMutex.RLock()
	{
		// Check if there's a logger already. If so, just return it.
		v, ok := smap[ctx]
		if ok {
			smapMutex.RUnlock()
			return v
		}
		smapMutex.RUnlock()
	}
	// Lock and check again, since we don't have mutex promotion
	smapMutex.Lock()
	defer smapMutex.Unlock()
	{
		v, ok := smap[ctx]
		if ok {
			return v
		}
	}
	l, _ := zap.NewProduction(logOptions...)
	smap[ctx] = l.Named(CtxToString(ctx))
	return l
}
