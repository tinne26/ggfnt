package glyphrule

import "testing"

import "github.com/tinne26/ggfnt"

func TestRuleScanner(t *testing.T) {
	var scanner RuleScanner

	tests := []struct{
		ReRuleDef []uint8 // glyph rewrite rule raw definition
		HeadLen uint8
		StartBlock RuleBlock
		Condition uint8
		NextGlyphs []ggfnt.GlyphIndex
		NextSets []GlyphSetIndex
	}{
		{ // simple rule replacing two glyph with one, only main block
			ReRuleDef: []uint8{
				255, // condition
				0, 2, 0, 1, // block and output lenghts
				11, 0, // output
				0b0000_0000, // head control
				0b0000_0010, // body control
				12, 0, 13, 0, // body content
				0b0000_0000, // tail control
			},
			HeadLen: 0,
			StartBlock: RuleBody,
			Condition: 255,
			NextGlyphs: []ggfnt.GlyphIndex{12, 13},
			NextSets: []GlyphSetIndex{GlyphSetNone, GlyphSetNone},
		},
		{ // simple rule replacing one glyph with one, only main block
			ReRuleDef: []uint8{
				255, // condition
				0, 1, 0, 1, // block and output lenghts
				6, 0, // output
				0b0000_0000, // head control
				0b0000_0001, // body control
				7, 0, // body content
				0b0000_0000, // tail control
			},
			HeadLen: 0,
			StartBlock: RuleBody,
			Condition: 255,
			NextGlyphs: []ggfnt.GlyphIndex{7},
			NextSets: []GlyphSetIndex{GlyphSetNone},
		},
		{ // replace glyph if head satisfied
			ReRuleDef: []uint8{
				0, // condition
				1, 1, 0, 1, // block and output lenghts
				8, 0, // output
				0b0000_0001, // head control
				3, 0, // head content
				0b0000_0001, // body control
				4, 0, // body content
				0b0000_0000, // tail control
			},
			HeadLen: 1,
			StartBlock: RuleHead,
			Condition: 0,
			NextGlyphs: []ggfnt.GlyphIndex{3, 4},
			NextSets: []GlyphSetIndex{GlyphSetNone, GlyphSetNone},
		},
		{ // replace glyph if tail satisfied
			ReRuleDef: []uint8{
				2, // condition
				0, 1, 1, 1, // block and output lenghts
				5, 0, // output
				0b0000_0000, // head control
				0b0000_0001, // body control
				6, 0, // body content
				0b0000_0001, // tail control
				7, 0,
			},
			HeadLen: 0,
			StartBlock: RuleBody,
			Condition: 2,
			NextGlyphs: []ggfnt.GlyphIndex{6, 7},
			NextSets: []GlyphSetIndex{GlyphSetNone, GlyphSetNone},
		},
		{ // replace glyph if both head and tail satisfied
			ReRuleDef: []uint8{
				1, // condition
				1, 2, 1, 2, // block and output lenghts
				4, 0, 5, 0, // output
				0b0000_0001, // head control
				3, 0, // head content
				0b0000_0010, // body control
				6, 0, 7, 0, // body content
				0b0000_0001, // tail control
				8, 0, // tail content
			},
			HeadLen: 1,
			StartBlock: RuleHead,
			Condition: 1,
			NextGlyphs: []ggfnt.GlyphIndex{3, 6, 7, 8},
			NextSets: []GlyphSetIndex{GlyphSetNone, GlyphSetNone, GlyphSetNone, GlyphSetNone},
		},
	}

	for n, test := range tests {
		rule := ggfnt.GlyphRewriteRule{ Data: test.ReRuleDef }
		err := scanner.Start(rule)
		
		if err != nil { t.Fatalf("t#%d unexpected start error: %s", n, err) }
		if scanner.HeadLen() != test.HeadLen {
			t.Fatalf("t#%d expected head length %d, got %d", n, test.HeadLen, scanner.HeadLen())
		}
	
		if scanner.Block() != test.StartBlock {
			t.Fatalf("t#%d expected start block %d, got %d", n, test.StartBlock, scanner.Block())
		}
	
		if scanner.Condition() != test.Condition {
			t.Fatalf("t#%d expected condition %d, got %d", n, test.Condition, scanner.Condition())
		}
		
		for i := 0; i < len(test.NextGlyphs); i++ {
			if !scanner.HasNext() {
				t.Fatalf("t#%d expected scanner to have next character", n)
			}
			glyphIndex, glyphSet, err := scanner.Next()
			if err != nil {
				t.Fatalf("t#%d unexpected error on next#%d: %s", n, i, err)
			}
			if glyphIndex != test.NextGlyphs[i] {
				t.Fatalf("t#%d expected glyph index %d, got %d instead", n, test.NextGlyphs[i], glyphIndex)
			}
			if glyphSet != test.NextSets[i] {
				t.Fatalf("t#%d expected glyph set %d, got %d instead", n, test.NextSets[i], glyphSet)
			}
		}

		if scanner.HasNext() {
			t.Fatalf("t#%d expected rule end, but it keeps going", n)
		}
	}
}

func TestRuleScannerMustFail(t *testing.T) {
	var scanner RuleScanner

	tests := [][]uint8{
		{
			255, // condition
			1, 1, 1, 1, // block and output lenghts
			8, 0, // output
			0b0000_0001, // head control
			1, 0, 
			0b0000_0001, // body control
			2, 0, // body content
			0b0000_0000, // tail control
		},
		{
			255, // condition
			1, 1, 1, 1, // block and output lenghts
			8, 0, // output
			0b0000_0010, // head control
			1, 0, 
			0b0000_0001, // body control
			2, 0, // body content
			0b0000_0001, // tail control
			1, 0,
		},
		{
			255, // condition
			1, 1, 1, 1, // block and output lenghts
			8, 0, // output
			0b0000_0001, // head control
			1, 0, 
			0b0000_0010, // body control
			1, 0, // body content
			0b0000_0001, // tail control
			1, 0,
		},
		{
			255, // condition
			1, 1, 1, 1, // block and output lenghts
			8, 0, // output
			0b0000_0001, // head control
			1, 0, 
			0b0000_0001, // body control
			1, 0, // body content
			0b0000_0010, // tail control
			1, 0,
		},
	}

	for i, test := range tests {
		rule := ggfnt.GlyphRewriteRule{ Data: test }
		err := scanner.Start(rule)
		if err != nil { continue }
		var failed bool
		for scanner.HasNext() {
			_, _, err := scanner.Next()
			if err != nil { failed = true ; break }
		}
		if !failed {
			t.Fatalf("test#%d expected invalid rule, but no error detected", i)
		}
	}
}
