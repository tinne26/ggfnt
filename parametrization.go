package ggfnt

import "image"

// A parametrization wraps a completely static [Font] with the
// values related to it that can change at runtime or between
// different instantiations of the font. These values include
// variables, custom glyphs, scale, interspacing values, etc.
//
// In other words: a [Font] is read-only and only has getters;
// a parametrization adds setters for a few font-related fields
// and configurations.
type Parametrization struct {
	font *Font
	variables []uint8
	customGlyphs []*image.Alpha
	scale uint8
	interVertShift int8 // vertical interspacing shift
	interHorzShift int8 // horz interspacing shift
	coloring *Coloring
}

// Returns the underlying [Font].
func (self *Parametrization) Font() *Font { return self.font }

func (self *Parametrization) Coloring() *Coloring { return self.coloring }
// TODO: maybe also Clone(). I was thinking about some "empty sibling", but
//       there's no real need, I can already do font.Parametrize()

// Sets a variable value. See also [FontVariables].
func (self *Parametrization) SetVar(key VarKey, value uint8) error { panic("unimplemented") }
func (self *Parametrization) GetVar(key VarKey) (uint8, error) { panic("unimplemented") }
func (self *Parametrization) SetScale(scale uint8) { panic("unimplemented") } // panics if 0
func (self *Parametrization) GetScale() uint8 { panic("unimplemented") }
func (self *Parametrization) SetHorzInterShift(int8) { panic("unimplemented") }
func (self *Parametrization) GetHorzInterShift() int8 { panic("unimplemented") }
func (self *Parametrization) SetVertInterShift(int8) { panic("unimplemented") }
func (self *Parametrization) GetVertInterShift() int8 { panic("unimplemented") }
func (self *Parametrization) Rescale(scale uint8) { panic("unimplemented") } // like SetScale, but also rescales interspacing shifts

// For custom glyphs. Mask bounds are what determine the positioning. Glyphs that are
// too big simply can't be added as glyphs. This is mostly for icons and stuff.
func (self *Parametrization) AddGlyph(mask *image.Alpha) (GlyphIndex, error) {
	panic("unimplemented")
}
