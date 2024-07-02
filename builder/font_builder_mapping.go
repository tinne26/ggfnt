package builder

import "slices"

import "github.com/tinne26/ggfnt"
import "github.com/tinne26/ggfnt/internal"

type mappingSwitchEntry struct {
	Settings []uint8
}

type mappingEntry struct {
	SwitchType uint8
	SwitchCases []mappingGroup
}

// Before calling this, the caller should cross check switch type with 
// the number of SwitchCases.
func (self *mappingEntry) AppendTo(data []byte, glyphLookup map[uint64]uint16, scratchBuffer []uint16) ([]byte, []uint16, error) {
	if len(scratchBuffer) != 0 { panic(brokenCode) }
	
	var err error
	data = append(data, self.SwitchType)
	
	// single glyph case
	if self.SwitchType == 255 {
		if len(self.SwitchCases) != 1 { panic(invalidInternalState) }
		if len(self.SwitchCases[0].Glyphs) != 1 { panic(invalidInternalState) }
		glyphIndex, found := glyphLookup[self.SwitchCases[0].Glyphs[0]]
		if !found { panic(invalidInternalState) }
		return internal.AppendUint16LE(data, glyphIndex), scratchBuffer, nil
	} else if self.SwitchType == 254 {
		if len(self.SwitchCases) != 1 { panic(invalidInternalState) }
		if len(self.SwitchCases[0].Glyphs) <= 1 { panic(invalidInternalState) }
		return self.SwitchCases[0].AppendTo(data, glyphLookup, scratchBuffer)
	}

	// more involved switch case
	for i, _ := range self.SwitchCases {
		data, scratchBuffer, err = self.SwitchCases[i].AppendTo(data, glyphLookup, scratchBuffer)
		if err != nil { return data, scratchBuffer[ : 0], err }
		if len(data) > ggfnt.MaxFontDataSize {
			return data, scratchBuffer[ : 0], errFontDataExceedsMax
		}
	}
	return data, scratchBuffer[ : 0], nil
}

type mappingGroup struct {
	Glyphs []uint64
	AnimationFlags ggfnt.AnimationFlags
}

func (self *mappingGroup) AppendTo(data []byte, glyphLookup map[uint64]uint16, scratchBuffer []uint16) ([]byte, []uint16, error) {
	if len(self.Glyphs) == 0 || len(self.Glyphs) > 128 { panic(invalidInternalState) }
	
	// single glyph case
	if len(self.Glyphs) == 1 {
		glyphIndex, found := glyphLookup[self.Glyphs[0]]
		if !found { panic(invalidInternalState) }
		return internal.AppendUint16LE(data, glyphIndex), scratchBuffer, nil
	}

	// get actual glyph indices
	scratchBuffer = internal.SetSliceSize(scratchBuffer, len(self.Glyphs))
	for i := 0; i < len(self.Glyphs); i++ {
		glyphIndex, found := glyphLookup[self.Glyphs[i]]
		if !found { panic(invalidInternalState) }
		scratchBuffer[i] = glyphIndex
	}

	// sort scratch buffer, makes it easier to see if glyphs are consecutive
	slices.Sort(scratchBuffer)
	if isContinuousSlice(scratchBuffer) {
		data = append(data, 0b1000_0000 | uint8(len(self.Glyphs) - 1))
		data = append(data, uint8(self.AnimationFlags))
		data = internal.AppendUint16LE(data, uint16(scratchBuffer[0]))
	} else {
		data = append(data, uint8(len(self.Glyphs) - 1))
		data = append(data, uint8(self.AnimationFlags))
		for _, glyphIndex := range scratchBuffer {
			data = internal.AppendUint16LE(data, uint16(glyphIndex))
		}
	}

	return data, scratchBuffer[ : 0], nil
}
