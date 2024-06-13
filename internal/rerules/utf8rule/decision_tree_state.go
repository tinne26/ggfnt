package utf8rule

import "fmt"

import "github.com/tinne26/ggfnt/internal"

type StateIndex uint8
const StateNone StateIndex = 255

type State struct {
	Transitions []Transition
	MinHeadLen uint8
	MaxHeadLen uint8
}

func (self *State) Reset() {
	self.Transitions = self.Transitions[ : 0]
	self.MinHeadLen = 255
	self.MaxHeadLen = 0
}

func (self *State) CopyDataFrom(other State) {
	numTransitions := len(other.Transitions)
	self.Transitions = internal.SetSliceSize(self.Transitions, numTransitions)
	copy(self.Transitions, other.Transitions)

	self.MinHeadLen = other.MinHeadLen
	self.MaxHeadLen = other.MaxHeadLen
}

// Must be called for each rule being applied to the state.
func (self *State) UpdateHeadLen(headLen uint8) {
	if headLen < self.MinHeadLen { self.MinHeadLen = headLen }
	if headLen > self.MaxHeadLen { self.MaxHeadLen = headLen }
}

// Given a coed point, the state returns the next state index
// and the matched rule index with the most recent code point.
func (self *State) Trigger(codePoint rune) (StateIndex, RuleIndex) {
	// find transition with binary search
	transIndex, found := self.findRuneTransition(codePoint)
	if !found { return StateNone, RuleNone }
	transition := self.Transitions[transIndex]
	return transition.NextState, transition.RuleMatch
}

func (self *State) SatisfiesMinRequiredHead(minRequiredHeadLen uint8) bool {
	return minRequiredHeadLen >= self.MinHeadLen && minRequiredHeadLen <= self.MaxHeadLen
}

// The new range might overlap previous empty areas or previously existing ranges,
// both fully, partially, or in many different combinations. We need to get each
// fragment and set up a new transition or adjust an already existing one. This
// is a tricky method.
func (self *State) AppendRange(runeRange RuneRange, ruleMatchIndex RuleIndex, stateLinker StateLinkerFn) error {
	// for the first part, we find candidate transitions and check if our range
	// begins earlier, later, how much overlap it has, etc.
	candidateTrans, _ := self.findRuneTransition(runeRange.First)
	for runeRange.First <= runeRange.Last && int(candidateTrans) < len(self.Transitions) {
		candidateRange := self.Transitions[candidateTrans].Range
		splitResult, splitRange := self.confrontRanges(candidateRange, runeRange)
		switch splitResult {
		case noSplitPre:
			err := self.insertNewTransition(splitRange, ruleMatchIndex, stateLinker, 0)
			if err != nil { return err }
			runeRange.First = splitRange.Last + 1
			// in theory we wouldn't want to change the candidateTrans, but we added
			// a new transition that will be placed exactly on candidateTrans, so
			// we want to skip it
			candidateTrans += 1
		case splitFirstMid:
			err := self.splitTransitionAt(candidateTrans, splitRange.Last)
			if err != nil { return err }
			return self.Transitions[candidateTrans].Link(ruleMatchIndex, stateLinker, 1)
			// ^ we terminate here
		case splitFirstLast:
			err := self.Transitions[candidateTrans].Link(ruleMatchIndex, stateLinker, 0)
			if err != nil { return err }
			runeRange.First = splitRange.Last + 1
			candidateTrans += 1
		case splitMidMid:
			err := self.splitTransitionBy(candidateTrans, splitRange)
			if err != nil { return err }
			return self.Transitions[candidateTrans + 1].Link(ruleMatchIndex, stateLinker, 2)
			// ^ we terminate here
		case splitMidLast:
			err := self.splitTransitionBy(candidateTrans, splitRange)
			if err != nil { return err }
			err  = self.Transitions[candidateTrans + 1].Link(ruleMatchIndex, stateLinker, 1)
			if err != nil { return err }
			runeRange.First = splitRange.Last + 1
			candidateTrans += 1
		case noSplitPost:
			// we have to see if the next transition applies
			candidateTrans += 1
		default:
			panic(BrokenCode)
		}
	}

	// for the final part, we check if there's any remaining range left.
	// if there is, we allocate it a completely new transition at the end
	if runeRange.First > runeRange.Last { return nil }
	return self.insertNewTransition(runeRange, ruleMatchIndex, stateLinker, 0)
}

// ---- internal helper methods ----

// Auxiliary type for AppendRange / confrontRanges.
type splitMode uint8
const (
	noSplitPre     splitMode = 0
	splitFirstMid  splitMode = 1 // from first to not last
	splitFirstLast splitMode = 2 // from first to last
	splitMidMid    splitMode = 3 // from not first to not last
	splitMidLast   splitMode = 4 // from not first to last
	noSplitPost    splitMode = 5
)

