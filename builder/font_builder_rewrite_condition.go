package builder

import "unsafe"
import "errors"
import "regexp"
import "strconv"

import "github.com/tinne26/ggfnt/internal"

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
