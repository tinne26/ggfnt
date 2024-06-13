package internal

import "testing"

func TestCircularBufferU16(t *testing.T) {
	var buffer CircularBufferU16[int]
	
	// basic set capacity test
	ok := buffer.SetCapacity(3)
	if !ok { t.Fatalf("failed to set capacity") }
	if buffer.Capacity() != 3 {
		t.Fatalf("expected capacity %d, got %d", 3, buffer.Capacity())
	}

	// basic push test
	buffer.Push(1)
	if buffer.Size() != 1 { t.Fatalf("expected size %d, got %d", 1, buffer.Size()) }
	buffer.Push(2)
	buffer.Push(3)
	if buffer.Size() != 3 { t.Fatalf("expected size %d, got %d", 3, buffer.Size()) }

	// basic lookup tests
	if buffer.Head() != 1 { t.Fatalf("expected head %d, got %d", 1, buffer.Head()) }
	if buffer.Tail() != 3 { t.Fatalf("expected tail %d, got %d", 3, buffer.Tail()) }
	peek, ok := buffer.PeekAhead()
	if !ok { t.Fatalf("expected peek to be possible") }
	if peek != 2 { t.Fatalf("expected peek %d, got %d", 2, peek) }
	buffer.ConfirmPeeks()
	if buffer.Head() != 2 { t.Fatalf("expected head %d, got %d", 2, buffer.Head()) }
	peek, ok = buffer.PeekAhead()
	if !ok { t.Fatalf("expected peek to be possible") }
	if peek != 3 { t.Fatalf("expected head %d, got %d", 3, buffer.Head()) }
	buffer.DiscardPeeks()
	if buffer.Head() != 2 { t.Fatalf("expected head %d, got %d", 2, buffer.Head()) }
	value := buffer.PopHead()
	if value != 2 { t.Fatalf("expected head pop %d, got %d", 2, value) }	
	if buffer.Size() != 1 { t.Fatalf("expected size %d, got %d", 1, buffer.Size()) }

	// wrapping pushes
	buffer.Push(4)
	if buffer.Size() != 2 { t.Fatalf("expected size %d, got %d", 2, buffer.Size()) }
	if buffer.Head() != 3 { t.Fatalf("expected head %d, got %d", 3, buffer.Head()) }
	if buffer.Tail() != 4 { t.Fatalf("expected tail %d, got %d", 4, buffer.Tail()) }
	if buffer.SetCapacity(1) {
		t.Fatalf("expected such low capacity to be impossible to set")
	}
	if !buffer.SetCapacity(2) { // shrink with wrap
		t.Fatalf("expected capacity to be possible to set")
	}
	if buffer.Head() != 3 { t.Fatalf("expected head %d, got %d", 3, buffer.Head()) }
	if buffer.Tail() != 4 { t.Fatalf("expected tail %d, got %d", 4, buffer.Tail()) }
	if buffer.PopTail() != 4 { t.Fatalf("expected tail %d, got %d", 4, buffer.Tail()) }
	if !buffer.SetCapacity(3) {
		t.Fatalf("expected capacity to be possible to set")
	}
	buffer.Push(6)
	if buffer.Size() != 2 { t.Fatalf("expected size %d, got %d", 2, buffer.Size()) }
	if buffer.Head() != 3 { t.Fatalf("expected head %d, got %d", 3, buffer.Head()) }
	if buffer.Tail() != 6 { t.Fatalf("expected tail %d, got %d", 6, buffer.Tail()) }
	if !buffer.SetCapacity(2) { // shrink with shift
		t.Fatalf("expected capacity to be possible to set")
	}
	if buffer.Size() != 2 { t.Fatalf("expected size %d, got %d", 2, buffer.Size()) }
	if buffer.Head() != 3 { t.Fatalf("expected head %d, got %d", 3, buffer.Head()) }
	if buffer.Tail() != 6 { t.Fatalf("expected tail %d, got %d", 6, buffer.Tail()) }
	
	value = buffer.PopHead()
	if value != 3 { t.Fatalf("expected head pop %d, got %d", 3, value) }
	buffer.Push(7)
	if !buffer.SetCapacity(5) {
		t.Fatalf("expected capacity to be possible to set")
	}
	if buffer.Size() != 2 { t.Fatalf("expected size %d, got %d", 2, buffer.Size()) }
	if buffer.Head() != 6 { t.Fatalf("expected head %d, got %d", 6, buffer.Head()) }
	if buffer.Tail() != 7 { t.Fatalf("expected tail %d, got %d", 7, buffer.Tail()) }

	//t.Fatalf("%v", buffer.buffer)
}
