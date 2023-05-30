package hogmild

import (
	"math/big"

	"github.com/stanford-ppl/DAM/datatypes"
)

type config struct {
	sendingTime  uint
	networkDelay uint

	foldLatency uint

	nSamples     uint
	nWorkers     uint
	nWeightBanks uint
}

type sample struct {
	sampleId      uint
	weightVersion uint
}

func (s sample) Validate() bool {
	return true
}

func (s sample) Size() *big.Int {
	return big.NewInt(42)
}

func (s sample) Payload() any {
	return s
}

var _ datatypes.DAMType = (*sample)(nil)

func hogmild() {
}
