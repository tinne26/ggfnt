package internal

import "cmp"
import "errors"
import "crypto/rand"

func GrowSliceByN[T any](buffer []T, increase int) []T {
	newSize := len(buffer) + increase
	if cap(buffer) >= newSize {
		return buffer[ : newSize]
	} else {
		newBuffer := make([]T, newSize)
		copy(newBuffer, buffer)
		return newBuffer
	}
}

// This method is only useful when T is a type that's either very
// large or it contains slices already allocated that we want to
// preserve and reuse. In those cases, GrowSliceByOne might help
// reduce GC and new allocations (it can also unnecessarily hog
// memory when misused).
func GrowSliceByOne[T any](buffer []T) []T {
	newSize := len(buffer) + 1
	if cap(buffer) >= newSize {
		return buffer[ : newSize]
	} else {
		var element T
		return append(buffer, element)
	}
}

// The contents might be cleared.
func SetSliceSize[T any](buffer []T, size int) []T {
	if cap(buffer) >= size {
		return buffer[ : size]
	} else {
		return make([]T, size)
	}
}

func DeleteElementAt[T any](buffer []T, index int) []T {
	// trivial cases
	size := len(buffer)
	if index + 1  > size { return buffer }
	if index + 1 == size { return buffer[ : size - 1] }
	
	// general case
	memo := buffer[index]
	copy(buffer[index : size - 1], buffer[index + 1 : ])
	buffer[size - 1] = memo // potentially preserve sub-allocated memory
	return buffer[ : size - 1]
}

// Like slices.BinarySearch, but without NaN nor overflow support.
// And better variable names, seriously...
func BadBinarySearch[T cmp.Ordered](slice []T, target T) (int, bool) {
	numElements := len(slice)
	minIndex, maxIndex := 0, numElements
	for minIndex < maxIndex {
		ctrIndex := (minIndex + maxIndex) >> 1
		if slice[ctrIndex] < target {
			minIndex = ctrIndex + 1
		} else {
			maxIndex = ctrIndex
		}
	}
	return minIndex, minIndex < numElements && slice[minIndex] == target
}

func BasicOrderedNonRepeatInsert[T cmp.Ordered](buffer []T, element T) []T {
	defaultInsertionIndex := len(buffer)
	insertionIndex := defaultInsertionIndex
	for insertionIndex > 0 { // could be a binary search instead
		candidate := buffer[insertionIndex - 1]
		if candidate < element { break }
		if candidate == element { return buffer } // redundant case
		insertionIndex -= 1
	}
	
	// grow slice, make space for new state if necessary by
	// shifting previously existing data, and set state index
	buffer = GrowSliceByOne(buffer)
	if insertionIndex != defaultInsertionIndex { // make space for new
		copy(buffer[insertionIndex + 1 : ], buffer[insertionIndex : ])
	}
	buffer[insertionIndex] = element
	return buffer
}

func BoolErrCheck(value uint8) error {
	if (value == 0 || value == 1) { return nil }
	return errors.New("bool value must be 0 or 1")
}

func BoolToUint8(truthy bool) uint8 {
	if truthy { return 1 }
	return 0
}

// Values <0.3 are fairly low entropy. anything <0.2 tends to be visibly low entropy
func LazyEntropyUint64(value uint64) float64 {
	var patterns [4]uint8
	for i := 0; i < 64; i += 2 {
		patterns[value & 0b11] += 1
		value >>= 1
	}
	
	var dist uint8
	for _, count := range patterns {
		if count <= 8 {
			dist += (8 - count)
		} else {
			dist += (count - 8)
		}
	}
	
	return 1.0 - float64(dist)/48.0
}

func CryptoRandUint64() (uint64, error) {
   randBytes := make([]byte, 8)
   _, err := rand.Read(randBytes)
   if err != nil { return 0, err }

   var id uint64
   for i := 0; i < 8; i++ {
      id |= (uint64(randBytes[i]) << (i << 3))
   }
   return id, nil
}

// LE stands for "little endian"

func DecodeUint16LE(buffer []byte) uint16 {
	if len(buffer) < 2 { panic(len(buffer)) }
	return uint16(buffer[0]) | (uint16(buffer[1]) << 8)
}

func DecodeUint24LE(buffer []byte) uint32 {
	if len(buffer) < 3 { panic(len(buffer)) }
	return (uint32(buffer[0]) <<  0) | (uint32(buffer[1]) <<  8) | (uint32(buffer[2]) << 16)
}

func DecodeUint32LE(buffer []byte) uint32 {
	if len(buffer) < 4 { panic(len(buffer)) }
	return (uint32(buffer[0]) <<  0) | (uint32(buffer[1]) <<  8) |
	       (uint32(buffer[2]) << 16) | (uint32(buffer[3]) << 24)
}

func DecodeDate(buffer []byte) (uint16, uint8, uint8) {
	if len(buffer) < 4 { panic(len(buffer)) }
	return DecodeUint16LE(buffer[0 : 2]), uint8(buffer[2]), uint8(buffer[3])
}

