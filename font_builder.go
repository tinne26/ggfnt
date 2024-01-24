package ggfnt

import "io"
import "slices"
import "time"
import "errors"

const invalidInternalState = "invalid internal state"

// A [Font] builder that allows modifying and exporting ggfnt fonts.
// It can also store and edit glyph category names, kerning classes
// and a few other elements not present in regular .ggfnt files. See
// .ggwkfnt in the spec document for more details.
//
// This object should never replace a [Font] outside the edition context.
type FontBuilder struct {
	// ---- internal glyph UID mapping ----
	glyphEditIDs map[uint16]uint64

	// ---- internal buffers to reduce allocations on operations ----
	tempGlyphIndexLookup map[uint64]uint16
	tempSortingBuffer []uint64

	// ---- font data ----
	// header
	formatVersion uint32
	fontID uint64
	versionMajor uint16
	versionMinor uint16
	firstVersionDateYear uint16
	firstVersionDateMonth uint8
	firstVersionDateDay uint8
	majorVersionDateYear uint16
	majorVersionDateMonth uint8
	majorVersionDateDay uint8
	minorVersionDateYear uint16
	minorVersionDateMonth uint8
	minorVersionDateDay uint8
	fontName string
	fontFamily string
	fontAuthor string
	fontAbout string

	// metrics
	hasVertLayout bool
	monoWidth uint8
	monoHeight uint8
	ascent uint8
	extraAscent uint8
	descent uint8
	extraDescent uint8
	lowercaseAscent uint8
	horzInterspacing uint8
	vertInterspacing uint8
	lineGap uint8

	// glyphs data
	glyphsData map[uint64]*glyphData

	// coloring
	dyes []string
	palettes []Palette
	coloringSectionStarts []uint8 // prevent modification if any options are assigned
	coloringSectionsEnd uint8 // inclusive
	coloringSectionNames []string
	coloringSectionOptions [][]uint8

	// variables
	variables []variableEntry // removing is expensive and requires many checks and reports

	// mapping
	mappingModes []mappingMode
	fastMappingTables []*fastMappingTable
	runeMapping map[rune]codePointMapping

	// FSM
	// ...

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
func NewFontBuilder() *FontBuilder {
	builder := &FontBuilder{}

	// --- header ---
	builder.formatVersion = FormatVersion
	fontID, err := cryptoRandUint64()
	if err != nil { panic(err) } // I'm not sure this can ever happen
	builder.fontID = fontID
	builder.versionMajor = 0
	builder.versionMinor = 1
	year, month, day := time.Now().Date()
	builder.firstVersionDateYear = uint16(year)
	builder.majorVersionDateYear = uint16(year)
	builder.minorVersionDateYear = uint16(year)
	builder.firstVersionDateMonth = uint8(month)
	builder.majorVersionDateMonth = uint8(month)
	builder.minorVersionDateMonth = uint8(month)
	builder.firstVersionDateDay = uint8(day)
	builder.majorVersionDateDay = uint8(day)
	builder.minorVersionDateDay = uint8(day)
	builder.fontName = "Unnamed"
	builder.fontFamily = "Unnamed"
	builder.fontAuthor = "Authorless"
	builder.fontAbout = "No information available."

	// --- metrics ---
	builder.ascent = 9
	builder.descent = 5
	builder.lowercaseAscent = 5
	builder.horzInterspacing = 1
	builder.lineGap = 1
	// (many omitted due to being 0)

	// --- glyphs data ---
	builder.glyphEditIDs = make(map[uint16]uint64, 32)
	builder.glyphsData = make(map[uint64]*glyphData, 32)

	// --- coloring ---
	builder.dyes = []string{"main"}
	builder.palettes = []Palette{
		Palette{
			key: 0,
			name: "default",
			colors: []byte{255, 255, 255, 255}, // pure white only
		},
	}
	builder.coloringSectionStarts = []uint8{255}
	builder.coloringSectionsEnd = 255
	builder.coloringSectionNames = []string{"main"}
	builder.coloringSectionOptions = [][]uint8{ []uint8{0} }

	// variables
	// (nothing to initialize here)

	// mapping
	// (nothing to initialize here)

	// kerning
	builder.horzKerningPairs = make(map[[2]uint64]*editionKerningPair)
	builder.vertKerningPairs =make(map[[2]uint64]*editionKerningPair)

	panic("unimplemented")
}

// Creates a [FontBuilder] already initialized with the given font
// values, to make it easier to modify an existing font.
func NewFontBuilderFromFont(font *Font) *FontBuilder {
	panic("unimplemented")
}

// Converts all the current data into a read-only [Font] object.
// This process can be quite expensive, so be careful how you use it.
func (self *FontBuilder) Build() (*Font, error) {
	// TODO: discrimination of what's an error and what's a panic is
	//       fairly arbitrary at the moment. I should clean it up

	var err error
	var data []byte = make([]byte, 0, 1024)
	var font Font

	// (signature is not part of the raw font data)
	// data = append(data, 'w', 'k', 'g', 'f', 'n', 't')

	// get num glyphs and check amount
	if len(self.glyphsData) > MaxGlyphs { panic(invalidInternalState) } // "font has too many glyphs"
	if len(self.glyphsData) == 0 { return nil, errors.New("can't build font with no glyphs") }
	numGlyphs := uint16(len(self.glyphsData))

	// build temp glyph index lookup, it's sometimes used for glyph names,
	// and it's often used for fast mapping tables and kernings, so we just
	// compute it and call it a day
	clear(self.tempGlyphIndexLookup)
	for index, uid := range self.glyphEditIDs {
		if int(index) >= len(self.glyphsData) { panic(invalidInternalState) } // "glyph ID exceeds NumGlyphs"
		self.tempGlyphIndexLookup[uid] = index
	}

	// --- header ---
	data = appendUint32LE(data, self.formatVersion)
	data = appendUint64LE(data, self.fontID)
	data = appendUint16LE(data, self.versionMajor)
	data = appendUint16LE(data, self.versionMinor)
	data = appendUint16LE(data, self.firstVersionDateYear)
	data = append(data, self.firstVersionDateMonth)
	data = append(data, self.firstVersionDateDay)
	data = appendUint16LE(data, self.majorVersionDateYear)
	data = append(data, self.majorVersionDateMonth)
	data = append(data, self.majorVersionDateDay)
	data = appendUint16LE(data, self.minorVersionDateYear)
	data = append(data, self.minorVersionDateMonth)
	data = append(data, self.minorVersionDateDay)
	data = appendShortString(data, self.fontName)
	data = appendShortString(data, self.fontFamily)
	data = appendShortString(data, self.fontAuthor)
	data = appendString(data, self.fontAbout)

	// --- metrics ---
	font.offsetToMetrics = uint32(len(data))
	data = appendUint16LE(data, numGlyphs)
	data = append(data, boolToUint8(self.hasVertLayout))
	data = append(data, self.monoWidth)
	data = append(data, self.monoHeight)
	data = append(data, self.ascent)
	data = append(data, self.extraAscent)
	data = append(data, self.descent)
	data = append(data, self.extraDescent)
	data = append(data, self.lowercaseAscent)
	data = append(data, self.horzInterspacing)
	data = append(data, self.vertInterspacing)
	data = append(data, self.lineGap)

	// --- glyphs data ---
	numNamedGlyphs := uint16(0) // guaranteed to fit by construction (numGlyphs is <= MaxGlyphs)
	self.tempSortingBuffer = self.tempSortingBuffer[ : 0]
	for _, uid := range self.glyphEditIDs {
		glyph, found := self.glyphsData[uid]
		if !found { panic(invalidInternalState) }
		if glyph.Name != "" {
			numNamedGlyphs += 1
			self.tempSortingBuffer = append(self.tempSortingBuffer, uid)
		}
	}
	
	font.offsetToGlyphNames = uint32(len(data))
	data = appendUint16LE(data, numNamedGlyphs)
	if numNamedGlyphs > 0 {
		// sort glyph uids by name
		slices.SortFunc(self.tempSortingBuffer, func(a, b uint64) int {
			nameA := self.glyphsData[self.tempSortingBuffer[a]].Name
			nameB := self.glyphsData[self.tempSortingBuffer[b]].Name
			if nameA < nameB { return -1 }
			if nameA > nameB { return  1 }
			return 0
		})
		for _, glyphUID := range self.tempSortingBuffer { // NamedGlyphIDs
			data = appendUint16LE(data, self.tempGlyphIndexLookup[glyphUID])
		}
		endOffset := uint32(0)
		for _, glyphUID := range self.tempSortingBuffer { // GlyphNameEndOffsets
			endOffset += uint32(len(self.glyphsData[glyphUID].Name))
			data = appendUint32LE(data, endOffset)
		}
		for _, glyphUID := range self.tempSortingBuffer { // GlyphNames
			data = append(data, self.glyphsData[glyphUID].Name...)
		}
	}

	// reserve space for glyph offsets
	font.offsetToGlyphMasks = uint32(len(data))
	glyphMaskEndOffsetsIndex := len(data)
	var offset32 uint32
	data = growSliceByN(data, int(numGlyphs)*4)
	baseGlyphMasksIndex := len(data)
	for i := uint16(0); i < numGlyphs; i++ {
		// safety checks
		glyph := self.glyphsData[self.glyphEditIDs[i]]
		w, h := glyph.Mask.Rect.Dx(), glyph.Mask.Rect.Dy()
		if w != int(glyph.Bounds.MaskWidth) || h != int(glyph.Bounds.MaskHeight) {
			panic(invalidInternalState) // "glyph mask does not match explicit bounds definition"
		}
		
		// append mask bounds
		if self.hasVertLayout {
			data = glyph.Bounds.appendWithVertLayout(data)
		} else {
			data = glyph.Bounds.appendWithoutVertLayout(data)
		}

		// append mask data (expensive to process the masks!)
		var err error
		data, err = AppendMaskRasterOps(data, glyph.Mask)
		if err != nil { return nil, err } // TODO: to be seen if this can really be an error

		// write offset back on the relevant index
		offset32 += uint32(len(data) - baseGlyphMasksIndex)
		encodeUint32LE(data[glyphMaskEndOffsetsIndex : glyphMaskEndOffsetsIndex + 4], offset32)
		glyphMaskEndOffsetsIndex += 4
	}

	// --- coloring ---
	// dyes
	if len(self.dyes) > 254 { panic(invalidInternalState) } // yes, max dye is 254, referenced as 255
	font.offsetToColoring = uint32(len(data))
	data = append(data, uint8(len(self.dyes)))
	var offset16 uint16
	for _, name := range self.dyes { // DyeNameEndOffsets
		if len(name) == 0 || len(name) > 32 { panic(invalidInternalState) }
		offset16 += uint16(len(name))
		data = appendUint16LE(data, offset16)
	}
	for _, name := range self.dyes { // DyeNames
		data = append(data, name...)
	}

	// palettes
	font.offsetToColoringPalettes = uint32(len(data))
	if len(self.palettes) > 255 { panic(invalidInternalState) }
	if len(self.palettes) == 0 { panic(invalidInternalState) }
	data = append(data, uint8(len(self.palettes)))
	for i := 0; i < len(self.palettes); i++ { // PaletteDyes
		data = append(data, uint8(self.palettes[i].dye))
	}
	offset16 = 0
	for i := 0; i < len(self.palettes); i++ { // PaletteEndOffsets
		if len(self.palettes[i].colors) > 255 { panic(invalidInternalState) }
		offset16 += uint16(len(self.palettes[i].colors))
		data = appendUint16LE(data, offset16)
	}
	self.tempSortingBuffer = self.tempSortingBuffer[ : 0]
	for i := uint64(0); i < uint64(len(self.palettes)); i++ { // Palettes
		self.tempSortingBuffer = append(self.tempSortingBuffer, i)
		data = append(data, self.palettes[i].colors...)
	}

	// (sort palette indices based on palette names)
	slices.SortFunc(self.tempSortingBuffer, func(a, b uint64) int {
		nameA := self.palettes[self.tempSortingBuffer[a]].name
		nameB := self.palettes[self.tempSortingBuffer[b]].name
		if nameA < nameB { return -1 }
		if nameA > nameB { return  1 }
		return 0
	})
	
	font.offsetToColoringPaletteNames = uint32(len(data))
	offset16 = 0
	for _, index := range self.tempSortingBuffer { // PaletteNameEndOffsets
		if len(self.palettes[index].name) > 32 { panic(invalidInternalState) }
		offset16 += uint16(len(self.palettes[index].name))
		data = appendUint16LE(data, offset16)
	}
	for _, index := range self.tempSortingBuffer { // PaletteNames
		data = append(data, self.palettes[index].name...)
	}

	// coloring section
	font.offsetToColoringSections	= uint32(len(data))
	if len(self.coloringSectionStarts) > 255 { panic(invalidInternalState) }
	if len(self.coloringSectionStarts) == 0 { panic(invalidInternalState) }
	data = append(data, uint8(len(self.coloringSectionStarts))) // NumSections
	var prevSectionStart uint8
	for i := 0; i < len(self.coloringSectionStarts); i++ { // SectionStarts
		if self.coloringSectionStarts[i] <= prevSectionStart && i > 0 {
			panic(invalidInternalState) // "coloring sections not in order"
		}
		data = append(data, self.coloringSectionStarts[i])
		prevSectionStart = self.coloringSectionStarts[i]
	}
	if self.coloringSectionsEnd < prevSectionStart {
		panic(invalidInternalState) // "coloring sections not in order"
	}
	data = append(data, self.coloringSectionsEnd) // SectionsEnd

	if len(self.coloringSectionNames) != len(self.coloringSectionStarts) {
		panic(invalidInternalState)
	}
	offset16 = 0
	for i := 0; i < len(self.coloringSectionNames); i++ { // SectionNameEndOffsets
		if len(self.coloringSectionNames[i]) > 32 { panic(invalidInternalState) }
		offset16 += uint16(len(self.coloringSectionNames[i]))
		data = appendUint16LE(data, offset16)
	}
	for i := 0; i < len(self.coloringSectionNames); i++ { // SectionNames
		data = append(data, self.coloringSectionNames[i]...)
	}
	
	// section options
	font.offsetToColoringSectionOptions = uint32(len(data))
	offset16 = 0
	for i := 0; i < len(self.coloringSectionOptions); i++ { // SectionOptionEndOffsets
		if len(self.coloringSectionOptions[i]) > 16 { panic(invalidInternalState) }
		offset16 += uint16(len(self.coloringSectionOptions[i]))
		data = appendUint16LE(data, offset16)
	}
	for i := 0; i < len(self.coloringSectionOptions); i++ { // SectionOptions
		data = append(data, self.coloringSectionOptions[i]...)
	}
	
	// --- variables ---
	if len(self.variables) > 255 { panic(invalidInternalState) }
	numVariables := uint8(len(self.variables))
	font.offsetToVariables = uint32(len(data))
	data = append(data, numVariables)
	self.tempSortingBuffer = self.tempSortingBuffer[ : 0]
	for i := uint64(0); i < uint64(len(self.variables)); i++ {
		data = self.variables[i].appendValuesTo(data)
		if self.variables[i].Name != "" {
			self.tempSortingBuffer = append(self.tempSortingBuffer, i)
		}
	}

	// here tempSortingBuffer contains the indices of the named variables
	data = append(data, uint8(len(self.tempSortingBuffer)))
	if len(self.tempSortingBuffer) > 0 {
		slices.SortFunc(self.tempSortingBuffer, func(a, b uint64) int {
			nameA := self.variables[self.tempSortingBuffer[a]].Name
			nameB := self.variables[self.tempSortingBuffer[b]].Name
			if nameA < nameB { return -1 }
			if nameA > nameB { return  1 }
			return 0
		})

		for _, index := range self.tempSortingBuffer { // NamedVarKeys
			data = append(data, uint8(index))
		}
		endOffset := uint16(0)
		for _, index := range self.tempSortingBuffer { // VarNameEndOffsets
			nameLen := len(self.variables[index].Name)
			if nameLen > 32 || nameLen == 0 { panic(invalidInternalState) }
			endOffset += uint16(nameLen)
			data = appendUint16LE(data, endOffset)
		}
		for _, index := range self.tempSortingBuffer { // VariableNames
			data = append(data, self.variables[index].Name...)
		}
	}
	
	// --- mapping ---
	if len(self.mappingModes) > 255 { panic(invalidInternalState) }
	font.offsetToMappingModes = uint32(len(data))
	data = append(data, uint8(len(self.mappingModes)))
	offset16 = 0
	for i, _ := range self.mappingModes { // MappingModeRoutineEndOffsets
		if len(self.mappingModes[i].Program) > 255 { panic(invalidInternalState) }
		offset16 += uint16(len(self.mappingModes[i].Program))
		data = appendUint16LE(data, offset16)
	}
	for i, _ := range self.mappingModes { // MappingModeRoutines
		data = append(data, self.mappingModes[i].Program...)
	}

	// fast mapping tables
	if len(self.fastMappingTables) > 255 { panic(invalidInternalState) }
	data = append(data, uint8(len(self.fastMappingTables)))
	totalUsedMem := 0
	for i := 0; i < len(self.fastMappingTables); i++ { // FastMappingTables
		font.offsetsToFastMapTables = append(font.offsetsToFastMapTables, uint32(len(data)))
		preLen := len(data)
		data, err = self.fastMappingTables[i].appendTo(data, self.tempGlyphIndexLookup)
		if err != nil { return nil, err }
		totalUsedMem += len(data) - preLen
		if totalUsedMem > maxFastMappingTablesSize {
			return nil, errors.New("fast mapping tables total size exceeds the limit") // TODO: err or panic?
		}
	}
	
	// main mapping
	if len(self.runeMapping) > 65535 { panic(invalidInternalState) }
	font.offsetToMainMappings = uint32(len(data))
	data = appendUint16LE(data, uint16(len(self.runeMapping)))
	self.tempSortingBuffer = self.tempSortingBuffer[ : 0]
	for codePoint, _ := range self.runeMapping { // CodePointList
		if codePoint < 0 { panic(invalidInternalState) }
		self.tempSortingBuffer = append(self.tempSortingBuffer, uint64(uint32(codePoint)))
	}
	slices.Sort(self.tempSortingBuffer) // regular sort
	for _, codePoint := range self.tempSortingBuffer {
		data = appendUint32LE(data, uint32(codePoint))
	}
	for _, codePoint := range self.tempSortingBuffer { // CodePointModes
		data = append(data, self.runeMapping[int32(uint32(codePoint))].Mode)
	}
	var offset int = 0
	for _, codePoint := range self.tempSortingBuffer { // CodePointMainIndices
		mapping := self.runeMapping[int32(uint32(codePoint))]
		if len(mapping.Indices) == 0 { panic(invalidInternalState) }
		if len(mapping.Indices) == 1 {
			if mapping.Mode != 255 { panic(invalidInternalState) }
			data = appendUint16LE(data, mapping.Indices[0])
		} else {
			if mapping.Mode == 255 { panic(invalidInternalState) }
			if len(mapping.Indices) > maxGlyphsPerCodePoint {
				panic(invalidInternalState)
			}
			offset += len(mapping.Indices)
			if offset > 65535 {
				return nil, errors.New("too many total glyph indices for custom mode code points")
			}
			data = appendUint16LE(data, uint16(offset))
		}
	}

	for _, codePoint := range self.tempSortingBuffer { // CodePointModeIndices
		mapping := self.runeMapping[int32(uint32(codePoint))]
		if mapping.Mode == 255 { continue }
		for _, index := range mapping.Indices {
			data = appendUint16LE(data, index)
		}
	}

	// --- FSMs ---
	// TODO: to be designed

	// --- kernings ---
	if len(self.horzKerningPairs) > MaxFontDataSize { panic(invalidInternalState) }
	if len(self.vertKerningPairs) > MaxFontDataSize { panic(invalidInternalState) }
	font.offsetToHorzKernings = uint32(len(data))
	data = appendUint32LE(data, uint32(len(self.horzKerningPairs)))
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
		data = appendUint32LE(data, pair)
	}
	for _, pair64 := range self.tempSortingBuffer { // HorzKerningValues
		// TODO: this looks soooo expensive (same for vert kerning values)
		glyphUID1, found := self.glyphEditIDs[uint16(pair64 >> 16)]
		if !found { panic(invalidInternalState) }
		glyphUID2, found := self.glyphEditIDs[uint16(pair64)]
		if !found { panic(invalidInternalState) }
		kerningInfo, found := self.horzKerningPairs[[2]uint64{glyphUID1, glyphUID2}]
		if !found { panic(invalidInternalState) }
		if kerningInfo.Class == 0 {
			data = append(data, uint8(kerningInfo.Value))
		} else {
			kerningClass := self.kerningClasses[kerningInfo.Class - 1]
			data = append(data, uint8(kerningClass.Value))
		}
	}
	
	font.offsetToVertKernings = uint32(len(data))
	data = appendUint32LE(data, uint32(len(self.vertKerningPairs)))
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
		data = appendUint32LE(data, pair)
	}
	for _, pair64 := range self.tempSortingBuffer { // VertKerningValues
		glyphUID1, found := self.glyphEditIDs[uint16(pair64 >> 16)]
		if !found { panic(invalidInternalState) }
		glyphUID2, found := self.glyphEditIDs[uint16(pair64)]
		if !found { panic(invalidInternalState) }
		kerningInfo, found := self.horzKerningPairs[[2]uint64{glyphUID1, glyphUID2}]
		if !found { panic(invalidInternalState) }
		if kerningInfo.Class == 0 {
			data = append(data, uint8(kerningInfo.Value))
		} else {
			kerningClass := self.kerningClasses[kerningInfo.Class - 1]
			data = append(data, uint8(kerningClass.Value))
		}
	}
	if len(data) > MaxFontDataSize {
		return nil, errors.New("font data exceeds maximum size")
	}

	font.data = data
	return &font, nil
}

