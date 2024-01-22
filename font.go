package ggfnt

import "errors"
import "image"
import "image/color"

// A [Font] is a read-only object that contains all the data required to
// use a font. To create a [Font], we use the [Parse]() method.
//
// Fonts contain multiple sections or tables, which are exposed through
// gateway methods and differentiated types:
//  - Use [Font.Header]() to access information about the [FontHeader].
//  - Use [Font.Metrics]() to access information about the [FontMetrics].
//  - Use [Font.Glyphs]() to access information about the [FontGlyphs].
//  - Use [Font.Coloring]() to access information about the [FontColoring].
//  - Use [Font.Vars]() to access information about the [FontVariables].
//  - Use [Font.Mapping]() to access information about the [FontMapping].
//  - Use [Font.Kerning]() to access information about the [FontKerning].
// 
// Additional structures for storing the font tend to result in around 12 64-bit
// words (96 bytes) of memory overhead compared to the data in the font itself,
// plus the [Parametrization], which are another 8 64-bit words plus as many bytes
// as variables there are in the font. On average this is around 22 64-bit words
// (176 bytes).
//
// Fonts themselves, once uncompressed, tend to take around XXX_KiB on average,
// but it depends a lot on the number of glyphs, features, glyph size and overall
// complexity.
type Font struct {
	data []byte // already ungzipped, starting from HEADER (signature is ignored)

	// offsets to specific points at which critical data appears
	// (offsetToHeader is always zero)
	offsetToMetrics uint32
	offsetToGlyphNames uint32
	offsetToGlyphMasks uint32
	offsetToColoring uint32
	offsetToColoringPalettes uint32
	offsetToColoringPaletteNames uint32
	offsetToColoringSections uint32
	offsetToColoringSectionOptions uint32
	offsetToVariables uint32
	offsetToMappings uint32
	offsetsToFastMapTables []uint32
	offsetToCodePointList uint32 // part of mappings table
	offsetToHorzKernings uint32
	offsetToVertKernings uint32
}

// --- general methods ---

