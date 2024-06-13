package internal

import "sync/atomic"

// Kind of a single-element pool. Basically, it instances
// one element and associates it to a lock. If we request
// more elements of that type, we are creating new ones
// instead. This means that non-concurrent operation can reuse
// an element, and concurrent access is not expected nor
// desirable, but not technically incorrect (only potentially
// inefficient).
type SingleInstancePool[T any] struct {
	instance T
	inUse uint32
}

func (self *SingleInstancePool[T]) Retrieve() *T {
	if atomic.CompareAndSwapUint32(&self.inUse, 0, 1) {
		return &self.instance
	} else {
		var untrackedInstance T
		return &untrackedInstance
	}
}

func (self *SingleInstancePool[T]) Release(instance *T) {
	if instance == &self.instance {
		if !atomic.CompareAndSwapUint32(&self.inUse, 1, 0) {
			panic(BrokenCode)
		}
	}
}
