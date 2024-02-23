package builder

import "errors"
import "image"
import "regexp"
import "strconv"
import "unsafe"

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

type settingEntry struct {
	Name string
	Values []uint8
}

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

type glyphRewriteRule struct {
	condition uint8 // (255 if none)
	replacement uint64 // glyph index
	sequence []uint64 // glyph UIDs (255 at most)
}

func (self *glyphRewriteRule) AppendTo(data []byte, glyphLookup map[uint64]uint16) []byte {
	data = append(data, self.condition, uint8(len(self.sequence)))
	glyphIndex, found := glyphLookup[self.replacement]
	if !found { panic(invalidInternalState) }
	data = internal.AppendUint16LE(data, glyphIndex)
	for _, glyphUID := range self.sequence {
		glyphIndex, found = glyphLookup[glyphUID]
		if !found { panic(invalidInternalState) }
		data = internal.AppendUint16LE(data, glyphIndex)
	}
	return data
}

type utf8RewriteRule struct {
	condition uint8
	replacement rune
	sequence []rune
}
func (self *utf8RewriteRule) AppendTo(data []byte) []byte {
	data = append(data, self.condition, uint8(len(self.sequence)))
	data = internal.AppendUint32LE(data, uint32(self.replacement))
	for _, codePoint := range self.sequence {
		data = internal.AppendUint32LE(data, uint32(codePoint))
	}
	return data
}

type rewriteCondition struct {
	EditorName string
	data []uint8
}

// Returns a human-friendly string representation of the rewrite condition.
// This takes a bit of time to create, but it's not crazy expensive or anything,
// so if you are only displaying one rewrite condition it's fine to call it
// every frame.
func (self *rewriteCondition) String() string {
	var str []byte = make([]byte, 0, 32)

	var index int
	switch self.data[index] >> 5 {
	case 0b000: // OR group
		numTerms := self.data[index] & 0b0001_1111
		if numTerms < 2 { panic(invalidInternalState) }
		index += 1
		for i := uint8(0); i < numTerms; i++ {
			index, str = self.appendSubTerm(index, str)
			if i != numTerms - 1 {
				str = append(str, []byte{' ', 'O', 'R', ' '}...)
			}
		}
	case 0b001: // AND group
		numTerms := self.data[index] & 0b0001_1111
		if numTerms < 2 { panic(invalidInternalState) }
		index += 1
		for i := uint8(0); i < numTerms; i++ {
			index, str = self.appendSubTerm(index, str)
			if i != numTerms - 1 {
				str = append(str, []byte{' ', 'A', 'N', 'D', ' '}...)
			}
		}
	default:
		index, str = self.appendSubTerm(index, str)
	}
	
	if index != len(self.data) { panic(invalidInternalState) }
	return unsafe.String(&str[0], len(str))
}

