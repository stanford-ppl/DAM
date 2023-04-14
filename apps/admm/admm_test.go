package admm

/*
extern int demo();
*/
import "C"

import (
	"testing"
)

func Test(t *testing.T) {
	ret := C.demo()
	if ret != 1 {
		t.Error("failed")
	}
}