// Returns a [Parametrization] for the font, which can store variables,
// scale and other values that can change at runtime but are still related
// to the font.
func (self *Font) Parametrize() *Parametrization {
	var parametrization Parametrization
	vars := self.Vars()

	parametrization.font = self
	parametrization.variables = make([]uint8, int(vars.Count()))
	for i := uint8(0); i < vars.Count(); i++ {
		parametrization.variables[i] = vars.GetInitValue(VarKey(i))
	}
	parametrization.customGlyphs = nil
	parametrization.scale = 1
	parametrization.interVertShift = 0
	parametrization.interHorzShift = 0
	
	return &parametrization
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
func (self *Font) Coloring() *FontColoring { return (*FontColoring)(self) }
func (self *Font) Vars() *FontVariables { return (*FontVariables)(self) }
func (self *Font) Mapping() *FontMapping { return (*FontMapping)(self) }
func (self *Font) Kerning() *FontKerning { return (*FontKerning)(self) }

// --- header section ---

type FontHeader Font
func (self *FontHeader) FormatVersion() uint32 { panic("unimplemented") }
func (self *FontHeader) ID() uint64 { panic("unimplemented") }
func (self *FontHeader) VersionMajor() uint16 { panic("unimplemented") }
func (self *FontHeader) VersionMinor() uint16 { panic("unimplemented") }
func (self *FontHeader) FirstVersionDate() (year uint16, month, day uint8) { panic("unimplemented") }
func (self *FontHeader) MajorVersionDate() (year uint16, month, day uint8) { panic("unimplemented") }
func (self *FontHeader) MinorVersionDate() (year uint16, month, day uint8) { panic("unimplemented") }
func (self *FontHeader) NumGlyphs() uint16 { panic("unimplemented") }
func (self *FontHeader) Name() string { panic("unimplemented") }
func (self *FontHeader) Family() string { panic("unimplemented") }
func (self *FontHeader) Author() string { panic("unimplemented") }
func (self *FontHeader) About() string { panic("unimplemented") }

func (self *FontHeader) Validate(mode FmtValidation) error {
	// default checks
	if self.FormatVersion() != FormatVersion { return errors.New("invalid FormatVersion") }
	if lazyEntropyUint64(self.ID()) < 0.26 { return errors.New("font ID entropy too low") }
	if self.NumGlyphs() == 0 { return errors.New("font must define at least one glyph") }
	if self.Name() == "" { return errors.New("font name can't be empty") }

	// strict checks
	if mode == FmtStrict {
		panic("unimplemented")
	}

	return nil
}

// --- metrics section ---

type FontMetrics Font
func (self *FontMetrics) HasVertLayout() bool { panic("unimplemented") }
func (self *FontMetrics) Monospaced() bool { return self.MonoWidth() != 0 }
func (self *FontMetrics) VertMonospaced() bool { return self.HasVertLayout() && self.MonoHeight() != 0 }
func (self *FontMetrics) MonoWidth() uint8 { panic("unimplemented") }
func (self *FontMetrics) MonoHeight() uint8 { panic("unimplemented") }
func (self *FontMetrics) Ascent() uint8 { panic("unimplemented") }
func (self *FontMetrics) ExtraAscent() uint8 { panic("unimplemented") }
func (self *FontMetrics) Descent() uint8 { panic("unimplemented") }
func (self *FontMetrics) ExtraDescent() uint8 { panic("unimplemented") }
func (self *FontMetrics) LowercaseAscent() uint8 { panic("unimplemented") }
func (self *FontMetrics) HorzInterspacing() uint8 { panic("unimplemented") }
func (self *FontMetrics) VertInterspacing() uint8 { panic("unimplemented") }
func (self *FontMetrics) LineGap() uint8 { panic("unimplemented") }

func (self *FontMetrics) Validate(mode FmtValidation) error {
	// default checks
	err := boolErrCheck(self.data[self.offsetToMetrics + 2])
	if err != nil { return err }
	if self.MonoHeight() != 0 && !self.HasVertLayout() {
		return errors.New("MonoHeight set without HasVertLayout")
	}
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
func (self *FontGlyphs) Count() uint16 { return ((*Font)(self)).Header().NumGlyphs() } // alias for Header().NumGlyphs()
func (self *FontGlyphs) NamedCount() uint16 { panic("unimplemented") }
func (self *FontGlyphs) FindIndexByName(name string) GlyphIndex { panic("unimplemented") } // notice: might return a control glyph
func (self *FontGlyphs) RasterizeMask(glyph GlyphIndex) *image.Alpha { panic("unimplemented") }
func (self *FontGlyphs) Bounds(glyph GlyphIndex) GlyphBounds { panic("unimplemented") }

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

// --- coloring section ---

type DyeKey uint8
type PaletteKey uint8
type ColorSectionKey uint8

type Palette struct {
	key PaletteKey
	name string // build with unsafe?
	colors []byte
}
func (self *Palette) Name() string { return self.name }
func (self *Palette) Key() PaletteKey { return self.key }
func (self *Palette) Size() int { return len(self.colors) }
func (self *Palette) EachColor(func(color.RGBA)) {
	panic("unimplemented")
}

type FontColoring Font
func (self *FontColoring) NumDyes() uint8 { return self.data[self.offsetToColoring + 0] }
func (self *FontColoring) DyeEntries(func(DyeKey, string)) { panic("unimplemented") }
// TODO: switch to Dyes() iters.Seq2[DyeKey, string] when that's available?
func (self *FontColoring) NumPalettes() uint8 { panic("unimplemented") }
func (self *FontColoring) PaletteEntries(func(PaletteKey, string)) {
	panic("unimplemented")
}
func (self *FontColoring) GetPalette(PaletteKey) (Palette, bool) {
	panic("unimplemented")
}
func (self *FontColoring) NumSections() uint8 { panic("unimplemented") }
func (self *FontColoring) SectionEntries(func(key ColorSectionKey, name string)) { panic("unimplemented") }
func (self *FontColoring) SectionPalettes(ColorSectionKey, func(PaletteKey, string)) {
	panic("unimplemented")
}
func (self *FontColoring) GetSectionRange(key ColorSectionKey) (inclusiveStart, inclusiveEnd uint8) {
	panic("unimplemented")
}
func (self *FontColoring) SectionDefaultColors(func(index uint8, rgba color.RGBA)) {
	panic("unimplemented")
}

func (self *FontColoring) Validate(mode FmtValidation) error {
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
// Variables can't be modified on the [*Font] object itself, the changing
// data is always managed through a [*Parametrization].
type FontVariables Font
func (self *FontVariables) Count() uint8 { panic("unimplemented") }
func (self *FontVariables) NamedCount() uint8 { panic("unimplemented") }
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

	// strict checks
	if mode == FmtStrict {
		// TODO: 
		panic("unimplemented")
	}

	panic("unimplemented")
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

	panic("unimplemented")
	return nil
}
