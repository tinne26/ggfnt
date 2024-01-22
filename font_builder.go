package ggfnt

import "io"
import "slices"
import "image"
import "time"
import "errors"

// A [Font] builder that allows modifying and exporting ggfnt fonts.
// It can also store and edit glyph category names, kerning classes
// and a few other elements not present in regular .ggfnt files. See
// .ggwkfnt in the spec document for more details.
//
// This object should never replace a [Font] outside the edition context.
type FontBuilder struct {
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
	glyphNames map[uint16]nameIndexEntry
	glyphBounds []GlyphBounds
	glyphMasks []*image.Alpha // indexed by uint16 glyph ids

	// coloring
	dyes []string
	palettes []Palette
	coloringSectionStarts []uint8
	coloringSectionsEnd uint8 // inclusive
	coloringSectionNames []string
	coloringSectionOptions [][]uint8

	// variables
	variables []variableEntry
	namedVariables map[uint8]nameIndexEntry

	// ---- edition-only data ----
	categories []editionCategory
	kerningClasses []editionKerningClass
	horzKerningPairs []editionKerningPair
	vertKerningPairs []editionKerningPair
	mappingModes []string
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
	builder.glyphNames = make(map[uint16]nameIndexEntry)
	builder.glyphBounds = make([]GlyphBounds, 32)
	builder.glyphMasks = make([]*image.Alpha, 32)

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

	// (signature is not part of the data)
	// data = append(data, 'w', 'k', 'g', 'f', 'n', 't')

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
	data, err = appendShortString(data, self.fontName)
	if err != nil { return nil, err }
	data, err = appendShortString(data, self.fontFamily)
	if err != nil { return nil, err }
	data, err = appendShortString(data, self.fontAuthor)
	if err != nil { return nil, err }
	data, err = appendString(data, self.fontAbout)
	if err != nil { return nil, err }

	// --- metrics ---
	numGlyphs := len(self.glyphMasks)
	if numGlyphs == 0 { return nil, errors.New("can't build font with no glyphs") }
	if numGlyphs > MaxGlyphs { return nil, errors.New("font has too many glyphs") }
	data = appendUint16LE(data, uint16(numGlyphs))
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
	numNamedGlyphs := len(self.glyphNames)
	if numNamedGlyphs > numGlyphs {
		return nil, errors.New("font can't have more named glyphs than glyphs")
	}
	data = appendUint16LE(data, uint16(numNamedGlyphs))
	if numNamedGlyphs > 0 {
		buffer := getNameIndexEntryBuffer()
		buffer.AppendAllMapUint16(self.glyphNames)
		buffer.Sort()
		for i, _ := range buffer.buffer { // NamedGlyphIDs
			if buffer.buffer[i].Index >= uint16(numGlyphs) {
				releaseNameIndexEntryBuffer(buffer)
				return nil, errors.New("named glyph ID beyond declared NumGlyphs")
			}
			data = appendUint16LE(data, buffer.buffer[i].Index)
		}
		endOffset := uint32(0)
		for i, _ := range buffer.buffer { // GlyphNameEndOffsets
			endOffset += uint32(len(buffer.buffer[i].Name))
			data = appendUint32LE(data, endOffset)
		}
		for i, _ := range buffer.buffer { // GlyphNames
			data = append(data, buffer.buffer[i].Name...)
		}
		releaseNameIndexEntryBuffer(buffer)
	}
	
	// some safety checks for glyphs
	if len(self.glyphMasks) != len(self.glyphBounds) {
		return nil, errors.New("number of glyph masks and glyph bounds doesn't match")
	}

	// reserve space for glyph offsets
	glyphMaskEndOffsetsIndex := len(data)
	var offset32 uint32
	data = growSliceByN(data, numGlyphs*4)
	baseGlyphMasksIndex := len(data)
	for i := 0; i < len(self.glyphMasks); i++ {
		// safety checks
		w, h := self.glyphMasks[i].Rect.Dx(), self.glyphMasks[i].Rect.Dy()
		if w != int(self.glyphBounds[i].MaskWidth) || h != int(self.glyphBounds[i].MaskHeight) {
			return nil, errors.New("glyph mask does not match explicit bounds definition")
		}
		
		// append mask bounds
		if self.hasVertLayout {
			data = self.glyphBounds[i].appendWithVertLayout(data)
		} else {
			data = self.glyphBounds[i].appendWithoutVertLayout(data)
		}

		// append mask data (expensive to process the masks!)
		var err error
		data, err = AppendMaskRasterOps(data, self.glyphMasks[i])
		if err != nil { return nil, err }

		// write offset back on the relevant index
		offset32 += uint32(len(data) - baseGlyphMasksIndex)
		encodeUint32(data[glyphMaskEndOffsetsIndex : glyphMaskEndOffsetsIndex + 4], offset32)
		glyphMaskEndOffsetsIndex += 4
	}

	// --- coloring ---
	// dyes
	if len(self.dyes) > 255 { panic("invalid internal state") }
	data = append(data, uint8(len(self.dyes)))
	var offset16 uint16
	for _, name := range self.dyes { // DyeNameEndOffsets
		if len(name) == 0 || len(name) > 32 { panic("invalid internal state") }
		offset16 += uint16(len(name))
		data = appendUint16LE(data, offset16)
	}
	for _, name := range self.dyes { // DyeNames
		data = append(data, name...)
	}

	// palettes
	if len(self.palettes) > 255 { panic("invalid internal state") }
	data = append(data, uint8(len(self.palettes)))
	offset16 = 0
	for i := 0; i < len(self.palettes); i++ {
		offset16 += uint16(len(self.palettes[i].colors))
		data = appendUint16LE(data, offset16)
	}
	for i := 0; i < len(self.palettes); i++ {
		data = append(data, self.palettes[i].colors...)
	}
	offset16 = 0
	var prevName string
	for i := 0; i < len(self.palettes); i++ { // PaletteNameEndOffsets
		if i > 0 && prevName >= self.palettes[i].name {
			return nil, errors.New("palette names not in order")
		}
		offset16 += uint16(len(self.palettes[i].name))
		data = appendUint16LE(data, offset16)
		prevName = self.palettes[i].name
	}
	for i := 0; i < len(self.palettes); i++ { // PaletteNames
		data = append(data, self.palettes[i].name...)
	}

	// coloring section
	if len(self.coloringSectionStarts) > 255 { panic("invalid internal state") }
	data = append(data, uint8(len(self.coloringSectionStarts)))
	var prevSectionStart uint8
	for i := 0; i < len(self.coloringSectionStarts); i++ {
		if self.coloringSectionStarts[i] < prevSectionStart && i > 0 {
			return nil, errors.New("coloring sections not in order")
		}
		data = append(data, self.coloringSectionStarts[i])
		prevSectionStart = self.coloringSectionStarts[i]
	}
	if self.coloringSectionsEnd < prevSectionStart {
		return nil, errors.New("coloring sections not in order")
	}
	data = append(data, self.coloringSectionsEnd)

	if len(self.coloringSectionNames) != len(self.coloringSectionStarts) {
		panic("invalid internal state")
	}
	offset16 = 0
	for i := 0; i < len(self.coloringSectionNames); i++ { // SectionNameEndOffsets
		if len(self.coloringSectionNames[i]) > 255 { panic("invalid internal state") }
		if i > 0 && prevName >= self.coloringSectionNames[i] {
			return nil, errors.New("invalid coloring section names order")
		}
		offset16 += uint16(len(self.coloringSectionNames[i]))
		data = appendUint16LE(data, offset16)
		prevName = self.coloringSectionNames[i]
	}
	for i := 0; i < len(self.coloringSectionNames); i++ { // SectionNames
		data = append(data, self.coloringSectionNames[i]...)
	}
	
	// section options
	offset16 = 0
	for i := 0; i < len(self.coloringSectionOptions); i++ { // SectionOptionEndOffsets
		offset16 += uint16(len(self.coloringSectionOptions[i]))
		data = appendUint16LE(data, offset16)
	}
	for i := 0; i < len(self.coloringSectionOptions); i++ { // SectionOptions
		data = append(data, self.coloringSectionOptions[i]...)
	}
	
	// --- variables ---
	if len(self.variables) > 255 { panic("invalid internal state") }
	numVariables := uint8(len(self.variables))
	data = append(data, numVariables)
	for i := 0; i < len(self.variables); i++ {
		varEntry := &self.variables[i]
		data = append(data, varEntry.DefaultValue, varEntry.MinValue, varEntry.MaxValue)
	}

	if len(self.namedVariables) > 255 { panic("invalid internal state") }
	data = append(data, uint8(len(self.namedVariables)))
	if len(self.namedVariables) > 0 {
		buffer := getNameIndexEntryBuffer()
		buffer.AppendAllMapUint8(self.namedVariables)
		buffer.Sort()
		for i, _ := range buffer.buffer { // NamedVarKeys
			if buffer.buffer[i].Index >= uint16(numVariables) {
				releaseNameIndexEntryBuffer(buffer)
				return nil, errors.New("named variable key exceeds declared NumVariables")
			}
			data = append(data, uint8(buffer.buffer[i].Index))
		}
		endOffset := uint16(0)
		for i, _ := range buffer.buffer { // VarNameEndOffsets
			nameLen := len(buffer.buffer[i].Name)
			if nameLen > 32 {
				releaseNameIndexEntryBuffer(buffer)
				return nil, errors.New("named variable name can't exceed 32 characters")
			}
			endOffset += uint16(nameLen)
			data = appendUint16LE(data, endOffset)
		}
		for i, _ := range buffer.buffer { // VariableNames
			data = append(data, buffer.buffer[i].Name...)
		}
		releaseNameIndexEntryBuffer(buffer)
	}
	
	// ...

	panic("unimplemented")
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
	var prevKerningPair uint32 = 0
	for i := uint32(0); i < numHorzKerningPairsWithClasses; i++ {
		kerningPair, err := parser.ReadUint32()
		if err != nil { return err }
		if kerningPair <= prevKerningPair {
			return parser.NewError("HorzKerningPairsWithClasses unordered")
		}
		self.horzKerningPairs = append(self.horzKerningPairs, editionKerningPair{ Pair: kerningPair })
	}
	for i := uint32(0); i < numHorzKerningPairsWithClasses; i++ {
		kerningClass, err := parser.ReadUint16()
		if err != nil { return err }
		(&self.horzKerningPairs[i]).Class = kerningClass
	}
	
	numVertKerningPairsWithClasses, err := parser.ReadUint32()
	if err != nil { return err }
	prevKerningPair = 0
	for i := uint32(0); i < numVertKerningPairsWithClasses; i++ {
		kerningPair, err := parser.ReadUint32()
		if err != nil { return err }
		if kerningPair <= prevKerningPair {
			return parser.NewError("VertKerningPairsWithClasses unordered")
		}
		self.vertKerningPairs = append(self.vertKerningPairs, editionKerningPair{ Pair: kerningPair })
	}
	for i := uint32(0); i < numVertKerningPairsWithClasses; i++ {
		kerningClass, err := parser.ReadUint16()
		if err != nil { return err }
		(&self.vertKerningPairs[i]).Class = kerningClass
	}
	
	numNamedMappingModes, err := parser.ReadUint8()
	if err != nil { return err }
	if numNamedMappingModes == 255 {
		return parser.NewError("MappingModeNames must have at most 254 elements")
	}
	for i := uint8(0); i < numNamedMappingModes; i++ {
		modeName, err := parser.ReadShortStr()
		if err != nil { return err }
		err = parser.ValidateBasicSpacedName(modeName)
		if err != nil { return err }
		self.mappingModes = append(self.mappingModes, modeName)
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
	self.horzKerningPairs = self.horzKerningPairs[ : 0]
	self.vertKerningPairs = self.vertKerningPairs[ : 0]
	self.mappingModes = self.mappingModes[ : 0]
}

func (self *FontBuilder) ValidateEditionData() error {
	// ...
	panic("unimplemented")
}
