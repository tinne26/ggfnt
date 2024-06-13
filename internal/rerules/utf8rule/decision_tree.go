package utf8rule

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

func (self *DecisionTree) Feed(codePoint rune) {
	if !self.isScanning { panic(PreViolation) }
	
	var ruleMatchIndex RuleIndex
	self.index, ruleMatchIndex = self.states[self.index].Trigger(codePoint)
	if ruleMatchIndex != RuleNone { self.bestMatch = ruleMatchIndex }
	if self.index == StateNone { self.isScanning = false }
}

func (self *DecisionTree) FeedHead(codePoint rune, minRequiredHeadLen uint8) {
	if !self.isScanning { panic(PreViolation) }
	
	if self.states[self.index].SatisfiesMinRequiredHead(minRequiredHeadLen) {
		self.Feed(codePoint)
	} else {
		self.isScanning = false
	}
}

// Basic algorithm for full match process, instead of stream-based
// with [DecisionTree.Feed](). Mainly for reference and testing.
func (self *DecisionTree) Match(runes []rune) RuleIndex {
	self.BeginSequence()
	for _, codePoint := range runes {
		self.Feed(codePoint)
		if !self.IsScanning() {
			return self.BestMatch()
		}
	}
	self.BreakSequence()
	return self.BestMatch()
}
