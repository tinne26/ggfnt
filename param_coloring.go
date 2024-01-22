package ggfnt

import "image/color"

// The full range of combined colors and palettes and so on,
// parametrized for a given font.
// TODO: I could technically include this into Parametrization
//       without any separation... it's unclear if I want the
//       renderer to have individualized control over this,
//       of if I want the parametrization to deal with it, or
//       if I want to expose shortcut methods or what.
type Coloring struct {
	palette [1024]byte
	dyes []color.RGBA
}

// (while I need to create this from an actual Font object, I 
// should do it alongside the Parametrization init, privately?)
// func NewColoring(font *Font) *Coloring {
// 	return &Coloring{
// 		// ...
// 	}
// 	// TODO
// }

func (self *Coloring) SetPalette(font *Font, key PaletteKey, colors ...color.RGBA) error {
	panic("unimplemented")
}

func (self *Coloring) GetPalette(font *Font, key PaletteKey) (Palette, bool) {
	panic("unimplemented")
}

func (self *Coloring) SetDye(font *Font, key DyeKey, value color.RGBA) error {
	panic("unimplemented")
}

func (self *Coloring) GetDye(font *Font, key DyeKey) (color.RGBA, bool) {
	panic("unimplemented")
}
