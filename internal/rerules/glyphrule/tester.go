package glyphrule

import "errors"
import "strconv"

import "github.com/tinne26/ggfnt"
import "github.com/tinne26/ggfnt/internal"
import "github.com/tinne26/ggfnt/internal/rerules"

type Tester struct {
	rules []ggfnt.GlyphRewriteRule
	trees []TesterTreeInstance

	accumulator internal.CircularBufferU16[ggfnt.GlyphIndex]
	unflushedTail uint8
	needsRecompile bool
	isOperating bool
}

// --- general operations ---

func (self *Tester) NumRules() int { return len(self.rules) }
func (self *Tester) IsOperating() bool { return self.isOperating }
func (self *Tester) NumPendingGlyphs() int { return int(self.accumulator.Size()) }
func (self *Tester) Resync(font *ggfnt.Font, settingsCache *ggfnt.SettingsCache) error {
	if self.isOperating { panic(PreViolation) }
	if !self.needsRecompile { return nil }
	return self.recompile(font, settingsCache)
}

// --- rule management ---

func (self *Tester) RemoveAllRules() {
	if self.isOperating { panic(PreViolation) }
	self.rules = self.rules[ : 0]
	self.trees = self.trees[ : 0]
	self.accumulator.Clear()
	self.needsRecompile = false
}

func (self *Tester) AddRule(rule ggfnt.GlyphRewriteRule) error {
	// initial assertions
	if self.isOperating { panic(PreViolation) }
	if len(rule.Data) < internal.MinGlyphReRuleFmtLen {
		return rerules.ErrInvalidReRule
	}
	if len(self.rules) >= 254 { return rerules.ErrTesterTooManyRules }

	// rule addition
	self.rules = append(self.rules, rule)
	treeIndex, found := self.findTreeByCondition(rule.Condition())
	if found {
		self.trees[treeIndex].NeedsResync = true
	} else { // must create a new branch for this
		self.trees = internal.GrowSliceByOne(self.trees)
		self.trees[len(self.trees) - 1].Reconfigure(rule.Condition())
		for i := len(self.trees) - 1; i > 0; i-- {
			if self.trees[i].Condition >= self.trees[i - 1].Condition { break }
			self.trees[i], self.trees[i - 1] = self.trees[i - 1], self.trees[i]
		}
	}
	self.needsRecompile = true
	return nil
}

func (self *Tester) RemoveRule(rule ggfnt.GlyphRewriteRule) bool {
	if self.isOperating { panic(PreViolation) }

	for i, _ := range self.rules {
		if self.rules[i].Equals(rule) {
			self.rules = internal.DeleteElementAt(self.rules, i)
			treeIndex, found := self.findTreeByCondition(rule.Condition())
			if !found { panic(BrokenCode) }
			self.trees[treeIndex].NeedsResync = true
			self.needsRecompile = true
			return true
		}
	}

	return false
}

// --- condition control ---

func (self *Tester) RefreshConditions(font *ggfnt.Font, settingsCache *ggfnt.SettingsCache) {
	for i, _ := range self.trees {
		condition := self.trees[i].Condition
		satisfied := self.testCondition(condition, font, settingsCache)
		self.trees[i].ConditionCachedValue = satisfied
		self.trees[i].ConditionIsCached = true
	}
}

// --- glyph sequence operations ---

type GlyphConfirmationFunc = func(ggfnt.GlyphIndex)

func (self *Tester) BeginSequence(font *ggfnt.Font, settingsCache *ggfnt.SettingsCache) error {
	if self.isOperating { panic(PreViolation) }
	
	self.isOperating = true
	if self.needsRecompile {
		err := self.recompile(font, settingsCache)
		if err != nil { return err }
	}
	self.treesRestartDetection()

	return nil
}

// Precondition: glyph index can't be a control index.
func (self *Tester) Feed(glyphIndex ggfnt.GlyphIndex, fn GlyphConfirmationFunc) error {
	if !self.isOperating { panic(PreViolation) }
	if glyphIndex == ggfnt.GlyphZilch { return nil } // ignore zilch glyphs
	if glyphIndex > ggfnt.GlyphMissing {
		return errors.New("unxpected control glyph index '" + strconv.Itoa(int(glyphIndex)) + "'")
	}
	
	// feed glyph and see if we stopped running
	self.accumulator.Push(glyphIndex)
	detectionRunning := self.treesFeedUnrestrictedGlyph(glyphIndex)
	if !detectionRunning {
		matchRuleIndex := self.treesGetBestMatch()
		if matchRuleIndex == RuleNone {
			self.popOldestGlyph(fn)
		} else { // match found, rewrite
			self.rewrite(matchRuleIndex, fn)
		}
		for self.rerunDetection(fn) {}
	}

	return nil
}

