package ggfnt

import "fmt"
import "image"
import "errors"
import "unicode/utf8"

import "github.com/tinne26/ggfnt/mask"

func (self *FontBuilder) GetFontID() uint64 { return self.fontID }
func (self *FontBuilder) GetFontIDStr() string { return fmt.Sprintf("%016X", self.fontID) }
func (self *FontBuilder) GetVersionMajor() uint16 { return self.versionMajor }
func (self *FontBuilder) GetVersionMinor() uint16 { return self.versionMinor }
func (self *FontBuilder) GetVersionStr() string {
	return fmt.Sprintf("v%d.%02d", self.versionMajor, self.versionMinor)
}
func (self *FontBuilder) GetFirstVerDate() Date {
	return self.firstVersionDate
}
func (self *FontBuilder) GetMajorVerDate() Date {
	return self.majorVersionDate
}
func (self *FontBuilder) GetMinorVerDate() Date {
	return self.minorVersionDate
}

// Also updates the relevant dates.
func (self *FontBuilder) RaiseMajorVersion() {
	self.versionMajor += 1
	self.versionMinor  = 0
	date := CurrentDate()
	self.majorVersionDate = date
	self.minorVersionDate = date
}

// Also updates the relevant dates.
func (self *FontBuilder) RaiseMinorVersion() {
	self.versionMinor += 1
	self.minorVersionDate = CurrentDate()
}
func (self *FontBuilder) GetName() string {
	return self.fontName
}
func (self *FontBuilder) SetName(name string) error {
	if len(name) > 255 { return errors.New("font name can't exceed 255 bytes") }
	err := checkStringValidity(name)
	if err != nil { return err }
	self.fontName = name
	return nil
}
func (self *FontBuilder) GetFamily() string {
	return self.fontFamily
}
func (self *FontBuilder) SetFamily(name string) error {
	if len(name) > 255 { return errors.New("family name can't exceed 255 bytes") }
	err := checkStringValidity(name)
	if err != nil { return err }
	self.fontFamily = name
	return nil
}
func (self *FontBuilder) GetAuthor() string {
	return self.fontAuthor
}
func (self *FontBuilder) SetAuthor(name string) error {
	if len(name) > 255 { return errors.New("author name can't exceed 255 bytes") }
	err := checkStringValidity(name)
	if err != nil { return err }
	self.fontAuthor = name
	return nil
}
func (self *FontBuilder) GetAbout() string {
	return self.fontAbout
}
func (self *FontBuilder) SetAbout(about string) error {
	if len(about) > 65535 { return errors.New("font 'about' can't exceed 65535 bytes") }
	err := checkStringValidity(about)
	if err != nil { return err }
	self.fontAbout = about
	return nil
}

func (self *FontBuilder) AddGlyph(glyphMask *image.Alpha) (uint64, error) {
	if len(self.glyphData) >= MaxGlyphs {
		return 0, errors.New("reached font glyph count limit")
	}

	// TODO: verify the mask for ascent, descent, and in case of monospacing,
	//       verify that it complies with that too.
	rect := mask.ComputeRect(glyphMask)
	if !rect.Empty() {
		if rect.Min.Y < 0 && -rect.Min.Y > int(self.ascent) + int(self.extraAscent) {
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
		glyphUID, err := cryptoRandUint64()
		if err != nil { return 0, err } // I'm not sure this can ever happen
		_, alreadyExists := self.glyphData[glyphUID]
		if !alreadyExists {
			self.glyphData[glyphUID] = &glyphData{
				Name: "",
				Placement: GlyphPlacement{
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

func (self *FontBuilder) SetGlyphPlacement(glyphUID uint64, placement GlyphPlacement) error {
	glyphData, found := self.glyphData[glyphUID]
	if !found { return errors.New("glyph not found") }
	glyphData.Placement = placement
	return nil
}

func (self *FontBuilder) SetGlyphName(glyphUID uint64, name string) error {
	glyphData, found := self.glyphData[glyphUID]
	if !found { return errors.New("glyph not found") }
	err := validateBasicName(name)
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
