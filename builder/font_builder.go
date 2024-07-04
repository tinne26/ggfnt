package builder

import "fmt"
import "io"
import "slices"
import "errors"
import "image/color"

import "github.com/tinne26/ggfnt"
import "github.com/tinne26/ggfnt/mask"
import "github.com/tinne26/ggfnt/internal"

const debugBuildGlyphEncoding = false

const invalidInternalState = "invalid internal state"
const fontBuilderDefaultFontName = "Unnamed"
const fontBuilderDefaultFontAuthor = "Authorless"
const fontBuilderDefaultFontAbout = "No information available."

var ErrBuildNoGlyphs = errors.New("can't build font with no glyphs")
var errFontDataExceedsMax = errors.New("font data exceeds maximum size")

// A [Font] builder that allows modifying and exporting ggfnt fonts.
// It can also store and edit glyph category names, kerning classes
// and a few other elements not present in regular .ggfnt files. See
// .ggwkfnt in the spec document for more details.
//
// This object should never replace a [Font] outside the edition context.
type Font struct {
	// ---- internal glyph UID mapping ----
	glyphOrder []uint64

	// ---- internal buffers to reduce allocations on operations ----
	tempGlyphIndexLookup map[uint64]uint16
	tempSortingBuffer []uint64
	tempMaskEncoder mask.Encoder

	// ---- font data ----
	// header
	fontID uint64
	versionMajor uint16
	versionMinor uint16
	firstVersionDate ggfnt.Date
	majorVersionDate ggfnt.Date
	minorVersionDate ggfnt.Date
	fontName string
	fontFamily string
	fontAuthor string
	fontAbout string

	// metrics
	hasVertLayout bool
	monoWidth uint8
	ascent uint8
	extraAscent uint8
	descent uint8
	extraDescent uint8
	uppercaseAscent uint8
	midlineAscent uint8
	horzInterspacing uint8
	vertInterspacing uint8
	lineGap uint8
	vertLineWidth uint8
	vertLineGap uint8

	// color sections
	numDyes uint8
	numPalettes uint8
	colorSectionStarts []uint8 // prevent modification if any options are assigned
	colorSectionNames []string
	dyeAlphas [][]uint8
	paletteColors [][]color.RGBA

	// glyphs data
	glyphData map[uint64]*glyphData

	// settings
	settings []settingEntry // removing is expensive and requires many checks and reports

	// mapping and switches
	mappingSwitches []mappingSwitchEntry // removing is expensive and requires many checks and reports
	runeMapping map[rune]mappingEntry
	// TODO: I could keep a list of "unsynced switches" to make edition "easier".
	//       it's not a particularly great idea.

	// rewrite rules
	rewriteConditions []rewriteCondition
	rewriteGlyphSets map[uint64]reGlyphSet
	rewriteRuneSets map[uint64]reRuneSet
	glyphSetsOrder []uint64
	runeSetsOrder []uint64
	glyphRules []glyphRewriteRule
	utf8Rules []utf8RewriteRule

	// kerning
	horzKerningPairs map[[2]uint64]*editionKerningPair
	vertKerningPairs map[[2]uint64]*editionKerningPair

	// ---- edition-only data ----
	categories []editionCategory
	kerningClasses []editionKerningClass
}

// TODO: the pain is that we need basically all the getter methods we
// already had in font + all the setter methods we didn't have there.

