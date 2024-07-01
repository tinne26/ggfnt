package builder

import "image"

import "github.com/tinne26/ggfnt"

type glyphData struct {
	Name string // can be empty
	Placement ggfnt.GlyphPlacement
	Mask *image.Alpha
}

type settingEntry struct {
	Name string
	Values []uint8
}

// --- edition subtypes ---

type editionCategory struct {
	Name string
	Size uint16
}

type editionKerningPair struct {
	First uint64
	Second uint64
	Class uint16 // if 0, value must be used instead
	Value int8
}
func (self *editionKerningPair) HasClass() bool {
	return self.Class != 0
}

type editionKerningClass struct {
	Name string
	Value int8
}
