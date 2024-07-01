package glyphrule

import "fmt"
import "errors"

import "github.com/tinne26/ggfnt"
import "github.com/tinne26/ggfnt/internal"

const CompilerHitLimits = "DecisionTreeCompiler hit a limit on number of states, transitions or a similar parameter"
var ErrCompilerHitLimits = errors.New(CompilerHitLimits)

// package level reusable decision tree (could have also been a pool)
var CommonDecisionTreeCompiler internal.SingleInstancePool[DecisionTreeCompiler]

// Creating decision trees is not trivial. Many states have to be reused, and
// we need a series of auxiliary data structures to keep track of everything
// and end up building a reasonably sized tree in a reasonable time.
type DecisionTreeCompiler struct {
	// compilation data
	states []State
	
	// compilation process aux variables
	run struct {
		Font *ggfnt.Font
		Scanner RuleScanner
		RuleIndex RuleIndex
		StateIndex StateIndex
		Depth uint8

		FromStates []StateIndex // for tracking state index branching throughout rule compilation
		ToStates   []StateIndex // only non-repeated states are inserted (in proper order)

		TrackerFreedStates []StateIndex // indices of states that can be repurposed due to no use
		// all the remaining slices begin with state index 1 (0 is root)
		TrackerStateUses []uint8 // use counts for each state. when 0, we can clean them up
		TrackerStateRules [][]RuleIndex // rules involved in the state, sorted (asc.)
		TrackerStatesByDepth [][]StateIndex // depth to list of states on that depth level, sorted (asc.)
	}

	// operating state
	compiling bool
}

// Can only be called if the compilation process isn't
// already active (after init or after a Finish()).
func (self *DecisionTreeCompiler) Begin(font *ggfnt.Font, buffer []State) {
	if self.compiling { panic(PreViolation) } // discretionary safety assertion

	self.compiling = true
	self.states = buffer[ : 0]
	
	self.run.Font = font
	self.run.TrackerStateRules = self.run.TrackerStateRules[ : 0]
	self.run.TrackerStateUses  = self.run.TrackerStateUses[ : 0]
	self.run.TrackerFreedStates = self.run.TrackerFreedStates[ : 0]
	self.run.TrackerStatesByDepth = self.run.TrackerStatesByDepth[ : 0]
}

// Can only be called within a compilation process
// (Begin() was invoked at some earlier point).
func (self *DecisionTreeCompiler) Finish() []State {
	if !self.compiling { panic(PreViolation) } // discretionary safety assertion
	
	// get output buffer
	buffer := self.states
	self.states = nil

	// reset all fields for subsequent uses
	self.run.Font = nil
	self.compiling = false // must be last

	return buffer
}


func (self *DecisionTreeCompiler) Feed(rule ggfnt.GlyphRewriteRule, index RuleIndex) error {
	if !self.compiling { panic(PreViolation) } // discretionary safety assertion

	// register temp operation helper values
	self.run.Depth = 0
	self.run.RuleIndex = index
	self.run.FromStates = self.run.FromStates[ : 0]
	self.run.ToStates   = self.run.ToStates[ : 0]
	self.run.FromStates = append(self.run.FromStates, 0) // start at the first state

	// first state special case
	if len(self.states) == 0 {
		self.states = internal.GrowSliceByOne(self.states)
		self.states[0].Reset()
	}

	// for each glyph set in the rule, for each range in the glyph set, append it
	err := self.run.Scanner.Start(rule)
	if err != nil { return err }

	for self.run.Scanner.HasNext() {
		glyphIndex, glyphSet, err := self.run.Scanner.Next()
		if err != nil { return err }
		var matchRule RuleIndex = RuleNone
		if !self.run.Scanner.HasNext() {
			matchRule = self.run.RuleIndex
		}

		// since the decision tree can widen during the process, we need
		// to keep updating all the possible branches that have been opening
		for _, state := range self.run.FromStates {
			// adjust helper variables and a few properties
			self.run.StateIndex = state
			self.states[state].UpdateHeadLen(self.run.Scanner.HeadLen())

			// append glyph or glyph range
			if glyphIndex != GlyphNone { // glyph index case
				if glyphSet != GlyphSetNone { panic(BrokenCode) } // discretionary safety assertion
	
				glyphRange := ggfnt.GlyphRange{ First: glyphIndex, Last: glyphIndex }
				err := self.states[state].AppendRange(glyphRange, matchRule, self.stateLinker)
				if err != nil { return err }
			} else { // glyph set case
				if glyphSet == GlyphSetNone { panic(BrokenCode) } // discretionary safety assertion
	
				set := self.run.Font.Rewrites().GetGlyphSet(uint8(glyphSet))
				err := set.EachRange(func(glyphRange ggfnt.GlyphRange) error {
					return self.states[state].AppendRange(glyphRange, matchRule, self.stateLinker)
				})
				if err != nil { return err }
				err = set.EachListGlyph(func(glyphIndex ggfnt.GlyphIndex) error {
					glyphRange := ggfnt.GlyphRange{ First: glyphIndex, Last: glyphIndex }
					return self.states[state].AppendRange(glyphRange, matchRule, self.stateLinker)
				})
				if err != nil { return err }
			}
		}
		
		// update run variables
		self.freeUnusedStates()
		self.run.Depth += 1
		if self.run.Depth == 0 { panic(BrokenCode) } // num states should break before this
		self.run.FromStates, self.run.ToStates = self.run.ToStates, self.run.FromStates
		self.run.ToStates = self.run.ToStates[ : 0]
	}

	return nil
}

