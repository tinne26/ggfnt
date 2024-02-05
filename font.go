package ggfnt

import "io"
import "errors"
import "image"
import "image/color"
import "compress/gzip"
import "unsafe"

import "github.com/tinne26/ggfnt/internal"

// A [Font] is a read-only object that contains all the data required to
// use a font. To create a [Font], we use the [Parse]() method.
//
// Fonts contain multiple sections or tables, which are exposed through
// gateway methods and differentiated types:
//  - Use [Font.Header]() to access information about the [FontHeader].
//  - Use [Font.Metrics]() to access information about the [FontMetrics].
//  - Use [Font.Glyphs]() to access information about the [FontGlyphs].
//  - Use [Font.Color]() to access information about the [FontColor].
//  - Use [Font.Vars]() to access information about the [FontVariables].
//  - Use [Font.Mapping]() to access information about the [FontMapping].
//  - Use [Font.Kerning]() to access information about the [FontKerning].
type Font internal.Font

// --- general methods ---

func (self *Font) Export(writer io.Writer) error {
	n, err := writer.Write([]byte{'t', 'g', 'g', 'f', 'n', 't'})
	if err != nil { return err }
	if n != 6 { return errors.New("short write") }

	gzipWriter := gzip.NewWriter(writer)
	n, err = gzipWriter.Write(self.Data)
	if err != nil { return err }
	if n != len(self.Data) { return errors.New("short write") }
	return gzipWriter.Close()
}

// TODO: don't worry about this until actually implementing validation, I'll
//       see there how easy it is to make, and what might or might not be reasonable
type FmtValidation bool
const (
	FmtDefault FmtValidation = false // basic and inexpensive checks only
	FmtStrict  FmtValidation = true  // check everything that can be checked
)

func (self *Font) Validate(mode FmtValidation) error {
	var err error

	err = self.Header().Validate(mode)
	if err != nil { return err }
	err = self.Metrics().Validate(mode)
	if err != nil { return err }
	err = self.Glyphs().Validate(mode)
	if err != nil { return err }
	err = self.Color().Validate(mode)
	if err != nil { return err }
	err = self.Vars().Validate(mode)
	if err != nil { return err }
	err = self.Mapping().Validate(mode)
	if err != nil { return err }
	err = self.Kerning().Validate(mode)
	if err != nil { return err }

	return nil
}

// --- data section gateways ---

func (self *Font) Header() *FontHeader { return (*FontHeader)(self) }
func (self *Font) Metrics() *FontMetrics { return (*FontMetrics)(self) }
func (self *Font) Glyphs() *FontGlyphs { return (*FontGlyphs)(self) }
func (self *Font) Color() *FontColor { return (*FontColor)(self) }
func (self *Font) Vars() *FontVariables { return (*FontVariables)(self) }
func (self *Font) Mapping() *FontMapping { return (*FontMapping)(self) }
func (self *Font) Kerning() *FontKerning { return (*FontKerning)(self) }

// --- header section ---

type FontHeader Font
func (self *FontHeader) FormatVersion() uint32 {
	return internal.DecodeUint32LE(self.Data[0 : 4])
}
func (self *FontHeader) ID() uint64 {
	return internal.DecodeUint64LE(self.Data[4 : 12])
}
func (self *FontHeader) VersionMajor() uint16 {
	return internal.DecodeUint16LE(self.Data[12 : 14])
}
func (self *FontHeader) VersionMinor() uint16 {
	return internal.DecodeUint16LE(self.Data[14 : 16])
}
func (self *FontHeader) FirstVersionDate() Date {
	y, m, d := internal.DecodeDate(self.Data[16 : 20])
	return Date{ Year: y, Month: m, Day: d }
}
func (self *FontHeader) MajorVersionDate() Date {
	y, m, d := internal.DecodeDate(self.Data[20 : 24])
	return Date{ Year: y, Month: m, Day: d }
}
func (self *FontHeader) MinorVersionDate() Date {
	y, m, d := internal.DecodeDate(self.Data[24 : 28])
	return Date{ Year: y, Month: m, Day: d }
}
func (self *FontHeader) Name() string {
	nameLen := self.Data[28]
	return unsafe.String(&self.Data[29], nameLen)
}
func (self *FontHeader) Family() string {
	nameLen   := self.Data[28]
	familyLen := self.Data[29 + nameLen]
	return unsafe.String(&self.Data[30 + nameLen], familyLen)
}
func (self *FontHeader) Author() string {
	nameLen   := self.Data[28]
	familyLen := self.Data[29 + nameLen]
	authorLen := self.Data[30 + nameLen + familyLen]
	return unsafe.String(&self.Data[31 + nameLen + familyLen], authorLen)
}
func (self *FontHeader) About() string {
	nameLen   := self.Data[28]
	familyLen := self.Data[29 + nameLen]
	authorLen := self.Data[30 + nameLen + familyLen]
	aboutIndex := 31 + nameLen + familyLen + authorLen
	aboutLen := internal.DecodeUint16LE(self.Data[aboutIndex : aboutIndex + 2])
	return unsafe.String(&self.Data[33 + nameLen + familyLen + authorLen], aboutLen)
}