// Returns the split mode and the relevant rune range.
func (self *State) confrontRanges(candidateRange, incomingRange RuneRange) (splitMode, RuneRange) {
	if incomingRange.First < candidateRange.First { // noSplitPre
		if incomingRange.Last < candidateRange.First {
			return noSplitPre, incomingRange
		} else {
			return noSplitPre, RuneRange{ incomingRange.First, candidateRange.Last - 1 }
		}
	} else if incomingRange.First == candidateRange.First { // splitFirst*
		if incomingRange.Last >= candidateRange.Last {
			return splitFirstLast, RuneRange{ incomingRange.First, candidateRange.Last }
		} else {
			return splitFirstMid, incomingRange
		}
	} else if incomingRange.First <= candidateRange.Last { // splitMid*
		if incomingRange.Last < candidateRange.Last {
			return splitMidMid, incomingRange
		} else {
			return splitMidLast, RuneRange{ incomingRange.First, candidateRange.Last }
		}
	} else { // noSplitPost
		return noSplitPost, incomingRange
	}
}

func (self *State) insertNewTransition(runeRange RuneRange, ruleMatchIndex RuleIndex, stateLinker StateLinkerFn, numSplits int) error {
	// insert new transition (only runeRange initialized)
	index, err := self.unsafeInsertUninitializedTransition(runeRange)
	if err != nil { return err }

	// initialize remaining transition properties
	self.Transitions[index].RuleMatch = ruleMatchIndex
	if ruleMatchIndex == RuleNone {
		self.Transitions[index].NextState, err = stateLinker(StateNone, numSplits)
		return err
	} else {
		self.Transitions[index].NextState = StateNone
		return nil
	}
}

func (self *State) findRuneTransition(codePoint rune) (TransitionIndex, bool) {
	numTrans := len(self.Transitions)
	minIndex, maxIndex := int(0), int(numTrans) - 1
	for minIndex < maxIndex {
		midIndex := (minIndex + maxIndex) >> 1 // no overflow possible due to numTrans being <255
		
		runeRange := self.Transitions[midIndex].Range
		if codePoint < runeRange.First {
			maxIndex = midIndex
		} else if codePoint > runeRange.Last {
			minIndex = midIndex + 1
		} else {
			return TransitionIndex(midIndex), true
		}
	}

	found := (minIndex < numTrans && self.Transitions[minIndex].Range.Contains(codePoint))
	if found && minIndex > 254 { panic(BrokenCode) } // discretionary safety assertion
	return TransitionIndex(minIndex), found
}

func (self *State) splitTransitionAt(transIndex TransitionIndex, lastRuneOfPre rune) error {
	// shorten existing transition and create new transition for the newly vacated range
	vacatedRange := self.Transitions[transIndex].ShortenRTL(lastRuneOfPre)
	newTransIndex, err := self.unsafeInsertUninitializedTransition(vacatedRange)
	if err != nil { return err }
	if transIndex != newTransIndex + 1 { panic(BrokenCode) }

	// initialize new transition to match the previously existing one
	self.Transitions[newTransIndex].NextState = self.Transitions[transIndex].NextState
	self.Transitions[newTransIndex].RuleMatch = self.Transitions[transIndex].RuleMatch
	return nil
}

func (self *State) splitTransitionBy(transIndex TransitionIndex, runeRange RuneRange) error {
	// note: there are more efficient ways to do this, but this 
	//       is comfy and kinda acceptable for the moment I guess
	err := self.splitTransitionAt(transIndex, runeRange.First - 1)
	if err != nil { return err }
	return self.splitTransitionAt(transIndex + 1, runeRange.Last)
}

// Helper method for insertNewTransition() and splitTransition*.
// The transition might have garbage on the NextState and RuleMatch fields,
// so they have to be manually set afterwards.
func (self *State) unsafeInsertUninitializedTransition(runeRange RuneRange) (TransitionIndex, error) {
	// ensure new transition within bounds
	newLastIndex := len(self.Transitions)
	if newLastIndex >= 254 { return TransitionNone, ErrCompilerHitLimits }

	// grow slice and find new transition position
	self.Transitions = internal.GrowSliceByOne(self.Transitions)
	orderedIndex := newLastIndex
	for orderedIndex > 0 {
		if self.Transitions[orderedIndex - 1].Range.First < runeRange.First {
			break
		} else {
			orderedIndex -= 1
		}
	}

	// shift data
	if orderedIndex != newLastIndex {
		copy(self.Transitions[orderedIndex + 1 : ], self.Transitions[orderedIndex : ])
	}

	// set only rune range
	self.Transitions[orderedIndex].Range = runeRange
	return TransitionIndex(orderedIndex), nil
}

// ---- debug ----

func (self *State) debugPrint(index StateIndex) {
	fmt.Printf("[state %03d] (head %d/%d)\n", index, self.MinHeadLen, self.MaxHeadLen)
	for i, _ := range self.Transitions {
		fmt.Print("\t")
		fmt.Print(self.Transitions[i].debugString(TransitionIndex(i)))
		fmt.Print("\n")
	}
}
