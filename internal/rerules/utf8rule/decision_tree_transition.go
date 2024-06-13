package utf8rule

import "fmt"

type TransitionIndex uint8
const TransitionNone TransitionIndex = 255

type Transition struct {
	Range RuneRange
	RuleMatch RuleIndex // RuleNone (255) is reserved for "none"
	NextState StateIndex // StateNone (255) is reserved for "none"
}

func (self *Transition) Link(ruleMatchIndex RuleIndex, stateLinker StateLinkerFn, numSplits int) error {
	if ruleMatchIndex != RuleNone { // on rule match, state addition stops
		if self.RuleMatch == RuleNone {
			self.RuleMatch = ruleMatchIndex
		}
		return nil
	} else { // add next state
		var err error
		self.NextState, err = stateLinker(self.NextState, numSplits)
		return err
	}
}

// Shorten the transition by moving the Range.Last more towards the beginning.
// Returns the newly vacated range.
// 
// Precondition: self.Range.First <= lastRuneOfPre < self.Range.Last
func (self *Transition) ShortenRTL(lastRuneOfPre rune) RuneRange {
	// discretionary safety assertion
	if lastRuneOfPre < self.Range.First { panic(PreViolation) }
	if lastRuneOfPre >= self.Range.Last { panic(PreViolation) }

	// adjust range and return new vacated range
	preLast := self.Range.Last
	self.Range.Last = lastRuneOfPre
	return RuneRange{ First: lastRuneOfPre + 1, Last: preLast }
}

// ---- debug ----

func (self *Transition) debugString(index TransitionIndex) string {
	if self.RuleMatch != RuleNone {
		if self.NextState == StateNone {
			return fmt.Sprintf("[trans. %03d] (%d - %d) => {match rule %d}", index, self.Range.First, self.Range.Last, self.RuleMatch)
		}
		return fmt.Sprintf("[trans. %03d] (%d - %d) => %d {match rule %d}", index, self.Range.First, self.Range.Last, self.NextState, self.RuleMatch)
	} else {
		return fmt.Sprintf("[trans. %03d] (%d - %d) => %d", index, self.Range.First, self.Range.Last, self.NextState)
	}
}
