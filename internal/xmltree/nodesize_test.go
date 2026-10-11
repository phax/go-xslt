package xmltree

import (
	"testing"
	"unsafe"
)

// TestNodeSizeStaysInItsAllocatorClass pins sizeof(Node) so a stray int field
// (or a reordering that reintroduces padding) can't silently push every node
// back into the next Go allocator size class: 160 bytes is a size class of its
// own, 176 and 144 are its neighbours — each byte over a class boundary costs
// the whole next step, on every node in every tree.
func TestNodeSizeStaysInItsAllocatorClass(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("layout budget is for 64-bit targets")
	}
	if got := unsafe.Sizeof(Node{}); got > 160 {
		t.Errorf("sizeof(Node) = %d, want <= 160", got)
	}
}
