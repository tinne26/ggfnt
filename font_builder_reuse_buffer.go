package ggfnt

import "slices"
import "sync/atomic"

type nameIndexEntryBuffer struct {
	buffer []nameIndexEntry
}
func (self *nameIndexEntryBuffer) Reset() {
	self.buffer = self.buffer[ : 0]
}
// func (self *nameIndexEntryBuffer) RequestMinExtraCapacity(n int) {
// 	missingCapacity := cap(self.buffer) - len(self.buffer)
// 	if missingCapacity <= 0 { return }
// 	newBuffer := make([]nameIndexEntry, len(self.buffer) + missingCapacity)
// 	copy(newBuffer, self.buffer)
// 	self.buffer = newBuffer
// }
func (self *nameIndexEntryBuffer) AppendAllMapUint16(set map[uint16]nameIndexEntry) {
	for _, entry := range set {
		self.buffer = append(self.buffer, entry)
	}
}
func (self *nameIndexEntryBuffer) AppendAllMapUint8(set map[uint8]nameIndexEntry) {
	for _, entry := range set {
		self.buffer = append(self.buffer, entry)
	}
}
func (self *nameIndexEntryBuffer) Sort() {
	slices.SortFunc(self.buffer, func(a, b nameIndexEntry) int {
		if a.Name < b.Name { return -1 }
		if a.Name > b.Name { return  1 }
		return 0 // ==
	})
}

var reusableNameIndexEntryBuffer nameIndexEntryBuffer
var usingNameIndexEntryBuffer uint32
func getNameIndexEntryBuffer() *nameIndexEntryBuffer {
	if atomic.CompareAndSwapUint32(&usingNameIndexEntryBuffer, 0, 1) {
		reusableNameIndexEntryBuffer.Reset()
		return &reusableNameIndexEntryBuffer
	} else {
		var buffer nameIndexEntryBuffer
		return &buffer
	}
}

func releaseNameIndexEntryBuffer(buffer *nameIndexEntryBuffer) {
	if buffer == &reusableNameIndexEntryBuffer {
		atomic.StoreUint32(&usingNameIndexEntryBuffer, 0)
	}
}
