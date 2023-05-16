package plasticine

import (
	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/datatypes"
	internal "github.com/stanford-ppl/DAM/templates/plasticine/internal"
	"github.com/stanford-ppl/DAM/templates/shared/accesstypes"
	"github.com/stanford-ppl/DAM/utils"
)

// Plasticine PMUs operate two parallel pipelines that can independently stall:
// one for reads, and one for writes. This significantly complicates the implementation
// In order to handle this, the PMU actually consists of TWO halves, which share a common data store.
// The PMUWriter simply takes values from the write requests and sends them along to the ack.

type (
	Behavior = internal.PMUBehavior
)

type PMU[T datatypes.DAMType] interface {
	core.Context
	AddReader(addr *core.CommunicationChannel, outputs []*core.CommunicationChannel, tp accesstypes.AccessType)
	AddWriter(addr *core.CommunicationChannel, data *core.CommunicationChannel,
		enable utils.Option[*core.CommunicationChannel], ack []*core.CommunicationChannel, tp accesstypes.AccessType,
	)
}

func MakeBehavior() Behavior {
	return Behavior{}
}

func MakePMU[T datatypes.DAMType](capacity int64, latency int64, behavior Behavior) PMU[T] {
	return internal.MakePMU[T](capacity, latency, behavior)
}