// Returns the next index (can be at most len(self.data)),
// and the str with the new term appended.
func (self *rewriteCondition) appendSubTerm(index int, str []byte) (int, []byte) {
	if index >= len(self.data) { panic(invalidInternalState) }

	switch self.data[index] >> 5 {
	case 0b000: // OR group
		numTerms := self.data[index] & 0b0001_1111
		if numTerms < 2 { panic(invalidInternalState) }
		index += 1
		str = append(str, '(')
		for i := uint8(0); i < numTerms; i++ {
			index, str = self.appendSubTerm(index, str)
			if i == numTerms - 1 {
				str = append(str, ')')
			} else {
				str = append(str, []byte{' ', 'O', 'R', ' '}...)
			}
		}
	case 0b001: // AND group
		numTerms := self.data[index] & 0b0001_1111
		if numTerms < 2 { panic(invalidInternalState) }
		index += 1
		str = append(str, '(')
		for i := uint8(0); i < numTerms; i++ {
			index, str = self.appendSubTerm(index, str)
			if i == numTerms - 1 {
				str = append(str, ')')
			} else {
				str = append(str, []byte{' ', 'A', 'N', 'D', ' '}...)
			}
		}
	case 0b010: // comparison
		// append first operand (setting)
		str = append(str, '#')
		str = internal.AppendByteDigits(self.data[index + 1], str)

		// append comparison operator
		str = append(str, ' ')
		switch self.data[index] & 0b0000_1111 {
		case 0b000: str = append(str, '=', '=')
		case 0b001: str = append(str, '!', '=')
		case 0b010: str = append(str, '<')
		case 0b011: str = append(str, '>')
		case 0b100: str = append(str, '<', '=')
		case 0b101: str = append(str, '>', '=')	
		default:
			panic(invalidInternalState)
		}
		str = append(str, ' ')

		// append second operand
		if (self.data[index] & 0b0001_0000) == 0 { // second operand is a setting too
			str = append(str, '#')	
		}
		str = internal.AppendByteDigits(self.data[index + 2], str)

		// advance index
		index += 3
	case 0b011: // quick 'setting == const'
		str = append(str, '#')
		str = internal.AppendByteDigits(self.data[index + 1], str)
		str = append(str, ' ', '=', '=', ' ')
		str = internal.AppendByteDigits(self.data[index] & 0b0001_1111, str)
		index += 2
	case 0b100: // quick 'setting != const'
		str = append(str, '#')
		str = internal.AppendByteDigits(self.data[index + 1], str)
		str = append(str, ' ', '!', '=', ' ')
		str = internal.AppendByteDigits(self.data[index] & 0b0001_1111, str)
		index += 2
	case 0b101: // quick 'setting < const'
		str = append(str, '#')
		str = internal.AppendByteDigits(self.data[index + 1], str)
		str = append(str, ' ', '<', ' ')
		str = internal.AppendByteDigits(self.data[index] & 0b0001_1111, str)
		index += 2
	case 0b110: // quick 'setting > const'
		str = append(str, '#')
		str = internal.AppendByteDigits(self.data[index + 1], str)
		str = append(str, ' ', '>', ' ')
		str = internal.AppendByteDigits(self.data[index] & 0b0001_1111, str)
		index += 2
	default:
		panic(invalidInternalState)
	}
	
	return index, str
}

// grammar:
// EXPR: (EXPR)
// EXPR: TERM
// EXPR: {INNER_EXPR OR}+ INNER_EXPR
// EXPR: {INNER_EXPR AND}+ INNER_EXPR
// TERM: #N {==|!=|<|>|<=|>=} #M
// TERM: #N {==|!=|<|>|<=|>=} M
// INNER_EXPR: TERM
// INNER_EXPR: (EXPR)

func compileRewriteCondition(definition string) (rewriteCondition, error) {
	var condition rewriteCondition
	if len(definition) > 1024 { return condition, errors.New("definitions limited to 1024 ascii characters max") }
	start, end := trimSpaces(definition, 0, len(definition) - 1)
	if end < start {
		return condition, errors.New("invalid empty definition")
	}

	var err error
	start, end, err = condition.appendNextExpr(definition, start, end, false)
	if err != nil { return condition, err }
	if start < end + 1 {
		return condition, errors.New("definition expected to end after '" + definition[0 : start] + "', but it continues")
	}
	if start != end + 1 {
		panic("broken code")
	}
	
	return condition, nil
}

func (self *rewriteCondition) appendNextExpr(definition string, start, end int, inner bool) (int, int, error) {
	start, end = trimSpaces(definition, start, end)
	if end < start {
		return start, end, errors.New("expected expression after '" + definition[: start] + "'")
	}
	if end + 1 > len(definition) { panic("broken code") }
	
	var err error
	if inner {
		if definition[start] == '(' {
			start, end, err = self.appendNextExpr(definition, start, end, false)
			if err != nil { return start, end, err }
			start, end = trimSpaces(definition, start, end)
			if end >= len(definition) || definition[end] != ')' {
				return start, end, errors.New("expected closing parenthesis after '" + definition[ : end] + "'")
			}
			return start, end, nil
		} else {
			return self.appendNextTerm(definition, start, end)
		}
	} else { // outer
		if definition[start] == '(' {
			start, end, err = self.appendNextExpr(definition, start, end, false)
			if err != nil { return start, end, err }
			start, end = trimSpaces(definition, start, end)
			if end >= len(definition) || definition[end] != ')' {
				return start, end, errors.New("expected closing parenthesis after '" + definition[ : end] + "'")
			}
			return start, end, nil
		} else if definition[start] == '#' {
			return self.appendNextTerm(definition, start, end)
		} else {
			start, end, err = self.appendNextExpr(definition, start, end, true) // inner expression
			if err != nil { return start, end, err }
			if startsWith(definition, start, end, "OR ") {
				index := len(self.data)
				self.data = append(self.data, 0)
				orCount := 2
				start += 3
				for {
					start, end, err = self.appendNextExpr(definition, start + 3, end, true)
					if err != nil { return start, end, err }
					if !startsWith(definition, start, end, "OR ") { break }
					start += 3
					orCount += 1
				}
				if orCount >= 32 { return start, end, errors.New("OR chain can't have more than 31 terms") }
				self.data[index] = uint8(orCount)
				return start, end, nil
			} else if startsWith(definition, start, end, "AND ") {
				index := len(self.data)
				self.data = append(self.data, 0)
				andCount := 2
				start += 4
				for {
					start, end, err = self.appendNextExpr(definition, start, end, true)
					if err != nil { return start, end, err }
					if !startsWith(definition, start, end, "AND ") { break }
					start += 4
					andCount += 1
				}
				if andCount >= 32 { return start, end, errors.New("AND chain can't have more than 31 terms") }
				self.data[index] = 0b0010_0000 | uint8(andCount)
				return start, end, nil
			} else {
				if end == start { return start, end, nil }
				// TODO: error message not great here, it should mention parens when relevant and more
				return start, end, errors.New("expected AND, OR or expression end after '" + definition[ : start] + "'")
			}
		}
	}
}

