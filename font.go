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
//  - Use [Font.Settings]() to access information about the [FontSettings].
//  - Use [Font.Mapping]() to access information about the [FontMapping].
//  - Use [Font.Kerning]() to access information about the [FontKerning].
type Font internal.Font

const invalidFontData = "invalid font data" // used for panics

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

func (self *Font) RawSize() int {
	return len(self.Data)
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
	err = self.Settings().Validate(mode)
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
func (self *Font) Color() *FontColor { return (*FontColor)(self) }
func (self *Font) Glyphs() *FontGlyphs { return (*FontGlyphs)(self) }
func (self *Font) Settings() *FontSettings { return (*FontSettings)(self) }
func (self *Font) Mapping() *FontMapping { return (*FontMapping)(self) }
func (self *Font) Rewrites() *FontRewrites { return (*FontRewrites)(self) }
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

// --- color section ---

// TODO: maybe given the importance of the "main" dye, either I change spec
// or I provide some function to easily search for it? Or "HasDyes?" hmm.

type FontColor Font

// OffsetToColorSections
// OffsetToColorSectionNames

func (self *FontColor) NumDyes() uint8 {
	return self.Data[self.OffsetToColorSections + 0]
}

func (self *FontColor) NumPalettes() uint8 {
	return self.Data[self.OffsetToColorSections + 1]
}

func (self *FontColor) numColorSections() uint8 {
	numDyes := self.NumDyes()
	numColorSections := numDyes + self.NumPalettes()
	if numColorSections < numDyes { panic(invalidFontData) } // overflow
	return numColorSections
}

func (self *FontColor) Count() uint8 {
	// the color count is not explicit, we need to find where the last
	// color section begins, and that reveals the number of colors used
	lastStart := self.Data[self.OffsetToColorSections + 2 + uint32(self.numColorSections()) - 1]
	if lastStart == 0 { panic(invalidFontData) }
	return 255 - lastStart + 1
}

// TODO: switch to Dyes() iters.Seq2[DyeKey, string] when that's available?
// TODO: the string is an unsafe.String, so don't store it indefinitely.
func (self *FontColor) EachDye(fn func(DyeKey, string)) {
	numDyes := uint32(self.NumDyes())
	numColorSections := uint32(self.numColorSections())
	offsetToColorSectionNamesData := self.OffsetToColorSectionNames + (numColorSections << 1)

	var startOffset uint32 = 0
	for i := uint32(0); i < numDyes; i++ {
		endOffset := uint32(internal.DecodeUint16LE(self.Data[self.OffsetToColorSectionNames + (i << 1) : ]))
		if endOffset <= startOffset { panic(invalidFontData) }
		nameLen := endOffset - startOffset
		dyeName := unsafe.String(&self.Data[offsetToColorSectionNamesData + startOffset], nameLen)
		fn(DyeKey(i), dyeName)
		startOffset = endOffset
	}
}

// TODO: mention the order, because the alphas order is unclear.
func (self *FontColor) EachDyeAlpha(key DyeKey, fn func(uint8)) {
	numDyes := uint32(self.NumDyes())
	numColorSections := uint32(self.numColorSections())
	offsetToColorSectionEndOffsets := self.OffsetToColorSections + 2 + numColorSections
	offsetToColorSectionsData := offsetToColorSectionEndOffsets + (numColorSections << 1)

	var startOffset uint32 = 0
	for i := uint32(0); i < numDyes; i++ {
		endOffset := uint32(internal.DecodeUint16LE(self.Data[offsetToColorSectionEndOffsets + (i << 1) : ]))
		if endOffset <= startOffset { panic(invalidFontData) }
		for offset := startOffset; offset < endOffset; offset++ {
			fn(self.Data[offsetToColorSectionsData + offset])
		}
		startOffset = endOffset
	}
}

// Returns the range of color indices taken by the dye in the global
// font color range [0, 255]. Dyes always start at 255, occupying
// the higher part of the range.
//
// An invalid dye key will always return (0, 0). A valid dye key will
// will always return start and ends > 0. Both start and end are inclusive.
// Given a valid dye key, the amount of alpha variants is (end - start + 1).
func (self *FontColor) GetDyeRange(key DyeKey) (start, end uint8) {
	if uint8(key) >= self.NumDyes() { panic("invalid DyeKey") }
	keyStart := self.Data[self.OffsetToColorSections + 2 + uint32(key)]
	if key == 0 { return keyStart, 255 }
	prevKeyStart := self.Data[self.OffsetToColorSections + 2 + uint32(key) - 1]
	if prevKeyStart <= keyStart { panic(invalidFontData) }
	return keyStart, prevKeyStart - 1
}

func (self *FontColor) EachPalette(fn func(PaletteKey, string)) {
	numDyes := uint32(self.NumDyes())
	numPalettes := uint32(self.NumPalettes())
	numColorSections := numDyes + numPalettes
	if numColorSections > 255 { panic(invalidFontData) }
	offsetToColorSectionNamesData := self.OffsetToColorSectionNames + (numColorSections << 1)

	var startOffset uint32 = 0
	for i := numDyes; i < numColorSections; i++ {
		endOffset := uint32(internal.DecodeUint16LE(self.Data[self.OffsetToColorSectionNames + (i << 1) : ]))
		if endOffset <= startOffset { panic(invalidFontData) }
		nameLen := endOffset - startOffset
		paletteName := unsafe.String(&self.Data[offsetToColorSectionNamesData + startOffset], nameLen)
		fn(PaletteKey(i - numDyes), paletteName)
		startOffset = endOffset
	}
}

func (self *FontColor) EachPaletteColor(key PaletteKey, fn func(color.RGBA)) {
	numDyes := uint32(self.NumDyes())
	numPalettes := uint32(self.NumPalettes())
	numColorSections := numDyes + numPalettes
	if numColorSections > 255 { panic(invalidFontData) } // discretional assertion
	offsetToColorSectionEndOffsets := self.OffsetToColorSections + 2 + numColorSections
	offsetToColorSectionsData := offsetToColorSectionEndOffsets + (numColorSections << 1)

	var startOffset uint32 = 0
	for i := numDyes; i < numColorSections; i++ {
		endOffset := uint32(internal.DecodeUint16LE(self.Data[offsetToColorSectionEndOffsets + (i << 1) : ]))
		if endOffset <= startOffset { panic(invalidFontData) }
		for offset := startOffset; offset < endOffset; offset += 4 {
			r := self.Data[offsetToColorSectionsData + offset + 0]
			g := self.Data[offsetToColorSectionsData + offset + 1]
			b := self.Data[offsetToColorSectionsData + offset + 2]
			a := self.Data[offsetToColorSectionsData + offset + 3]
			fn(color.RGBA{r, g, b, a})
		}
		startOffset = endOffset
	}
}

// Returns the range of color indices taken by the palette in the global
// font color range [0, 255]. Palettes always start after dyes.
//
// An invalid palette key will always return (0, 0). A valid palette
// key will always return start and ends > 0. Both start and end are
// inclusive. Given a valid palette key, the size is (end - start + 1).
func (self *FontColor) GetPaletteRange(key PaletteKey) (start, end uint8) {
	if uint8(key) >= self.NumPalettes() { panic("invalid PaletteKey") }
	
	numDyes := self.NumDyes()
	relKey := uint32(numDyes) + uint32(key)
	keyStart := self.Data[self.OffsetToColorSections + 2 + relKey]
	if 255 - keyStart < numDyes { panic(invalidFontData) }
	if key == 0 && numDyes == 0 { return keyStart, 255 }
	prevKeyStart := self.Data[self.OffsetToColorSections + 2 + relKey - 1]
	if prevKeyStart <= keyStart { panic(invalidFontData) }
	return keyStart, prevKeyStart - 1
}

func (self *FontColor) Validate(mode FmtValidation) error {
	// default checks
	numDyes     := uint32(self.NumDyes())
	numPalettes := uint32(self.NumPalettes())
	numColorSections := numDyes + numPalettes
	if numColorSections > 255 {
		return errors.New("NumDyes + NumPalettes can't exceed 255")
	}

	// check section sizes vs offsets
	startsIndex  := self.OffsetToColorSections + 2
	offsetsIndex := startsIndex + numColorSections
	prevSectionEnd := uint16(256) // exclusive
	prevOffset := uint16(0)
	for i := uint32(0); i < numColorSections; i++ {
		sectionStart := uint16(self.Data[startsIndex])
		if sectionStart >= prevSectionEnd {
			return errors.New("invalid ColorSectionStarts sequence")
		}
		sectionSize := prevSectionEnd - sectionStart
		endOffset := internal.DecodeUint16LE(self.Data[offsetsIndex : ])

		if i < numDyes {
			if endOffset - prevOffset != sectionSize {
				return errors.New("mismatch between ColorSectionStarts and ColorSectionEndOffsets (dye shades count)")
			}
		} else {
			if endOffset - prevOffset != sectionSize*4 {
				return errors.New("mismatch between ColorSectionStarts and ColorSectionEndOffsets (palette color count)")
			}
		}

		offsetsIndex += 2
		startsIndex  += 1
		prevOffset = endOffset
		prevSectionEnd = sectionStart
	}

	// verify paletted RGBA values (premult alpha checks)
	// TODO ...

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

// --- settings section ---

// Index to a font setting. See [FontSettings].
type SettingKey uint8

// Obtained through [Font.Settings]().
// 
// Settings can't be modified on the [*Font] object itself, that
// kind of state must be managed by a renderer or similar.
type FontSettings Font
func (self *FontSettings) Count() uint8 {
	return self.Data[self.OffsetToSettingDefinitions]
}
//func (self *FontSettings) FindKeyByName(name string) SettingKey { panic("unimplemented") }
//func (self *FontSettings) GetInitValue(key SettingKey) uint8 { panic("unimplemented") }
func (self *FontSettings) GetNumOptions(key SettingKey) uint8 { panic("unimplemented") }
func (self *FontSettings) Each(func(key SettingKey, name string)) {
	panic("unimplemented")
}
func (self *FontSettings) EachOption(key SettingKey, each func(optionIndex uint8, optionName string)) {
	panic("unimplemented")
}

func (self *FontSettings) Validate(mode FmtValidation) error {
	// default checks
	// ...

	// strict checks
	if mode == FmtStrict {
		// TODO:
		// - go through var defs and ensure init value is in range
		// - make sure every setting is not repeated and is within numVars
		// - make sure every setting name is correct
		// - make sure every setting name comes in order? nah.
		// - make sure the offsets to names are correct
		panic("unimplemented")
	}

	return nil
}

// --- mapping section ---

type FontMapping Font

// More than this might be needed for more complex switch caches, but
// it might not even be relevant, maybe the default cache is ok, without
// any interface at all.
func (self *FontMapping) NumSwitchTypes() uint8 { panic("unimplemented") }

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

	return nil
}

// --- rewrite rules section ---

type FontRewrites Font
func (self *FontRewrites) NumGlyphRules() uint16 { panic("unimplemented") }
func (self *FontRewrites) NumUTF8Rules() uint16 { panic("unimplemented") }

type GlyphRewriteRule struct { data []uint8 }
func (self *GlyphRewriteRule) Condition() (uint8, bool) { panic("unimplemented") }
func (self *GlyphRewriteRule) Replacement() GlyphIndex { panic("unimplemented") }
func (self *GlyphRewriteRule) Sequence(each func(GlyphIndex)) { panic("unimplemented") }
func (self *GlyphRewriteRule) SequenceSize() uint8 { panic("unimplemented") }

func (self *FontRewrites) GetGlyphRule(index uint16) GlyphRewriteRule {
	panic("unimplemented")
}

type Utf8RewriteRule struct { data []uint8 }
func (self *Utf8RewriteRule) Condition() (uint8, bool) { panic("unimplemented") }
func (self *Utf8RewriteRule) Replacement() rune { panic("unimplemented") }
func (self *Utf8RewriteRule) Sequence(each func(rune)) { panic("unimplemented") }
func (self *Utf8RewriteRule) SequenceSize() uint8 { panic("unimplemented") }

func (self *FontRewrites) GetUtf8Rule(index uint16) Utf8RewriteRule {
	panic("unimplemented")
}

func (self *FontRewrites) Validate(mode FmtValidation) error {
	// default checks
	// ...

	// strict checks
	if mode == FmtStrict {
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
