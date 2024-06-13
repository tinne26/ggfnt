package glyphrule

import "testing"

import "slices"

import "github.com/tinne26/ggfnt"

func TestTester(t *testing.T) {
	var font *ggfnt.Font
	var settingsCache *ggfnt.SettingsCache

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
	ruleData1a2t5to6 := []uint8{
		255, // condition
		0, 2, 1, 1, // block and output lenghts
		6, 0, // output
		0b0000_0000, // head control
		0b0000_0010, // body control
		1, 0, 2, 0, // body content
		0b0000_0001, // tail control
		5, 0,
	}
	ruleData1a2t1to9 := []uint8{
		255, // condition
		0, 2, 1, 1, // block and output lenghts
		9, 0, // output
		0b0000_0000, // head control
		0b0000_0010, // body control
		1, 0, 2, 0, // body content
		0b0000_0001, // tail control
		1, 0,
	}
	ruleData_h1a2to8 := []uint8{
		255, // condition
		1, 1, 0, 1, // block and output lenghts
		8, 0, // output
		0b0000_0001, // head control
		1, 0, 
		0b0000_0001, // body control
		2, 0, // body content
		0b0000_0000, // tail control
	}

	// tests table
	type inOut struct {
		Input []ggfnt.GlyphIndex
		Output []ggfnt.GlyphIndex
	}

	var tests = []struct{
		Rules [][]uint8
		Tests []inOut
	}{
		{ // tester test set 0 (basic detection)
			Rules: [][]uint8{ruleData1a2to3},
			Tests: []inOut{
				{
					Input: []ggfnt.GlyphIndex{1, 2},
					Output: []ggfnt.GlyphIndex{3},
				},
				{
					Input: []ggfnt.GlyphIndex{0, 1, 2},
					Output: []ggfnt.GlyphIndex{0, 3},
				},
				{
					Input: []ggfnt.GlyphIndex{0, 1, 2, 3},
					Output: []ggfnt.GlyphIndex{0, 3, 3},
				},
				{
					Input: []ggfnt.GlyphIndex{1, 2, 6},
					Output: []ggfnt.GlyphIndex{3, 6},
				},
			},
		},
		{ // tester test set 1 (multiple rules detection)
			Rules: [][]uint8{ruleData1a2to3, ruleData1a2a3to4},
			Tests: []inOut{
				{
					Input: []ggfnt.GlyphIndex{1, 2, 3},
					Output: []ggfnt.GlyphIndex{4},
				},
				{
					Input: []ggfnt.GlyphIndex{0, 1, 2, 1, 2, 3},
					Output: []ggfnt.GlyphIndex{0, 3, 4},
				},
				{
					Input: []ggfnt.GlyphIndex{0, 1, 2, 3, 0, 1, 2},
					Output: []ggfnt.GlyphIndex{0, 4, 0, 3},
				},
			},
		},
		{ // tester test set 2 (multiple rules with overlap)
			Rules: [][]uint8{ruleData1a2to3, ruleData1a2t5to6},
			Tests: []inOut{
				{
					Input: []ggfnt.GlyphIndex{1, 2, 5},
					Output: []ggfnt.GlyphIndex{6, 5},
				},
				{
					Input: []ggfnt.GlyphIndex{0, 1, 2, 1, 2, 5, 3},
					Output: []ggfnt.GlyphIndex{0, 3, 6, 5, 3},
				},
				{
					Input: []ggfnt.GlyphIndex{0, 1, 2, 5, 1, 2},
					Output: []ggfnt.GlyphIndex{0, 6, 5, 3},
				},
			},
		},
		{ // tester test set 3 (multiple rules, tail/head overlap)
			Rules: [][]uint8{ruleData1a2t1to9, ruleData_h1a2to8},
			Tests: []inOut{
				{
					Input: []ggfnt.GlyphIndex{1, 2, 3},
					Output: []ggfnt.GlyphIndex{1, 8, 3},
				},
				{
					Input: []ggfnt.GlyphIndex{0, 1, 2, 1},
					Output: []ggfnt.GlyphIndex{0, 9, 1},
				},
			},
		},
		// ...
	}

	// run tests
	var tester Tester
	var outBuffer []ggfnt.GlyphIndex
	for testIndex, test := range tests {
		tester.RemoveAllRules()
		for i, ruleData := range test.Rules {
			var rule ggfnt.GlyphRewriteRule
			rule.Data = ruleData
			err := tester.AddRule(rule)
			if err != nil {
				t.Fatalf("test#%d, on AddRule#%d: %s", testIndex, i, err)
			}
		}

		for subtestIndex, subtest := range test.Tests {
			outBuffer = outBuffer[ : 0]
			var fn GlyphConfirmationFunc = func(glyphIndex ggfnt.GlyphIndex) {
				outBuffer = append(outBuffer, glyphIndex)
			}

			err := tester.BeginSequence(font, settingsCache)
			if err != nil {
				t.Fatalf("test#%d, subtest#%d, unexpected error on BeginSequence(): %s", testIndex, subtestIndex, err)
			}

			for i, glyphIndex := range subtest.Input {
				err := tester.Feed(glyphIndex, fn)
				if err != nil {
					t.Fatalf("test#%d, subtest#%d, input %#v, glyph#%d | feed error: %s", testIndex, subtestIndex, subtest.Input, i, err)
				}
			}
			tester.FinishSequence(fn)

			if !slices.Equal(outBuffer, subtest.Output) {
				t.Fatalf("test#%d, subtest#%d, input %#v, expected out %#v, got %#v", testIndex, subtestIndex, subtest.Input, subtest.Output, outBuffer)
			}
		}
	}
}