// Creates an almost empty [FontBuilder].
func New() *Font {
	builder := &Font{}

	// font ID random generation
	var fontID uint64
	var err error
	const MaxRerolls = 8
	for i := 1; i <= MaxRerolls; i++ {
		fontID, err = internal.CryptoRandUint64()
		if err != nil { panic(err) } // I'm not sure this can ever happen
		if internal.LazyEntropyUint64(fontID) >= internal.MinEntropyID { break }
		if i == MaxRerolls { panic("failed to generate font ID with sufficient entropy") }
	}

	// internal
	builder.tempGlyphIndexLookup = make(map[uint64]uint16, 32)

	// --- header ---
	builder.fontID = fontID
	builder.versionMajor = 0
	builder.versionMinor = 1
	date := ggfnt.CurrentDate()
	builder.firstVersionDate = date
	builder.majorVersionDate = date
	builder.minorVersionDate = date
	builder.fontName = fontBuilderDefaultFontName
	builder.fontFamily = fontBuilderDefaultFontName
	builder.fontAuthor = fontBuilderDefaultFontAuthor
	builder.fontAbout = fontBuilderDefaultFontAbout

	// --- metrics ---
	builder.ascent = 9
	builder.descent = 5
	builder.uppercaseAscent = 9
	builder.midlineAscent = 5
	builder.horzInterspacing = 1
	builder.lineGap = 1
	// (many omitted due to being 0)

	// --- glyphs data ---
	builder.glyphOrder = make([]uint64, 0, 64)
	builder.glyphData = make(map[uint64]*glyphData, 32)

	// --- color sections ---
	builder.numDyes = 1
	builder.colorSectionStarts = []uint8{255} // inclusive
	builder.colorSectionNames = []string{"main"}
	builder.dyeAlphas = [][]uint8{[]uint8{255}}

	// settings
	// (nothing to initialize here)

	// mapping
	builder.runeMapping = make(map[rune]mappingEntry, 32)

	// kerning
	builder.horzKerningPairs = make(map[[2]uint64]*editionKerningPair)
	builder.vertKerningPairs =make(map[[2]uint64]*editionKerningPair)

	return builder
}

// Creates a [Font] builder already initialized with the given font
// values, to make it easier to modify an existing font.
func NewFrom(font *Font) *Font {
	panic("unimplemented")
}

