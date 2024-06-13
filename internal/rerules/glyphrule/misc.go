package glyphrule

import "github.com/tinne26/ggfnt"
import "github.com/tinne26/ggfnt/internal"
import "github.com/tinne26/ggfnt/internal/rerules"

type GlyphSetIndex uint8
const GlyphSetNone GlyphSetIndex = 255
const GlyphNone ggfnt.GlyphIndex = 65535

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

func IsValidRuleGlyphIndex(glyphIndex ggfnt.GlyphIndex) bool {
	return glyphIndex < internal.MaxGlyphs
}