func (self *Tester) Break(fn GlyphConfirmationFunc) {
	if !self.isOperating { panic(PreViolation) }
	
	for self.accumulator.Size() > uint16(self.unflushedTail) {
		matchRuleIndex := self.treesGetBestMatch()
		if matchRuleIndex == RuleNone {
			self.popOldestGlyph(fn)		
		} else {
			self.rewrite(matchRuleIndex, fn)
		}
		for self.rerunDetection(fn) {}
	}
}

func (self *Tester) FinishSequence(fn GlyphConfirmationFunc) {
	self.Break(fn)
	self.isOperating = false
}

// ---- internal helper methods ----

func (self *Tester) findTreeByCondition(condition uint8) (int, bool) {
	numTrees := len(self.trees)
	start, end := 0, numTrees
	for start < end {
		mid := (start + end) >> 1 // can't overflow, can't have more than 255 conditions
		if self.trees[mid].Condition < condition {
			start = mid + 1
		} else {
			end = mid
		}
	}
	
	return start, start < numTrees && self.trees[start].Condition == condition
}

func (self *Tester) testCondition(condition uint8, font *ggfnt.Font, settingsCache *ggfnt.SettingsCache) bool {
	// base case, not needed in general, but allows font and settings to be nil
	if condition == 255 { return true }

	// general case
	satisfied, cached := settingsCache.GetRewriteCondition(condition)
	if !cached {
		satisfied = font.Rewrites().EvaluateCondition(condition, settingsCache.UnsafeSlice())
		settingsCache.CacheRewriteCondition(condition, satisfied)
	}
	return satisfied
}

func (self *Tester) recompile(font *ggfnt.Font, settingsCache *ggfnt.SettingsCache) error {
	// assume no concurrent tester recompilations will happen.
	// if they happen.. performance might struggle but behavior
	// will still be correct
	compiler := CommonDecisionTreeCompiler.Retrieve()
	defer CommonDecisionTreeCompiler.Release(compiler)

	// check each branch
	var treeIndex int = 0
	for treeIndex < len(self.trees) {
		tree := &self.trees[treeIndex]

		// test and cache condition if uncached
		if !tree.ConditionIsCached {
			satisfied := self.testCondition(tree.Condition, font, settingsCache)
			tree.ConditionCachedValue = satisfied
			tree.ConditionIsCached = true
		}
		
		// recompile tree
		if tree.NeedsResync {
			var numRulesFound int = 0
			compiler.Begin(font, tree.DecisionTree.states)
			for ruleIndex, _ := range self.rules {
				// (NOTE: this is not particularly efficient)
				if self.rules[ruleIndex].Condition() == tree.Condition {
					err := compiler.Feed(self.rules[ruleIndex], RuleIndex(ruleIndex))
					if err != nil { return err }
					numRulesFound += 1
				}
			}
			tree.DecisionTree.states = compiler.Finish()

			// if no rules found, delete tree
			if numRulesFound == 0 {
				self.trees = internal.DeleteElementAt(self.trees, treeIndex)
				continue
			}
		}

		treeIndex += 1
	}
	
	// ensure sufficient accumulator capacity
	// (NOTE: this is not particularly efficient)
	var maxAccLen uint8 = 0
	for i, _ := range self.rules {
		maxAccLen = max(maxAccLen, self.rules[i].InLen())
	}
	self.accumulator.Clear()
	self.accumulator.SetCapacity(uint16(maxAccLen))

	// success, return
	self.needsRecompile = false
	return nil
}

func (self *Tester) treesFeedGlyph(glyphIndex ggfnt.GlyphIndex, depth uint8) bool {
	if depth < self.unflushedTail {
		return self.treesFeedHeadGlyph(glyphIndex)
	} else {
		return self.treesFeedUnrestrictedGlyph(glyphIndex)
	}
}

