package builder

import "errors"

import "github.com/tinne26/ggfnt/internal"

// --- glyph rewrite rules ---

type glyphRewriteRule struct {
	condition uint8 // (255 if none)
	headLen uint8
	bodyLen uint8
	tailLen uint8
	inElemsAreGroups boolList // if true, they are groups, otherwise, they are single values
	inGlyphs []uint64
	inGroups []uint64
	output []uint64 // glyph UIDs (255 at most)
}

func (self *glyphRewriteRule) AppendTo(data []byte, groupLookup map[uint64]uint8, glyphLookup map[uint64]uint16) ([]byte, error) {
	// safety assertions
	if len(self.output) > 255 { panic(invalidInternalState) }
	if self.bodyLen == 0 { panic(invalidInternalState) }
	if len(self.inGlyphs) + len(self.inGroups) != int(self.headLen) + int(self.bodyLen) + int(self.tailLen) {
		panic(invalidInternalState)
	}

	// append rewrite rule header info
	data = append(data, self.condition, self.headLen, self.bodyLen, self.tailLen, uint8(len(self.output)))
	
	// append rewrite rule output
	for _, glyphUID := range self.output {
		glyphIndex, found := glyphLookup[glyphUID]
		if !found { return data, errors.New("glyph rewrite rule output sequence is using an undefined glyph") }
		data = internal.AppendUint16LE(data, glyphIndex)
	}

	// prepare for rewrite rule input
	var inIndex int = 0
	var groupsIndex, listIndex int = 0, 0
	var accumulatedGroups, accumulatedElements int

	// append rewrite rule head, body and tail, fragment by fragment
	var err error
	for _, blockSize := range [3]uint8{self.headLen, self.bodyLen, self.tailLen} {
		// for each element in the block, consider the fragment append
		// (notice that we have the 'appendBlockFragment' basic idea repeated thrice)
		var blockHasFragment bool = false
		for i := uint8(0); i < blockSize; i++ {
			isGroup := self.inElemsAreGroups.Get(inIndex)
			if isGroup {
				if accumulatedElements != 0 || accumulatedGroups == 15 {
					blockHasFragment = true
					groups := self.inGroups[groupsIndex : groupsIndex + accumulatedGroups]
					elems  := self.inGlyphs[listIndex : listIndex + accumulatedElements]
					data, err = self.appendAccumulatedInBlock(data, groups, elems, groupLookup, glyphLookup)
					if err != nil { return data, err }
					groupsIndex += accumulatedGroups
					listIndex += accumulatedElements
					accumulatedGroups, accumulatedElements = 0, 0
				}
				accumulatedGroups += 1
			} else {
				if accumulatedElements == 15 {
					blockHasFragment = true
					groups := self.inGroups[groupsIndex : groupsIndex + accumulatedGroups]
					elems  := self.inGlyphs[listIndex : listIndex + accumulatedElements]
					data, err = self.appendAccumulatedInBlock(data, groups, elems, groupLookup, glyphLookup)
					if err != nil { return data, err }
					groupsIndex += accumulatedGroups
					listIndex += accumulatedElements
					accumulatedGroups, accumulatedElements = 0, 0
				}
				accumulatedElements += 1
			}
			inIndex += 1
		}
		
		// block-closing append
		if accumulatedGroups > 0 || accumulatedElements > 0 {
			groups := self.inGroups[groupsIndex : groupsIndex + accumulatedGroups]
			elems  := self.inGlyphs[listIndex : listIndex + accumulatedElements]
			data, err = self.appendAccumulatedInBlock(data, groups, elems, groupLookup, glyphLookup)
			if err != nil { return data, err }
			groupsIndex += accumulatedGroups
			listIndex += accumulatedElements
			accumulatedGroups, accumulatedElements = 0, 0
		} else if !blockHasFragment {
			data = append(data, 0)
		}
	}

	return data, nil
}

func (self *glyphRewriteRule) appendAccumulatedInBlock(data []byte, groups, elements []uint64, groupLookup map[uint64]uint8, glyphLookup map[uint64]uint16) ([]byte, error) {
	if groups == nil && elements == nil { return data, nil }
	data = append(data, (uint8(len(groups)) << 4) | uint8(len(elements)))
	for _, groupUID := range groups {
		group, found := groupLookup[groupUID]
		if !found { return data, errors.New("glyph rewrite rule input sequence is using an undefined glyph group") }
		data = append(data, group)
	}
	for _, glyphUID := range elements {
		glyphIndex, found := glyphLookup[glyphUID]
		if !found { return data, errors.New("glyph rewrite rule input sequence is using an undefined glyph") }
		data = internal.AppendUint16LE(data, glyphIndex)
	}
	return data, nil
}

// --- utf8 rewrite rules ---

type utf8RewriteRule struct {
	condition uint8 // (255 if none)
	headLen uint8
	bodyLen uint8
	tailLen uint8
	inElemsAreGroups boolList
	inRunes []rune
	inGroups []uint64
	output []rune
}

