package glyphrule

import "testing"

import "github.com/tinne26/ggfnt"

func TestDecisionTree(t *testing.T) {
	var tree DecisionTree
	var compiler DecisionTreeCompiler

	var font *ggfnt.Font
	var rule ggfnt.GlyphRewriteRule
	var err error

	// reused rules
	ruleData1a2to3 := []uint8{
		255, // condition
		0, 2, 0, 1, // block and output lenghts
		3, 0, // output
		0b0000_0000, // head control
		0b0000_0010, // body control
		1, 0, 2, 0, // body content
		0b0000_0000, // tail control
	}
	ruleData1a2a3to4 := []uint8{
		255, // condition
		0, 3, 0, 1, // block and output lenghts
		4, 0, // output
		0b0000_0000, // head control
		0b0000_0011, // body control
		1, 0, 2, 0, 3, 0, // body content
		0b0000_0000, // tail control
	}
	ruleData2a3to5 := []uint8{
		255, // condition
		0, 2, 0, 1, // block and output lenghts
		5, 0, // output
		0b0000_0000, // head control
		0b0000_0010, // body control
		2, 0, 3, 0, // body content
		0b0000_0000, // tail control
	}

	// tests table
	var tests = []struct{
		InRuleData1 []uint8
		InRuleData2 []uint8
		InGlyphs []ggfnt.GlyphIndex
		OutMatch RuleIndex
	}{
		{
			InRuleData1: ruleData1a2to3,
			InRuleData2: nil,
			InGlyphs: []ggfnt.GlyphIndex{ 0, 1, 2, 3, 4, 5, 6, 7, 8, 9 },
			OutMatch: RuleNone,
		},
		{
			InRuleData1: ruleData1a2to3,
			InRuleData2: nil,
			InGlyphs: []ggfnt.GlyphIndex{ 1, 2, 3, 4, 5, 6, 7, 8, 9 },
			OutMatch: 0,
		},
		{
			InRuleData1: ruleData1a2to3,
			InRuleData2: ruleData1a2a3to4,
			InGlyphs: []ggfnt.GlyphIndex{ 1, 2, 3, 4, 5, 6, 7, 8, 9 },
			OutMatch: 1,
		},
		{
			InRuleData1: ruleData1a2a3to4,
			InRuleData2: ruleData2a3to5,
			InGlyphs: []ggfnt.GlyphIndex{ 1, 2, 3, 4, 1, 2, 3 },
			OutMatch: 0,
		},
		{
			InRuleData1: ruleData2a3to5,
			InRuleData2: nil,
			InGlyphs: []ggfnt.GlyphIndex{ 2, 3, 4, 1, 2, 3 },
			OutMatch: 0,
		},
		{
			InRuleData1: ruleData1a2a3to4,
			InRuleData2: ruleData2a3to5,
			InGlyphs: []ggfnt.GlyphIndex{ 2, 3, 4, 1, 2, 3 },
			OutMatch: 1,
		},
	}

	// run tests
	for i, test := range tests {
		// compile
		compiler.Begin(font, tree.states)
		rule.Data = test.InRuleData1
		err = compiler.Feed(rule, 0)
		if err != nil { t.Fatalf("test#%d rule 1 feed failure: %s", i, err) }
		if test.InRuleData2 != nil {
			rule.Data = test.InRuleData2
			err = compiler.Feed(rule, 1)
			if err != nil { t.Fatalf("test#%d rule 2 feed failure: %s", i, err) }
		}
		tree.states = compiler.Finish()

		// test
		matchedRule := tree.Match(test.InGlyphs)
		if matchedRule != test.OutMatch {
			t.Fatalf("test#%d expected to match %d, got %d", i, test.OutMatch, matchedRule)
		}
	}
}