func DecodeUint64LE(buffer []byte) uint64 {
	if len(buffer) < 8 { panic(len(buffer)) }
	return (uint64(buffer[0]) <<  0) | (uint64(buffer[1]) <<  8) | (uint64(buffer[2]) << 16) |
	       (uint64(buffer[3]) << 24) | (uint64(buffer[4]) << 32) | (uint64(buffer[5]) << 40) |
	       (uint64(buffer[6]) << 48) | (uint64(buffer[7]) << 56)
}

func AppendUint8(buffer []byte, value byte) []byte {
	return append(buffer, value)
}

func AppendUint16LE(buffer []byte, value uint16) []byte {
	return append(buffer, byte(value), byte(value >> 8))
}

func EncodeUint16LE(buffer []byte, value uint16) {
	if len(buffer) < 2 { panic("invalid usage of encodeUint16LE") }
	buffer[0] = byte(value)
	buffer[1] = byte(value >> 8)
}

func AppendUint24LE(buffer []byte, value uint32) []byte {
	return append(buffer, byte(value), byte(value >> 8), byte(value >> 16))
}

func EncodeUint24LE(buffer []byte, value uint32) {
	if len(buffer) < 3 { panic("invalid usage of encodeUint24LE") }
	buffer[0] = byte(value)
	buffer[1] = byte(value >>  8)
	buffer[2] = byte(value >> 16)
}

func AppendUint32LE(buffer []byte, value uint32) []byte {
	return append(buffer, byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24))
}

func EncodeUint32LE(buffer []byte, value uint32) {
	if len(buffer) < 4 { panic("invalid usage of encodeUint32LE") }
	buffer[0] = byte(value)
	buffer[1] = byte(value >>  8)
	buffer[2] = byte(value >> 16)
	buffer[3] = byte(value >> 24)
}

func AppendUint64LE(buffer []byte, value uint64) []byte {
	return append(
		buffer, 
		byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24),
		byte(value >> 32), byte(value >> 40), byte(value >> 48), byte(value >> 56),
	)
}

func AppendShortString(buffer []byte, str string) []byte {
	if len(str) > 255 { panic("appendShortString() can't append string with len() > 255") }
	return append(append(buffer, uint8(len(str))), str...)
}

func AppendString(buffer []byte, str string) []byte {
	if len(str) > 65535 { panic("appendString() can't append string with len() > 65535") }
	return append(AppendUint16LE(buffer, uint16(len(str))), str...)
}

func AppendByteDigits(value byte, str []byte) []byte {
	if value > 99 {
		digit := value/100
		str = append(str, digit + '0')
	}
	if value > 9 {
		digit := (value/10) % 10
		str = append(str, digit + '0')
	}
	return append(str, (value % 10) + '0')
}

type BoolList struct {
	size int
	bools []uint8
}

func NewBoolList(size int) BoolList {
	if size <= 0 { return BoolList{} }
	return BoolList{
		size: size,
		bools: make([]uint8, (size + 7) >> 3),
	}
}

func (self *BoolList) Push(value bool) {
	if (self.size >> 3) >= len(self.bools) {
		self.bools = append(self.bools, 0)
	}
	self.size += 1
	self.Set(self.size - 1, value)
}

func (self *BoolList) SetAllFalse() {
	for i, _ := range self.bools {
		self.bools[i] = 0
	}
}

func (self *BoolList) SetAllTrue() {
	for i, _ := range self.bools {
		self.bools[i] = 0b1111_1111
	}
}

func (self *BoolList) IsSet(i int) bool { return self.Get(i) }
func (self *BoolList) IsUnset(i int) bool { return !self.Get(i) }

func (self *BoolList) Get(i int) bool {
	if i < 0 || i >= self.size { panic("out of bounds") }
	bitMask := uint8(0b0000_00001) << uint8(i & 0b111)
	return (self.bools[i >> 3] & bitMask) != 0
}

func (self *BoolList) IsSetU8(i uint8) bool { return self.GetU8(i) }
func (self *BoolList) IsUnsetU8(i uint8) bool { return !self.GetU8(i) }

func (self *BoolList) GetU8(i uint8) bool {
	bitMask := uint8(0b0000_00001) << (i & 0b111)
	return (self.bools[i >> 3] & bitMask) != 0
}

func (self *BoolList) Set(i int, value bool) {
	if i < 0 || i >= self.size { panic("out of bounds") }
	bitMask := uint8(0b0000_00001) << uint8(i & 0b111)
	if value {
		self.bools[i >> 3] |= bitMask
	} else {
		self.bools[i >> 3] &= ^bitMask
	}
}

func (self *BoolList) SetU8(i uint8, value bool) {
	bitMask := uint8(0b0000_00001) << (i & 0b111)
	if value {
		self.bools[i >> 3] |= bitMask
	} else {
		self.bools[i >> 3] &= ^bitMask
	}
}

func (self *BoolList) RawGetU8(word, bitMask uint8) bool {
	return (self.bools[word] & bitMask) != 0
}

func (self *BoolList) GetWordAndBitMaskU8(i uint8) (uint8, uint8) {
	byteIndex := (i >> 3)
	bitMask   := uint8(0b0000_00001 << (i & 0b111))
	return byteIndex, bitMask
}