// Converts all the current data into a read-only [Font] object.
// This process can be quite expensive, so be careful how you use it.
func (self *Font) Build() (*ggfnt.Font, error) {
	// TODO: discrimination of what's an error and what's a panic is
	//       fairly arbitrary at the moment. I should clean it up

	var err error
	var data []byte = make([]byte, 0, 1024)
	var font internal.Font

	// (signature is not part of the raw font data)
	// data = append(data, 'w', 'k', 'g', 'f', 'n', 't')

	// get num glyphs and check amount
	if len(self.glyphData) > ggfnt.MaxGlyphs { panic(invalidInternalState) } // "font has too many glyphs"
	if len(self.glyphData) != len(self.glyphOrder) { panic(invalidInternalState) }
	if len(self.glyphData) == 0 { return nil, ErrBuildNoGlyphs }
	numGlyphs := uint16(len(self.glyphData))

	// build temp glyph index lookup, it's sometimes used for glyph names,
	// and it's often used for fast mapping tables and kernings, so we just
	// compute it and call it a day
	clear(self.tempGlyphIndexLookup)
	for index, uid := range self.glyphOrder {
		self.tempGlyphIndexLookup[uid] = uint16(index)
	}

	// --- header ---
	var appendDateTo = func(date ggfnt.Date, outBuff []byte) []byte {
		return append(internal.AppendUint16LE(outBuff, date.Year), date.Month, date.Day)
	}
	data = internal.AppendUint32LE(data, ggfnt.FormatVersion)
	data = internal.AppendUint64LE(data, self.fontID)
	data = internal.AppendUint16LE(data, self.versionMajor)
	data = internal.AppendUint16LE(data, self.versionMinor)
	data = appendDateTo(self.firstVersionDate, data)
	data = appendDateTo(self.majorVersionDate, data)
	data = appendDateTo(self.minorVersionDate, data)
	data = internal.AppendShortString(data, self.fontName)
	data = internal.AppendShortString(data, self.fontFamily)
	data = internal.AppendShortString(data, self.fontAuthor)
	data = internal.AppendString(data, self.fontAbout)

	// --- metrics ---
	err = self.GetMetricsStatus()
	if err != nil { return nil, err }
	font.OffsetToMetrics = uint32(len(data))
	data = internal.AppendUint16LE(data, numGlyphs)
	data = append(data, internal.BoolToUint8(self.hasVertLayout))
	data = append(data, self.monoWidth)
	data = append(data, self.ascent)
	data = append(data, self.extraAscent)
	data = append(data, self.descent)
	data = append(data, self.extraDescent)
	data = append(data, self.uppercaseAscent)
	data = append(data, self.midlineAscent)
	data = append(data, self.horzInterspacing)
	data = append(data, self.vertInterspacing)
	data = append(data, self.lineGap)
	data = append(data, self.vertLineWidth)
	data = append(data, self.vertLineGap)

	// --- colors ---
	numColorSections := int(self.numDyes) + int(self.numPalettes)
	if numColorSections > 255 { panic(invalidInternalState) }
	if numColorSections == 0 { panic(invalidInternalState) }
	if len(self.dyeAlphas) != int(self.numDyes) { panic(invalidInternalState) }
	if len(self.paletteColors) != int(self.numPalettes) { panic(invalidInternalState) }
	if len(self.colorSectionStarts) != numColorSections { panic(invalidInternalState) }
	if len(self.colorSectionNames) != numColorSections { panic(invalidInternalState) }
	font.OffsetToColorSections = uint32(len(data))
	data = append(data, self.numDyes) // NumDyes
	data = append(data, self.numPalettes) // NumPalettes

	if numColorSections > 0 {
		data = append(data, self.colorSectionStarts...) // ColorSectionStarts

		var offset16 uint16
		for i := uint8(0); i < uint8(numColorSections); i++ { // // ColorSectionEndOffsets
			if i < self.numDyes {
				offset16 += uint16(len(self.dyeAlphas[i]))
			} else {
				offset16 += uint16(len(self.paletteColors[i - self.numDyes])) << 2
			}
			data = internal.AppendUint16LE(data, offset16)
		}
	
		for i := uint8(0); i < uint8(numColorSections); i++ { // // ColorSections
			if i < self.numDyes { // alpha
				for _, alpha := range self.dyeAlphas[i] {
					data = append(data, alpha)
				}
			} else {
				for _, rgba := range self.paletteColors[i - self.numDyes] {
					data = append(data, rgba.R, rgba.G, rgba.B, rgba.A)
				}
			}
		}
	
		font.OffsetToColorSectionNames = uint32(len(data))
		offset16 = 0
		for i, _ := range self.colorSectionNames { // ColorSectionNameEndOffsets
			nameLen := len(self.colorSectionNames[i])
			if nameLen == 0 || nameLen > 32 { panic(invalidInternalState) }
			// notice: we aren't validating, hopefully no one messed with anything
			offset16 += uint16(nameLen)
			data = internal.AppendUint16LE(data, offset16)
		}
	
		for i, _ := range self.colorSectionNames { // ColorSectionNames
			data = append(data, self.colorSectionNames[i]...)
		}
	}

	// --- glyphs data ---
	numNamedGlyphs := uint16(0) // guaranteed to fit by construction (numGlyphs is <= MaxGlyphs)
	self.tempSortingBuffer = self.tempSortingBuffer[ : 0]
	for _, uid := range self.glyphOrder {
		glyph, found := self.glyphData[uid]
		if !found { panic(invalidInternalState) }
		if glyph.Name != "" {
			numNamedGlyphs += 1
			self.tempSortingBuffer = append(self.tempSortingBuffer, uid)
		}
	}
	
	font.OffsetToGlyphNames = uint32(len(data))
	data = internal.AppendUint16LE(data, numNamedGlyphs)
	if numNamedGlyphs > 0 {
		// sort glyph uids by name
		slices.SortFunc(self.tempSortingBuffer, func(a, b uint64) int {
			nameA := self.glyphData[a].Name
			nameB := self.glyphData[b].Name
			if nameA < nameB { return -1 }
			if nameA > nameB { return  1 }
			return 0
		})
		for _, glyphUID := range self.tempSortingBuffer { // NamedGlyphIDs
			data = internal.AppendUint16LE(data, self.tempGlyphIndexLookup[glyphUID])
		}
		endOffset := uint32(0)
		for _, glyphUID := range self.tempSortingBuffer { // GlyphNameEndOffsets
			endOffset += uint32(len(self.glyphData[glyphUID].Name))
			data = internal.AppendUint24LE(data, endOffset)
		}
		var prevName string
		for _, glyphUID := range self.tempSortingBuffer { // GlyphNames
			name := self.glyphData[glyphUID].Name
			if name == prevName {
				return nil, errors.New("duplicated glyph name '" + name + "'")
			}
			data = append(data, name...)
			prevName = name
		}
	}

	// reserve space for glyph offsets
	font.OffsetToGlyphMasks = uint32(len(data))
	glyphMaskEndOffsetsIndex := len(data)
	var offset32 uint32
	data = internal.GrowSliceByN(data, int(numGlyphs)*3)
	baseGlyphMasksIndex := len(data)
	for i := uint16(0); i < numGlyphs; i++ {
		// safety checks
		glyph := self.glyphData[self.glyphOrder[i]]
		
		// append glyph placement
		gp := glyph.Placement
		if self.hasVertLayout {
			data = append(data, gp.Advance, gp.TopAdvance, gp.BottomAdvance, gp.HorzCenter)
		} else {
			data = append(data, gp.Advance)
		}

		// append mask data (expensive to process the masks!)
		if debugBuildGlyphEncoding { fmt.Printf("encoding glyph %d\n", i) }
		data = self.tempMaskEncoder.AppendRasterOps(data, glyph.Mask)

		// write offset back on the relevant index
		offset32 = uint32(len(data) - baseGlyphMasksIndex)
		internal.EncodeUint24LE(data[glyphMaskEndOffsetsIndex : glyphMaskEndOffsetsIndex + 3], offset32)
		glyphMaskEndOffsetsIndex += 3
	}
	
	// --- settings ---
	words := make(map[string]int16, 16)
	for i, _ := range self.settings {
		self.settings[i].AppendWords(words)
	}
	var builtinWords map[string]int16
	if len(words) > 0 {
		builtinWords = make(map[string]int16, 256)
		for i := 0; i < 256; i++ {
			builtinWords[ggfnt.GetPredefinedWord(uint8(i))] = int16(i)
		}
		err := organizeWords(builtinWords, words)
		if err != nil { return nil, err }
	}
	if len(words) > 255 { panic(invalidInternalState) }
	numWords := uint8(len(words))
	font.OffsetToWords = uint32(len(data))
	data = append(data, numWords)
	if len(words) > 0 {
		wordsList := make([]string, len(words))
		for word, index := range words {
			wordsList[index] = word
		}
		slices.Sort(wordsList)

		// WordEndOffsets
		var offset uint16
		for word, _ := range words {
			offset += uint16(len(word))
			data = internal.AppendUint16LE(data, offset)
		}

		// Words | append actual words
		for word, _ := range words {
			data = append(data, word...)
		}
	}
	
	if len(self.settings) > 255 { panic(invalidInternalState) }
	numSettings := uint8(len(self.settings))
	font.OffsetToSettingNames = uint32(len(data))
	data = append(data, numSettings)
	font.OffsetToSettingDefinitions = uint32(len(data)) // temporary assign in case no settings are defined
	if numSettings > 0 {
		// SettingNameEndOffsets
		var offset uint16
		for i, _ := range self.settings {
			offset += uint16(len(self.settings[i].Name))
			data = internal.AppendUint16LE(data, offset)
		}

		// SettingNames
		for i, _ := range self.settings {
			data = append(data, self.settings[i].Name...)
		}

		// SettingEndOffsets
		font.OffsetToSettingDefinitions = uint32(len(data))
		offset = 0
		for i, _ := range self.settings {
			offset += uint16(len(self.settings[i].Options))
			data = internal.AppendUint16LE(data, offset)
		}

		// Settings
		for i, _ := range self.settings {
			data = self.settings[i].AppendTo(data, builtinWords, words)
		}
	}
	
	// --- mapping ---
	if len(self.mappingSwitches) > 255 { panic(invalidInternalState) }
	font.OffsetToMappingSwitches = uint32(len(data))
	numMappingSwitches := uint8(len(self.mappingSwitches))
	data = append(data, numMappingSwitches)
	if numMappingSwitches > 0 {
		// MappingSwitchEndOffsets
		var offset uint16
		for _, mappingSwitch := range self.mappingSwitches {
			if len(mappingSwitch.Settings) > int(numSettings) { panic(invalidInternalState) }
			offset += uint16(len(mappingSwitch.Settings))
			data = internal.AppendUint16LE(data, offset)
		}

		// MappingSwitches
		for _, mappingSwitch := range self.mappingSwitches {
			data = append(data, mappingSwitch.Settings...)
		}
	}
	
	// main mapping
	if len(self.runeMapping) > ggfnt.MaxGlyphs { panic(invalidInternalState) }
	font.OffsetToMapping = uint32(len(data))
	numMappingEntries := uint16(len(self.runeMapping))
	data = internal.AppendUint16LE(data, numMappingEntries)
	if numMappingEntries > 0 {
		// gather all code points and sort
		self.tempSortingBuffer = self.tempSortingBuffer[ : 0]
		for codePoint, _ := range self.runeMapping {
			if codePoint < 0 { panic(invalidInternalState) }
			self.tempSortingBuffer = append(self.tempSortingBuffer, uint64(uint32(codePoint)))
		}
		slices.Sort(self.tempSortingBuffer)
		
		// CodePointsIndex
		for _, codePoint := range self.tempSortingBuffer {
			data = internal.AppendUint32LE(data, uint32(codePoint))
		}

		// reserve space for MappingEndOffsets
		nextMappingEndOffsetIndex := len(data)
		data = internal.GrowSliceByN(data, int(numMappingEntries)*3)
		
		// append Mappings
		var scratchBuffer []uint16 // TODO: not very efficient
		var offset int = 0
		for _, codePoint := range self.tempSortingBuffer {
			mapping := self.runeMapping[int32(uint32(codePoint))]
			numCases := self.computeNumSwitchCases(mapping.SwitchType)
			if numCases != len(mapping.SwitchCases) {
				panic(invalidInternalState) // TODO: maybe we need an error instead of a panic here
			}

			// append mapping data
			preLen := len(data)
			var err error
			data, scratchBuffer, err = mapping.AppendTo(data, self.tempGlyphIndexLookup, scratchBuffer)
			if err != nil { return nil, err }
			offset += len(data) - preLen

			// append offset
			internal.EncodeUint24LE(data[nextMappingEndOffsetIndex : nextMappingEndOffsetIndex + 3], uint32(offset))
			nextMappingEndOffsetIndex += 3
		}
	}

	// --- rewrite rules ---
	if len(self.rewriteConditions) > 254 { panic(invalidInternalState) }
	font.OffsetToRewriteConditions = uint32(len(data))
	numConditions := uint8(len(self.rewriteConditions))
	data = append(data, numConditions) // NumConditions
	if numConditions > 0 {
		// ConditionEndOffsets
		var offset uint16
		for _, condition := range self.rewriteConditions {
			newOffset := offset + uint16(len(condition.data))
			if newOffset < offset {
				return nil, errors.New("rewrite rule conditions contain too much data (can't exceed 65535 bytes)")
			}
			offset = newOffset
			data = internal.AppendUint16LE(data, offset)
		}

		// Conditions
		for _, condition := range self.rewriteConditions {
			data = append(data, condition.data...)
		}
	}

	// rewrite sets
	font.OffsetToRewriteUtf8Sets = uint32(len(data))
	var runeSetsMap = make(map[uint64]uint8)
	data = append(data, 0) // NumUtf8Sets
	// ... (TODO)
	if len(self.rewriteRuneSets) > 0 {
		panic("rune sets encoding unimplemented")
	}
	
	if len(self.rewriteGlyphSets) > 255 { panic(invalidInternalState) }
	if len(self.rewriteGlyphSets) != len(self.glyphSetsOrder) { panic(invalidInternalState) }
	numGlyphSets := uint8(len(self.rewriteGlyphSets))
	font.OffsetToRewriteGlyphSets = uint32(len(data))
	data = append(data, numGlyphSets) // NumGlyphSets
	var glyphSetsMap = make(map[uint64]uint8)
	if len(self.rewriteGlyphSets) > 0 {
		
		// GlyphSetEndOffsets
		var offset uint32
		for index, setUID := range self.glyphSetsOrder {
			glyphSetsMap[setUID] = uint8(index)
			set, found := self.rewriteGlyphSets[setUID]
			if !found { panic(invalidInternalState) }
			offset += set.GetSize()
			if offset > 65535 {
				return nil, errors.New("rewrite glyph sets contain too much data (can't exceed 65535 bytes)")
			}
			data = internal.AppendUint16LE(data, uint16(offset))
		}

		// GlyphSets
		for _, setUID := range self.glyphSetsOrder {
			set := self.rewriteGlyphSets[setUID]
			data, err = set.AppendTo(data, self.tempGlyphIndexLookup)
			if err != nil { return nil, err }
		}
	}
	
	// utf8 rules
	if len(self.utf8Rules) > 65535 { panic(invalidInternalState) }
	font.OffsetToUtf8Rewrites = uint32(len(data))
	numUtf8Rules := uint16(len(self.utf8Rules))
	data = internal.AppendUint16LE(data, numUtf8Rules) // NumUTF8Rules
	if numUtf8Rules > 0 {
		// pad for offsets
		data = internal.GrowSliceByN(data, int(numUtf8Rules)*3)

		// UTF8Rules and UTF8RuleEndOffsets
		baseRulesIndex := len(data)
		for i, _ := range self.utf8Rules {
			var err error
			data, err = self.utf8Rules[i].AppendTo(data, runeSetsMap)
			if err != nil { return nil, err }
			offset := uint32(len(data) - baseRulesIndex)
			if offset > 16777215 {
				return nil, errors.New("utf8 rewrite rules exceed maximum allowed size of 16MiB")
			}
			offsetIndex := font.OffsetToUtf8Rewrites + 2 + uint32(i)*3
			internal.EncodeUint24LE(data[offsetIndex : offsetIndex + 3], offset)
		}
	}

	// glyph rules
	if len(self.glyphRules) > 65535 { panic(invalidInternalState) }
	font.OffsetToGlyphRewrites = uint32(len(data))
	numGlyphRules := uint16(len(self.glyphRules))
	data = internal.AppendUint16LE(data, numGlyphRules) // NumGlyphRules
	if numGlyphRules > 0 {
		// pad for offsets
		data = internal.GrowSliceByN(data, int(numGlyphRules)*3)

		// GlyphRuless and GlyphRuleEndOffsets
		baseRulesIndex := len(data)
		for i, _ := range self.glyphRules {
			var err error
			data, err = self.glyphRules[i].AppendTo(data, glyphSetsMap, self.tempGlyphIndexLookup)
			if err != nil { return nil, err }
			offset := uint32(len(data) - baseRulesIndex)
			if offset > 16777215 {
				return nil, errors.New("glyph rewrite rules exceed maximum allowed size of 16MiB")
			}
			offsetIndex := font.OffsetToGlyphRewrites + 2 + uint32(i)*3
			internal.EncodeUint24LE(data[offsetIndex : offsetIndex + 3], offset)
		}
	}

	// --- kernings ---
	if len(self.horzKerningPairs) > 16777215 { panic(invalidInternalState) }
	if len(self.vertKerningPairs) > 16777215 { panic(invalidInternalState) }
	font.OffsetToHorzKernings = uint32(len(data))
	data = internal.AppendUint24LE(data, uint32(len(self.horzKerningPairs)))
	self.tempSortingBuffer = self.tempSortingBuffer[ : 0]
	for key, _ := range self.horzKerningPairs {
		i1, found := self.tempGlyphIndexLookup[key[0]]
		if !found { panic(invalidInternalState) }
		i2, found := self.tempGlyphIndexLookup[key[1]]
		if !found { panic(invalidInternalState) }
		self.tempSortingBuffer = append(self.tempSortingBuffer, uint64(uint32(i1) << 16 | uint32(i2)))
	}
	slices.Sort(self.tempSortingBuffer)
	var prevPair uint32
	for i, pair64 := range self.tempSortingBuffer { // HorzKerningPairs
		pair := uint32(pair64)
		if pair == prevPair && i > 0 { panic(invalidInternalState) } // repeated pair (remove check?)
		data = internal.AppendUint32LE(data, pair)
	}
	for _, pair64 := range self.tempSortingBuffer { // HorzKerningValues
		// TODO: this looks soooo expensive (same for vert kerning values)
		glyphUID1 := self.glyphOrder[uint16(pair64 >> 16)]
		glyphUID2 := self.glyphOrder[uint16(pair64)]
		kerningInfo, found := self.horzKerningPairs[[2]uint64{glyphUID1, glyphUID2}]
		if !found { panic(invalidInternalState) }
		if kerningInfo.Class == 0 {
			data = append(data, uint8(kerningInfo.Value))
		} else {
			kerningClass := self.kerningClasses[kerningInfo.Class - 1]
			data = append(data, uint8(kerningClass.Value))
		}
	}
	
	font.OffsetToVertKernings = uint32(len(data))
	data = internal.AppendUint24LE(data, uint32(len(self.vertKerningPairs)))
	self.tempSortingBuffer = self.tempSortingBuffer[ : 0]
	for key, _ := range self.vertKerningPairs {
		i1, found := self.tempGlyphIndexLookup[key[0]]
		if !found { panic(invalidInternalState) }
		i2, found := self.tempGlyphIndexLookup[key[1]]
		if !found { panic(invalidInternalState) }
		self.tempSortingBuffer = append(self.tempSortingBuffer, uint64(uint32(i1) << 16 | uint32(i2)))
	}
	slices.Sort(self.tempSortingBuffer)
	prevPair = 0
	for i, pair64 := range self.tempSortingBuffer { // VertKerningPairs
		pair := uint32(pair64)
		if pair == prevPair && i > 0 { panic(invalidInternalState) } // repeated pair (remove check?)
		data = internal.AppendUint32LE(data, pair)
	}
	for _, pair64 := range self.tempSortingBuffer { // VertKerningValues
		glyphUID1 := self.glyphOrder[uint16(pair64 >> 16)]
		glyphUID2 := self.glyphOrder[uint16(pair64)]
		kerningInfo, found := self.horzKerningPairs[[2]uint64{glyphUID1, glyphUID2}]
		if !found { panic(invalidInternalState) }
		if kerningInfo.Class == 0 {
			data = append(data, uint8(kerningInfo.Value))
		} else {
			kerningClass := self.kerningClasses[kerningInfo.Class - 1]
			data = append(data, uint8(kerningClass.Value))
		}
	}
	if len(data) > ggfnt.MaxFontDataSize {
		return nil, errFontDataExceedsMax
	}

	font.Data = data
	out := ggfnt.Font(font)
	return &out, nil
}

