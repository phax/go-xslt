package xpath

import (
	"testing"
	"unsafe"
)

// TestAtomicSizeStaysSmall pins sizeof(Atomic): the rare large payloads live
// behind ext so a string/number/boolean pays for 64 bytes, not ~190.
func TestAtomicSizeStaysSmall(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("layout budget is for 64-bit targets")
	}
	if got := unsafe.Sizeof(Atomic{}); got > 64 {
		t.Errorf("sizeof(Atomic) = %d, want <= 64", got)
	}
}
