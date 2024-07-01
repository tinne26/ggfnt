package builder

import "errors"
import "slices"

import "github.com/tinne26/ggfnt/internal"

type uidRange struct { First, Last uint64 }
type runeRange struct { First, Last rune }

// --- glyph set ---

// rewrite rule glyph set
type reGlyphSet struct {
	ranges []uidRange
	list []uint64
}

func (self *reGlyphSet) GetSize() uint32 {
	return uint32(2 + (len(self.ranges) << 1) + len(self.ranges) + (len(self.list) << 1))
}

func (self *reGlyphSet) AppendTo(data []byte, glyphLookup map[uint64]uint16) ([]byte, error) {
	if len(self.ranges) >= 255 { panic(invalidInternalState) }
	if len(self.list) >= 255 { panic(invalidInternalState) }

	// ranges
	data = append(data, uint8(len(self.ranges)))
	for _, uidRange := range self.ranges {	
		glyphIndex, found := glyphLookup[uidRange.First]
		if !found { return data, errors.New("rewrite glyph set range contains invalid glyph UID") }
		if uidRange.First == uidRange.Last {
			data = internal.AppendUint16LE(data, glyphIndex)
			data = append(data, 0)
		} else {
			toIndex, found := glyphLookup[uidRange.Last]
			if !found { return data, errors.New("rewrite glyph set range contains invalid glyph UID") }
			if toIndex < glyphIndex {
				return data, errors.New("rewrite glyph set range start and end points are reversed")
			}
			rangeLen := toIndex - glyphIndex
			if rangeLen > 255 {
				return data, errors.New("rewrite glyph set range can't exceed length 255")
			}
			data = internal.AppendUint16LE(data, glyphIndex)
			data = append(data, uint8(rangeLen))
		}
	}

	// list
	data = append(data, uint8(len(self.list)))
	for _, uid := range self.list {	
		glyphIndex, found := glyphLookup[uid]
		if !found { return data, errors.New("rewrite glyph set list contains invalid glyph UID") }
		data = internal.AppendUint16LE(data, glyphIndex)
	}

	return data, nil
}

func (self *Font) CreateGlyphSet() (uint64, error) {
	if len(self.rewriteGlyphSets) >= 255 {
		return 0, errors.New("font can't contain more than 255 glyph sets")
	}
	
	uid, err := internal.CryptoRandUint64()
	if err != nil { return 0, err } // don't think this can ever happen
	_, found := self.rewriteGlyphSets[uid]
	if found { return 0, errors.New("failed to generate unique glyph set UID") } // *
	// * if this is triggered, start playing the lottery more often
	if self.rewriteGlyphSets == nil { self.rewriteGlyphSets = make(map[uint64]reGlyphSet) }
	self.rewriteGlyphSets[uid] = reGlyphSet{}
	self.glyphSetsOrder = append(self.glyphSetsOrder, uid)
	return uid, nil
}

func (self *Font) RemoveGlyphSet(setUID uint64) bool {
	_, found := self.rewriteGlyphSets[setUID]
	if !found { return false }
	delete(self.rewriteGlyphSets, setUID)
	return true
}

func (self *Font) AddGlyphSetRange(setUID, rangeStartGlyphUID, rangeEndGlyphUID uint64) error {
	set, found := self.rewriteGlyphSets[setUID]
	if !found { return errors.New("invalid glyph set UID") }
	if len(set.ranges) >= 255 { return errors.New("glyph set can't contain more than 255 ranges") }
	_, found = self.glyphData[rangeStartGlyphUID]
	if !found { return errors.New("invalid glyph range starting UID") }
	_, found = self.glyphData[rangeEndGlyphUID]
	if !found { return errors.New("invalid glyph range ending UID") }
	for _, glyphRange := range set.ranges {
		if glyphRange.First == rangeStartGlyphUID && glyphRange.Last == rangeEndGlyphUID {
			return errors.New("glyph range already included")
		}
	}
	set.ranges = append(set.ranges, uidRange{ rangeStartGlyphUID, rangeEndGlyphUID })
	self.rewriteGlyphSets[setUID] = set
	return nil
}

func (self *Font) RemoveGlyphSetRange(setUID, rangeStartGlyphUID, rangeEndGlyphUID uint64) bool {
	set, found := self.rewriteGlyphSets[setUID]
	if !found { return false }
	var rangeIndex int = -1
	for index, glyphRange := range set.ranges {
		if glyphRange.First == rangeStartGlyphUID && glyphRange.Last == rangeEndGlyphUID {
			rangeIndex = index
			break
		}
	}
	if rangeIndex == -1 { return false }
	set.ranges = slices.Delete(set.ranges, rangeIndex, rangeIndex)
	self.rewriteGlyphSets[setUID] = set
	return true
}

func (self *Font) EachGlyphSetRange(setUID uint64, each func(start, end uint64)) {
	set, found := self.rewriteGlyphSets[setUID]
	if !found { return }
	for index, _ := range set.ranges {
		each(set.ranges[index].First, set.ranges[index].Last)
	}
}

