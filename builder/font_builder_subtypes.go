package builder

import "errors"
import "image"
import "regexp"

import "github.com/tinne26/ggfnt"
import "github.com/tinne26/ggfnt/internal"

type editionCategory struct {
	Name string
	Size uint16
}

type editionKerningPair struct {
	First uint64
	Second uint64
	Class uint16 // if 0, value must be used instead
	Value int8
}
func (self *editionKerningPair) HasClass() bool {
	return self.Class != 0
}

type editionKerningClass struct {
	Name string
	Value int8
}

type glyphData struct {
	Name string // can be empty
	Placement ggfnt.GlyphPlacement
	Mask *image.Alpha
}

type variableEntry struct {
	Name string
	DefaultValue uint8
	MinValue uint8
	MaxValue uint8
}

type codePointMapping struct {
	Mode uint8
	Glyphs []uint64 // glyph UIDs
}

func (self *variableEntry) appendValuesTo(buffer []byte) []byte {
	return append(buffer, self.DefaultValue, self.MinValue, self.MaxValue)
}

type mappingMode struct {
	Name string
	Program []uint8
}

type FastMappingTable struct {
	builder *Font // back reference
	condition [3]uint8
	startCodePoint rune // inclusive
	endCodePoint   rune // exclusive
	codePointModes []uint8 // if 0, no mode is used
	codePointMainIndices []uint64
	codePointModeIndices [][]uint64
	// notice: ^ we use as many mode indices as table length, leaving empty if necessary.
	//           tables are not that big and this makes general edition easier
	numCodePointModeIndices int
}

func newFastMappingTable(start, end rune) (*FastMappingTable, error) {
	var table FastMappingTable
	err := table.validateRange(start, end)
	table.startCodePoint = start
	table.endCodePoint   = end
	if err != nil { return nil, err }
	size := int((end + 1) - start)
	table.codePointModes = make([]uint8, size)
	for i := 0; i < len(table.codePointModes); i++ {
		table.codePointModes[i] = 255
	}
	table.codePointMainIndices = make([]uint64, size)
	for i := 0; i < size; i++ {
		table.codePointMainIndices[i] = uint64(ggfnt.GlyphMissing)
	}
	table.codePointModeIndices = make([][]uint64, size)
	table.recomputeNumCodePointModeIndices()
	table.condition[0] = internal.MappingConditionArg1Const | internal.MappingConditionArg2Const
	return &table, nil
}

func (self *FastMappingTable) recomputeNumCodePointModeIndices() {
	self.numCodePointModeIndices = 0
	for i, _ := range self.codePointModeIndices {
		codePointNumIndices := len(self.codePointModeIndices[i])
		if self.codePointModes[i] == 255 {
			if codePointNumIndices != 0 {
				panic(invalidInternalState)
			}
		} else if codePointNumIndices < 2 {
			panic(invalidInternalState)
		} else {
			self.numCodePointModeIndices += codePointNumIndices
		}
	}
}

func (self *FastMappingTable) dataSize() int {
	tableLen := self.tableLen()
	return 11 + tableLen + (tableLen << 1) + self.numCodePointModeIndices*2
}

func (self *FastMappingTable) validateRange(start, end rune) error {
	// start point validation
	if start < ' ' { return errors.New("mapping range can't start before ' ' (space)") }
	if end < start { return errors.New("invalid range") }

	// size validation
	newSize := (end + 1) - start
	if newSize > internal.MaxFastMappingTableCodePoints {
		return errors.New("fast mapping table can't be so big")
	}
	return nil
}

// Both start and end are included.
func (self *FastMappingTable) GetRange() (start, end rune) {
	return self.startCodePoint, self.endCodePoint
}

// Reset the fast mapping table data so you can safely set a new range
// and fill again.
func (self *FastMappingTable) ClearIndices() {
	for i, _ := range self.codePointMainIndices {
		self.codePointMainIndices[i] = uint64(ggfnt.GlyphMissing)
	}
	for i, _ := range self.codePointModeIndices {
		self.codePointModeIndices[i] = self.codePointModeIndices[i][ : 0]
	}
}

func (self *FastMappingTable) Map(codePoint rune, glyphUID uint64) error {
	if codePoint < self.startCodePoint || codePoint > self.endCodePoint {
		return errors.New("code point out of range")
	}
	_, hasData := self.builder.glyphData[glyphUID]
	if !hasData {
		return errors.New("attempted to map '" + string(codePoint) + "' to an undefined glyph")
	}

	index := codePoint - self.startCodePoint
	if self.codePointModes[index] != 255 {
		assignedIndices := len(self.codePointModeIndices[index])
		if assignedIndices < 2 { panic(invalidInternalState) }
		self.codePointModeIndices[index] = self.codePointModeIndices[index][ : 0] // clear
		self.numCodePointModeIndices -= assignedIndices
	}
	self.codePointModes[index] = 255 // set mapping in direct mode
	self.codePointMainIndices[index] = glyphUID
	return nil
}