// Exports the current data into a .ggfnt file or data blob.
func (self *Font) Export(writer io.Writer) error {
	// TODO: unclear if we can make this substantially more 
	//       efficient and if that would even be worth it.
	font, err := self.Build()
	if err != nil { return err }
	return font.Export(writer)
}

// Exports the current edition data into a .ggwkfnt file or data blob.
func (self *Font) ExportEditionData(writer io.Writer) error {
	panic("unimplemented")
}

// Clears any existing edition data and tries to parse the given
// data. If the process fails, edition data will be cleared again.
func (self *Font) ParseEditionData(reader io.Reader) error {
	self.ClearEditionData()
	var completedWithoutErrors bool
	defer func() { if !completedWithoutErrors { self.ClearEditionData() } }()
	
	var parser internal.ParsingBuffer
	parser.InitBuffers()
	parser.FileType = "ggwkfnt"

	// read signature first (this is not gzipped, so it's important)
	n, err := reader.Read(parser.TempBuff[0 : 6])
	if err != nil || n != 6 {
		return parser.NewError("failed to read file signature")
	}
	if !slices.Equal(parser.TempBuff[0 : 6], []byte{'w', 'k', 'g', 'f', 'n', 't'}) {
		return parser.NewError("invalid signature")
	}

	err = parser.InitGzipReader(reader)
	if err != nil { return parser.NewError(err.Error()) }

	// --- categories ---
	
	fontID, err := parser.ReadUint64()
	if err != nil { return err }
	if fontID != self.fontID {
		return errors.New("edition data doesn't match current font ID")
	}

	numCategories, err := parser.ReadUint8()
	if err != nil { return err }
	for i := uint8(0); i < numCategories; i++ {
		categoryName, err := parser.ReadShortStr()
		if err != nil { return err }
		err = parser.ValidateBasicSpacedName(categoryName)
		if err != nil { return err }
		self.categories = append(self.categories, editionCategory{ Name: categoryName })
	}
	for i := uint8(0); i < numCategories; i++ {
		size, err := parser.ReadUint16()
		if err != nil { return err }
		(&self.categories[i]).Size = size
	}

	numKerningClasses, err := parser.ReadUint16()
	if err != nil { return err }
	for i := uint16(0); i < numKerningClasses; i++ {
		kerningClassName, err := parser.ReadShortStr()
		if err != nil { return err }
		err = parser.ValidateBasicSpacedName(kerningClassName)
		if err != nil { return err }
		self.kerningClasses = append(self.kerningClasses, editionKerningClass{ Name: kerningClassName })
	}
	for i := uint16(0); i < numKerningClasses; i++ {
		value, err := parser.ReadInt8() // TODO: could be more efficient?
		if err != nil { return err }
		(&self.kerningClasses[i]).Value = value
	}
	
	numHorzKerningPairsWithClasses, err := parser.ReadUint32()
	if err != nil { return err }
	for i := uint32(0); i < numHorzKerningPairsWithClasses; i++ {
		firstIndex, err := parser.ReadUint16()
		if err != nil { return err }
		secondIndex, err := parser.ReadUint16()
		if err != nil { return err }
		kerningClass, err := parser.ReadUint16()
		if err != nil { return err }
		if kerningClass == 0 {
			return parser.NewError("kerning pair class can't be zero")
		}
		
		first  := self.glyphOrder[firstIndex]
		second := self.glyphOrder[secondIndex]
		kerningPair, found := self.horzKerningPairs[[2]uint64{first, second}]
		if found {
			kerningPair.Class = kerningClass
		} else {
			self.horzKerningPairs[[2]uint64{first, second}] = &editionKerningPair{ First: first, Second: second, Class: kerningClass }
		}
	}
	
	numVertKerningPairsWithClasses, err := parser.ReadUint32()
	if err != nil { return err }
	for i := uint32(0); i < numVertKerningPairsWithClasses; i++ {
		firstIndex, err := parser.ReadUint16()
		if err != nil { return err }
		secondIndex, err := parser.ReadUint16()
		if err != nil { return err }
		kerningClass, err := parser.ReadUint16()
		if err != nil { return err }
		if kerningClass == 0 {
			return parser.NewError("kerning pair class can't be zero")
		}
		
		first  := self.glyphOrder[firstIndex]
		second := self.glyphOrder[secondIndex]
		kerningPair, found := self.vertKerningPairs[[2]uint64{first, second}]
		if found {
			kerningPair.Class = kerningClass
		} else {
			self.vertKerningPairs[[2]uint64{first, second}] = &editionKerningPair{ First: first, Second: second, Class: kerningClass }
		}
	}
	
	numRewriteConditions, err := parser.ReadUint8()
	if err != nil { return err }
	if numRewriteConditions == 255 {
		return parser.NewError("ConditionNames must have at most 254 elements")
	}
	if int(numRewriteConditions) != len(self.rewriteConditions) {
		if len(self.rewriteConditions) > 254 { panic(invalidInternalState) }

		nBytes := internal.AppendByteDigits(uint8(len(self.rewriteConditions)), nil)
		return parser.NewError("ConditionNames expected to have exactly " + string(nBytes) + " elements")
	}
	for i := uint8(0); i < numRewriteConditions; i++ {
		name, err := parser.ReadShortStr()
		if err != nil { return err }
		err = parser.ValidateBasicSpacedName(name)
		if err != nil { return err }
		self.rewriteConditions[i].EditorName = name
	}

	// --- EOF ---
	// ensure we reach EOF exactly at the right time
	err = parser.EnsureEOF()
	if err != nil { return parser.NewError(err.Error()) }

	// done
	completedWithoutErrors = true
	return nil
}

func (self *Font) ClearEditionData() {
	self.categories = self.categories[ : 0]
	self.kerningClasses = self.kerningClasses[ : 0]
	for _, kerningPair := range self.horzKerningPairs {
		kerningPair.Class = 0
	}
	for _, kerningPair := range self.vertKerningPairs {
		kerningPair.Class = 0
	}
	for i, _ := range self.rewriteConditions {
		self.rewriteConditions[i].EditorName = ""
	}
}

func (self *Font) ValidateEditionData() error {
	// ...
	panic("unimplemented")
}

func (self *Font) computeNumSwitchCases(switchIndex uint8) int {
	// base special case
	if switchIndex >= 254 { return 1 }

	// general case
	var numCases int
	for i, settingIndex := range self.mappingSwitches[switchIndex].Settings {
		numOptions := len(self.settings[settingIndex].Options)
		if numOptions == 0 { panic(invalidInternalState) }
		if i == 0 { numCases = numOptions } else { numCases *= numOptions }
	}
	return numCases
}