// Exports the current data into a .ggfnt file or data blob.
func (self *FontBuilder) Export(writer io.Writer) error {
	// TODO: using Build() and just export? lazy dev?
	panic("unimplemented")
}

// Exports the current edition data into a .ggwkfnt file or data blob.
func (self *FontBuilder) ExportEditionData(writer io.Writer) error {
	panic("unimplemented")
}

// Clears any existing edition data and tries to parse the given
// data. If the process fails, edition data will be cleared again.
func (self *FontBuilder) ParseEditionData(reader io.Reader) error {
	self.ClearEditionData()
	var completedWithoutErrors bool
	defer func() { if !completedWithoutErrors { self.ClearEditionData() } }()
	
	var parser parsingBuffer
	parser.InitBuffers()
	parser.fileType = "ggwkfnt"

	// read signature first (this is not gzipped, so it's important)
	n, err := reader.Read(parser.tempBuff[0 : 6])
	if err != nil || n != 6 {
		return parser.NewError("failed to read file signature")
	}
	if !slices.Equal(parser.tempBuff[0 : 6], []byte{'w', 'k', 'g', 'f', 'n', 't'}) {
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
		
		first, found := self.glyphEditIDs[firstIndex]
		if !found { return parser.NewError("kerning pair glyph not found") }
		second, found := self.glyphEditIDs[secondIndex]
		if !found { return parser.NewError("kerning pair glyph not found") }

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
		
		first, found := self.glyphEditIDs[firstIndex]
		if !found { return parser.NewError("kerning pair glyph not found") }
		second, found := self.glyphEditIDs[secondIndex]
		if !found { return parser.NewError("kerning pair glyph not found") }

		kerningPair, found := self.vertKerningPairs[[2]uint64{first, second}]
		if found {
			kerningPair.Class = kerningClass
		} else {
			self.vertKerningPairs[[2]uint64{first, second}] = &editionKerningPair{ First: first, Second: second, Class: kerningClass }
		}
	}
	
	numNamedMappingModes, err := parser.ReadUint8()
	if err != nil { return err }
	if numNamedMappingModes == 255 {
		return parser.NewError("MappingModeNames must have at most 254 elements")
	}
	if int(numNamedMappingModes) != len(self.mappingModes) {
		return parser.NewError("MappingModeNames must have at most 254 elements")
	}
	for i := uint8(0); i < numNamedMappingModes; i++ {
		modeName, err := parser.ReadShortStr()
		if err != nil { return err }
		err = parser.ValidateBasicSpacedName(modeName)
		if err != nil { return err }
		self.mappingModes[i].Name = modeName
	}

	// --- EOF ---
	// ensure we reach EOF exactly at the right time
	err = parser.EnsureEOF()
	if err != nil { return parser.NewError(err.Error()) }

	// done
	completedWithoutErrors = true
	return nil
}

func (self *FontBuilder) ClearEditionData() {
	self.categories = self.categories[ : 0]
	self.kerningClasses = self.kerningClasses[ : 0]
	for _, kerningPair := range self.horzKerningPairs {
		kerningPair.Class = 0
	}
	for _, kerningPair := range self.vertKerningPairs {
		kerningPair.Class = 0
	}
	for i, _ := range self.mappingModes {
		self.mappingModes[i].Name = ""
	}
}

func (self *FontBuilder) ValidateEditionData() error {
	// ...
	panic("unimplemented")
}
