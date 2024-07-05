package ggfnt

import "io"
import "fmt"
import "errors"
import "image"
import "image/color"
import "compress/gzip"
import "unsafe"

import "github.com/tinne26/ggfnt/internal"
import "github.com/tinne26/ggfnt/mask"

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

// Utility method returning the ascent + descent + line gap.
func (self *FontMetrics) LineHeight() int {
	return int(self.Ascent()) + int(self.Descent()) + int(self.LineGap())
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
func (self *FontMetrics) UppercaseAscent() uint8 {
	return self.Data[self.OffsetToMetrics + 8]
}
func (self *FontMetrics) MidlineAscent() uint8 {
	return self.Data[self.OffsetToMetrics + 9]
}
func (self *FontMetrics) HorzInterspacing() uint8 {
	return self.Data[self.OffsetToMetrics + 10]
}
func (self *FontMetrics) VertInterspacing() uint8 {
	return self.Data[self.OffsetToMetrics + 11]
}
func (self *FontMetrics) LineGap() uint8 {
	return self.Data[self.OffsetToMetrics + 12]
}
func (self *FontMetrics) VertLineWidth() uint8 {
	return self.Data[self.OffsetToMetrics + 13]
}
func (self *FontMetrics) VertLineGap() uint8 {
	return self.Data[self.OffsetToMetrics + 14]
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

type FontColor Font

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

func (self *FontColor) DyeAlphasCount() uint8 {
	lastDyeStart := self.Data[self.OffsetToColorSections + 2 + uint32(self.NumDyes()) - 1]
	if lastDyeStart == 0 { panic(invalidFontData) }
	dyeAlphasCount := 255 - lastDyeStart + 1
	if dyeAlphasCount < self.NumDyes() { panic(invalidFontData) }
	return dyeAlphasCount
}

// Notice: the string is an unsafe.String, so don't store it indefinitely.
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
// font color range [0, 255].
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
	if numColorSections > 255 { panic(invalidFontData) } // discretionary assertion
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

// local equivalent for (*Font)(self).Metrics().HasVertLayout()
func (self *FontGlyphs) hasVertLayout() bool {
	return self.Data[self.OffsetToMetrics + 2] == 1
}

// Same as [FontMetrics.NumGlyphs]() (it actually refers to the data in the font metrics section).
func (self *FontGlyphs) Count() uint16 {
	return internal.DecodeUint16LE(self.Data[self.OffsetToMetrics + 0 : self.OffsetToMetrics + 2])
}

func (self *FontGlyphs) NamedCount() uint16 {
	return internal.DecodeUint16LE(self.Data[self.OffsetToGlyphNames + 0 : self.OffsetToGlyphNames + 2])
}

// If not found, the return value will be [GlyphMissing].
func (self *FontGlyphs) FindIndexByName(name string) GlyphIndex {
	// binary search the name
	numEntries := uint32(self.NamedCount())
	if numEntries == 0 { return GlyphMissing }
	minIndex, maxIndex := uint32(0), numEntries - 1
	for minIndex < maxIndex {
		midIndex := (minIndex + maxIndex) >> 1 // no overflow possible due to numEntries being uint16
		value := self.getNthGlyphName(midIndex, numEntries)
		if bytesSmallerThanStr(value, name) {
			minIndex = midIndex + 1
		} else {
			maxIndex = midIndex
		}
	}

	if minIndex >= numEntries { return GlyphMissing }
	value := self.getNthGlyphName(minIndex, numEntries)
	if !bytesEqStr(value, name) { return GlyphMissing }
	idOffset := self.OffsetToGlyphNames + 2 + (minIndex << 1)
	return GlyphIndex(internal.DecodeUint16LE(self.Data[idOffset : idOffset + 2]))
}

func (self *FontGlyphs) getNthGlyphName(nth uint32, numNamedGlyphs uint32) []byte {
	endOffsetsIndex := self.OffsetToGlyphNames + 2 + (numNamedGlyphs << 1)
	glyphNameEndOffsetIndex := endOffsetsIndex + (nth << 1) + nth
	endOffset := internal.DecodeUint24LE(self.Data[glyphNameEndOffsetIndex : glyphNameEndOffsetIndex + 3])
	namesIndex := endOffsetsIndex + (numNamedGlyphs << 1) + numNamedGlyphs
	if nth == 0 { return self.Data[namesIndex : namesIndex + endOffset] }
	prevEndOffset := internal.DecodeUint24LE(self.Data[glyphNameEndOffsetIndex - 3 : glyphNameEndOffsetIndex])
	return self.Data[namesIndex + prevEndOffset : namesIndex + endOffset]
}

func bytesSmallerThanStr(bytes []byte, str string) bool {
	for i := 0; i < len(bytes); i++ {
		if i >= len(str) { return true }
		if bytes[i] > str[i] { return false }
	}
	if len(bytes) == len(str) { return false } // equal
	return true // smaller
}

func bytesEqStr(bytes []byte, str string) bool {
	if len(bytes) != len(str) { return false }
	for i := 0; i < len(bytes); i++ {
		if bytes[i] != str[i] { return false }
	}
	return true
}

func (self *FontGlyphs) RasterizeMask(glyphIndex GlyphIndex) *image.Alpha {
	startOffset, endOffset := self.getGlyphDataOffsets(glyphIndex)
	if self.hasVertLayout() { startOffset += 4 } else { startOffset += 1 }
	numGlyphs := uint32(self.Count())
	offsetToMasksData := self.OffsetToGlyphMasks + (numGlyphs << 1) + numGlyphs
	glyphMask, err := mask.Rasterize(self.Data[offsetToMasksData + startOffset : offsetToMasksData + endOffset])
	if err != nil { panic(err) }
	return glyphMask
}

func (self *FontGlyphs) Advance(glyphIndex GlyphIndex) uint8 {
	numGlyphs := self.Count()
	if uint16(glyphIndex) >= numGlyphs { panic("glyphIndex out of range") }  // discretionary assertion
	
	glyphDataStartOffset := self.getGlyphDataStartOffset(glyphIndex)
	numGlyphs32 := uint32(numGlyphs)
	return self.Data[self.OffsetToGlyphMasks + (numGlyphs32 << 1) + numGlyphs32 + glyphDataStartOffset]
}

func (self *FontGlyphs) Placement(glyphIndex GlyphIndex) GlyphPlacement {
	numGlyphs := self.Count()
	if uint16(glyphIndex) >= numGlyphs { panic("glyphIndex out of range") } // discretionary assertion

	glyphDataStartOffset := self.getGlyphDataStartOffset(glyphIndex)

	var placement GlyphPlacement
	numGlyphs32 := uint32(numGlyphs)
	placementDataIndex := self.OffsetToGlyphMasks + (numGlyphs32 << 1) + numGlyphs32 + glyphDataStartOffset
	placement.Advance = self.Data[placementDataIndex]
	if self.hasVertLayout() {
		placement.TopAdvance = self.Data[placementDataIndex + 1]
		placement.BottomAdvance = self.Data[placementDataIndex + 2]
		placement.HorzCenter = self.Data[placementDataIndex + 3]
	}
	return placement
}

func (self *FontGlyphs) getGlyphDataOffsets(glyphIndex GlyphIndex) (uint32, uint32) {
	index := uint32(glyphIndex)
	index = (index << 1) + index
	
	glyphDataEndOffset := internal.DecodeUint24LE(self.Data[self.OffsetToGlyphMasks + index : ])
	if index == 0 { return 0, glyphDataEndOffset }
	glyphDataStartOffset := internal.DecodeUint24LE(self.Data[self.OffsetToGlyphMasks + index - 3 : ])
	if glyphDataEndOffset <= glyphDataStartOffset { panic(invalidFontData) } // discretionary assertion
	return glyphDataStartOffset, glyphDataEndOffset
}

func (self *FontGlyphs) getGlyphDataStartOffset(glyphIndex GlyphIndex) uint32 {
	if glyphIndex == 0 { return 0 }
	index := uint32(glyphIndex)
	index = (index << 1) + index
	return internal.DecodeUint24LE(self.Data[self.OffsetToGlyphMasks + index - 3 : ])
}

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

func (self *FontSettings) NumWords() uint8 {
	return self.Data[self.OffsetToWords + 0]
}

func (self *FontSettings) GetWord(index uint8) string {
	numWords := self.NumWords()
	if index < numWords {
		offsetToWordEndOffset := self.OffsetToWords + 1 + (uint32(index) << 1)
		wordEndOffsetIndex := internal.DecodeUint16LE(self.Data[offsetToWordEndOffset : ])
		var wordStartOffsetIndex uint16
		if index > 0 {
			wordStartOffsetIndex = internal.DecodeUint16LE(self.Data[offsetToWordEndOffset - 2 : ])
		}
		if wordEndOffsetIndex <= wordStartOffsetIndex { panic(invalidFontData) }
		wordStartIndex := self.OffsetToWords + 1 + (uint32(numWords) << 1) + uint32(wordStartOffsetIndex)
		wordLen := (wordEndOffsetIndex - wordStartOffsetIndex)
		return unsafe.String(&self.Data[wordStartIndex], wordLen)
	} else {
		return GetPredefinedWord(index)
	}
}

func (self *FontSettings) Count() uint8 {
	return self.Data[self.OffsetToSettingNames]
}
//func (self *FontSettings) FindKeyByName(name string) SettingKey { panic("unimplemented") }
//func (self *FontSettings) GetInitValue(key SettingKey) uint8 { panic("unimplemented") }
func (self *FontSettings) GetNumOptions(key SettingKey) uint8 {
	if uint8(key) >= self.Count() { return 0 }

	var startOffset uint16
	keyEndOffsetIndex := self.OffsetToSettingDefinitions + (uint32(key) << 1)
	endOffset := internal.DecodeUint16LE(self.Data[keyEndOffsetIndex : ])
	if key > 0 {
		startOffset = internal.DecodeUint16LE(self.Data[keyEndOffsetIndex - 2 : ])
	}
	if endOffset <= startOffset { panic(invalidFontData) }
	numOpts := endOffset - startOffset
	if numOpts > 255 { panic(invalidFontData) }
	return uint8(numOpts)
}

func (self *FontSettings) GetOptionName(key SettingKey, option uint8) string {
	// the first part is basically the same as GetNumOptions
	numSettings := uint32(self.Count())
	key32 := uint32(key)
	if key32 >= numSettings { panic("invalid setting key") }
	keyEndOffsetIndex := self.OffsetToSettingDefinitions + (key32 << 1)
	endOffset := internal.DecodeUint16LE(self.Data[keyEndOffsetIndex : ])
	var startOffset uint16
	if key != 0 {
		startOffset = internal.DecodeUint16LE(self.Data[keyEndOffsetIndex - 2 : ])
	}
	if endOffset <= startOffset { panic(invalidFontData) }
	numOpts := endOffset - startOffset
	if numOpts > 255 { panic(invalidFontData) }

	// validate the given option
	opt32 := uint32(option)
	if opt32 >= uint32(numOpts) { panic("invalid setting option (out of bounds)") }
	optionValue := self.Data[self.OffsetToSettingDefinitions + (numSettings << 1) + uint32(startOffset) + opt32]
	return self.GetWord(optionValue)
}

func (self *FontSettings) Each(fn func(key SettingKey, name string)) {
	numSettings := int(self.Count())
	var prevNameEnd int
	var offsetToNameEnds int = int(self.OffsetToSettingNames + 1)
	var offsetToNames int = offsetToNameEnds + numSettings*2
	for i := 0; i < numSettings; i++ {
		nameEndOffset := int(internal.DecodeUint16LE(self.Data[offsetToNameEnds + i*2 : ]))
		nameLen := nameEndOffset - prevNameEnd
		fn(SettingKey(i), unsafe.String(&self.Data[offsetToNames + prevNameEnd], nameLen))
		prevNameEnd = nameEndOffset
	}
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

// Returned from [FontMapping.Utf8]().
type GlyphMappingGroup struct {
	font *Font
	offset uint32
	switchType uint8
	caseBranch uint8
	directMapping bool
}
func (self *GlyphMappingGroup) Size() uint8 {
	if self.directMapping { return 1 }
	return (0b0111_1111 & self.font.Data[self.offset + 0]) + 1
}
func (self *GlyphMappingGroup) AnimationFlags() AnimationFlags {
	if self.directMapping { return 0 }
	if (0b0111_1111 & self.font.Data[self.offset + 0]) == 0 { return 0 }
	return AnimationFlags(self.font.Data[self.offset + 2])
}
func (self *GlyphMappingGroup) CaseBranch() uint8 {
	return self.caseBranch
}

// Precondition: choice must be between 0 and GlyphMappingGroup.Size() - 1
func (self *GlyphMappingGroup) Select(choice uint8) GlyphIndex {
	// basic case
	if self.directMapping {
		if choice != 0 { panic("choice outside valid range") } // discretionary assertion
		return GlyphIndex(internal.DecodeUint16LE(self.font.Data[self.offset : ]))
	}

	// general case
	info := self.font.Data[self.offset + 0]
	size := (info & 0b0111_1111) + 1
	if choice >= size { panic("choice outside valid range") } // discretionary assertion

	// no animation case (single glyph)
	if size == 1 { return GlyphIndex(internal.DecodeUint16LE(self.font.Data[self.offset + 1 : ])) }

	// multi-glyph case (group)
	if (info & 0b1000_0000) != 0 { // range case
		return GlyphIndex(internal.DecodeUint16LE(self.font.Data[self.offset + 2 : ]) + uint16(choice))
	} else {
		return GlyphIndex(internal.DecodeUint16LE(self.font.Data[self.offset + 2 + (uint32(choice) << 2) : ]))
	}
}

type FontMapping Font

// More than this might be needed for more complex switch caches, but
// it might not even be relevant, maybe the default cache is ok, without
// any interface at all.
func (self *FontMapping) NumSwitchTypes() uint8 {
	return self.Data[self.OffsetToMappingSwitches + 0]
}

func (self *FontMapping) EvaluateSwitch(switchKey uint8, settings []uint8) uint8 {
	numSwitchTypes := self.NumSwitchTypes()
	if switchKey >= numSwitchTypes { panic("invalid switch key") }

	switchEndOffsetIndex := self.OffsetToMappingSwitches + 1 + (uint32(switchKey) << 1)
	endOffset := internal.DecodeUint16LE(self.Data[switchEndOffsetIndex : ])
	var startOffset uint16 = 0
	if switchKey > 0 {
		startOffset = internal.DecodeUint16LE(self.Data[switchEndOffsetIndex - 2 : ])
	}
	if endOffset <= startOffset { panic(invalidFontData) }
	
	offsetToMappingSwitchesData := self.OffsetToMappingSwitches + 1 + (uint32(numSwitchTypes) << 1)
	var caseCombinations uint8 = 1
	var result uint8
	endOffset -= 1 // we will evaluate from last to first
	for {
		settingKey := SettingKey(self.Data[offsetToMappingSwitchesData + uint32(endOffset)])
		result += settings[settingKey]*caseCombinations
		if endOffset == startOffset { break }
		caseCombinations *= (*FontSettings)(self).GetNumOptions(SettingKey(settingKey))
		endOffset -= 1
	}
	return result
}

func (self *FontMapping) NumEntries() uint16 {
	return internal.DecodeUint16LE(self.Data[self.OffsetToMapping : ])
}

func (self *FontMapping) Utf8WithCache(codePoint rune, settingsCache *SettingsCache) (GlyphMappingGroup, bool) {
	// binary search the code point
	target := uint32(int32(codePoint))
	numEntries := self.NumEntries()
	offsetToSearchIndex := int(self.OffsetToMapping + 2)
	minIndex, maxIndex := int(0), int(numEntries) - 1
	for minIndex < maxIndex {
		midIndex := (minIndex + maxIndex) >> 1 // no overflow possible due to numEntries being uint16
		value := internal.DecodeUint32LE(self.Data[offsetToSearchIndex + (midIndex << 2) : ])
		if value < target {
			minIndex = midIndex + 1
		} else {
			maxIndex = midIndex
		}
	}

	if minIndex >= int(numEntries) { return GlyphMappingGroup{}, false }
	value := internal.DecodeUint32LE(self.Data[offsetToSearchIndex + (minIndex << 2) : ])
	if value != target { return GlyphMappingGroup{}, false }
	
	// target found at minIndex
	offsetToMappingEndOffsets := offsetToSearchIndex + (int(numEntries) << 2)
	codePointEndOffsetIndex := offsetToMappingEndOffsets + minIndex + (minIndex << 1)
	endOffset := internal.DecodeUint24LE(self.Data[codePointEndOffsetIndex : ])
	var startOffset uint32
	if minIndex > 0 {
		startOffset = internal.DecodeUint24LE(self.Data[codePointEndOffsetIndex - 3 : ])
	}
	if endOffset <= startOffset { panic(invalidFontData) }
	offsetToMappingData := offsetToMappingEndOffsets + int(numEntries) + (int(numEntries) << 1)
	switchType := self.Data[offsetToMappingData + int(startOffset)]
	startOffset += 1 // move from switch type offset index to actual data
	
	// basic case: inconditional mapping
	if switchType == 255 {
		offset := uint32(offsetToMappingData) + startOffset
		return GlyphMappingGroup{
			font: (*Font)(self),
			offset: offset,
			switchType: switchType,
			directMapping: true,
		}, true
	}

	// general case: switch staircase
	var targetSwitchCase uint8
	if switchType != 254 {
		var cached bool
		targetSwitchCase, cached = settingsCache.GetMappingCase(switchType)
		if !cached {
			targetSwitchCase = self.EvaluateSwitch(switchType, settingsCache.UnsafeSlice())
			settingsCache.CacheMappingCase(switchType, targetSwitchCase)
		}
	}

	group := GlyphMappingGroup{ font: (*Font)(self), switchType: switchType, caseBranch: targetSwitchCase }
	for targetSwitchCase > 0 {
		groupInfo := self.Data[offsetToMappingData + int(startOffset)]
		groupSize := (groupInfo & 0b0111_1111) + 1
		groupDefinedAsRange := ((groupSize & 0b1000_0000) != 0)
		if groupDefinedAsRange {
			startOffset += 3 // 1 byte group size, 2 bytes base glyph
		} else {
			startOffset += 1 + (uint32(groupSize) << 1)
		}
		if groupSize > 1 { startOffset += 1 } // anim flag skip
		targetSwitchCase -= 1

		if startOffset >= endOffset { panic(invalidFontData) } // discretionary assertion
	}
	group.offset = uint32(offsetToMappingData) + startOffset
	return group, true
}

// Notice: line breaks and other control codes shouldn't be requested here,
// but manually taken into account by the caller instead.
func (self *FontMapping) Utf8(codePoint rune, settings []uint8) (GlyphMappingGroup, bool) {
	// binary search the code point
	target := uint32(int32(codePoint))
	numEntries := self.NumEntries()
	offsetToSearchIndex := int(self.OffsetToMapping + 2)
	minIndex, maxIndex := int(0), int(numEntries) - 1
	for minIndex < maxIndex {
		midIndex := (minIndex + maxIndex) >> 1 // no overflow possible due to numEntries being uint16
		value := internal.DecodeUint32LE(self.Data[offsetToSearchIndex + (midIndex << 2) : ])
		if value < target {
			minIndex = midIndex + 1
		} else {
			maxIndex = midIndex
		}
	}

	if minIndex >= int(numEntries) { return GlyphMappingGroup{}, false }
	value := internal.DecodeUint32LE(self.Data[offsetToSearchIndex + (minIndex << 2) : ])
	if value != target { return GlyphMappingGroup{}, false }
	
	// target found at minIndex
	offsetToMappingEndOffsets := offsetToSearchIndex + (int(numEntries) << 2)
	codePointEndOffsetIndex := offsetToMappingEndOffsets + minIndex + (minIndex << 1)
	endOffset := internal.DecodeUint24LE(self.Data[codePointEndOffsetIndex : ])
	var startOffset uint32
	if minIndex > 0 {
		startOffset = internal.DecodeUint24LE(self.Data[codePointEndOffsetIndex - 3 : ])
	}
	if endOffset <= startOffset { panic(invalidFontData) }
	offsetToMappingData := offsetToMappingEndOffsets + int(numEntries) + (int(numEntries) << 1)
	switchType := self.Data[offsetToMappingData + int(startOffset)]
	startOffset += 1 // move from switch type offset index to actual data
	
	// basic case: inconditional mapping
	if switchType == 255 {
		offset := uint32(offsetToMappingData) + startOffset
		return GlyphMappingGroup{
			font: (*Font)(self),
			offset: offset,
			switchType: switchType,
			directMapping: true,
		}, true
	}

	// general case: switch staircase
	var targetSwitchCase uint8
	if switchType != 254 {
		targetSwitchCase = self.EvaluateSwitch(switchType, settings)
	}

	group := GlyphMappingGroup{ font: (*Font)(self), switchType: switchType, caseBranch: targetSwitchCase }
	for targetSwitchCase > 0 {
		groupInfo := self.Data[offsetToMappingData + int(startOffset)]
		groupSize := (groupInfo & 0b0111_1111) + 1
		groupDefinedAsRange := ((groupSize & 0b1000_0000) != 0)
		if groupDefinedAsRange {
			startOffset += 3 // 1 byte group size, 2 bytes base glyph
		} else {
			startOffset += 1 + (uint32(groupSize) << 1)
		}
		if groupSize > 1 { startOffset += 1 } // anim flag skip
		targetSwitchCase -= 1

		if startOffset >= endOffset { panic(invalidFontData) } // discretionary assertion
	}
	group.offset = uint32(offsetToMappingData) + startOffset
	return group, true
}

func (self *FontMapping) Ascii(codePoint byte, settings []uint8) (GlyphMappingGroup, bool) {
	return self.Utf8(rune(codePoint), settings)
}

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

func (self *FontRewrites) NumConditions() uint8 {
	return self.Data[self.OffsetToRewriteConditions + 0]
}

func (self *FontRewrites) NumGlyphRules() uint16 {
	return internal.DecodeUint16LE(self.Data[self.OffsetToGlyphRewrites + 0 : ])
}

func (self *FontRewrites) NumUTF8Rules() uint16 {
	return internal.DecodeUint16LE(self.Data[self.OffsetToUtf8Rewrites + 0 : ])
}

func (self *FontRewrites) EvaluateCondition(conditionKey uint8, settings []uint8) bool {
	numConditions := self.NumConditions()
	if conditionKey >= numConditions { panic("invalid condition key") }

	var startOffset uint16 = 0
	endOffset := internal.DecodeUint16LE(self.Data[self.OffsetToRewriteConditions + 1 + uint32(conditionKey) : ])
	if conditionKey > 0 {
		startOffset = internal.DecodeUint16LE(self.Data[self.OffsetToRewriteConditions + uint32(conditionKey) : ])
	}
	if endOffset <= startOffset { panic(invalidFontData) }
	endOffset -= 1 // we will evaluate from last to first

	offsetToRewriteConditionsData := self.OffsetToRewriteConditions + 1 + (uint32(numConditions) << 1)
	maxDataIndex := offsetToRewriteConditionsData + uint32(endOffset)
	if int(maxDataIndex) >= len(self.Data) { panic(invalidFontData) } // discretionary assertion
	dataIndex := offsetToRewriteConditionsData + uint32(startOffset)
	endDataIndex, satisfied := self.evalConditionSubexpr(dataIndex, maxDataIndex, settings)
	if endDataIndex != maxDataIndex { panic(brokenCode) }
	return satisfied

	// - 0b000X_XXXX: `OR` condition group. The X's indicate the number of terms in the expression (can't be < 2).
	// - 0b001X_XXXX: `AND` condition group. The X's indicate the number of terms in the expression (can't be < 2).
}

// Returns the new dataIndex and the result.
func (self *FontRewrites) evalConditionSubexpr(dataIndex, maxDataIndex uint32, settings []uint8) (uint32, bool) {
	if dataIndex > maxDataIndex { panic(invalidFontData) }

	ctrl := self.Data[dataIndex]
	switch ctrl & 0b1110_0000 {
	case 0b0000_0000: // OR
		dataIndex += 1
		numTerms := (ctrl & 0b0001_1111)
		for i := uint8(0); i < numTerms; i++ {
			var satisfied bool
			dataIndex, satisfied = self.evalConditionSubexpr(dataIndex, maxDataIndex, settings)
			if dataIndex > maxDataIndex { panic(invalidFontData) } // discretionary assertion
			if satisfied {
				numSettings := uint8(min(len(settings), 255))
				for i < numTerms { // could be optimized in some cases, but it's a pain
					dataIndex = self.skipConditionSubexpr(dataIndex, maxDataIndex, numSettings)
					i += 1
				}
				return dataIndex, true
			}
		}
		return dataIndex, false
	case 0b0010_0000: // AND
		dataIndex += 1
		numTerms := (ctrl & 0b0001_1111)
		for i := uint8(0); i < numTerms; i++ {
			var satisfied bool
			dataIndex, satisfied = self.evalConditionSubexpr(dataIndex, maxDataIndex, settings)
			if dataIndex > maxDataIndex { panic(invalidFontData) } // discretionary assertion
			if !satisfied {
				numSettings := uint8(min(len(settings), 255))
				for i < numTerms { // could be optimized in some cases, but it's a pain
					dataIndex = self.skipConditionSubexpr(dataIndex, maxDataIndex, numSettings)
					i += 1
				}
				return dataIndex, false
			}
		}
		return dataIndex, true
	case 0b0100_0000: // comparison
		setting := self.Data[dataIndex + 1]
		if int(setting) > len(settings) { panic(invalidFontData) } // discretionary assertion
		operand := self.Data[dataIndex + 2]
		if (ctrl & 0b0001_0000) == 0 { // comparing two settings
			if int(operand) > len(settings) { panic(invalidFontData) } // discretionary assertion
			operand = settings[operand]
		}
		switch (ctrl & 0b0000_1111) {
		case 0b000: return dataIndex + 3, setting == operand
		case 0b001: return dataIndex + 3, setting != operand
		case 0b010: return dataIndex + 3, setting  < operand
		case 0b011: return dataIndex + 3, setting  > operand
		case 0b100: return dataIndex + 3, setting <= operand
		case 0b101: return dataIndex + 3, setting >= operand
		default:
			panic(invalidFontData)
		}
	case 0b0110_0000: // quick comparison `setting == const`
		constValue := (ctrl & 0b0001_1111)
		setting := self.Data[dataIndex + 1]
		if int(setting) > len(settings) { panic(invalidFontData) } // discretionary assertion
		return dataIndex + 2, settings[setting] == constValue
	case 0b1000_0000: // quick comparison `setting != const`
		constValue := (ctrl & 0b0001_1111)
		setting := self.Data[dataIndex + 1]
		if int(setting) > len(settings) { panic(invalidFontData) } // discretionary assertion
		return dataIndex + 2, settings[setting] != constValue
	case 0b1010_0000: // quick comparison `setting < const`
		constValue := (ctrl & 0b0001_1111)
		setting := self.Data[dataIndex + 1]
		if int(setting) > len(settings) { panic(invalidFontData) } // discretionary assertion
		return dataIndex + 2, settings[setting] < constValue
	case 0b1100_0000: // quick comparison `setting > const`
		constValue := (ctrl & 0b0001_1111)
		setting := self.Data[dataIndex + 1]
		if int(setting) > len(settings) { panic(invalidFontData) } // discretionary assertion
		return dataIndex + 2, settings[setting] > constValue
	default: // undefined control mode
		panic(invalidFontData)
	}
}

func (self *FontRewrites) skipConditionSubexpr(dataIndex, maxDataIndex uint32, numSettings uint8) uint32 {
	if dataIndex > maxDataIndex { panic(invalidFontData) }

	ctrl := self.Data[dataIndex]
	switch ctrl & 0b1110_0000 {
	case 0b0000_0000: // OR
		dataIndex += 1
		numTerms := (ctrl & 0b0001_1111)
		for i := uint8(0); i < numTerms; i++ {
			dataIndex = self.skipConditionSubexpr(dataIndex, maxDataIndex, numSettings)
			if dataIndex > maxDataIndex { panic(invalidFontData) } // discretionary assertion
		}
		return dataIndex
	case 0b0010_0000: // AND
		dataIndex += 1
		numTerms := (ctrl & 0b0001_1111)
		for i := uint8(0); i < numTerms; i++ {
			dataIndex = self.skipConditionSubexpr(dataIndex, maxDataIndex, numSettings)
			if dataIndex > maxDataIndex { panic(invalidFontData) } // discretionary assertion
		}
		return dataIndex
	case 0b0100_0000: // comparison
		if self.Data[dataIndex + 1] > numSettings { panic(invalidFontData) } // discretionary assertion
		if (ctrl & 0b0001_0000) == 0 { // comparing two settings
			if self.Data[dataIndex + 2] > numSettings { panic(invalidFontData) } // discretionary assertion
		}
		switch (ctrl & 0b0000_1111) {
		case 0b000: return dataIndex + 3
		case 0b001: return dataIndex + 3
		case 0b010: return dataIndex + 3
		case 0b011: return dataIndex + 3
		case 0b100: return dataIndex + 3
		case 0b101: return dataIndex + 3
		default:
			panic(invalidFontData)
		}
	case 0b0110_0000: // quick comparison `setting == const`
		if self.Data[dataIndex + 1] > numSettings { panic(invalidFontData) } // discretionary assertion
		return dataIndex + 2
	case 0b1000_0000: // quick comparison `setting != const`
		if self.Data[dataIndex + 1] > numSettings { panic(invalidFontData) } // discretionary assertion
		return dataIndex + 2
	case 0b1010_0000: // quick comparison `setting < const`
		if self.Data[dataIndex + 1] > numSettings { panic(invalidFontData) } // discretionary assertion
		return dataIndex + 2
	case 0b1100_0000: // quick comparison `setting > const`
		if self.Data[dataIndex + 1] > numSettings { panic(invalidFontData) } // discretionary assertion
		return dataIndex + 2
	default: // undefined control mode
		panic(invalidFontData)
	}
}

type GlyphRewriteRule internal.RawBlock
// TODO: a validation method would be nice
func (self *GlyphRewriteRule) Condition() uint8 { return self.Data[0] } // 255 means no condition
func (self *GlyphRewriteRule) HeadLen() uint8 { return self.Data[1] }
func (self *GlyphRewriteRule) BodyLen() uint8 { return self.Data[2] }
func (self *GlyphRewriteRule) TailLen() uint8 { return self.Data[3] }
func (self *GlyphRewriteRule) InLen() uint8 { return self.Data[1] + self.Data[2] + self.Data[3] }
func (self *GlyphRewriteRule) OutLen() uint8 { return self.Data[4] } // sequence size
func (self *GlyphRewriteRule) EachOut(each func(GlyphIndex)) {
	outSize := int(self.Data[4])
	for i := 5; i < 5 + outSize; i += 2 {
		each(GlyphIndex(internal.DecodeUint16LE(self.Data[i : ])))
	}
}
func (self *GlyphRewriteRule) Equals(other GlyphRewriteRule) bool {
	if len(self.Data) != len(other.Data) { return false }
	for i := 0; i < len(self.Data); i++ {
		if self.Data[i] != other.Data[i] { return false }
	}
	return true
}

func (self *FontRewrites) GetGlyphRule(index uint16) GlyphRewriteRule {
	numRules := uint32(self.NumGlyphRules())
	index32 := uint32(index)
	if index32 >= numRules { panic("invalid glyph rule index") }

	ruleEndOffsetIndex := (self.OffsetToGlyphRewrites + 2) + ((index32 << 1) + index32)
	ruleEndOffset := internal.DecodeUint24LE(self.Data[ruleEndOffsetIndex : ruleEndOffsetIndex + 3])
	var ruleStartOffset uint32
	if index > 0 {
		ruleStartOffset = internal.DecodeUint24LE(self.Data[ruleEndOffsetIndex - 3 : ruleEndOffsetIndex])
	}

	rulesBaseOffset := self.OffsetToGlyphRewrites + 2 + ((numRules << 1) + numRules)
	ruleData := self.Data[rulesBaseOffset + ruleStartOffset : rulesBaseOffset + ruleEndOffset]
	return GlyphRewriteRule(internal.RawBlock{ ruleData })
}

type Utf8RewriteRule internal.RawBlock
func (self *Utf8RewriteRule) Condition() uint8 { return self.Data[0] } // 255 means no condition
func (self *Utf8RewriteRule) HeadLen() uint8 { return self.Data[1] }
func (self *Utf8RewriteRule) BodyLen() uint8 { return self.Data[2] }
func (self *Utf8RewriteRule) TailLen() uint8 { return self.Data[3] }
func (self *Utf8RewriteRule) InLen() uint8 { return self.Data[1] + self.Data[2] + self.Data[3] }
func (self *Utf8RewriteRule) OutLen() uint8 { return self.Data[4] } // sequence size
func (self *Utf8RewriteRule) EachOut(each func(rune)) {
	outSize := int(self.Data[4])
	for i := 5; i < 5 + outSize; i += 2 {
		each(rune(internal.DecodeUint32LE(self.Data[i : ])))
	}
}
func (self *Utf8RewriteRule) Equals(other Utf8RewriteRule) bool {
	if len(self.Data) != len(other.Data) { return false }
	for i := 0; i < len(self.Data); i++ {
		if self.Data[i] != other.Data[i] { return false }
	}
	return true
}
func (self *Utf8RewriteRule) String() string {
	return fmt.Sprintf("Utf8RewriteRule%v", self.Data)
}

func (self *FontRewrites) GetUtf8Rule(index uint16) Utf8RewriteRule {
	numRules := uint32(self.NumUTF8Rules())
	index32 := uint32(index)
	if index32 >= numRules { panic("invalid utf8 rule index") }

	ruleEndOffsetIndex := (self.OffsetToUtf8Rewrites + 2) + ((index32 << 1) + index32)
	ruleEndOffset := internal.DecodeUint24LE(self.Data[ruleEndOffsetIndex : ruleEndOffsetIndex + 3])
	var ruleStartOffset uint32
	if index > 0 {
		ruleStartOffset = internal.DecodeUint24LE(self.Data[ruleEndOffsetIndex - 3 : ruleEndOffsetIndex])
	}

	rulesBaseOffset := self.OffsetToUtf8Rewrites + 2 + ((numRules << 1) + numRules)
	ruleData := self.Data[rulesBaseOffset + ruleStartOffset : rulesBaseOffset + ruleEndOffset]
	return Utf8RewriteRule(internal.RawBlock{ ruleData })
}

type GlyphRewriteSet internal.RawBlock
func (self *GlyphRewriteSet) EachRange(each func(GlyphRange) error) error {
	numRanges := int(self.Data[0])
	for i := 1; i < 1 + numRanges*3; i += 3 {
		glyphIndex := GlyphIndex(internal.DecodeUint16LE(self.Data[i : i + 2]))
		err := each(GlyphRange{ First: glyphIndex, Last: glyphIndex + GlyphIndex(self.Data[i + 2]) })
		if err != nil { return err }
	}
	return nil
}

func (self *GlyphRewriteSet) EachListGlyph(each func(GlyphIndex) error) error {
	numRanges := int(self.Data[0])
	elemsIndex := 1 + numRanges*3
	numElems := int(self.Data[elemsIndex])
	for i := 0; i < numElems; i += 1 {
		dataIndex := elemsIndex + 1 + (i << 1)
		glyphIndex := GlyphIndex(internal.DecodeUint16LE(self.Data[dataIndex : dataIndex + 2]))
		err := each(glyphIndex)
		if err != nil { return err }
	}
	return nil
}

func (self *FontRewrites) NumGlyphSets() uint8 {
	return self.Data[self.OffsetToRewriteGlyphSets + 0]
}

func (self *FontRewrites) GetGlyphSet(set uint8) GlyphRewriteSet {
	numSets := uint32(self.NumGlyphSets())
	set32 := uint32(set)
	if set32 >= numSets { panic("invalid glyph set") }
	endOffsetIndex := self.OffsetToRewriteGlyphSets + 1 + (uint32(set) << 1)
	setEndOffset := uint32(internal.DecodeUint16LE(self.Data[endOffsetIndex : endOffsetIndex + 2]))
	baseSetsDataIndex := self.OffsetToRewriteGlyphSets + 1 + (numSets << 1)
	if set32 == 0 {
		return GlyphRewriteSet{ Data: self.Data[baseSetsDataIndex : baseSetsDataIndex + setEndOffset] }
	} else {
		startOffsetIndex := self.OffsetToRewriteGlyphSets + 1 + (uint32(set - 1) << 1)
		setStartOffset := uint32(internal.DecodeUint16LE(self.Data[startOffsetIndex : startOffsetIndex + 2]))
		if setStartOffset >= setEndOffset { panic(invalidFontData) }
		return GlyphRewriteSet{ Data: self.Data[baseSetsDataIndex + setStartOffset : baseSetsDataIndex + setEndOffset] }
	}
}

type Utf8RewriteSet internal.RawBlock
func (self *Utf8RewriteSet) EachRange(each func(start, end rune) error) error {
	numRanges := int(self.Data[0])
	for i := 1; i < numRanges*5; i += 5 {
		codePoint := rune(internal.DecodeUint32LE(self.Data[i : i + 4]))
		err := each(codePoint, codePoint + rune(self.Data[i + 3]))
		if err != nil { return err }
	}
	return nil
}

func (self *Utf8RewriteSet) EachListRune(each func(rune) error) error {
	numRanges := int(self.Data[0])
	elemsIndex := 1 + numRanges*5
	numElems := int(self.Data[elemsIndex])
	for i := 0; i < numElems; i += 1 {
		dataIndex := elemsIndex + 1 + (i << 2)
		err := each(rune(internal.DecodeUint32LE(self.Data[dataIndex : dataIndex + 4])))
		if err != nil { return err }
	}
	return nil
}

func (self *FontRewrites) NumUTF8Sets() uint8 {
	return self.Data[self.OffsetToRewriteUtf8Sets + 0]
}

func (self *FontRewrites) GetUtf8Set(set uint8) Utf8RewriteSet {
	numSets := uint32(self.NumUTF8Sets())
	set32 := uint32(set)
	if set32 >= numSets { panic("invalid rune set") }
	endOffsetIndex := self.OffsetToRewriteUtf8Sets + 1 + (uint32(set) << 1)
	setEndOffset := uint32(internal.DecodeUint16LE(self.Data[endOffsetIndex : endOffsetIndex + 2]))
	baseSetsDataIndex := self.OffsetToRewriteUtf8Sets + 1 + (numSets << 1)
	if set32 == 0 {
		return Utf8RewriteSet{ Data: self.Data[baseSetsDataIndex : baseSetsDataIndex + setEndOffset] }
	} else {
		startOffsetIndex := self.OffsetToRewriteUtf8Sets + 1 + (uint32(set - 1) << 1)
		setStartOffset := uint32(internal.DecodeUint16LE(self.Data[startOffsetIndex : startOffsetIndex + 2]))
		if setStartOffset >= setEndOffset { panic(invalidFontData) }
		return Utf8RewriteSet{ Data: self.Data[baseSetsDataIndex + setStartOffset : baseSetsDataIndex + setEndOffset] }
	}
}

func (self *FontRewrites) Validate(mode FmtValidation) error {
	// default checks
	// ...

	// strict checks
	if mode == FmtStrict {
		panic("unimplemented")
		// I need to validate that rewrite rules don't contain control indices,
		// with the only exception of the initial 'GlyphMissing'.
	}

	return nil
}

// --- kerning section ---

type FontKerning Font
func (self *FontKerning) NumPairs() uint32 {
	return internal.DecodeUint24LE(self.Data[self.OffsetToHorzKernings : ])
}

func (self *FontKerning) NumVertPairs() uint32 {
	return internal.DecodeUint24LE(self.Data[self.OffsetToVertKernings : ])
}

func (self *FontKerning) Get(prev, curr GlyphIndex) int8 { // binary search based
	target := (uint32(prev) << 16) | uint32(curr)
	numPairs := self.NumPairs()
	offsetToSearchIndex := int(self.OffsetToHorzKernings + 3)
	minIndex, maxIndex := int(0), int(numPairs) - 1
	for minIndex < maxIndex {
		midIndex := (minIndex + maxIndex) >> 1 // no overflow possible due to numPairs being uint24
		value := internal.DecodeUint32LE(self.Data[offsetToSearchIndex + (midIndex << 2) : ])
		if value < target {
			minIndex = midIndex + 1
		} else {
			maxIndex = midIndex
		}
	}

	if minIndex >= int(numPairs) { return 0 } // no kerning
	value := internal.DecodeUint32LE(self.Data[offsetToSearchIndex + (minIndex << 2) : ])
	if value != target { return 0 } // no kerning

	// yes kerning
	return int8(self.Data[self.OffsetToHorzKernings + 3 + (numPairs << 2) + uint32(minIndex)])
}

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
