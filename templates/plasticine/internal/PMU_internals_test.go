package plasticine

import (
	"testing"

	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/utils"
)

func PktLT2[T any](p1, p2 PMUPacket[T], t *testing.T) bool {
	cmp := p1.Time.Cmp(&p2.Time)
	t.Log("Cmp", cmp)
	t.Log("Statuses:", p1.Status, p2.Status)

	if cmp == 0 {
		// In the case that they're at the same time, favor the packet that actually has data.
		if p1.Status == core.Nothing {
			return false
		}
		if p2.Status == core.Nothing {
			return true
		}
	}
	return cmp < 0
}

func TestPKTLT(t *testing.T) {
	pkt1 := PMUPacket[any]{}
	pkt1.Time.Set(core.OneTick)
	pkt1.Status = core.Nothing

	pkt2 := PMUPacket[any]{}
	pkt2.Time.Set(core.OneTick)
	pkt2.Status = core.Ok

	if PktLT2(pkt1, pkt2, t) {
		t.Error("Expected a packet with nothing to be deprioritized over a packet that has data!")
	}
	min := utils.MinElem([]PMUPacket[any]{pkt1, pkt2}, PktLT[any])
	t.Logf("Minimum packet: %+v", min)
}
