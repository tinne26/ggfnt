package internal

import "io"
import "errors"
import "unsafe"
import "compress/gzip"

// creating a reusable buffer doesn't make much sense because
// then we will unnecessary keep a tempBuff, and the cost of
// parsing exceeds the cost of allocating <2KiB each time that
// it's needed

type ParsingBuffer struct {
	TempBuff []byte // size 1024, for temporary reads immediately copied to 'bytes'
	gzipReader *gzip.Reader
	FileType string

	Bytes []byte
	Index int // index of processed data within 'bytes'. unprocessed data == len(bytes) - index
	eof bool
}

func (self *ParsingBuffer) NewError(details string) error {
	return errors.New(self.FileType + " parsing error: " + details)
}

func (self *ParsingBuffer) InitBuffers() {
	self.TempBuff = make([]byte, 1024)
	self.Bytes    = make([]byte, 0, 1024)
	self.Index = 0
	self.eof = false
}

func (self *ParsingBuffer) InitGzipReader(reader io.Reader) error {
	var err error
	self.gzipReader, err = gzip.NewReader(reader)
	return err
}

func (self *ParsingBuffer) EnsureEOF() error {
	if self.eof { return nil }
	preIndex := self.Index
	err := self.readMore()
	if err != nil { return err }
	if self.Index > preIndex {
		return errors.New("file continues beyond the expected end")
	}
	if !self.eof { panic("broken code") }
	return nil
}

// utility function called to read more data
func (self *ParsingBuffer) readMore() error {
	for retries := 0; retries < 3; retries++ {
		// read and process read bytes
		n, err := self.gzipReader.Read(self.TempBuff)
		if n > 0 {
			self.Bytes = GrowSliceByN(self.Bytes, n)
			if len(self.Bytes) > MaxFontDataSize {
				return self.NewError("font data size exceeds limit")
			}
			k := copy(self.Bytes[len(self.Bytes) - n : ], self.TempBuff[ : n])
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
	return self.NewError("repeated empty reads")
}

func (self *ParsingBuffer) readUpTo(newIndex int) error {
	if newIndex <= self.Index { panic("readUpTo() misuse") }
	for len(self.Bytes) < newIndex {
		if self.eof {
			return self.NewError("premature end of file (or font offsets are wrong)")
		}
		err := self.readMore()
		if err != nil { return err }	
	}
	self.Index = newIndex
	return nil
}

func (self *ParsingBuffer) AdvanceBytes(n int) error {
	if n == 0 { return nil }
	if n < 0 { panic("AdvanceBytes(N) where N < 0") }
	return self.readUpTo(self.Index + n)
}

func (self *ParsingBuffer) AdvanceShortSlice() error {
	sliceLen, err := self.ReadUint8()
	if err != nil { return err }	
	return self.AdvanceBytes(int(sliceLen))
}

func (self *ParsingBuffer) ReadUint64() (uint64, error) {
	index := self.Index
	err := self.readUpTo(index + 8)
	if err != nil { return 0, err }
	return DecodeUint64LE(self.Bytes[index : ]), nil
}

func (self *ParsingBuffer) ReadUint32() (uint32, error) {
	index := self.Index
	err := self.readUpTo(index + 4)
	if err != nil { return 0, err }
	return DecodeUint32LE(self.Bytes[index : ]), nil
}

func (self *ParsingBuffer) ReadInt32() (int32, error) {
	n, err := self.ReadUint32()
	return int32(n), err
}

func (self *ParsingBuffer) ReadUint16() (uint16, error) {
	index := self.Index
	err := self.readUpTo(index + 2)
	if err != nil { return 0, err }
	return DecodeUint16LE(self.Bytes[index : ]), nil
}

func (self *ParsingBuffer) ReadUint8() (uint8, error) {
	index := self.Index
	err := self.readUpTo(index + 1)
	if err != nil { return 0, err }
	return self.Bytes[index], nil
}

func (self *ParsingBuffer) ReadInt8() (int8, error) {
	index := self.Index
	err := self.readUpTo(index + 1)
	if err != nil { return 0, err }
	return int8(self.Bytes[index]), nil
}

// Returns bool value, new index, and an error if format was incorrect.
func (self *ParsingBuffer) ReadBool() (bool, error) {
	value, err := self.ReadUint8()
	if err != nil { return false, err }
	if value == 0 { return false, nil }
	if value == 1 { return true , nil }
	return false, self.NewError(BoolErrCheck(value).Error())
}

func (self *ParsingBuffer) ReadShortStr() (string, error) {
	length, err := self.ReadUint8()
	if err != nil { return "", err }
	if length == 0 { return "", nil }
	index := self.Index
	err = self.readUpTo(index + int(length))
	if err != nil { return "", err }
	return unsafe.String(&self.Bytes[index], int(length)), nil
}

func (self *ParsingBuffer) ReadString() (string, error) {
	length, err := self.ReadUint16()
	if err != nil { return "", err }
	if length == 0 { return "", nil }
	index := self.Index
	err = self.readUpTo(index + int(length))
	if err != nil { return "", err }
	return unsafe.String(&self.Bytes[index], int(length)), nil
}

func (self *ParsingBuffer) ValidateBasicName(name string) error {
	err := ValidateBasicName(name)
	if err == nil { return nil }
	return self.NewError(err.Error())
}

func ValidateBasicName(name string) error {
	if len(name) > 32 { return errors.New("basic name can't exceed 32 characters") }
	if len(name) == 0 { return errors.New("basic name can't be empty") }
	if name[0] == '-' { return errors.New("basic name can't start with a hyphen") }

	var prevIsHyphen bool
	for i := 0; i < len(name); i++ {
		if isAZaz09(name[i]) {
			prevIsHyphen = false
			continue
		}
		if name[i] == '-' {
			if prevIsHyphen {
				errors.New("basic name can't contain consecutive hyphens")
			}
			prevIsHyphen = true
			continue
		}
		return errors.New("basic name contains invalid character")
	}
	
	if prevIsHyphen {
		return errors.New("basic name can't end with a hyphen")
	}

	return nil
}

// Repeated spaces and hyphens and stuff are ok on basic names.
func (self *ParsingBuffer) ValidateBasicSpacedName(name string) error {
	err := validateBasicSpacedName(name)
	if err == nil { return nil }
	return self.NewError(err.Error())
}

func validateBasicSpacedName(name string) error {
	if len(name) > 32 { return errors.New("name can't exceed 32 characters") }
	if len(name) == 0 { return errors.New("name can't be empty") }

	for i := 0; i < len(name); i++ {
		if isAZaz09(name[i]) || name[i] == '-' || name[i] == ' ' { continue }
		return errors.New("name contains invalid character")
	}
	
	return nil
}

func isAZaz09(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')
}
