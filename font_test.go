package ggfnt

import "testing"

func TestExpectedParsing(t *testing.T) {
	// TODO: I might need to write font_work.go first? Because creating stuff
	//       manually is kind of a pain, and it might be better to simply do
	//       everything through the built-in types, even if testing gets kinda
	//       circular then.

	// TODO: we pass t to a sub-testing function for parsing
	testingParseFontWithoutErrors(t, nil)
}

func testingParseFontWithoutErrors(t *testing.T, data []byte) {
	// ...
}
