package builder

import "fmt"
import "image"
import "errors"
import "unicode/utf8"

import "github.com/tinne26/ggfnt"
import "github.com/tinne26/ggfnt/mask"
import "github.com/tinne26/ggfnt/internal"

// ---- header ----

func (self *Font) GetFontID() uint64 { return self.fontID }
func (self *Font) GetFontIDStr() string { return fmt.Sprintf("%016X", self.fontID) }
func (self *Font) GetVersionMajor() uint16 { return self.versionMajor }
func (self *Font) GetVersionMinor() uint16 { return self.versionMinor }
func (self *Font) GetVersionStr() string {
	return fmt.Sprintf("v%d.%02d", self.versionMajor, self.versionMinor)
}
func (self *Font) GetFirstVerDate() ggfnt.Date { return self.firstVersionDate }
func (self *Font) GetMajorVerDate() ggfnt.Date { return self.majorVersionDate }
func (self *Font) GetMinorVerDate() ggfnt.Date { return self.minorVersionDate }

// Also updates the relevant dates.
func (self *Font) RaiseMajorVersion() {
	self.versionMajor += 1
	self.versionMinor  = 0
	date := ggfnt.CurrentDate()
	self.majorVersionDate = date
	self.minorVersionDate = date
}

// Also updates the relevant dates.
func (self *Font) RaiseMinorVersion() {
	self.versionMinor += 1
	self.minorVersionDate = ggfnt.CurrentDate()
}
func (self *Font) GetName() string { return self.fontName }
func (self *Font) SetName(name string) error {
	if len(name) > 255 { return errors.New("font name can't exceed 255 bytes") }
	err := checkStringValidity(name)
	if err != nil { return err }
	self.fontName = name
	return nil
}
func (self *Font) GetFamily() string { return self.fontFamily }
func (self *Font) SetFamily(name string) error {
	if len(name) > 255 { return errors.New("family name can't exceed 255 bytes") }
	err := checkStringValidity(name)
	if err != nil { return err }
	self.fontFamily = name
	return nil
}
func (self *Font) GetAuthor() string { return self.fontAuthor }
func (self *Font) SetAuthor(name string) error {
	if len(name) > 255 { return errors.New("author name can't exceed 255 bytes") }
	err := checkStringValidity(name)
	if err != nil { return err }
	self.fontAuthor = name
	return nil
}
func (self *Font) GetAbout() string { return self.fontAbout }
func (self *Font) SetAbout(about string) error {
	if len(about) > 65535 { return errors.New("font 'about' can't exceed 65535 bytes") }
	err := checkStringValidity(about)
	if err != nil { return err }
	self.fontAbout = about
	return nil
}

// ---- metrics ----

func (self *Font) GetNumGlyphs() int { return len(self.glyphData) }
func (self *Font) SetVertLayoutUsed(used bool) {
	// TODO: unclear if I need to check or update anything
	self.hasVertLayout = used
}
func (self *Font) GetMonoWidth() uint8 { return self.monoWidth }
func (self *Font) SetMonoWidth(width uint8) {
	self.monoWidth = width
}