func (self *FastMappingTable) MapWithMode(codePoint rune, mode uint8, modeGlyphUIDs ...uint64) error {
	// all error checks
	if codePoint < self.startCodePoint || codePoint > self.endCodePoint {
		return errors.New("code point out of range")
	}
	if len(modeGlyphUIDs) < 2 {
		return errors.New("custom mode mapping requires mapping at least 2 glyphs")
	}
	if int(mode) >= len(self.builder.mappingModes) {
		return errors.New("attempted to use undefined custom mode mapping")
	}
	for _, modeGlyphUID := range modeGlyphUIDs {
		_, hasData := self.builder.glyphData[modeGlyphUID]
		if !hasData {
			return errors.New("attempted to map '" + string(codePoint) + "' to an undefined glyph")
		}
	}

	// TODO: I haven't actually checked the size excess
	
	// everything should be smooth now
	index := codePoint - self.startCodePoint
	if self.codePointModes[index] != 255 {
		assignedIndices := len(self.codePointModeIndices[index])
		if assignedIndices < 2 { panic(invalidInternalState) }
		self.codePointModeIndices[index] = self.codePointModeIndices[index][ : 0] // clear
		self.numCodePointModeIndices -= assignedIndices
	}
	
	self.codePointModes[index] = mode
	self.codePointMainIndices[index] = 0 // clear
	if len(self.codePointModeIndices[index]) != 0 {
		panic(invalidInternalState)
	}
	for _, modeGlyphUID := range modeGlyphUIDs {
		self.codePointModeIndices[index] = append(self.codePointModeIndices[index], modeGlyphUID)
		self.numCodePointModeIndices += 1
	}

	return nil
}

var fastMapTableConditionRegexp = regexp.MustCompile(
	`(VAR\[\d+\]|RAND\(\d+\)|\d+) ?(==|!=|<|<=) ?(VAR\[\d+\]|RAND\(\d+\)|\d+)`,
)

// Uses a very similar condition format as mapmode.Compile():
//  - (VAR[ID] | RAND(VALUE) | {CONST}) (==/!=/</<=) (VAR[ID] | RAND(VALUE) | {CONST})
// 
// Some examples:
//   _ = table.SetCondition("0 == 0") // always true, this is the default condition
//   _ = table.SetCondition("VAR[2] == 1")
//   _ = table.SetCondition("RAND(100) < 66")
//   _ = table.SetCondition("VAR[0] != RAND(1)")
//
// Notice that ids, values and constants must be non-negative values not exceeding 255.
func (self *FastMappingTable) SetCondition(condition string) error {
	panic("unimplemented")
}

// Returns the current condition as a string. This method computes and allocates a
// new string each time it's called, so make sure to cache the result if necessary.
func (self *FastMappingTable) GetCondition() string {
	panic("unimplemented")
}

