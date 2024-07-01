package ggfnt

import "errors"

type GlyphIndex uint16
const (
	GlyphMissing   GlyphIndex = 56789
	GlyphZilch     GlyphIndex = 56790
	GlyphNewLine   GlyphIndex = 56791

	GlyphCustomMin GlyphIndex = 60000
	GlyphCustomMax GlyphIndex = 62000
)

type GlyphRange struct {
	First GlyphIndex // included
	Last  GlyphIndex // included
}

func (self *GlyphRange) Contains(index GlyphIndex) bool {
	return index >= self.First && index <= self.Last
}

func (self *GlyphRange) Validate(maxGlyphs uint16) error {
	if self.First > self.Last { return errors.New("invalid glyph range: First > Last") }
	if uint16(self.Last) >= maxGlyphs { return errors.New("invalid glyph range: Last exceeds maxGlyphs") }
	return nil
}

func NewRange(first, last GlyphIndex) GlyphRange {
	if last < first { panic("invalid glyph range: last < first") }
	return GlyphRange{ First: first, Last: last }
}

type GlyphPlacement struct {
	Advance uint8
	
	// vertical bounds fields: these will be zero
	// unless the font includes vertical layout data
	TopAdvance, BottomAdvance uint8
	HorzCenter uint8 // offset to the glyph's center pixel, from the origin
}

func (self *GlyphPlacement) appendWithoutVertLayout(buffer []byte) []byte {
	return append(buffer, self.Advance)
}

func (self *GlyphPlacement) appendWithVertLayout(buffer []byte) []byte {
	return append(buffer, self.Advance, self.TopAdvance, self.BottomAdvance, self.HorzCenter)
}


