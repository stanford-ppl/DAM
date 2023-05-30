package core

import "math/big"

type Time struct {
	time big.Int
	done bool // done means infinite time
}

func (t *Time) Cmp(ot *Time) int {
	if t.done && ot.done {
		return 0
	}
	if t.done && !ot.done {
		return 1
	}
	if !t.done && ot.done {
		return -1
	}
	return t.time.Cmp(&ot.time)
}

func (t *Time) Set(ot *Time) *Time {
	t.time.Set(&ot.time)
	t.done = ot.done
	return t
}

func (t *Time) Add(a, b *Time) *Time {
	t.time.Add(&a.time, &b.time)
	t.done = a.done || b.done
	return t
}

func (t *Time) IsInf() bool {
	return t.done
}

func (t *Time) String() string {
	if t.done {
		return "Inf"
	} else {
		return t.time.String()
	}
}

func InfiniteTime() *Time {
	t := new(Time)
	t.done = true
	return t
}

func NewTime(t int64) *Time {
	time := Time{}
	time.time.SetInt64(t)
	return &time
}

func (t *Time) GetUnderlying() big.Int {
	return t.time
}

var OneTick = NewTime(1)
