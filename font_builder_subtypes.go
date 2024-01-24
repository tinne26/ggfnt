package ggfnt

import "errors"
import "image"

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
	Bounds GlyphBounds
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
	Indices []uint16 // glyph indices
}

func (self *variableEntry) appendValuesTo(buffer []byte) []byte {
	return append(buffer, self.DefaultValue, self.MinValue, self.MaxValue)
}

type mappingMode struct {
	Name string
	Program []uint8
}

type fastMappingTable struct {
	Condition [3]uint8
	StartCodePoint rune // inclusive
	EndCodePoint   rune // exclusive
	CodePointModes []uint8
	CodePointMainIndices []uint64
	CodePointModeIndices [][]uint64
	// notice: ^ we use as many mode indices as table length, leaving empty if necessary.
	//           tables are not that big and this makes general edition easier
}

func (self *fastMappingTable) tableLen() int {
	return int(self.EndCodePoint) - int(self.StartCodePoint)
}

func (self *fastMappingTable) appendTo(data []byte, glyphIndexLookup map[uint64]uint16) ([]byte, error) {
	data = append(data, self.Condition[0 : 3]...)
	if self.EndCodePoint <= self.StartCodePoint { panic(invalidInternalState) }
	data = appendUint32LE(data, uint32(self.StartCodePoint))
	data = appendUint32LE(data, uint32(self.EndCodePoint))
	tableLen := self.tableLen()
	if tableLen > maxFastMappingTableCodePoints { panic(invalidInternalState) }

	if len(self.CodePointModes) != tableLen { panic(invalidInternalState) }
	data = append(data, self.CodePointModes...) // CodePointModes
	if len(self.CodePointMainIndices) != tableLen { panic(invalidInternalState) }
	mainIndicesOffset := len(data)
	for i, uid := range self.CodePointMainIndices { // CodePointMainIndices
		// note: some additional checks could be done here, but it gets messy quick
		if self.CodePointModes[i] == 255 {
			glyphIndex, found := glyphIndexLookup[uid]
			if !found { panic(invalidInternalState) }
			data = appendUint16LE(data, glyphIndex)
		} else {
			data = append(data, 0, 0)
		}
	}

	var offset uint16
	for i, _ := range self.CodePointMainIndices {
		if self.CodePointModes[i] == 255 {
			if len(self.CodePointModeIndices[i]) != 0 { panic(invalidInternalState) }
			continue
		}
		
		if len(self.CodePointModeIndices[i]) > maxGlyphsPerCodePoint {
			panic(invalidInternalState)
		}
		if len(self.CodePointModeIndices[i]) < 2 {
			cp := string((self.StartCodePoint + rune(i)))
			return nil, errors.New("code point \"" + cp + "\" must have at least two indices for its mapping mode")
		}

		// append indices to data
		for _, uid := range self.CodePointModeIndices[i] {
			glyphIndex, found := glyphIndexLookup[uid]
			if !found { panic(invalidInternalState) }
			data = appendUint16LE(data, glyphIndex)
		}

		// set CodePointMainIndices retroactively
		offset += uint16(len(self.CodePointModeIndices[i])) // TODO: am I sure this can't overflow?
		retroIndex := mainIndicesOffset + i*2
		encodeUint16LE(data[retroIndex : retroIndex + 2], offset)
	}

	return data, nil
}