// ---- unexported ----

func (self *DecisionTreeCompiler) freeUnusedStates() {
	if self.run.Depth == 0 { return } // no state at depth 0

	// the main idea of this little snippet is to iterate the states
	// at the target depth to see if they can be removed, and while
	// we are at it, we also keep "compacting" the slice by moving
	// remaining state indices to fill any gaps created
	var freedCount int = 0
	depth := self.run.Depth - 1 // TrackerStatesByDepth begins at depth 1
	
	for i, state := range self.run.TrackerStatesByDepth[depth] {
		if state == 0 { panic(BrokenCode) }
		if self.run.TrackerStateUses[state - 1] == 0 {
			// note: could optimize if it's the last state, but whatever
			// notice: we don't reset state info here yet, it will have
			//         to be reset later when we need it again
			self.run.TrackerFreedStates = append(self.run.TrackerFreedStates, state)
			freedCount += 1
		} else if freedCount > 0 {
			self.run.TrackerStatesByDepth[depth][i - freedCount] = state
		}
	}
	
	if freedCount > 0 {
		numStates := len(self.run.TrackerStatesByDepth[depth])
		self.run.TrackerStatesByDepth[depth] = self.run.TrackerStatesByDepth[depth][ : numStates - freedCount]
	}
}

type StateLinkerFn = func(StateIndex, int) (StateIndex, error)
func (self *DecisionTreeCompiler) stateLinker(referenceStateIndex StateIndex, numSplits int) (StateIndex, error) {
	// check if the desired state already exists
	resultStateIndex := self.fetchReusableState(referenceStateIndex)
	
	// new state case
	if resultStateIndex == StateNone {
		var err error
		resultStateIndex, err = self.fetchFreedOrNewState()
		if err != nil { return StateNone, err }
		if referenceStateIndex != StateNone {
			self.states[resultStateIndex].CopyDataFrom(self.states[referenceStateIndex])
			self.initializeStateRulesWithRef(resultStateIndex, referenceStateIndex)
		} else {
			self.states[resultStateIndex].Reset()
			self.initializeStateRules(resultStateIndex)
		}
		self.registerNewStateAtDepth(resultStateIndex)
	}
	
	// update tracking and helper variables
	self.appendStateToNextRound(resultStateIndex)
	self.adjustStateUseCount(resultStateIndex, 1)
	if referenceStateIndex == StateNone && numSplits != 0 { panic(PreViolation) }
	if referenceStateIndex != StateNone {
		if numSplits < 0 || numSplits > 2 { panic(PreViolation) }
		self.adjustStateUseCount(referenceStateIndex, numSplits - 1)
	}

	// return resulting state
	return resultStateIndex, nil
}