func (self *Tester) treesFeedHeadGlyph(glyphIndex ggfnt.GlyphIndex) bool {
	var running bool = false
	minRequiredHeadLen := self.unflushedTail
	for i, _ := range self.trees {
		if !self.trees[i].IsScanning() { continue }
		self.trees[i].FeedHead(glyphIndex, minRequiredHeadLen)
		if self.trees[i].IsScanning() {
			running = true
		}
	}
	return running
}

func (self *Tester) treesFeedUnrestrictedGlyph(glyphIndex ggfnt.GlyphIndex) bool {
	var running bool = false
	for i, _ := range self.trees {
		if !self.trees[i].IsScanning() { continue }
		self.trees[i].Feed(glyphIndex)
		if self.trees[i].IsScanning() {
			running = true
		}
	}
	return running
}

func (self *Tester) treesGetBestMatch() RuleIndex {
	var bestMatch RuleIndex = RuleNone
	for i, _ := range self.trees {
		match := self.trees[i].BestMatch()
		if match < bestMatch { bestMatch = match }
	}
	return bestMatch
}

func (self *Tester) rewrite(matchRuleIndex RuleIndex, fn GlyphConfirmationFunc) {
	if matchRuleIndex == RuleNone { panic(PreViolation) }
	
	// report head
	headLen := self.rules[matchRuleIndex].HeadLen()
	for i := uint8(0); i < headLen; i++ {
		if self.unflushedTail > 0 {
			self.unflushedTail -= 1
		} else {
			fn(self.accumulator.PopHead())
		}
	}

	// safety assertion
	if self.unflushedTail > 0 { panic(BrokenCode) }
	
	// report rewritten glyphs and skip replaces from the body block
	self.rules[matchRuleIndex].EachOut(fn)
	for i := uint8(0); i < self.rules[matchRuleIndex].BodyLen(); i++ {
		self.accumulator.PopHead()
	}
	
	// restart detection before reporting the tail.
	// the tail must be re-detected if necessary.
	self.treesRestartDetection()

	// report tail
	self.unflushedTail = uint8(self.rules[matchRuleIndex].TailLen())
	if self.unflushedTail > 0 { // we can't pop the tail, but we still gotta report it
		fn(self.accumulator.Head())
		for i := uint8(1); i < self.unflushedTail; i++ {
			glyphIndex, found := self.accumulator.PeekAhead()
			if !found { panic(BrokenCode) }
			fn(glyphIndex)
		}
		//self.accumulator.DiscardPeeks()
	}
	
	// notice that we are not feeding any possible remaining accumulated glyphs,
	// which are a possibility if we were testing a longer rewrite rule that
	// wasn't satisfied in the end. that has to be dealt with externally.
}

func (self *Tester) treesRestartDetection() {
	for i, _ := range self.trees {
		if self.trees[i].IsScanning() {
			self.trees[i].BreakSequence()
		}
		self.trees[i].BeginSequence()
	}
}

// Returns whether further rerun detection is possible or not.
func (self *Tester) rerunDetection(fn GlyphConfirmationFunc) bool {
	if self.accumulator.IsEmpty() { return false }
	
	self.accumulator.DiscardPeeks()
	glyphIndex := self.accumulator.Head()
	var depth uint8 = 0
	for {
		detectionRunning := self.treesFeedGlyph(glyphIndex, depth)
		if detectionRunning {
			var hasNext bool
			glyphIndex, hasNext = self.accumulator.PeekAhead()
			if !hasNext { return false } // halt for now, process will continue with the next feeds
			depth += 1
		} else {
			matchRuleIndex := self.treesGetBestMatch()
			if matchRuleIndex == RuleNone {
				// discard peeks and report initial head
				self.accumulator.DiscardPeeks()
				self.popOldestGlyph(fn)
			} else {
				// rewrite match (this will also advance the accumulator as necessary)
				self.rewrite(matchRuleIndex, fn)
			}
			return !self.accumulator.IsEmpty() // some action has been performed, that's enough for now
		}
	}
}

func (self *Tester) popOldestGlyph(fn GlyphConfirmationFunc) {
	if self.unflushedTail > 0 {
		_ = self.accumulator.PopHead()
		self.unflushedTail -= 1
	} else {
		fn(self.accumulator.PopHead())
	}
	self.treesRestartDetection()
}