func (self *utf8RewriteRule) AppendTo(data []byte, groupLookup map[uint64]uint8) ([]byte, error) {
	// safety assertions
	if len(self.output) > 255 { panic(invalidInternalState) }
	if self.bodyLen == 0 { panic(invalidInternalState) }
	if len(self.inRunes) + len(self.inGroups) != int(self.headLen) + int(self.bodyLen) + int(self.tailLen) {
		panic(invalidInternalState)
	}

	// append rewrite rule header info
	data = append(data, self.condition, self.headLen, self.bodyLen, self.tailLen, uint8(len(self.output)))
	
	// append rewrite rule output
	for _, codePoint := range self.output {
		data = internal.AppendUint32LE(data, uint32(codePoint))
	}

	// prepare for rewrite rule input
	var inIndex int = 0
	var groupsIndex, listIndex int = 0, 0
	var accumulatedGroups, accumulatedElements int

	// append rewrite rule head, body and tail, fragment by fragment
	var err error
	for _, blockSize := range [3]uint8{self.headLen, self.bodyLen, self.tailLen} {
		// for each element in the block, consider the fragment append
		// (notice that we have the 'appendBlockFragment' basic idea repeated thrice)
		var blockHasFragment bool = false
		for i := uint8(0); i < blockSize; i++ {
			isGroup := self.inElemsAreGroups.Get(inIndex)
			if isGroup {
				if accumulatedElements != 0 || accumulatedGroups == 15 {
					blockHasFragment = true
					groups := self.inGroups[groupsIndex : groupsIndex + accumulatedGroups]
					elems  := self.inRunes[listIndex : listIndex + accumulatedElements]
					data, err = self.appendAccumulatedInBlock(data, groups, elems, groupLookup)
					if err != nil { return data, err }
					groupsIndex += accumulatedGroups
					listIndex += accumulatedElements
					accumulatedGroups, accumulatedElements = 0, 0
				}
				accumulatedGroups += 1
			} else {
				if accumulatedElements == 15 {
					blockHasFragment = true
					groups := self.inGroups[groupsIndex : groupsIndex + accumulatedGroups]
					elems  := self.inRunes[listIndex : listIndex + accumulatedElements]
					data, err = self.appendAccumulatedInBlock(data, groups, elems, groupLookup)
					if err != nil { return data, err }
					groupsIndex += accumulatedGroups
					listIndex += accumulatedElements
					accumulatedGroups, accumulatedElements = 0, 0
				}
				accumulatedElements += 1
			}
			inIndex += 1
		}
		
		// block-closing append
		if accumulatedGroups > 0 || accumulatedElements > 0 {
			groups := self.inGroups[groupsIndex : groupsIndex + accumulatedGroups]
			elems  := self.inRunes[listIndex : listIndex + accumulatedElements]
			data, err = self.appendAccumulatedInBlock(data, groups, elems, groupLookup)
			if err != nil { return data, err }
			groupsIndex += accumulatedGroups
			listIndex += accumulatedElements
			accumulatedGroups, accumulatedElements = 0, 0
		} else if !blockHasFragment {
			data = append(data, 0)
		}
	}

	return data, nil
}

func (self *utf8RewriteRule) appendAccumulatedInBlock(data []byte, groups []uint64, elements []rune, groupLookup map[uint64]uint8) ([]byte, error) {
	if groups == nil && elements == nil { return data, nil }
	data = append(data, (uint8(len(groups)) << 4) | uint8(len(elements)))
	for _, groupUID := range groups {
		group, found := groupLookup[groupUID]
		if !found { return data, errors.New("utf8 rewrite rule input sequence is using an undefined group") }
		data = append(data, group)
	}
	for _, codePoint := range elements {
		data = internal.AppendUint32LE(data, uint32(codePoint))
	}
	return data, nil
}

// --- public API ---

func (self *Font) AddSimpleUtf8RewriteRule(replacement rune, sequence ...rune) error {
	if len(sequence) == 0 { return errors.New("rewrite rule sequence can't be empty") }
	if len(sequence) > 255 { return errors.New("rewrite rule sequence can't exceed 255 runes") }
	
	rule := utf8RewriteRule{ condition: 255, bodyLen: uint8(len(sequence)), inRunes: sequence, output: []rune{replacement} }
	for i := 0; i < len(sequence); i++ {
		rule.inElemsAreGroups.Push(false)
	}
	self.utf8Rules = append(self.utf8Rules, rule)
	return nil
}

// The 'any' values can be only runes or uint64 ids for groups.
func (self *Font) AddUtf8RewriteRule(headLen, bodyLen, tailLen uint8, input []any, sequence ...rune) error {
	panic("unimplemented")
}

// Ids can be for glyphs or glyph sets, we assume they won't collide.
func (self *Font) AddGlyphRewriteRule(headLen, bodyLen, tailLen uint8, input []uint64, out ...uint64) error {
	// validate input sizes
	if bodyLen == 0 { errors.New("rewrite rule exceeds input body must have len >= 1") }
	headBodyLen := headLen + bodyLen
	if headBodyLen < bodyLen || headBodyLen + tailLen < tailLen {
		return errors.New("rewrite rule exceeds 255 input elements")
	}
	if int(headBodyLen + tailLen) != len(input) {
		return errors.New("rewrite rule input block lengths don't match given input")
	}
	
	// validate output glyphs
	for _, glyphUID := range out {
		_, found := self.glyphData[glyphUID]
		if !found { return errors.New("invalid rewrite rule output glyph UID") }
	}

	// create rule
	rule := glyphRewriteRule{ condition: 255 }

	// create in glyphs and groups
	var inGlyphs []uint64
	var inGroups []uint64
	for _, in := range input {
		_, isGlyph := self.glyphData[in]
		if isGlyph {
			rule.inElemsAreGroups.Push(false)
			inGlyphs = append(inGlyphs, in)
		} else {
			_, isGroup := self.rewriteGlyphSets[in]
			if !isGroup {
				return errors.New("given input UID is neither glyph nor set")
			}
			rule.inElemsAreGroups.Push(true)
			inGroups = append(inGroups, in)
		}
	}
	
	rule.headLen = headLen
	rule.bodyLen = bodyLen
	rule.tailLen = tailLen
	rule.inGlyphs = inGlyphs
	rule.inGroups = inGroups
	rule.output = out
	self.glyphRules = append(self.glyphRules, rule)
	return nil
}