func (self *Font) GetAscent() uint8 { return self.ascent }
func (self *Font) GetExtraAscent() uint8 { return self.extraAscent }
func (self *Font) GetDescent() uint8 { return self.descent }
func (self *Font) GetExtraDescent() uint8 { return self.extraDescent }
func (self *Font) GetLowercaseAscent() uint8 { return self.lowercaseAscent } // aka xheight
func (self *Font) SetAscent(value uint8) error {
	// TODO: shouldn't I check for existing glyph collisions?
	if value == 0 { return errors.New("ascent value must be strictly positive") }
	if value <= self.extraAscent { return errors.New("ascent value must be greater than extra ascent") }
	if value < self.lowercaseAscent { return errors.New("ascent value must be equal or greater than lowercase ascent") }
	self.ascent = value
	return nil
}
func (self *Font) SetExtraAscent(value uint8) error {
	// TODO: shouldn't I check for existing glyph collisions?
	if value >= self.ascent { return errors.New("extra ascent value can't be equal or greater than ascent") }
	self.extraAscent = value
	return nil
}
func (self *Font) SetDescent(value uint8) error {
	// TODO: shouldn't I check for existing glyph collisions?
	if value <= self.extraDescent { return errors.New("descent value must be greater than extra descent") }
	self.descent = value
	return nil
}
func (self *Font) SetExtraDescent(value uint8) error {
	// TODO: shouldn't I check for existing glyph collisions?
	if value >= self.descent { return errors.New("extra descent value can't be equal or greater than descent") }
	self.extraDescent = value
	return nil
}
func (self *Font) SetLowercaseAscent(value uint8) error {
	// TODO: shouldn't I check for existing glyph collisions?
	if value > self.ascent { return errors.New("lowercase ascent can't be greater than ascent") }
	self.lowercaseAscent = value
	return nil
}

func (self *Font) GetHorzInterspacing() uint8 { return self.horzInterspacing }
func (self *Font) GetVertInterspacing() uint8 { return self.vertInterspacing }
func (self *Font) SetHorzInterspacing(value uint8) {
	self.horzInterspacing = value
}
func (self *Font) SetVertInterspacing(value uint8) {
	self.vertInterspacing = value
}
func (self *Font) GetLineGap() uint8 { return self.lineGap }
func (self *Font) SetLineGap(value uint8) {
	self.lineGap = value
}

func (self *Font) GetVertLineWidth() uint8 { return self.vertLineWidth }
func (self *Font) GetVertLineGap() uint8 { return self.vertLineGap }
func (self *Font) SetVertLineWidth(value uint8) error {
	if value == 0 && self.hasVertLayout {
		return errors.New("can't set vert line width to zero when vert layout is enabled")
	}
	if value != 0 && !self.hasVertLayout {
		return errors.New("can't set vert line width without enabling vert layout first")
	}
	self.vertLineWidth = value
	return nil
}
func (self *Font) SetVertLineGap(value uint8) error {
	if value != 0 && !self.hasVertLayout {
		return errors.New("can't set vert line gap without enabling vert layout first")
	}
	self.vertLineGap = value
	return nil
}

// ---- glyph data ----

func (self *Font) AddGlyph(glyphMask *image.Alpha) (uint64, error) {
	if len(self.glyphData) >= ggfnt.MaxGlyphs {
		return 0, errors.New("reached font glyph count limit")
	}

	rect := mask.ComputeRect(glyphMask)
	if !rect.Empty() {
		if rect.Min.Y < 0 && -rect.Min.Y > int(self.ascent) + int(self.extraAscent) {
			fmt.Printf("rect: %v, ascent: %d, extraAscent: %d\n", rect, self.ascent, self.extraAscent)
			return 0, errors.New("glyph exceeds font ascent")
		}
		if rect.Max.Y > 0 && rect.Max.Y > int(self.descent) + int(self.extraDescent) {
			return 0, errors.New("glyph exceeds font descent")
		}
		if self.monoWidth != 0 && (rect.Min.X < 0 || rect.Max.X > int(self.monoWidth)) {
			return 0, errors.New("glyph doesn't respect monospacing width")
		}
		// TODO: ok, monoHeight could actually be used to ensure that placement pre and
		//       post offsets add to the relevant value. unclear how valuable that is
	}

	const MaxRerolls = 4
	for i := 1; i <= MaxRerolls; i++ {
		glyphUID, err := internal.CryptoRandUint64()
		if err != nil { return 0, err } // I'm not sure this can ever happen
		_, alreadyExists := self.glyphData[glyphUID]
		if !alreadyExists && glyphUID != uint64(ggfnt.GlyphMissing) {
			self.glyphData[glyphUID] = &glyphData{
				Name: "",
				Placement: ggfnt.GlyphPlacement{
					Advance: uint8(min(255, glyphMask.Bounds().Dx())),
					TopAdvance: self.ascent,
					BottomAdvance: self.descent,
					HorzCenter: uint8(min(255, glyphMask.Bounds().Dx()/2)),
				},
				Mask: glyphMask,
			}
			self.glyphOrder = append(self.glyphOrder, glyphUID)
			return glyphUID, nil
		}
	}

	return 0, errors.New("failed to generate unique glyph UID")
}

