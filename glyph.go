package ggfnt

type GlyphIndex uint16
const (
	GlyphMissing GlyphIndex = 56789
	GlyphZilch   GlyphIndex = 56790
	GlyphNewLine GlyphIndex = 56791
)
func (self GlyphIndex) String() string { panic("unimplemented") }
func (self GlyphIndex) Type() GlyphType { panic("unimplemented") }

type GlyphType uint8
const (
	GlyphTypeNormal  GlyphType = 0b0000_0001
	GlyphTypeControl GlyphType = 0b0000_0010
		// TODO: should we provide any way to distinguish all the subtypes?
		// GlyphTypeControlPredef
		// GlyphTypeControlPredefUndef
		// GlyphTypeControlFont
		// GlyphTypeControlLibrary
		// GlyphTypeControlCustom
		// GlyphTypeControlUndef
	GlyphTypeCustom    GlyphType = 0b0000_0100
	GlyphTypeUndefined GlyphType = 0b0000_1000 // >65k
)

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


