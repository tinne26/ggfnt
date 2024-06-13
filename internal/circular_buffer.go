package internal

type CircularBufferU16[T any] struct {
	buffer []T
	capacity uint16
	headIndex uint16
	numElements uint16
	numPeeks uint16
}

func (self *CircularBufferU16[T]) IsEmpty() bool {
	return self.numElements == 0
}

func (self *CircularBufferU16[T]) Capacity() uint16 {
	return self.capacity
}

func (self *CircularBufferU16[T]) Size() uint16 {
	return self.numElements
}

func (self *CircularBufferU16[T]) Clear() {
	self.headIndex   = 0
	self.numElements = 0
	self.numPeeks    = 0
}

// This method panics if we go above the current capacity.
func (self *CircularBufferU16[T]) Push(element T) {
	// this is an expensive check that maybe shouldn't be happening
	// here, but let's keep it safe and simple for the moment
	if self.Size() >= self.Capacity() {
		panic("can't Push() into CircularBufferU16, buffer already full")
	}

	self.numElements += 1
	self.buffer[self.unsafeLastElemIndex()] = element
}

func (self *CircularBufferU16[T]) PeekAhead() (T, bool) {
	if self.IsEmpty() {
		var zero T
		return zero, false
	}

	if self.numPeeks == self.numElements - 1 {
		return self.buffer[self.unsafePeekElemIndex()], false
	} else {
		self.numPeeks += 1
		return self.buffer[self.unsafePeekElemIndex()], true
	}
}

func (self *CircularBufferU16[T]) DiscardPeeks() {
	self.numPeeks = 0
}

func (self *CircularBufferU16[T]) ConfirmPeeks() {
	self.headIndex = self.unsafePeekElemIndex()
	self.numElements -= self.numPeeks
	self.numPeeks = 0
}

func (self *CircularBufferU16[T]) Head() T {
	if self.IsEmpty() {
		panic("can't Head() on empty CircularBufferU16")
	}

	return self.buffer[self.headIndex]
}

func (self *CircularBufferU16[T]) Tail() T {
	if self.IsEmpty() {
		panic("can't Tail() on empty CircularBufferU16")
	}

	return self.buffer[self.unsafeLastElemIndex()]
}

func (self *CircularBufferU16[T]) PopHead() T {
	if self.IsEmpty() {
		panic("can't PopHead() on empty CircularBufferU16")
	}

	element := self.buffer[self.headIndex]
	self.numElements -= 1
	self.headIndex += 1
	if self.headIndex == self.capacity {
		self.headIndex = 0
	}
	return element
}

func (self *CircularBufferU16[T]) PopTail() T {
	if self.IsEmpty() {
		panic("can't PopTail() on empty CircularBufferU16")
	}

	element := self.buffer[self.unsafeLastElemIndex()]
	self.numElements -= 1
	return element
}

func (self *CircularBufferU16[T]) SetMinCapacity(capacity uint16) bool {
	if self.Capacity() >= capacity { return true }
	return self.SetCapacity(capacity)
}

// Returns false if capacity < CircularBufferU16.Len().
func (self *CircularBufferU16[T]) SetCapacity(capacity uint16) bool {
	// trivial case
	if self.capacity == capacity { return true }
	
	// get current length and check that it's compatible with the requested capacity
	if capacity < self.numElements { return false }
	
	// see if we are growing or shrinking
	if capacity < self.capacity { // shrinking
		lenReduction := self.capacity - capacity
		if self.numElements > (self.capacity - self.headIndex) {
			copy(self.buffer[self.headIndex - lenReduction : ], self.buffer[self.headIndex : ])
			self.headIndex -= lenReduction
		} else if self.headIndex > capacity - self.numElements {
			copy(self.buffer[ : self.numElements], self.buffer[self.headIndex : self.headIndex + self.numElements])
			self.headIndex = 0
		}
		self.buffer = self.buffer[ : capacity]
	} else { // growing
		extraLen := capacity - self.capacity
		self.buffer = GrowSliceByN(self.buffer, int(extraLen))
		
		// correct indices if necessary
		if self.numElements > (self.capacity - self.headIndex) {
			copy(self.buffer[self.headIndex + extraLen : ], self.buffer[self.headIndex : self.capacity])
			self.headIndex += extraLen
		}
	}

	self.capacity = capacity
	return true
}

// --- helper methods ---

// Precondition: circular buffer is not empty.
func (self *CircularBufferU16[T]) unsafeLastElemIndex() uint16 {
	index := self.headIndex + self.numElements - 1
	
	if index < self.headIndex { // overflow case
		return self.numElements - (self.capacity - self.headIndex)
	} else if index < self.capacity { // simple case
		return index
	} else { // wrap case
		return index - self.capacity
	}
}

// Precondition: circular buffer is not empty.
func (self *CircularBufferU16[T]) unsafePeekElemIndex() uint16 {
	index := self.headIndex + self.numPeeks
	
	if index < self.headIndex { // overflow case
		return self.numPeeks - (self.capacity - self.headIndex)
	} else if index < self.capacity { // simple case
		return index
	} else { // wrap case
		return index - self.capacity
	}
}
