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
	lowercaseAscent uint8
	horzInterspacing uint8
	vertInterspacing uint8
	lineGap uint8
	vertLineWidth uint8
	vertLineGap uint8

	// glyphs data
	glyphData map[uint64]*glyphData

	// color sections
	colorSectionModes []uint8
	colorSectionStarts []uint8 // prevent modification if any options are assigned
	colorSectionNames []string
	colorSections [][]color.Color // either color.RGBA or color.Alpha

	// variables
	variables []variableEntry // removing is expensive and requires many checks and reports

	// mapping
	mappingModes []mappingMode
	fastMappingTables []*FastMappingTable
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
	builder.lowercaseAscent = 5
	builder.horzInterspacing = 1
	builder.lineGap = 1
	// (many omitted due to being 0)

	// --- glyphs data ---
	builder.glyphOrder = make([]uint64, 0, 64)
	builder.glyphData = make(map[uint64]*glyphData, 32)

	// --- color sections ---
	builder.colorSectionModes = []uint8{0} // 0 for alpha scale (dye), 1 for palette
	builder.colorSectionStarts = []uint8{255} // inclusive
	builder.colorSectionNames = []string{"main"}
	builder.colorSections = [][]color.Color{[]color.Color{color.Alpha{255}}} // either color.RGBA or color.Alpha

	// variables
	// (nothing to initialize here)

	// mapping
	builder.runeMapping = make(map[rune]codePointMapping, 32)

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
	font.OffsetToMetrics = uint32(len(data))
	data = internal.AppendUint16LE(data, numGlyphs)
	data = append(data, internal.BoolToUint8(self.hasVertLayout))
	data = append(data, self.monoWidth)
	data = append(data, self.ascent)
	data = append(data, self.extraAscent)
	data = append(data, self.descent)
	data = append(data, self.extraDescent)
	data = append(data, self.lowercaseAscent)
	data = append(data, self.horzInterspacing)
	data = append(data, self.vertInterspacing)
	data = append(data, self.lineGap)
	data = append(data, self.vertLineWidth)
	data = append(data, self.vertLineGap)

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
			nameA := self.glyphData[self.tempSortingBuffer[a]].Name
			nameB := self.glyphData[self.tempSortingBuffer[b]].Name
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
			data = internal.AppendUint32LE(data, endOffset)
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
	data = internal.GrowSliceByN(data, int(numGlyphs)*4)
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
		offset32 += uint32(len(data) - baseGlyphMasksIndex)
		internal.EncodeUint32LE(data[glyphMaskEndOffsetsIndex : glyphMaskEndOffsetsIndex + 4], offset32)
		glyphMaskEndOffsetsIndex += 4
	}

	// --- color sections ---
	if len(self.colorSectionModes) > 255 { panic(invalidInternalState) }
	if len(self.colorSectionModes) != len(self.colorSectionStarts) { panic(invalidInternalState) }
	if len(self.colorSectionModes) != len(self.colorSectionNames) { panic(invalidInternalState) }
	if len(self.colorSectionModes) != len(self.colorSections) { panic(invalidInternalState) }
	font.OffsetToColorSections = uint32(len(data))
	data = append(data, uint8(len(self.colorSectionModes))) // NumColorSections
	data = append(data, self.colorSectionModes...) // ColorSectionModes
	data = append(data, self.colorSectionStarts...) // ColorSectionStarts

	var offset16 uint16
	for i, _ := range self.colorSections { // // ColorSectionEndOffsets
		if len(self.colorSections[i]) > 255 { panic(invalidInternalState) }
		colorSectionNumColors := uint16(len(self.colorSections[i]))
		switch self.colorSectionModes[i] {
		case 0: // alpha
			offset16 += colorSectionNumColors
		case 1: // palette
			offset16 += (colorSectionNumColors << 2)
		default:
			panic(invalidInternalState)
		}
		data = internal.AppendUint16LE(data, offset16)
	}

	for i, _ := range self.colorSections { // // ColorSections
		switch self.colorSectionModes[i] {
		case 0: // alpha
			for j, _ := range self.colorSections[i] {
				data = append(data, self.colorSections[i][j].(color.Alpha).A)
			}
		case 1: // palette
			for j, _ := range self.colorSections[i] {
				rgba := self.colorSections[i][j].(color.RGBA)
				data = append(data, rgba.R, rgba.G, rgba.B, rgba.A)
			}
		default:
			panic(invalidInternalState)
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

	for i, _ := range self.colorSectionNames { // ColorSectionNameEndOffsets
		data = append(data, self.colorSectionNames[i]...)
	}
	
	// --- variables ---
	if len(self.variables) > 255 { panic(invalidInternalState) }
	numVariables := uint8(len(self.variables))
	font.OffsetToVariables = uint32(len(data))
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
			data = internal.AppendUint16LE(data, endOffset)
		}
		var prevName string
		for _, index := range self.tempSortingBuffer { // VariableNames
			name := self.variables[index].Name
			if name == prevName {
				return nil, errors.New("duplicated variable name '" + name + "'")
			}
			data = append(data, name...)
			prevName = name
		}
	}
	
	// --- mapping ---
	if len(self.mappingModes) > 255 { panic(invalidInternalState) }
	font.OffsetToMappingModes = uint32(len(data))
	data = append(data, uint8(len(self.mappingModes)))
	offset16 = 0
	for i, _ := range self.mappingModes { // MappingModeRoutineEndOffsets
		if len(self.mappingModes[i].Program) > 255 { panic(invalidInternalState) }
		offset16 += uint16(len(self.mappingModes[i].Program))
		data = internal.AppendUint16LE(data, offset16)
	}
	for i, _ := range self.mappingModes { // MappingModeRoutines
		data = append(data, self.mappingModes[i].Program...)
	}

	// fast mapping tables
	if len(self.fastMappingTables) > 255 { panic(invalidInternalState) }
	data = append(data, uint8(len(self.fastMappingTables)))
	totalUsedMem := 0
	for i := 0; i < len(self.fastMappingTables); i++ { // FastMappingTables
		font.OffsetsToFastMapTables = append(font.OffsetsToFastMapTables, uint32(len(data)))
		preLen := len(data)
		data, err = self.fastMappingTables[i].appendTo(data, self.tempGlyphIndexLookup)
		if err != nil { return nil, err }
		totalUsedMem += len(data) - preLen
		if totalUsedMem > internal.MaxFastMappingTablesSize {
			return nil, errors.New("fast mapping tables total size exceeds the limit") // TODO: err or panic?
		}
	}
	
	// main mapping
	if len(self.runeMapping) > 65535 { panic(invalidInternalState) }
	font.OffsetToMainMappings = uint32(len(data))
	data = internal.AppendUint16LE(data, uint16(len(self.runeMapping)))
	self.tempSortingBuffer = self.tempSortingBuffer[ : 0]
	for codePoint, _ := range self.runeMapping { // CodePointList
		if codePoint < 0 { panic(invalidInternalState) }
		self.tempSortingBuffer = append(self.tempSortingBuffer, uint64(uint32(codePoint)))
	}
	slices.Sort(self.tempSortingBuffer) // regular sort
	for _, codePoint := range self.tempSortingBuffer {
		data = internal.AppendUint32LE(data, uint32(codePoint))
	}
	for _, codePoint := range self.tempSortingBuffer { // CodePointModes
		data = append(data, self.runeMapping[int32(uint32(codePoint))].Mode)
	}
	var offset int = 0
	for _, codePoint := range self.tempSortingBuffer { // CodePointMainIndices
		mapping := self.runeMapping[int32(uint32(codePoint))]
		if len(mapping.Glyphs) == 0 { panic(invalidInternalState) }
		if len(mapping.Glyphs) == 1 {
			if mapping.Mode != 255 { panic(invalidInternalState) }
			glyphIndex, found := self.tempGlyphIndexLookup[mapping.Glyphs[0]]
			if !found { panic(invalidInternalState) }
			data = internal.AppendUint16LE(data, glyphIndex)
		} else {
			if mapping.Mode == 255 { panic(invalidInternalState) }
			if len(mapping.Glyphs) > internal.MaxGlyphsPerCodePoint {
				panic(invalidInternalState)
			}
			offset += len(mapping.Glyphs)
			if offset > 65535 {
				return nil, errors.New("too many total glyph indices for custom mode code points")
			}
			data = internal.AppendUint16LE(data, uint16(offset))
		}
	}

	data = internal.AppendUint16LE(data, uint16(offset)) // CodePointModeIndices slice size
	for _, codePoint := range self.tempSortingBuffer { // CodePointModeIndices
		mapping := self.runeMapping[int32(uint32(codePoint))]
		if mapping.Mode == 255 { continue }
		for _, glyphUID := range mapping.Glyphs {
			glyphIndex, found := self.tempGlyphIndexLookup[glyphUID]
			if !found { panic(invalidInternalState) }
			data = internal.AppendUint16LE(data, glyphIndex)
		}
	}

	// --- FSMs ---
	// TODO: to be designed

	// --- kernings ---
	if len(self.horzKerningPairs) > ggfnt.MaxFontDataSize { panic(invalidInternalState) }
	if len(self.vertKerningPairs) > ggfnt.MaxFontDataSize { panic(invalidInternalState) }
	font.OffsetToHorzKernings = uint32(len(data))
	data = internal.AppendUint32LE(data, uint32(len(self.horzKerningPairs)))
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
	data = internal.AppendUint32LE(data, uint32(len(self.vertKerningPairs)))
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
		return nil, errors.New("font data exceeds maximum size")
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

func (self *Font) ClearEditionData() {
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

func (self *Font) ValidateEditionData() error {
	// ...
	panic("unimplemented")
}