// Will return StateNone if no reusable state found.
// The reference state index must be a next state, so it can't ever be zero (root).
func (self *DecisionTreeCompiler) fetchReusableState(referenceStateIndex StateIndex) StateIndex {
	// for a state to be reusable, it must contain the union of rules
	// from the given reference state + self.run.RuleIndex

	if referenceStateIndex == 0 { panic(PreViolation) }

	depth := self.run.Depth // it should be + 1, but TrackerStatesByDepth begins at 1
	if int(depth) >= len(self.run.TrackerStatesByDepth) { return StateNone }
	if len(self.run.TrackerStatesByDepth[depth]) == 0 { return StateNone }
	
	// target rule set for reusability checks would be all the referenceStateIndex
	// rules + run.RuleIndex. we don't really need to build the set, though, figuring
	// out in which position run.RuleIndex would appear is enough for later comparisons
	var comparisonRulePosition int = 0
	var rules []RuleIndex
	if referenceStateIndex != StateNone {
		rules = self.run.TrackerStateRules[referenceStateIndex - 1]
		for i, _ := range rules {
			if rules[i] < self.run.RuleIndex { comparisonRulePosition += 1 }
			if rules[i] == self.run.RuleIndex { panic(BrokenCode) }
			break
		}
	}

	// for each state at the right depth, see if it can be reused
	targetRulesLen := 1 + len(rules)
outerLoop:
	for _, stateIndex := range self.run.TrackerStatesByDepth[depth] {
		candidateRules := self.run.TrackerStateRules[stateIndex - 1]
		if len(candidateRules) != targetRulesLen { continue }

		// somewhat clever comparison mechanism: rules are ordered,
		// so we don't need to create an aux slice with the expected
		// rules for state reusability, only know at which position
		// the "extra rule" (the fromRule) would appear, in order
		// to compare the current state rule set with the desired one
		for i, ruleIndex := range candidateRules {
			if i < comparisonRulePosition {
				if ruleIndex != rules[i] { continue outerLoop }
			} else if i == comparisonRulePosition {
				if ruleIndex != self.run.RuleIndex { continue outerLoop }
			} else { // i > comparisonRulePosition
				if ruleIndex != rules[i - 1] { continue outerLoop }
			}
		}
		return stateIndex // reusable state found!
	}
	
	return StateNone
}

// Notice: the returned state index might contain junk within states,
// TrackerStateRules and TrackerStateUses. All that data should be
// manually overwritten or set.
func (self *DecisionTreeCompiler) fetchFreedOrNewState() (StateIndex, error) {	
	switch {
	case len(self.run.TrackerFreedStates) > 0: // reuse freed state
		freedCount := len(self.run.TrackerFreedStates)
		resultStateIndex := self.run.TrackerFreedStates[freedCount - 1]
		self.run.TrackerFreedStates = self.run.TrackerFreedStates[ : freedCount - 1]
		return resultStateIndex, nil
	case len(self.states) >= 254: // compiler limit reached
		return StateNone, ErrCompilerHitLimits
	default: // general case if no freed states nor issues detected
		resultStateIndex := StateIndex(len(self.states))
		self.states = internal.GrowSliceByOne(self.states)
		return resultStateIndex, nil
	}
}

func (self *DecisionTreeCompiler) appendStateToNextRound(stateIndex StateIndex) {
	self.run.ToStates = internal.BasicOrderedNonRepeatInsert(self.run.ToStates, stateIndex)
}

// The state can't be 0, as 0 is the root, which is treated separately.
func (self *DecisionTreeCompiler) adjustStateUseCount(stateIndex StateIndex, delta int) {
	if stateIndex == 0 { panic(PreViolation) }
	stateIndex -= 1

	var result int
	if len(self.run.TrackerStateUses) <= int(stateIndex) {
		if len(self.run.TrackerStateUses) != int(stateIndex) { panic(BrokenCode) }
		self.run.TrackerStateUses = internal.GrowSliceByOne(self.run.TrackerStateUses)
		result = delta
	} else {
		result = int(self.run.TrackerStateUses[stateIndex]) + delta
	}
	if result > 255 || result < 0 { panic(BrokenCode) }
	self.run.TrackerStateUses[stateIndex] = uint8(result)
}

