package glyphrule

import "github.com/tinne26/ggfnt"
import "github.com/tinne26/ggfnt/internal"
import "github.com/tinne26/ggfnt/internal/rerules"

type RuleScanner struct {
	rule ggfnt.GlyphRewriteRule
	dataIndex uint16
	block RuleBlock
	
	remainingSets uint8
	remainingGlyphs uint8
	remainingBlockElements uint8
	accumulatedBlockSets uint8 // verification purposes only
	accumulatedBlockGlyphs uint8 // verification purposes only
}

// Returns any error that might be detected while processing the rule
// (ErrInvalidReRule is often returned as a catch-all for any issue found).
func (self *RuleScanner) Start(rule ggfnt.GlyphRewriteRule) error {
	// ensure that rule has enough data and the output sequence is not empty
	if len(rule.Data) < internal.MinGlyphReRuleFmtLen || rule.Data[4] == 0 {
		return rerules.ErrInvalidReRule
	}

	// ensure that sum of lengths doesn't exceed 255
	lenSum := rule.Data[1] + rule.Data[2]
	if rule.Data[1] > lenSum { return rerules.ErrInvalidReRule }
	lenSum += rule.Data[3]
	if rule.Data[3] > lenSum { return rerules.ErrInvalidReRule }

	// ensure that output length doesn't exceed body length
	if rule.BodyLen() < rule.OutLen() { return rerules.ErrInvalidReRule }
	
	// reset some fields
	self.rule = rule
	self.remainingSets   = 0
	self.remainingGlyphs = 0
	self.accumulatedBlockSets   = 0
	self.accumulatedBlockGlyphs = 0
	self.remainingBlockElements = self.rule.Data[1] // head len
	self.block = RuleHead
	self.dataIndex = 5 + (uint16(self.rule.Data[4]) << 1)
	
	// skip to body if head is empty
	if self.rule.Data[1] == 0 {
		self.dataIndex += 1
		self.block = RuleBody
		self.remainingBlockElements = self.rule.Data[2]
		if self.remainingBlockElements == 0 || self.remainingBlockElements < self.rule.Data[4] {
			return rerules.ErrInvalidReRule
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

// Returns the next element in the rule (a ggfnt.GlyphIndex or an
// uint8 glyph set). Returns [ErrInvalidReRule] if something is wrong
// in the rule. Will panic if !HasNext().
func (self *RuleScanner) Next() (ggfnt.GlyphIndex, GlyphSetIndex, error) {
	// glyph set case
	if self.remainingSets > 0 {
		self.remainingSets -= 1
		if int(self.dataIndex) >= len(self.rule.Data) {
			return GlyphNone, GlyphSetNone, rerules.ErrInvalidReRule
		}

		set := GlyphSetIndex(self.rule.Data[self.dataIndex])
		self.dataIndex += 1
		if int(self.dataIndex) >= len(self.rule.Data) {
			err := self.checkTermination()
			if err != nil { return GlyphNone, GlyphSetNone, err }
		} else if self.remainingSets == 0 && self.remainingGlyphs == 0 {
			err := self.advanceControl()
			if err != nil { return GlyphNone, GlyphSetNone, err }
		}

		return GlyphNone, set, nil
	}

	// glyph case
	if self.remainingGlyphs > 0 {
		self.remainingGlyphs -= 1
		if int(self.dataIndex) + 1 >= len(self.rule.Data) {
			return GlyphNone, GlyphSetNone, rerules.ErrInvalidReRule
		}

		glyphIndex := ggfnt.GlyphIndex(internal.DecodeUint16LE(self.rule.Data[self.dataIndex : ]))
		if !IsValidRuleGlyphIndex(glyphIndex) {
			return GlyphNone, GlyphSetNone, rerules.ErrInvalidReRule
		}
		self.dataIndex += 2
		if int(self.dataIndex) >= len(self.rule.Data) {
			err := self.checkTermination()
			if err != nil { return GlyphNone, GlyphSetNone, err }
		} else if self.remainingSets == 0 && self.remainingGlyphs == 0 {
			err := self.advanceControl()
			if err != nil { return GlyphNone, GlyphSetNone, err }
		}

		return glyphIndex, GlyphSetNone, nil
	}

	if !self.HasNext() { panic("precondition violation") }
	panic(BrokenCode)
}

func (self *RuleScanner) advanceControl() error {
	if self.remainingBlockElements == 0 { // new fragment
		// safety check
		if self.rule.Data[1 + int(self.block)] != self.accumulatedBlockGlyphs + self.accumulatedBlockSets {
			return rerules.ErrInvalidReRule
		}

		var err error
		self.block, err = self.block.Next()
		if err != nil { return err }
		self.remainingBlockElements = self.rule.Data[1 + int(self.block)] // max hacks
		self.accumulatedBlockSets   = 0
		self.accumulatedBlockGlyphs = 0
	}

	return self.parseFragmentControl()
}

func (self *RuleScanner) parseFragmentControl() error {
	ctrl := self.rule.Data[self.dataIndex]
	self.dataIndex += 1
	self.remainingSets   = (ctrl & 0b1111_0000) >> 4
	self.remainingGlyphs = (ctrl & 0b0000_1111)
	self.accumulatedBlockSets   += self.remainingSets
	self.accumulatedBlockGlyphs += self.remainingGlyphs
	numElems := self.remainingSets + self.remainingGlyphs
	if numElems == 0 {
		if self.accumulatedBlockSets + self.accumulatedBlockGlyphs != self.rule.Data[1 + int(self.block)] {
			return rerules.ErrInvalidReRule
		}
	} else {
		if self.accumulatedBlockSets + self.accumulatedBlockGlyphs < self.accumulatedBlockSets {
			return rerules.ErrInvalidReRule // ^ overflow
		}
		if self.accumulatedBlockSets + self.accumulatedBlockGlyphs > self.rule.Data[1 + int(self.block)] {
			return rerules.ErrInvalidReRule
		}
		if numElems > self.remainingBlockElements {
			return rerules.ErrInvalidReRule
		}
	}
	
	self.remainingBlockElements -= numElems
	return nil
}

// Returns an error if the current point is not a valid termination point for the rule.
func (self *RuleScanner) checkTermination() error {
	if self.remainingSets > 0 || self.remainingGlyphs > 0 || self.remainingBlockElements > 0 {
		return rerules.ErrInvalidReRule
	}

	switch self.block {
	case RuleHead:
		return rerules.ErrInvalidReRule 
	case RuleBody:
		if self.rule.Data[3] != 0 {
			return rerules.ErrInvalidReRule
		}
	case RuleTail:
		if self.rule.Data[3] != self.accumulatedBlockSets + self.accumulatedBlockGlyphs {
			return rerules.ErrInvalidReRule
		}
	default:
		panic(BrokenCode)
	}
	return nil
}