func (self *Font) AddGlyphSetListGlyph(setUID, glyphUID uint64) error {
	set, found := self.rewriteGlyphSets[setUID]
	if !found { return errors.New("invalid glyph set UID") }
	if len(set.list) >= 255 { return errors.New("glyph set can't contain more than 255 glyphs") }
	_, found = self.glyphData[glyphUID]
	if !found { return errors.New("invalid glyph UID") }
	for _, listGlyphUID := range set.list {
		if listGlyphUID == glyphUID {
			return errors.New("glyph already included")
		}
	}
	set.list = append(set.list, glyphUID)
	self.rewriteGlyphSets[setUID] = set
	return nil
}

func (self *Font) RemoveGlyphSetListGlyph(setUID, glyphUID uint64) bool {
	set, found := self.rewriteGlyphSets[setUID]
	if !found { return false }
	var listIndex int = -1
	for index, listGlyphUID := range set.list {
		if listGlyphUID == glyphUID {
			listIndex = index
			break
		}
	}
	if listIndex == -1 { return false }
	set.list = slices.Delete(set.list, listIndex, listIndex)
	self.rewriteGlyphSets[setUID] = set
	return true
}

func (self *Font) EachGlyphSetListGlyph(setUID uint64, each func(uid uint64)) {
	set, found := self.rewriteGlyphSets[setUID]
	if !found { return }
	for index, _ := range set.list {
		each(set.list[index])
	}
}

// --- rune set ---

// rewrite rule rune set
type reRuneSet struct {
	ranges []runeRange
	list []rune
}

func (self *Font) CreateRuneSet() (uint64, error) {
	if len(self.rewriteRuneSets) >= 255 {
		return 0, errors.New("font can't contain more than 255 rune sets")
	}
	
	uid, err := internal.CryptoRandUint64()
	if err != nil { return 0, err } // don't think this can ever happen
	_, found := self.rewriteGlyphSets[uid]
	if found {
		return 0, errors.New("failed to generate unique rune set UID")
	}
	self.rewriteRuneSets[uid] = reRuneSet{}
	return uid, nil
}

func (self *Font) RemoveRuneSet(setUID uint64) bool {
	_, found := self.rewriteRuneSets[setUID]
	if !found { return false }
	delete(self.rewriteRuneSets, setUID)
	return true
}

func (self *Font) AddRuneSetRange(setUID uint64, rangeStart, rangeEnd rune) error {
	set, found := self.rewriteRuneSets[setUID]
	if !found { return errors.New("invalid rune set UID") }
	if rangeStart > rangeEnd {
		return errors.New("invalid rune range (start > end)")
	}
	for _, runeRange := range set.ranges {
		if runeRange.First == rangeStart && runeRange.Last == rangeEnd {
			return errors.New("rune range already included")
		}
	}
	set.ranges = append(set.ranges, runeRange{ rangeStart, rangeEnd })
	self.rewriteRuneSets[setUID] = set
	return nil
}

func (self *Font) RemoveRuneSetRange(setUID uint64, rangeStart, rangeEnd rune) bool {
	set, found := self.rewriteRuneSets[setUID]
	if !found { return false }
	var rangeIndex int = -1
	for index, runeRange := range set.ranges {
		if runeRange.First == rangeStart && runeRange.Last == rangeEnd {
			rangeIndex = index
			break
		}
	}
	if rangeIndex == -1 { return false }
	set.ranges = slices.Delete(set.ranges, rangeIndex, rangeIndex)
	self.rewriteRuneSets[setUID] = set
	return true
}

func (self *Font) EachRuneSetRange(setUID uint64, each func(start, end rune)) {
	set, found := self.rewriteRuneSets[setUID]
	if !found { return }
	for index, _ := range set.ranges {
		each(set.ranges[index].First, set.ranges[index].Last)
	}
}

func (self *Font) AddRuneSetListRune(setUID uint64, codePoint rune) error {
	set, found := self.rewriteRuneSets[setUID]
	if !found { return errors.New("invalid rune set UID") }
	for _, listGlyphUID := range set.list {
		if listGlyphUID == codePoint {
			return errors.New("rune already included")
		}
	}
	set.list = append(set.list, codePoint)
	self.rewriteRuneSets[setUID] = set
	return nil
}

func (self *Font) RemoveRuneSetListRune(setUID uint64, codePoint rune) bool {
	set, found := self.rewriteRuneSets[setUID]
	if !found { return false }
	var listIndex int = -1
	for index, listRune := range set.list {
		if listRune == codePoint {
			listIndex = index
			break
		}
	}
	if listIndex == -1 { return false }
	set.list = slices.Delete(set.list, listIndex, listIndex)
	self.rewriteRuneSets[setUID] = set
	return true
}

func (self *Font) EachRuneSetListRune(setUID uint64, each func(codePoint rune)) {
	set, found := self.rewriteRuneSets[setUID]
	if !found { return }
	for index, _ := range set.list {
		each(set.list[index])
	}
}

