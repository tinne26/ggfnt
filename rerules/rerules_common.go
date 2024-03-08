package rerules

import "strconv"

import "github.com/tinne26/ggfnt"

const fsmFlagUnsynced  = 0b01
const fsmFlagOperating = 0b10
const brokenCode = "broken code"

// you still need to manually check for glyph zilch first
func ensureValidFsmGlyphIndex(glyphIndex ggfnt.GlyphIndex) {
	if glyphIndex <= ggfnt.GlyphMissing { return }
	panic("unxpected control glyph index '" + strconv.Itoa(int(glyphIndex)) + "'")
}
