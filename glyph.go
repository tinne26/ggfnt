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

type GlyphBounds struct {
	MaskWidth, MaskHeight uint8
	LeftOffset, RightOffset int8
	
	// vertical bounds fields: these will be zero
	// unless the font includes vertical layout data
	TopOffset, BottomOffset int8
	VertHorzOffset int8
}

func (self *GlyphBounds) appendWithoutVertLayout(buffer []byte) []byte {
	return append(buffer, self.MaskWidth, self.MaskHeight, uint8(self.LeftOffset), uint8(self.RightOffset))
}

func (self *GlyphBounds) appendWithVertLayout(buffer []byte) []byte {
	return append(
		buffer,
		self.MaskWidth, self.MaskHeight, uint8(self.LeftOffset), uint8(self.RightOffset), // regular bounds
		uint8(self.TopOffset), uint8(self.BottomOffset), uint8(self.VertHorzOffset), // vert-layout related
	)
}