// Both start and end are included.
func (self *FastMappingTable) SetRange(start, end rune) error {
	// trivial case
	if start == self.startCodePoint && end == self.endCodePoint { return nil }

	// range validation
	err := self.validateRange(start, end)
	if err != nil { return err }
	newSize := int((end + 1) - start)
	
	// check that new start doesn't overlap existing data
	if start > self.startCodePoint {
		for i := self.startCodePoint; i < min(self.endCodePoint, start); i++ {
			if self.codePointModes[i] != 0 {
				return errors.New("can't modify mapping range: rune '" + string(i) + "' is using a custom mode")
			}
			if self.codePointMainIndices[i] != 0 {
				return errors.New("can't modify mapping range: rune '" + string(i) + "' is already assigned")
			}
		}
	}

	// check that new end doesn't overlap existing data
	if end < self.endCodePoint {
		for i := max(self.startCodePoint, end); i < self.endCodePoint; i++ {
			if self.codePointModes[i] != 0 {
				return errors.New("can't modify mapping range: rune '" + string(i) + "' is using a custom mode")
			}
			if self.codePointMainIndices[i] != 0 {
				return errors.New("can't modify mapping range: rune '" + string(i) + "' is already assigned")
			}
		}
	}

	// ensure that we have enough capacity on the slices
	indexingShift := int(start - self.startCodePoint)
	tableLen := self.tableLen()
	if newSize > tableLen {
		internal.GrowSliceByN(self.codePointMainIndices, newSize - tableLen)
		internal.GrowSliceByN(self.codePointModeIndices, newSize - tableLen)
		internal.GrowSliceByN(self.codePointModes, newSize - tableLen)
	}
	
	if indexingShift > 0 { // moving range right (data is moved left)
		// notice: you kinda have to draw this to understand it
		relevantLen := min(tableLen, newSize) - indexingShift
		if relevantLen > 0 {
			// TODO: code point modes should be adapted too, no?
			copy(
				self.codePointMainIndices[ : relevantLen],
				self.codePointMainIndices[indexingShift : indexingShift + relevantLen],
			)
			for i := relevantLen; i < newSize; i++ {
				self.codePointMainIndices[i] = uint64(ggfnt.GlyphMissing)
			}
			copy(
				self.codePointModeIndices[ : relevantLen],
				self.codePointModeIndices[indexingShift : indexingShift + relevantLen],
			)
			copy(
				self.codePointModes[ : relevantLen],
				self.codePointModes[indexingShift : indexingShift + relevantLen],
			)
		}
		
	} else if indexingShift < 0 { // moving range left (data is moved right)
		relevantLen := min(tableLen, newSize) + indexingShift
		if relevantLen > 0 {
			copy(
				self.codePointMainIndices[-indexingShift : -indexingShift + relevantLen],
				self.codePointMainIndices[ : relevantLen],
			)
			for i := 0; i < -indexingShift; i++ {
				self.codePointMainIndices[i] = uint64(ggfnt.GlyphMissing)
			}
			copy(
				self.codePointModeIndices[-indexingShift : -indexingShift + relevantLen],
				self.codePointModeIndices[ : relevantLen],
			)
			copy(
				self.codePointModes[-indexingShift : -indexingShift + relevantLen],
				self.codePointModes[ : relevantLen],
			)
		}
	} else { // this can happen if we are only extending the end point
		for i := tableLen; i < newSize; i++ {
			self.codePointMainIndices[i] = uint64(ggfnt.GlyphMissing)
		}
		
	}
	self.startCodePoint, self.endCodePoint = start, end

	// remove unused space
	if newSize < tableLen {
		self.codePointMainIndices = self.codePointMainIndices[ : newSize]
		self.codePointModeIndices = self.codePointModeIndices[ : newSize]
		self.codePointModes = self.codePointModes[ : newSize]
	}

	// some final sanity checks
	if len(self.codePointMainIndices) != newSize || len(self.codePointModeIndices) != newSize || len(self.codePointModes) != newSize {
		panic("broken code")
	}

	return nil
}

func (self *FastMappingTable) tableLen() int {
	return (int(self.endCodePoint) + 1) - int(self.startCodePoint)
}

func (self *FastMappingTable) appendTo(data []byte, glyphIndexLookup map[uint64]uint16) ([]byte, error) {
	data = append(data, self.condition[0 : 3]...)
	if self.endCodePoint < self.startCodePoint { panic(invalidInternalState) }
	data = internal.AppendUint32LE(data, uint32(self.startCodePoint))
	data = internal.AppendUint32LE(data, uint32(self.endCodePoint))
	tableLen := self.tableLen()
	if tableLen > internal.MaxFastMappingTableCodePoints { panic(invalidInternalState) }

	if len(self.codePointModes) != tableLen { panic(invalidInternalState) }
	data = append(data, self.codePointModes...) // CodePointModes
	if len(self.codePointMainIndices) != tableLen { panic(invalidInternalState) }
	mainIndicesOffset := len(data)
	for i, uid := range self.codePointMainIndices { // CodePointMainIndices
		// note: some additional checks could be done here, but it gets messy quick
		if self.codePointModes[i] == 255 {
			if uid == uint64(ggfnt.GlyphMissing) {
				data = internal.AppendUint16LE(data, uint16(ggfnt.GlyphMissing))
			} else {
				glyphIndex, found := glyphIndexLookup[uid]
				if !found { panic(invalidInternalState) }
				data = internal.AppendUint16LE(data, glyphIndex)
			}
		} else {
			data = append(data, 0, 0)
		}
	}

	var offset uint16
	for i, _ := range self.codePointMainIndices {
		if self.codePointModes[i] == 255 {
			if len(self.codePointModeIndices[i]) != 0 { panic(invalidInternalState) }
			continue
		}
		
		if len(self.codePointModeIndices[i]) > internal.MaxGlyphsPerCodePoint {
			panic(invalidInternalState)
		}
		if len(self.codePointModeIndices[i]) < 2 {
			cp := string((self.startCodePoint + rune(i)))
			return nil, errors.New("code point \"" + cp + "\" must have at least two indices for its mapping mode")
		}

		// append indices to data
		for _, uid := range self.codePointModeIndices[i] {
			glyphIndex, found := glyphIndexLookup[uid]
			if !found { panic(invalidInternalState) }
			data = internal.AppendUint16LE(data, glyphIndex)
		}

		// set CodePointMainIndices retroactively
		offset += uint16(len(self.codePointModeIndices[i])) // TODO: am I sure this can't overflow?
		retroIndex := mainIndicesOffset + i*2
		internal.EncodeUint16LE(data[retroIndex : retroIndex + 2], offset)
	}

	return data, nil
}
