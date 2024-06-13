package glyphrule

import "testing"

import "github.com/tinne26/ggfnt"

func TestDecisionTreeCompiler(t *testing.T) {
	var err error
	var compiler DecisionTreeCompiler
	var states []State = make([]State, 0, 32)
	var font *ggfnt.Font
	var rule ggfnt.GlyphRewriteRule

	// NOTICE: this test is not reliable at all, it only checks
	//         the number of output states. I manually checked
	//         the results with DebugPrint()s first, though.

	// test #1
	rule.Data = []uint8{
		255, // condition
		0, 1, 0, 1, // block and output lenghts
		11, 0, // output
		0b0000_0000, // head control
		0b0000_0001, // body control
		12, 0, // body content
		0b0000_0000, // tail control
	}
	
	compiler.Begin(font, states)
	err = compiler.Feed(rule, 0)
	if err != nil { t.Fatalf("test#1 unexpected error: %s", err) }
	states = compiler.Finish()
	if compiler.debugNumUsedStates() != 1 {
		t.Fatalf("expected %d states, found %d", 1, compiler.debugNumUsedStates())
	}

	// test #2
	compiler.Begin(font, states)
	
	rule.Data = []uint8{
		255, // condition
		0, 2, 0, 1, // block and output lenghts
		13, 0, // output
		0b0000_0000, // head control
		0b0000_0010, // body control
		11, 0, 12, 0, // body content
		0b0000_0000, // tail control
	}
	err = compiler.Feed(rule, 0)
	if err != nil { t.Fatalf("test#2 unexpected error: %s", err) }

	rule.Data = []uint8{
		255, // condition
		0, 2, 0, 1, // block and output lenghts
		9, 0, // output
		0b0000_0000, // head control
		0b0000_0010, // body control
		11, 0, 10, 0, // body content
		0b0000_0000, // tail control
	}
	err = compiler.Feed(rule, 1)
	if err != nil { t.Fatalf("test#2 unexpected error: %s", err) }

	states = compiler.Finish()
	if compiler.debugNumUsedStates() != 2 {
		t.Fatalf("expected %d states, found %d", 2, compiler.debugNumUsedStates())
	}

	// test #3
	compiler.Begin(font, states)
	
	rule.Data = []uint8{
		255, // condition
		0, 2, 0, 1, // block and output lenghts
		13, 0, // output
		0b0000_0000, // head control
		0b0000_0010, // body control
		11, 0, 12, 0, // body content
		0b0000_0000, // tail control
	}
	err = compiler.Feed(rule, 0)
	if err != nil { t.Fatalf("test#3 unexpected error: %s", err) }

	rule.Data = []uint8{
		255, // condition
		0, 2, 0, 1, // block and output lenghts
		9, 0, // output
		0b0000_0000, // head control
		0b0000_0010, // body control
		10, 0, 11, 0, // body content
		0b0000_0000, // tail control
	}
	err = compiler.Feed(rule, 1)
	if err != nil { t.Fatalf("test#3 unexpected error: %s", err) }

	states = compiler.Finish()
	if compiler.debugNumUsedStates() != 3 {
		t.Fatalf("expected %d states, found %d", 3, compiler.debugNumUsedStates())
	}

	// test #4 (complex, including heads)
	compiler.Begin(font, states)
	
	rule.Data = []uint8{
		255, // condition
		1, 2, 0, 1, // block and output lenghts
		30, 0, // output
		0b0000_0001, // head control
		8, 0,
		0b0000_0010, // body control
		9, 0, 10, 0, // body content
		0b0000_0000, // tail control
	}
	err = compiler.Feed(rule, 0)
	if err != nil { t.Fatalf("test#4 unexpected error: %s", err) }

	rule.Data = []uint8{
		255, // condition
		0, 2, 0, 2, // block and output lenghts
		31, 0, 32, 0, // output
		0b0000_0000, // head control
		0b0000_0010, // body control
		8, 0, 9, 0, // body content
		0b0000_0000, // tail control
	}
	err = compiler.Feed(rule, 1)
	if err != nil { t.Fatalf("test#4 unexpected error: %s", err) }

	rule.Data = []uint8{
		255, // condition
		2, 1, 0, 1, // block and output lenghts
		33, 0, // output
		0b0000_0010, // head control
		4, 0, 9, 0,
		0b0000_0001, // body control
		8, 0, // body content
		0b0000_0000, // tail control
	}
	err = compiler.Feed(rule, 2)
	if err != nil { t.Fatalf("test#4 unexpected error: %s", err) }

	states = compiler.Finish()
	if compiler.debugNumUsedStates() != 5 {
		t.Fatalf("expected %d states, found %d", 5, compiler.debugNumUsedStates())
	}

	// test #5
	compiler.Begin(font, states)
	rule.Data = []uint8{
		255, // condition
		0, 2, 0, 1, // block and output lenghts
		5, 0, // output
		0b0000_0000, // head control
		0b0000_0010, // body control
		2, 0, 3, 0, // body content
		0b0000_0000, // tail control
	}
	err = compiler.Feed(rule, 0)
	if err != nil { t.Fatalf("test#5 unexpected error: %s", err) }
	
	states = compiler.Finish()
	if compiler.debugNumUsedStates() != 2 {
		t.Fatalf("expected %d states, found %d", 2, compiler.debugNumUsedStates())
	}

	// test #6
	compiler.Begin(font, states)
	rule.Data = []uint8{
		255, // condition
		0, 3, 0, 1, // block and output lenghts
		4, 0, // output
		0b0000_0000, // head control
		0b0000_0011, // body control
		1, 0, 2, 0, 3, 0, // body content
		0b0000_0000, // tail control
	}
	err = compiler.Feed(rule, 0)
	if err != nil { t.Fatalf("test#6 unexpected error: %s", err) }

	rule.Data = []uint8{
		255, // condition
		0, 2, 0, 1, // block and output lenghts
		5, 0, // output
		0b0000_0000, // head control
		0b0000_0010, // body control
		2, 0, 3, 0, // body content
		0b0000_0000, // tail control
	}
	err = compiler.Feed(rule, 1)
	if err != nil { t.Fatalf("test#6 unexpected error: %s", err) }

	states = compiler.Finish()
	if compiler.debugNumUsedStates() != 4 {
		t.Fatalf("expected %d states, found %d", 4, compiler.debugNumUsedStates())
	}

	// use the following code right before compiler.Finish()
	// to see the internal state and debug potential issues:
	// compiler.fullDebugPrint()
}

func TestDecisionTreeCompilerMustFail(t *testing.T) {
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

	var compiler DecisionTreeCompiler
	var states []State = make([]State, 0, 32)
	var font *ggfnt.Font
	for i, test := range tests {
		rule := ggfnt.GlyphRewriteRule{ Data: test }
		compiler.Begin(font, states)
		err := compiler.Feed(rule, 0)
		if err == nil {
			t.Fatalf("test #%d expected failure, but no error detected", i)
		}
		states = compiler.Finish()
	}
}