// The states can't be 0, as 0 is the root, which is treated separately.
func (self *DecisionTreeCompiler) initializeStateRulesWithRef(stateIndex StateIndex, referenceIndex StateIndex) {
	if stateIndex == 0 || referenceIndex == 0 { panic(PreViolation) }
	stateIndex -= 1
	referenceIndex -= 1

	refRules := self.run.TrackerStateRules[referenceIndex] // this can't be empty if I'm right
	numRules := len(refRules) + 1

	// expand tracker state rules slice size if necessary
	if len(self.run.TrackerStateRules) <= int(stateIndex) {
		if len(self.run.TrackerStateRules) != int(stateIndex) { panic(BrokenCode) }
		self.run.TrackerStateRules = internal.GrowSliceByOne(self.run.TrackerStateRules)
	}
	
	// three cases for copying rules and adding the new one
	self.run.TrackerStateRules[stateIndex] = internal.SetSliceSize(self.run.TrackerStateRules[stateIndex], numRules)
	if self.run.RuleIndex < refRules[0] { // set and copy the rest
		self.run.TrackerStateRules[stateIndex][0] = self.run.RuleIndex
		copy(self.run.TrackerStateRules[stateIndex][1 : ], refRules[:])
	} else if self.run.RuleIndex > refRules[len(refRules) - 1] { // copy and set last
		self.run.TrackerStateRules[stateIndex][len(refRules)] = self.run.RuleIndex
		copy(self.run.TrackerStateRules[stateIndex][ : numRules - 1], refRules[:])
	} else { // copy start, set middle, copy end
		insertionIndex, found := internal.BadBinarySearch(refRules, self.run.RuleIndex)
		if found { panic(BrokenCode) }
		copy(self.run.TrackerStateRules[stateIndex][ : insertionIndex], refRules[ : insertionIndex])
		copy(self.run.TrackerStateRules[stateIndex][insertionIndex + 1 : ], refRules[insertionIndex : ])
		self.run.TrackerStateRules[stateIndex][insertionIndex] = self.run.RuleIndex
	}
}

// The states can't be 0, as 0 is the root, which is treated separately.
func (self *DecisionTreeCompiler) initializeStateRules(stateIndex StateIndex) {
	if stateIndex == 0 { panic(PreViolation) }
	stateIndex -= 1

	if len(self.run.TrackerStateRules) <= int(stateIndex) {
		if len(self.run.TrackerStateRules) != int(stateIndex) { panic(BrokenCode) }
		self.run.TrackerStateRules = internal.GrowSliceByOne(self.run.TrackerStateRules)
	}
	self.run.TrackerStateRules[stateIndex] = self.run.TrackerStateRules[stateIndex][ : 0]
	self.run.TrackerStateRules[stateIndex] = append(self.run.TrackerStateRules[stateIndex], self.run.RuleIndex)
}

func (self *DecisionTreeCompiler) registerNewStateAtDepth(stateIndex StateIndex) {
	depth := int(self.run.Depth) // should be + 1, but TrackerStatesByDepth begins at 1
	if len(self.run.TrackerStatesByDepth) <= depth {
		if len(self.run.TrackerStatesByDepth) != depth { panic(BrokenCode) }
		self.run.TrackerStatesByDepth = internal.GrowSliceByOne(self.run.TrackerStatesByDepth)
		self.run.TrackerStatesByDepth[depth] = self.run.TrackerStatesByDepth[depth][ : 0]
	}
	self.run.TrackerStatesByDepth[depth] = internal.BasicOrderedNonRepeatInsert(self.run.TrackerStatesByDepth[depth], stateIndex)
}

// ---- debug ----

func (self *DecisionTreeCompiler) debugNumStateUses(stateIndex StateIndex) int {
	if stateIndex == 0 { return 1 }
	return int(self.run.TrackerStateUses[stateIndex - 1])
}

func (self *DecisionTreeCompiler) debugNumUsedStates() int {
	var count int = 1
	for _, uses := range self.run.TrackerStateUses {
		if uses != 0 { count += 1 }
	}
	return count
}

func ( self *DecisionTreeCompiler) fullDebugPrint() {
	for i, _ := range self.states {
		//fmt.Printf("[%d uses] ", self.debugNumStateUses(StateIndex(i)))
		self.states[i].debugPrint(StateIndex(i))
	}
	for i, rules := range self.run.TrackerStateRules {
		fmt.Printf("(state %d) (uses: %d) (rules: %v)\n", i + 1, self.run.TrackerStateUses[i], rules)
	}
	for i, _ := range self.run.TrackerStatesByDepth {
		fmt.Printf("(states at depth %d) %v\n", i + 1, self.run.TrackerStatesByDepth[i])
	}
}
