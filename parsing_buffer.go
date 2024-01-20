package ggfnt

import "io"
import "errors"
import "unsafe"
import "encoding/binary"
import "compress/gzip"

// creating a reusable buffer doesn't make much sense because
// then we will unnecessary keep a tempBuff, and the cost of
// parsing exceeds the cost of allocating <2KiB each time that
// it's needed

type parsingBuffer struct {
	tempBuff []byte // size 1024, for temporary reads immediately copied to 'bytes'
	gzipReader *gzip.Reader

	bytes []byte
	index int // index of processed data within 'bytes'. unprocessed data == len(bytes) - index
	eof bool
}

func (self *parsingBuffer) InitBuffers() {
	self.tempBuff = make([]byte, 1024)
	self.bytes    = make([]byte, 1024)
	self.index = 0
	self.eof = false
}

func (self *parsingBuffer) InitGzipReader(reader io.Reader) error {
	var err error
	self.gzipReader, err = gzip.NewReader(reader)
	return err
}

func (self *parsingBuffer) EnsureEOF() error {
	if self.eof { return nil }
	preIndex := self.index
	err := self.readMore()
	if err != nil { return err }
	if self.index > preIndex {
		return errors.New("file continues beyond the expected end")
	}
	if !self.eof { panic("broken code") }
	return nil
}

// utility function called to read more data
func (self *parsingBuffer) readMore() error {
	for retries := 0; retries < 3; retries++ {
		// read and process read bytes
		n, err := self.gzipReader.Read(self.tempBuff)
		if n > 0 {
			self.bytes = growSliceByN(self.bytes, n)
			if len(self.bytes) > MaxSize {
				return newParseErr("font data size exceeds limit")
			}
			k := copy(self.bytes[: len(self.bytes) - n], self.tempBuff[ : n])
			if k != n { panic("broken code") }
		}

		// handle errors
		if err == io.EOF {
			self.eof = true
			return nil
		} else if err != nil {
			return err
		}

		// return if we have read something
		if n != 0 { return nil }
	}

	// fallback error case if repeated reads still don't lead us anywhere
	return newParseErr("repeated empty reads")
}

func (self *parsingBuffer) readUpTo(newIndex int) error {
	if newIndex <= self.index { panic("readUpTo() misuse") }
	for len(self.bytes) < newIndex {
		if self.eof {
			return newParseErr("premature end of file (or font offsets are wrong)")
		}
		err := self.readMore()
		if err != nil { return err }	
	}
	self.index = newIndex
	return nil
}

func (self *parsingBuffer) AdvanceBytes(n int) error {
	if n <= 0 { panic("AdvanceBytes(N) where N <= 0") }
	return self.readUpTo(self.index + n)
}

func (self *parsingBuffer) AdvanceShortSlice() error {
	sliceLen, err := self.ReadUint8()
	if err != nil { return err }	
	return self.AdvanceBytes(int(sliceLen))
}

func (self *parsingBuffer) ReadUint64() (uint64, error) {
	index := self.index
	err := self.readUpTo(index + 8)
	if err != nil { return 0, err }
	return binary.LittleEndian.Uint64(self.bytes[index : ]), nil
}

func (self *parsingBuffer) ReadUint32() (uint32, error) {
	index := self.index
	err := self.readUpTo(index + 4)
	if err != nil { return 0, err }
	return binary.LittleEndian.Uint32(self.bytes[index : ]), nil
}

func (self *parsingBuffer) ReadInt32() (int32, error) {
	n, err := self.ReadUint32()
	return int32(n), err
}

func (self *parsingBuffer) ReadUint16() (uint16, error) {
	index := self.index
	err := self.readUpTo(index + 2)
	if err != nil { return 0, err }
	return binary.LittleEndian.Uint16(self.bytes[index : ]), nil
}

func (self *parsingBuffer) ReadUint8() (uint8, error) {
	index := self.index
	err := self.readUpTo(index + 1)
	if err != nil { return 0, err }
	return self.bytes[index], nil
}

// Returns bool value, new index, and an error if format was incorrect.
func (self *parsingBuffer) ReadBool() (bool, error) {
	value, err := self.ReadUint8()
	if err != nil { return false, err }
	if value == 0 { return false, nil }
	if value == 1 { return true , nil }
	return false, newParseErr(boolErrCheck(value).Error())
}

func (self *parsingBuffer) ReadShortStr() (string, error) {
	length, err := self.ReadUint8()
	if err != nil { return "", err }
	index := self.index
	err = self.readUpTo(index + int(length))
	if err != nil { return "", err }
	return unsafe.String(&self.bytes[index], int(length)), nil
}

func (self *parsingBuffer) ReadString() (string, error) {
	length, err := self.ReadUint16()
	if err != nil { return "", err }
	index := self.index
	err = self.readUpTo(index + int(length))
	if err != nil { return "", err }
	return unsafe.String(&self.bytes[index], int(length)), nil
}

// TODO: slice reads/skips, signed integer types, date parsing, basic-name-regexp checks, etc.