func (self *Font) SetGlyphPlacement(glyphUID uint64, placement ggfnt.GlyphPlacement) error {
	glyphData, found := self.glyphData[glyphUID]
	if !found { return errors.New("glyph not found") }
	glyphData.Placement = placement
	return nil
}

func (self *Font) SetGlyphName(glyphUID uint64, name string) error {
	glyphData, found := self.glyphData[glyphUID]
	if !found { return errors.New("glyph not found") }
	err := internal.ValidateBasicName(name)
	if err != nil { return err }
	glyphData.Name = name
	return nil
}

func checkStringValidity(str string) error {
	if !utf8.ValidString(str) { return errors.New("string contains invalid characters") }
	for _, codePoint := range str {
		if codePoint < ' ' {
			return errors.New("string can't contain control characters")
		}
	}
	return nil
}

// ---- mapping ----

func (self *Font) AddMappingMode(definition []byte) (uint8, error) {
	panic("unimplemented")
}

func (self *Font) DeleteMappingMode(index uint8) error {
	panic("unimplemented")
}

func (self *Font) SwapMappingModes(a, b uint8) error {
	panic("unimplemented")
}

func (self *Font) GetMappingMode(index uint8) {
	// ????
	panic("unimplemented")
}

func (self *Font) Map(codePoint rune, glyphUID uint64) error {
	if codePoint < ' ' {
		return errors.New("can't map code points before ' ' (space)") 
	}
	_, hasData := self.glyphData[glyphUID]
	if !hasData {
		return errors.New("attempted to map '" + string(codePoint) + "' to an undefined glyph")
	}
	self.runeMapping[codePoint] = codePointMapping{ Mode: 255, Glyphs: []uint64{ glyphUID } }
	return nil
}

func (self *Font) MapWithMode(codePoint rune, mode uint8, modeGlyphUIDs ...uint64) error {
	// all error checks
	if len(modeGlyphUIDs) < 2 {
		return errors.New("custom mode mapping requires mapping at least 2 glyphs")
	}
	if int(mode) >= len(self.mappingModes) {
		return errors.New("attempted to use undefined custom mode mapping")
	}
	for _, modeGlyphUID := range modeGlyphUIDs {
		_, hasData := self.glyphData[modeGlyphUID]
		if !hasData {
			return errors.New("attempted to map '" + string(codePoint) + "' to an undefined glyph")
		}
	}
	
	self.runeMapping[codePoint] = codePointMapping{ Mode: mode, Glyphs: modeGlyphUIDs }
	return nil
}

func (self *Font) NumFastMappingTables() int {
	return len(self.fastMappingTables)
}

func (self *Font) GetFastMappingTable(i int) *FastMappingTable {
	return self.fastMappingTables[i]
}

func (self *Font) SwitchFastMappingTables(i, j int) error {
	numTables := self.NumFastMappingTables()
	if i < 0 || i >= numTables { return fmt.Errorf("fast mapping table #%d does not exist", i) }
	if j < 0 || j >= numTables { return fmt.Errorf("fast mapping table #%d does not exist", j) }
	if i == j { return nil }
	self.fastMappingTables[i], self.fastMappingTables[j] = self.fastMappingTables[j], self.fastMappingTables[i]
	return nil
}

func (self *Font) AddFastMappingTable(start, end rune) error {
	table, err := newFastMappingTable(start, end)
	if err != nil { return err }
	table.builder = self
	// TODO: check for max size error?
	self.fastMappingTables = append(self.fastMappingTables, table)
	return nil
}
