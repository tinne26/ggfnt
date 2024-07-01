package builder

func startsWith(definition string, start, end int, expr string) bool {
	if start < 0 || end + 1 > len(definition) { panic("broken code usage") }
	if len(expr) > (end + 1 - start) { return false }
	for i := 0; i < len(expr); i++ {
		if definition[start + i] != expr[i] { return false }
	}
	return true
}

// end is inclusive
func trimSpaces(definition string, start, end int) (int, int) {
	var changed bool = true
	for start <= end && changed {
		changed = false

		// trim spaces
		for start <= end && definition[start] == ' ' {
			start += 1
			changed = true
		}
		for end >= 0 && definition[end] == ' ' {
			end -= 1
			changed = true
		}
	}

	return start, end
}

type boolList struct {
	size int
	bools []uint8
}

func (self *boolList) Push(value bool) {
	if (self.size >> 3) >= len(self.bools) {
		self.bools = append(self.bools, 0)
	}
	self.size += 1
	self.Set(self.size - 1, value)
}

func (self *boolList) SetAllFalse() {
	for i, _ := range self.bools {
		self.bools[i] = 0
	}
}

func (self *boolList) Get(i int) bool {
	if i < 0 || i >= self.size { panic("out of bounds") }
	byteIndex := (i >> 3)
	bitMask   := uint8(0b0000_00001 << uint8(i - (byteIndex << 3)))
	return (self.bools[byteIndex] & bitMask) != 0
}

func (self *boolList) Set(i int, value bool) {
	if i < 0 || i >= self.size { panic("out of bounds") }
	byteIndex := (i >> 3)
	bitMask   := uint8(0b0000_00001 << uint8(i - (byteIndex << 3)))
	if value {
		self.bools[byteIndex] |= bitMask
	} else {
		self.bools[byteIndex] &= ^bitMask
	}
}