func (self *FontHeader) Validate(mode FmtValidation) error {
	// default checks
	if self.FormatVersion() != FormatVersion { return errors.New("invalid FormatVersion") }
	if internal.LazyEntropyUint64(self.ID()) < internal.MinEntropyID {
		return errors.New("font ID entropy too low")
	}
	if self.Name() == "" { return errors.New("font name can't be empty") }

	// strict checks
	if mode == FmtStrict {
		panic("unimplemented")
	}

	return nil
}

// --- metrics section ---

type FontMetrics Font
func (self *FontMetrics) NumGlyphs() uint16 {
	return internal.DecodeUint16LE(self.Data[self.OffsetToMetrics + 0 : self.OffsetToMetrics + 2])
}
func (self *FontMetrics) HasVertLayout() bool {
	return self.Data[self.OffsetToMetrics + 2] == 1
}
func (self *FontMetrics) Monospaced() bool { return self.MonoWidth() != 0 }
func (self *FontMetrics) MonoWidth() uint8 {
	return self.Data[self.OffsetToMetrics + 3]
}
func (self *FontMetrics) Ascent() uint8 {
	return self.Data[self.OffsetToMetrics + 4]
}
func (self *FontMetrics) ExtraAscent() uint8 {
	return self.Data[self.OffsetToMetrics + 5]
}
func (self *FontMetrics) Descent() uint8 {
	return self.Data[self.OffsetToMetrics + 6]
}
func (self *FontMetrics) ExtraDescent() uint8 {
	return self.Data[self.OffsetToMetrics + 7]
}
func (self *FontMetrics) LowercaseAscent() uint8 {
	return self.Data[self.OffsetToMetrics + 8]
}
func (self *FontMetrics) HorzInterspacing() uint8 {
	return self.Data[self.OffsetToMetrics + 9]
}
func (self *FontMetrics) VertInterspacing() uint8 {
	return self.Data[self.OffsetToMetrics + 10]
}
func (self *FontMetrics) LineGap() uint8 {
	return self.Data[self.OffsetToMetrics + 11]
}
func (self *FontMetrics) VertLineWidth() uint8 {
	return self.Data[self.OffsetToMetrics + 12]
}
func (self *FontMetrics) VertLineGap() uint8 {
	return self.Data[self.OffsetToMetrics + 13]
}

func (self *FontMetrics) Validate(mode FmtValidation) error {
	// default checks
	if self.NumGlyphs() == 0 { return errors.New("font must define at least one glyph") }
	err := internal.BoolErrCheck(self.Data[self.OffsetToMetrics + 2])
	if err != nil { return err }
	if self.Ascent() == 0 { return errors.New("Ascent can't be zero") }
	if self.ExtraAscent() > self.Ascent() {
		return errors.New("ExtraAscent can't be bigger than Ascent")
	}
	if self.VertInterspacing() != 0 && !self.HasVertLayout() {
		return errors.New("VertInterspacing set without HasVertLayout")
	}

	// strict checks
	if mode == FmtStrict {
		panic("unimplemented")
	}

	return nil
}

// --- glyphs section ---

type FontGlyphs Font

// Alias for Metrics().NumGlyphs()
func (self *FontGlyphs) Count() uint16 {
	return ((*Font)(self)).Metrics().NumGlyphs()
}
func (self *FontGlyphs) NamedCount() uint16 {
	return internal.DecodeUint16LE(self.Data[self.OffsetToGlyphNames + 0 : self.OffsetToGlyphNames + 2])
}
func (self *FontGlyphs) FindIndexByName(name string) GlyphIndex { panic("unimplemented") } // notice: might return a control glyph
func (self *FontGlyphs) RasterizeMask(glyph GlyphIndex) *image.Alpha { panic("unimplemented") }
func (self *FontGlyphs) Placement(glyph GlyphIndex) GlyphPlacement { panic("unimplemented") }