var termRegexp = regexp.MustCompile(`^#([0-9]+) *(==|!=|<|>|<=|>=) *(#?)([0-9]+)`)
func (self *rewriteCondition) appendNextTerm(definition string, start, end int) (int, int, error) {
	matches := termRegexp.FindStringSubmatch(definition[start : end + 1])
	if matches == nil {
		return start, end, errors.New("invalid setting comparison (e.g. \"#3 != 0\", \"#0 > #1\", \"#0 == 42\", etc) after '" + definition[ : start] + "'")
	}

	leftOpSettingIndex, err := strconv.Atoi(matches[1])
	if err != nil { panic("broken code") } // guaranteed by regexp
	if leftOpSettingIndex > 254 {
		return start, end, errors.New("setting '" + matches[1] + "' out of range")
	}
	operator := matches[2]
	rightOpIsSetting := (matches[3] == "#")
	rightOpValue, err := strconv.Atoi(matches[4])
	if err != nil { panic("broken code") } // guaranteed by regexp
	if rightOpValue > 254 {
		if rightOpIsSetting {
			return start, end, errors.New("setting '" + matches[4] + "' out of range")
		} else if rightOpValue > 255 {
			return start, end, errors.New("constant '" + matches[4] + "' out of range")
		}
	}

	if !rightOpIsSetting && rightOpValue < 32 && operator != "<=" && operator != ">=" {
		var ctrlCode byte
		switch operator {
		case "==": ctrlCode = 0b0110_0000
		case "!=": ctrlCode = 0b1000_0000
		case  "<": ctrlCode = 0b1010_0000
		case  ">": ctrlCode = 0b1100_0000
		default:
			panic("broken code")
		}
		self.data = append(self.data, ctrlCode | uint8(rightOpValue), uint8(leftOpSettingIndex))
	} else {
		var opCode byte
		switch operator {
		case "==": opCode = 0b000
		case "!=": opCode = 0b001
		case  "<": opCode = 0b010
		case  ">": opCode = 0b011
		case "<=": opCode = 0b100
		case ">=": opCode = 0b101
		default:
			panic("broken code")
		}

		if rightOpIsSetting {
			self.data = append(self.data, 0b0100_0000 | opCode, uint8(leftOpSettingIndex))
		} else {
			self.data = append(self.data, 0b0101_0000 | opCode, uint8(leftOpSettingIndex))
		}
	}

	return start + len(matches[0]), end, nil
}

func startsWith(definition string, start, end int, expr string) bool {
	if start < 0 || end + 1 > len(definition) { panic("broken code usage") }
	if len(expr) > (end + 1 - start) { return false }
	for i := 0; i < len(expr); i++ {
		if definition[start + i] != expr[i] { return false }
	}
	return true
}

// end is inclusive
func trimSpaces(definition string, start, end int) (int, int) {
	var changed bool = true
	for start <= end && changed {
		changed = false

		// trim spaces
		for start <= end && definition[start] == ' ' {
			start += 1
			changed = true
		}
		for end >= 0 && definition[end] == ' ' {
			end -= 1
			changed = true
		}
	}

	return start, end
}
