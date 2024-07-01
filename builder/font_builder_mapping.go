package builder

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
func (self *mappingEntry) AppendTo(data []byte, glyphLookup map[uint64]uint16, scratchBuffer []uint64) ([]byte, []uint64, error) {
	var err error
	data = append(data, self.SwitchType)
	
	// single glyph case
	if self.SwitchType == 255 {
		glyphIndex, found := glyphLookup[self.SwitchCases[0].Glyphs[0]]
		if !found { panic(invalidInternalState) }
		return internal.AppendUint16LE(data, glyphIndex), scratchBuffer, nil
	}

	// more involved switch case
	for i, _ := range self.SwitchCases {
		data, scratchBuffer, err = self.SwitchCases[i].AppendTo(data, glyphLookup, scratchBuffer)
		if err != nil { return data, scratchBuffer, err }
		if len(data) > ggfnt.MaxFontDataSize {
			return data, scratchBuffer, errFontDataExceedsMax
		}
	}
	return data, scratchBuffer, nil
}

type mappingGroup struct {
	Glyphs []uint64
	AnimationFlags ggfnt.AnimationFlags
}

func (self *mappingGroup) AppendTo(data []byte, glyphLookup map[uint64]uint16, scratchBuffer []uint64) ([]byte, []uint64, error) {
	if len(self.Glyphs) == 0 || len(self.Glyphs) > 128 { panic(invalidInternalState) }
	
	// append group size
	data = append(data, uint8(len(self.Glyphs)))

	// single glyph case
	if len(self.Glyphs) == 1 {
		glyphIndex, found := glyphLookup[self.Glyphs[0]]
		if !found { panic(invalidInternalState) }
		return internal.AppendUint16LE(data, glyphIndex), scratchBuffer, nil
	}
	
	// --- group cases ---

	// add animations info
	data = append(data, uint8(self.AnimationFlags))

	// get actual glyph indices
	scratchBuffer = internal.SetSliceSize(scratchBuffer, len(self.Glyphs))
	for i := 0; i < len(self.Glyphs); i++ {
		glyphIndex, found := glyphLookup[self.Glyphs[i]]
		if !found { panic(invalidInternalState) }
		scratchBuffer[i] = uint64(glyphIndex)
	}

	// find biggest continuous groups (if any)
	firstGroupStart, firstGroupLen, secondGroupStart, secondGroupLen := findContinuousGroups(scratchBuffer)

	// --- append data for all 5 potential subgroups ---
	// first potential non-consecutive group
	if firstGroupStart != 0 {
		data = append(data, firstGroupStart) // append group size
		for i := uint8(0); i < firstGroupStart; i++ { // append group glyphs
			data = internal.AppendUint16LE(data, uint16(scratchBuffer[i]))
		}
	}

	// first potential consecutive group
	if firstGroupLen > 0 {
		data = append(data, 0b1000_0000 | firstGroupLen)
		data = internal.AppendUint16LE(data, uint16(scratchBuffer[firstGroupStart]))
	}

	// second potential non-consecutive group
	firstGroupEnd := firstGroupStart + firstGroupLen
	if firstGroupEnd < firstGroupStart { panic(invalidInternalState) } // unreasonable overflow
	if secondGroupStart > firstGroupEnd {
		data = append(data, secondGroupStart - firstGroupEnd + 1)
		for i := firstGroupEnd; i < secondGroupStart; i++ {
			data = internal.AppendUint16LE(data, uint16(scratchBuffer[i]))
		}
	}

	// second potential consecutive group
	if secondGroupLen > 0 {
		data = append(data, 0b1000_0000 | secondGroupLen)
		data = internal.AppendUint16LE(data, uint16(scratchBuffer[secondGroupStart]))
	}

	// third potential non-consecutive group
	secondGroupEnd := secondGroupStart + secondGroupLen
	if secondGroupEnd < secondGroupStart { panic(invalidInternalState) } // unreasonable overflow
	if len(self.Glyphs) > int(secondGroupEnd) {
		data = append(data, uint8(len(self.Glyphs)) - secondGroupEnd + 1)
		for i := int(secondGroupEnd); i < len(self.Glyphs); i++ {
			data = internal.AppendUint16LE(data, uint16(scratchBuffer[i]))
		}
	}

	// clean up and return
	scratchBuffer = scratchBuffer[ : 0]
	return data, scratchBuffer, nil
}

// Returns first consecutive group start index, then its length, and then
// second consecutive group start index, and its length.
func findContinuousGroups(glyphIndices []uint64) (uint8, uint8, uint8, uint8) {
	if len(glyphIndices) > 128 { panic(invalidInternalState) }

	var longestGroupStart uint8 = uint8(len(glyphIndices))
	var longestGroupLen uint8
	var secondLongestGroupStart uint8 = uint8(len(glyphIndices))
	var secondLongestGroupLen uint8
	
	var currentGroupActive bool = false
	var currentGroupLen uint8
	var currentGroupStart uint8
	var prevIndex uint64 = 65536 // outside uint16 range
	for i := 0; i < len(glyphIndices); i++ {
		if glyphIndices[i] == prevIndex + 1 {
			if currentGroupActive {
				currentGroupLen += 1
			} else {
				currentGroupActive = true
				currentGroupStart = uint8(i) - 1
				currentGroupLen = 2
			}
		} else {
			if currentGroupActive {
				if currentGroupLen > longestGroupLen {

				} else if currentGroupLen > secondLongestGroupLen {
					secondLongestGroupStart = currentGroupStart
					secondLongestGroupLen = currentGroupLen
				}
				currentGroupActive = false
			}
		}
	}

	if longestGroupStart < secondLongestGroupStart {
		return longestGroupStart, longestGroupLen, secondLongestGroupStart, secondLongestGroupLen
	} else {
		return secondLongestGroupStart, secondLongestGroupLen, longestGroupStart, longestGroupLen
	}
}
