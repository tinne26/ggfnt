package glyphrule

import "github.com/tinne26/ggfnt"

// TODO: it should be possible to store the lowest rule index to bound
//       the search process by comparing to the best current match.
//       I don't know if it's worth or not, but it would be a reasonable
//       optimization to attempt for complex cases.

// Evaluates a set of replacement rules.
type DecisionTree struct {
	states []State
	
	index StateIndex
	bestMatch RuleIndex
	isScanning bool
}

func (self *DecisionTree) IsScanning() bool {
	return self.isScanning
}

// This can be queried even after [DecisionTree.BreakSequence](),
// it's only reset on [DecisionTree.BeginSequence]().
func (self *DecisionTree) BestMatch() RuleIndex {
	return self.bestMatch
}

func (self *DecisionTree) BeginSequence() {
	if self.isScanning { panic(PreViolation) }
	self.index = 0
	self.bestMatch = RuleNone
	self.isScanning = true
}

func (self *DecisionTree) BreakSequence() {
	if !self.isScanning { panic(PreViolation) }
	self.isScanning = false
}

func (self *DecisionTree) Feed(glyphIndex ggfnt.GlyphIndex) {
	if !self.isScanning { panic(PreViolation) }
	
	var ruleMatchIndex RuleIndex
	self.index, ruleMatchIndex = self.states[self.index].Trigger(glyphIndex)
	if ruleMatchIndex != RuleNone { self.bestMatch = ruleMatchIndex }
	if self.index == StateNone { self.isScanning = false }
}

func (self *DecisionTree) FeedHead(glyphIndex ggfnt.GlyphIndex, minRequiredHeadLen uint8) {
	if !self.isScanning { panic(PreViolation) }
	
	if self.states[self.index].SatisfiesMinRequiredHead(minRequiredHeadLen) {
		self.Feed(glyphIndex)
	} else {
		self.isScanning = false
	}
}

// Basic algorithm for full match process, instead of stream-based
// with [DecisionTree.Feed](). Mainly for reference and testing.
func (self *DecisionTree) Match(glyphs []ggfnt.GlyphIndex) RuleIndex {	
	self.BeginSequence()
	for _, glyphIndex := range glyphs {
		self.Feed(glyphIndex)
		if !self.IsScanning() {
			return self.BestMatch()
		}
	}
	self.BreakSequence()
	return self.BestMatch()
}