func (self *FontGlyphs) Validate(mode FmtValidation) error {
	// default checks
	if self.NamedCount() > self.Count() {
		return errors.New("can't have more named glyphs than glyphs")
	}

	// strict checks
	if mode == FmtStrict {
		panic("unimplemented")
	}

	return nil
}

// --- color section ---

type FontColor Font
func (self *FontColor) EachDye(func(DyeKey, string)) {
	// TODO: switch to Dyes() iters.Seq2[DyeKey, string] when that's available?
	panic("unimplemented")
}
func (self *FontColor) GetDyeRange(key DyeKey) (start, end uint8) {
	panic("unimplemented")
}

func (self *FontColor) EachPalette(func(PaletteKey, string)) {
	panic("unimplemented")
}
func (self *FontColor) EachPaletteColor(PaletteKey, func(color.RGBA)) {
	panic("unimplemented")
}

// An invalid palette key will always return (0, 0). A valid palette
// key will always return start and ends > 0. Both start and end are
// inclusive. Given a valid palette key, the size is (end - start + 1).
func (self *FontColor) GetPaletteRange(key PaletteKey) (start, end uint8) {
	panic("unimplemented")
}

func (self *FontColor) NumColors() uint8 {
	panic("unimplemented") // (255 - ColorSectionStarts[last]) + 1
}

func (self *FontColor) Validate(mode FmtValidation) error {
	// default checks
	// ...

	// strict checks
	if mode == FmtStrict {
		panic("unimplemented")
	}

	return nil
}

// --- variables section ---

// Index to a font variable. See [FontVariables].
type VarKey uint8

// Obtained through [Font.Variables]().
// 
// Variables can't be modified on the [*Font] object itself, that
// kind of state must be managed by a renderer or similar.
type FontVariables Font
func (self *FontVariables) Count() uint8 {
	return self.Data[self.OffsetToVariables]
}
func (self *FontVariables) NamedCount() uint8 {
	index := self.OffsetToVariables + 1 + uint32(self.Count())*3
	return self.Data[index]
}
func (self *FontVariables) FindIndexByName(name string) VarKey { panic("unimplemented") }
func (self *FontVariables) GetInitValue(index VarKey) uint8 { panic("unimplemented") }
func (self *FontVariables) GetRange(index VarKey) (minValue, maxValue uint8) { panic("unimplemented") }
func (self *FontVariables) Each(func(index VarKey, name string)) { panic("unimplemented") } // only named variables are exposed

func (self *FontVariables) Validate(mode FmtValidation) error {
	// default checks
	if self.NamedCount() > self.Count() {
		return errors.New("can't have more named variables than variables")
	}

	// strict checks
	if mode == FmtStrict {
		// TODO:
		// - go through var defs and ensure init value is in range
		// - make sure every named variable is not repeated and is within numVars
		// - make sure every named variable name is correct
		// - make sure every named variable name comes in order
		// - make sure the offsets to names are correct
		panic("unimplemented")
	}

	return nil
}

// --- mapping section ---

type FontMapping Font
func (self *FontMapping) NumCodePoints() uint32 { panic("unimplemented") }
func (self *FontMapping) Utf8(codePoint rune) GlyphIndex { panic("unimplemented") }
func (self *FontMapping) Ascii(codePoint byte) GlyphIndex { panic("unimplemented") }

func (self *FontMapping) Validate(mode FmtValidation) error {
	// default checks
	// ...
	// TODO: check fast table ranges and conditions?

	// strict checks
	if mode == FmtStrict {
		// TODO: 
		panic("unimplemented")
	}

	return nil
}

// --- kerning section ---

type FontKerning Font
func (self *FontKerning) NumPairs() uint32 { panic("unimplemented") }
func (self *FontKerning) NumVertPairs() uint32 { panic("unimplemented") }
func (self *FontKerning) Get(prev, curr GlyphIndex) int8 { panic("unimplemented") } // binary search based
func (self *FontKerning) GetVert(prev, curr GlyphIndex) int8 { panic("unimplemented") } // binary search based
func (self *FontKerning) EachPair(func (prev, curr GlyphIndex, kern int8)) { panic("unimplemented") }
func (self *FontKerning) EachVertPair(func (prev, curr GlyphIndex, kern int8)) { panic("unimplemented") }

func (self *FontKerning) Validate(mode FmtValidation) error {
	// default checks
	// ...

	// strict checks
	if mode == FmtStrict {
		panic("unimplemented")
	}

	return nil
}
