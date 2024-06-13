package utf8rule

import "github.com/tinne26/ggfnt/internal/rerules"

type RuneRange struct {
	First rune
	Last rune
}

func (self *RuneRange) Contains(codePoint rune) bool {
	return codePoint >= self.First && codePoint <= self.Last
}

const RuneError = '\uFFFD'
type Utf8SetIndex uint8
const Utf8SetNone Utf8SetIndex = 255

type RuleIndex uint8
const RuleNone RuleIndex = 255

type RuleBlock uint8
const (
	RuleHead RuleBlock = 0
	RuleBody RuleBlock = 1
	RuleTail RuleBlock = 2
)

func (self RuleBlock) Next() (RuleBlock, error) {
	switch self {
	case RuleHead: return RuleBody, nil
	case RuleBody: return RuleTail, nil
	default:
		return 0, rerules.ErrInvalidReRule
	}
}

const BrokenCode = "broken code"
const PreViolation = "precondition violation"

func IsValidRuleCodePoint(codePoint rune) bool {
	return codePoint != RuneError
}
