package builder

import "errors"
import "image/color"

// --- dyes ---

type dyeSection struct {
	name string
	alphas []uint8
}

func (self *Font) EachDyeAlpha(name string, fn func(alpha, index uint8)) error {
	clrIndex := 255
	for index, _ := range self.dyes {
		// color index check
		if len(self.dyes[index].alphas) > clrIndex {
			return errors.New("dye sections exceed 255 alpha values")
		}

		// section check
		if self.dyes[index].name == name {
			for index, alpha := range self.dyes[index].alphas {
				fn(alpha, uint8(clrIndex - index)) // TODO: this might be an inverse order...
			}
		} else {
			clrIndex -= len(self.dyes[index].alphas)
		}
	}
	return errors.New("dye section not found")
}

func (self *Font) AddDye(name string, alphas ...uint8) error {
	// ensure no name collisions
	err := self.checkColorSectionNameCollision(name)
	if err != nil { return err }
	count := self.getColorIndexCount()
	if count + len(alphas) > 255 {
		return errors.New("font colors can't exceed 255 indices")
	}
	dyeSection := dyeSection{ name: name }
	dyeSection.alphas = make([]uint8, len(alphas))
	copy(dyeSection.alphas, alphas)
	self.dyes = append(self.dyes, dyeSection)
	return nil
}

// --- palettes ---

type paletteSection struct {
	name string
	colors []color.RGBA
}

func (self *Font) EachPaletteColor(name string, fn func(rgba color.RGBA, index uint8)) error {
	// skip dye sections first
	clrIndex := 255
	for index, _ := range self.dyes {
		clrIndex -= len(self.dyes[index].alphas)
	}
	if clrIndex < 0 { return errors.New("dye sections exceed 255 alpha values") }

	// iterate palette sections
	for index, _ := range self.palettes {
		// color index check
		if len(self.palettes[index].colors) > clrIndex {
			return errors.New("dye + palette sections exceed 255 alpha values")
		}

		// section check
		if self.palettes[index].name == name {
			for index, clr := range self.palettes[index].colors {
				fn(clr, uint8(clrIndex - index)) // TODO: this might be an inverse order...
			}
		} else {
			clrIndex -= len(self.dyes[index].alphas)
		}
	}
	return errors.New("palette section not found")
}

func (self *Font) AddPalette(name string, colors ...color.RGBA) error {
	// ensure no name collisions
	err := self.checkColorSectionNameCollision(name)
	if err != nil { return err }
	count := self.getColorIndexCount()
	if count + len(colors) > 255 {
		return errors.New("font colors can't exceed 255 indices")
	}
	paletteSection := paletteSection{ name: name }
	paletteSection.colors = make([]color.RGBA, len(colors))
	copy(paletteSection.colors, colors)
	self.palettes = append(self.palettes, paletteSection)
	return nil
}

// func (self *Font) RenameColorSection(oldName, newName string) error {
// 	// TODO
// }

// --- shared helpers ---

func (self *Font) checkColorSectionNameCollision(name string) error {
	for index, _ := range self.dyes {
		if self.dyes[index].name == name {
			return errors.New("the given name is already taken by an existing dye section")
		}
	}
	for index, _ := range self.palettes {
		if self.palettes[index].name == name {
			return errors.New("the given name is already taken by an existing palette section")
		}
	}
	return nil
}

func (self *Font) getColorIndexCount() int {
	var count int
	for index, _ := range self.dyes {
		count += len(self.dyes[index].alphas)
	}
	for index, _ := range self.palettes {
		count += len(self.palettes[index].colors)
	}
	return count
}
