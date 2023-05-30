package core

import (
	"math/big"
	"testing"
)

func TestSimpleTime(t *testing.T) {
	timeZero := NewTime(0)
	if timeZero.time.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Expected: %d, received: %d\n", 0, timeZero.time.Int64())
	}
	if timeZero.done {
		t.Errorf("Expected: not done, received: done\n")
	}

	timeOne := timeZero.Add(timeZero, NewTime(1))
	if timeOne.time.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Expected: %d, received: %d\n", 1, timeZero.time.Int64())
	}

	timeTwo := timeOne.Add(timeOne, NewTime(1))
	if timeTwo.time.Cmp(big.NewInt(2)) != 0 {
		t.Errorf("Expected: %d, received: %d\n", 2, timeZero.time.Int64())
	}
}
