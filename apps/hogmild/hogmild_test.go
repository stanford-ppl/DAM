package hogmild

import (
	"fmt"
	"testing"
)

func assert_uint_eq(t *testing.T, exp, got uint) {
	if exp != got {
		t.Errorf("Expected: %d, got: %d\n", exp, got)
	}
}

func TestHogmildSequential(t *testing.T) {
	conf := config{
		sendingTime:     8,
		networkDelay:    32,
		foldLatency:     32,
		gradientLatency: 64,
		fifo_depth:      8,
		nSamples:        4,
		nWorkers:        1,
		nWeightBanks:    1,
	}

	updateLogs := hogmild(&conf)

	assert_uint_eq(t, conf.nSamples, uint(len(updateLogs)))

	fmt.Printf("Got %d updates\n", len(updateLogs))
	for i, s := range updateLogs {
		assert_uint_eq(t, uint(i), s.sampleId)
		assert_uint_eq(t, uint(i), s.weightVersion)
		fmt.Println(s)
	}
}
