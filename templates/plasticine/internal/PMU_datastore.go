package plasticine

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/adam-lavrik/go-imath/ix"

	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/datatypes"
	"github.com/stanford-ppl/DAM/templates/shared/accesstypes"
	"github.com/stanford-ppl/DAM/utils"
)

func (pmu *PMUDataStore[T]) HandleRead(addr datatypes.DAMType, readInfo PMURead, time *core.Time) (result datatypes.DAMType) {
	switch accessType := readInfo.Type.(type) {
	case accesstypes.Gather:
		// we're in gather read mode
		addrVec := addr.(datatypes.Vector[datatypes.FixedPoint])
		tmp := datatypes.NewVector[T](addrVec.Width())
		for i := 0; i < addrVec.Width(); i++ {
			addr := addrVec.Get(i).ToInt().Int64()
			tmp.Set(i, pmu.Read(addr, time))
		}
		result = tmp

	case accesstypes.Scalar:
		// scalar read mode
		intAddr := addr.(datatypes.FixedPoint).ToInt().Int64()
		result = pmu.Read(intAddr, time)
	case accesstypes.Vector:
		//  addr, vector result
		tmp := datatypes.NewVector[T](accessType.Width)
		addr := addr.(datatypes.FixedPoint).ToInt().Int64()
		for i := 0; i < accessType.Width; i++ {
			tmp.Set(i, pmu.Read(addr+int64(i), time))
		}
		result = tmp
	}
	return
}

func (pmu *PMUDataStore[T]) HandleWrite(addr datatypes.DAMType, enable utils.Option[datatypes.DAMType], data datatypes.DAMType, writeInfo PMUWrite, time *core.Time) {
	addrScalar, _ := addr.(datatypes.FixedPoint)
	addrVec, _ := addr.(datatypes.Vector[datatypes.FixedPoint])
	dataVec, isVec := data.(datatypes.Vector[T])
	var width int = 1
	if isVec {
		width = dataVec.Width()
	}
	enables := broadcastEnable(enable, width)
	switch writeInfo.Type.(type) {
	case accesstypes.Scalar:
		if !enables[0] {
			break
		}
		pmu.Write(addrScalar.ToInt().Int64(), data.(T), time)
	case accesstypes.Vector:
		addrOffset := addrScalar.ToInt().Int64()
		for i := 0; i < dataVec.Width(); i++ {
			if !enables[i] {
				continue
			}
			pmu.Write(addrOffset+int64(i), dataVec.Get(i), time)
		}
	case accesstypes.Scatter:
		if dataVec.Width() != addrVec.Width() {
			panic(fmt.Sprintf("Mismatch between data and addr widths in scatter: %d vs %d", dataVec.Width(), dataVec.Width()))
		}
		for i := 0; i < dataVec.Width(); i++ {
			if !enables[i] {
				continue
			}
			pmu.Write(addrVec.Get(i).ToInt().Int64(), dataVec.Get(i), time)
		}
	}
}

// An EntryHistory is a list of prior values for a "real" value, each associated with
// a timestamp.
type historyEntry[T any] struct {
	Time  core.Time
	value T
}

func (he *historyEntry[T]) String() string {
	return fmt.Sprintf("<%v, %v>", &he.Time, he.value)
}

type EntryHistory[T any] struct {
	history []historyEntry[T]
	lock    sync.RWMutex
}

func (eh *EntryHistory[T]) String() string {
	eh.lock.RLock()
	defer eh.lock.RUnlock()
	if len(eh.history) == 0 {
		return "History{Empty}"
	}
	hist := make([]string, len(eh.history))
	for i, he := range eh.history {
		hist[i] = he.String()
	}
	return fmt.Sprintf("History{%s}", strings.Join(hist, ", "))
}

func (eh *EntryHistory[T]) AddEntry(value T, time *core.Time) {
	eh.lock.Lock()
	defer eh.lock.Unlock()
	curLen := len(eh.history)
	if curLen > 0 {
		// check if the last entry's time is less than current time
		if eh.history[curLen-1].Time.Cmp(time) >= 0 {
			panic("The history needs to be monotonically increasing for each entry!")
		}
	}
	newEntry := historyEntry[T]{
		value: value,
	}
	newEntry.Time.Set(time)
	eh.history = append(eh.history, newEntry)
}

func (eh *EntryHistory[T]) ReadEntry(time *core.Time, policy PMUBehavior) T {
	eh.lock.RLock()
	defer eh.lock.RUnlock()
	ind, _ := sort.Find(len(eh.history), func(i int) int {
		return time.Cmp(&eh.history[i].Time)
	})
	// ind is now one-past the write we care about
	if ind == 0 {
		if policy.USE_DEFAULT_VALUE {
			var x T
			return x
		} else {
			panic(fmt.Sprintf("Trying to read a value before any writes have occurred! Time: %v History: %s", time, eh))
		}
	}
	return eh.history[ind-1].value
}

// Deletes all the history that happened before time. However, the latest entry is kept regardless.
func (eh *EntryHistory[T]) PurgeHistory(time *core.Time) {
	eh.lock.Lock()
	defer eh.lock.Unlock()
	ind, _ := sort.Find(len(eh.history), func(i int) int {
		return time.Cmp(&eh.history[i].Time)
	})
	// ind-1 is the last visible write before time.
	sLower := ix.Max(ind-1, 0)
	eh.history = eh.history[sLower:]
}

type PMUDataStore[T datatypes.DAMType] struct {
	dataStore []EntryHistory[T]
	Behavior  PMUBehavior
	Capacity  int64
}

func (pmu *PMUDataStore[T]) mapAndCheckIndex(index int64) int64 {
	if !pmu.Behavior.NO_MOD_ADDRESS {
		return index % pmu.Capacity
	}
	if index >= pmu.Capacity {
		panic(fmt.Sprintf("Out of bounds access at address %d (PMU Size %d)", index, pmu.Capacity))
	}
	return index
}

func (pmu *PMUDataStore[T]) Write(index int64, value T, time *core.Time) {
	pmu.dataStore[pmu.mapAndCheckIndex(index)].AddEntry(value, time)
}

func (pmu *PMUDataStore[T]) Read(index int64, time *core.Time) T {
	return pmu.dataStore[pmu.mapAndCheckIndex(index)].ReadEntry(time, pmu.Behavior)
}

func (pmu *PMUDataStore[T]) Init() {
	pmu.dataStore = make([]EntryHistory[T], pmu.Capacity)
}
