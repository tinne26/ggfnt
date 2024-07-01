package utf8rule

import "github.com/tinne26/ggfnt"
import "github.com/tinne26/ggfnt/internal"
import "github.com/tinne26/ggfnt/internal/rerules"

type RuleScanner struct {
	rule ggfnt.Utf8RewriteRule
	dataIndex uint16
	block RuleBlock
	
	remainingSets uint8
	remainingCodePoints uint8
	remainingBlockElements uint8
	accumulatedBlockSets uint8 // verification purposes only
	accumulatedBlockCodePoints uint8 // verification purposes only
}

// Returns any error that might be detected while processing the rule.
func (self *RuleScanner) Start(rule ggfnt.Utf8RewriteRule) error {
	// ensure that rule has enough data and the output sequence is not empty
	if len(rule.Data) < internal.MinUtf8ReRuleFmtLen || rule.Data[4] == 0 {
		return rerules.InvalidUtf8RuleErr(rule)
	}

	// ensure that sum of lengths doesn't exceed 255
	lenSum := rule.Data[1] + rule.Data[2]
	if rule.Data[1] > lenSum { return rerules.InvalidUtf8RuleErr(rule) }
	lenSum += rule.Data[3]
	if rule.Data[3] > lenSum { return rerules.InvalidUtf8RuleErr(rule) }

	// ensure that output length doesn't exceed body length
	if rule.BodyLen() < rule.OutLen() { return rerules.InvalidUtf8RuleErr(rule) }
	
	// reset some fields
	self.rule = rule
	self.remainingSets = 0
	self.remainingCodePoints = 0
	self.accumulatedBlockSets = 0
	self.accumulatedBlockCodePoints = 0
	self.remainingBlockElements = self.rule.Data[1] // head len
	self.block = RuleHead
	self.dataIndex = 5 + (uint16(self.rule.Data[4]) << 2)
	
	// skip to body if head is empty
	if self.rule.Data[1] == 0 {
		self.dataIndex += 1
		self.block = RuleBody
		self.remainingBlockElements = self.rule.Data[2]
		if self.remainingBlockElements == 0 || self.remainingBlockElements < self.rule.Data[4] {
			return rerules.InvalidUtf8RuleErr(rule)
		}
	}

	// parse head or body fragment control
	self.parseFragmentControl()

	return nil
}

func (self *RuleScanner) Condition() uint8 { return self.rule.Condition() }
func (self *RuleScanner) HeadLen() uint8 { return self.rule.HeadLen() }
func (self *RuleScanner) BodyLen() uint8 { return self.rule.BodyLen() }
func (self *RuleScanner) TailLen() uint8 { return self.rule.TailLen() }
func (self *RuleScanner) Block() RuleBlock { return self.block }

func (self *RuleScanner) HasNext() bool {
	return int(self.dataIndex) < len(self.rule.Data)
}

// Returns the next element in the rule (a code point or
// an uint8 utf8 set), or an error. Panics if !HasNext().
func (self *RuleScanner) Next() (rune, Utf8SetIndex, error) {
	// utf8 set case
	if self.remainingSets > 0 {
		self.remainingSets -= 1
		if int(self.dataIndex) >= len(self.rule.Data) {
			return RuneError, Utf8SetNone, rerules.InvalidUtf8RuleErr(self.rule)
		}

		set := Utf8SetIndex(self.rule.Data[self.dataIndex])
		self.dataIndex += 1
		if int(self.dataIndex) >= len(self.rule.Data) {
			err := self.checkTermination()
			if err != nil { return RuneError, Utf8SetNone, err }
		} else if self.remainingSets == 0 && self.remainingCodePoints == 0 {
			err := self.advanceControl()
			if err != nil { return RuneError, Utf8SetNone, err }
		}

		return RuneError, set, nil
	}

	// code point case
	if self.remainingCodePoints > 0 {
		self.remainingCodePoints -= 1
		if int(self.dataIndex) + 3 >= len(self.rule.Data) {
			return RuneError, Utf8SetNone, rerules.InvalidUtf8RuleErr(self.rule)
		}

		codePoint := rune(internal.DecodeUint32LE(self.rule.Data[self.dataIndex : ]))
		if !IsValidRuleCodePoint(codePoint) {
			return RuneError, Utf8SetNone, rerules.InvalidUtf8RuleErr(self.rule)
		}
		self.dataIndex += 4
		if int(self.dataIndex) >= len(self.rule.Data) {
			err := self.checkTermination()
			if err != nil { return RuneError, Utf8SetNone, err }
		} else if self.remainingSets == 0 && self.remainingCodePoints == 0 {
			err := self.advanceControl()
			if err != nil { return RuneError, Utf8SetNone, err }
		}

		return codePoint, Utf8SetNone, nil
	}

	if !self.HasNext() { panic("precondition violation") }
	panic(BrokenCode)
}

func (self *RuleScanner) advanceControl() error {
	if self.remainingBlockElements == 0 { // new fragment
		// safety check
		if self.rule.Data[1 + int(self.block)] != self.accumulatedBlockCodePoints + self.accumulatedBlockSets {
			return rerules.InvalidUtf8RuleErr(self.rule)
		}

		var err error
		self.block, err = self.block.Next()
		if err != nil { return err }
		self.remainingBlockElements = self.rule.Data[1 + int(self.block)] // max hacks
		self.accumulatedBlockSets   = 0
		self.accumulatedBlockCodePoints = 0
	}

	return self.parseFragmentControl()
}

func (self *RuleScanner) parseFragmentControl() error {
	ctrl := self.rule.Data[self.dataIndex]
	self.dataIndex += 1
	self.remainingSets = (ctrl & 0b1111_0000) >> 4
	self.remainingCodePoints = (ctrl & 0b0000_1111)
	self.accumulatedBlockSets += self.remainingSets
	self.accumulatedBlockCodePoints += self.remainingCodePoints
	numElems := self.remainingSets + self.remainingCodePoints
	if numElems == 0 {
		if self.accumulatedBlockSets + self.accumulatedBlockCodePoints != self.rule.Data[1 + int(self.block)] {
			return rerules.InvalidUtf8RuleErr(self.rule)
		}
	} else {
		if self.accumulatedBlockSets + self.accumulatedBlockCodePoints < self.accumulatedBlockSets {
			return rerules.InvalidUtf8RuleErr(self.rule) // ^ overflow
		}
		if self.accumulatedBlockSets + self.accumulatedBlockCodePoints > self.rule.Data[1 + int(self.block)] {
			return rerules.InvalidUtf8RuleErr(self.rule)
		}
		if numElems > self.remainingBlockElements {
			return rerules.InvalidUtf8RuleErr(self.rule)
		}
	}
	
	self.remainingBlockElements -= numElems
	return nil
}

// Returns an error if the current point is not a valid termination point for the rule.
func (self *RuleScanner) checkTermination() error {
	if self.remainingSets > 0 || self.remainingCodePoints > 0 || self.remainingBlockElements > 0 {
		return rerules.InvalidUtf8RuleErr(self.rule)
	}

	switch self.block {
	case RuleHead:
		return rerules.InvalidUtf8RuleErr(self.rule)
	case RuleBody:
		if self.rule.Data[3] != 0 {
			return rerules.InvalidUtf8RuleErr(self.rule)
		}
	case RuleTail:
		if self.rule.Data[3] != self.accumulatedBlockSets + self.accumulatedBlockCodePoints {
			return rerules.InvalidUtf8RuleErr(self.rule)
		}
	default:
		panic(BrokenCode)
	}
	return nil
}
