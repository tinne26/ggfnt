package ggfnt

import "fmt"
import "io"
import "io/fs"
import "slices"
import "errors"

const traceParsing = false

// Utility method for parsing from a fs.FS, like when using embed.
func ParseFS(filesys fs.FS, filename string) (*Font, error) {
	file, err := filesys.Open(filename)
	if err != nil { return nil, err }
	stat, err := file.Stat()
	if err != nil { return nil, err }
	if stat.Size() > MaxFontDataSize {
		return nil, errors.New("file size exceeds limit")
	}
	
	font, err := Parse(file)
	if err != nil { return font, err }
	return font, file.Close()
}

func Parse(reader io.Reader) (*Font, error) {
	var font Font
	var parser parsingBuffer
	parser.InitBuffers()
	parser.fileType = "ggfnt"

	if traceParsing { fmt.Printf("starting parsing...\n") }

	// read signature first (this is not gzipped, so it's important)
	n, err := reader.Read(parser.tempBuff[0 : 6])
	if err != nil || n != 6 {
		return &font, parser.NewError("failed to read file signature")
		// if debug is required: return font, err
	}
	if !slices.Equal(parser.tempBuff[0 : 6], []byte{'t', 'g', 'g', 'f', 'n', 't'}) {
		return &font, parser.NewError("invalid signature")
	}

	err = parser.InitGzipReader(reader)
	if err != nil { return &font, parser.NewError(err.Error()) }

	// --- header ---
	if traceParsing { fmt.Printf("parsing header...\n") }
	err = parser.AdvanceBytes(28)
	if err != nil { return &font, err }
	for i := 0; i < 3; i++ {
		_, err = parser.ReadShortStr()
		if err != nil { return &font, err }
	}
	_, err = parser.ReadString()
	if err != nil { return &font, err }
	
	font.data = parser.bytes // initial assignation (required before validation)
	err = font.Header().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }

	// --- metrics ---
	if traceParsing { fmt.Printf("parsing metrics... (index = %d)\n", parser.index) }
	font.offsetToMetrics = uint32(parser.index)
	err = parser.AdvanceBytes(14)
	if err != nil { return &font, err }

	font.data = parser.bytes // possible slice reallocs
	err = font.Metrics().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }

	// --- glyphs ---
	if traceParsing { fmt.Printf("parsing glyphs... (index = %d)\n", parser.index) }
	font.offsetToGlyphNames = uint32(parser.index)
	numNamedGlyphs, err := parser.ReadUint16()
	if err != nil { return &font, err }
	
	numGlyphs := font.Metrics().NumGlyphs()
	hasVertLayout := font.Metrics().HasVertLayout()

	// (named glyphs)
	if numNamedGlyphs > 0 {
		if numNamedGlyphs > numGlyphs {
			return &font, parser.NewError("NumNamedGlyphs can't exceed NumGlyphs")
		}

		err = parser.AdvanceBytes(int(numNamedGlyphs)*2) // advance NamedGlyphIDs
		if err != nil { return &font, err }
		err = parser.AdvanceBytes(int(numNamedGlyphs - 1)*4) // advance NamedGlyphEndOffsets except last
		if err != nil { return &font, err }
		glyphNamesLen, err := parser.ReadUint32()
		if err != nil { return &font, err }
		if glyphNamesLen > uint32(numNamedGlyphs)*32 {
			return &font, parser.NewError("GlyphNameEndOffsets declares GlyphNames to end beyond allowed")
		}

		// skip glyph names based on last index
		err = parser.AdvanceBytes(int(glyphNamesLen))
		if err != nil { return &font, err }
	}
	
	// (glyph masks)
	font.offsetToGlyphMasks = uint32(parser.index)
	err = parser.AdvanceBytes(int(numGlyphs - 1)*4)
	if err != nil { return &font, err }
	glyphMasksLen, err := parser.ReadUint32()
	if err != nil { return &font, err }
	placementSize := 1
	if hasVertLayout { placementSize += 3 }
	if glyphMasksLen > (uint32(placementSize) + 255)*uint32(numGlyphs) {
		return &font, parser.NewError("GlyphMaskEndOffsets declares GlyphMasks to end beyond allowed")
	}
	err = parser.AdvanceBytes(int(glyphMasksLen))
	if err != nil { return &font, err }

	// (glyphs validation)
	font.data = parser.bytes // possible slice reallocs
	err = font.Glyphs().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }

	// --- color sections ---
	if traceParsing { fmt.Printf("parsing color sections... (index = %d)\n", parser.index) }
	font.offsetToColorSections = uint32(parser.index)
	numColorSections, err := parser.ReadUint8()
	if err != nil { return &font, err }
	if numColorSections == 0 { parser.NewError("NumColorSections must be at least 1") }

	err = parser.AdvanceBytes(int(numColorSections)) // advance ColorSectionModes
	if err != nil { return &font, err }
	err = parser.AdvanceBytes(int(numColorSections - 1)) // advance ColorSectionStarts
	if err != nil { return &font, err }
	lastRangeStart, err := parser.ReadUint8()
	if err != nil { return &font, err }
	if lastRangeStart == 0 { return &font, parser.NewError("ColorSectionStarts can't reach 0") }
	numColors := (255 - lastRangeStart) + 1
	err = parser.AdvanceBytes(int(numColorSections - 1)*2) // advance ColorSectionEndOffsets
	if err != nil { return &font, err }
	colorSectionsLen, err := parser.ReadUint16()
	if err != nil { return &font, err }
	if colorSectionsLen > uint16(numColors)*4 {
		return &font, parser.NewError("ColorSectionEndOffsets declares ColorSections to end beyond allowed")
	}
	err = parser.AdvanceBytes(int(colorSectionsLen)) // skip color sections
	if err != nil { return &font, err }

	// (color section names)
	font.offsetToColorSectionNames = uint32(parser.index)
	err = parser.AdvanceBytes(int(numColorSections - 1)*2) // advance ColorSectionNameEndOffsets
	if err != nil { return &font, err }
	sectionNamesLen, err := parser.ReadUint16()
	if err != nil { return &font, err }
	if sectionNamesLen > uint16(numColorSections)*32 {
		return &font, parser.NewError("ColorSectionNameEndOffsets declares ColorSectionNames to end beyond allowed")
	}
	err = parser.AdvanceBytes(int(sectionNamesLen)) // advance ColorSectionNames
	if err != nil { return &font, err }
	
	font.data = parser.bytes // possible slice reallocs
	err = font.Color().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }
	

	// --- variables ---
	if traceParsing { fmt.Printf("parsing variables... (index = %d)\n", parser.index) }
	font.offsetToVariables = uint32(parser.index)
	numVars, err := parser.ReadUint8()
	if err != nil { return &font, err }
	err = parser.AdvanceBytes(int(numVars)*3) // advance variable defs
	if err != nil { return &font, err }
	
	numNamedVars, err := parser.ReadUint8()
	if err != nil { return &font, err }
	if numNamedVars > 0 {
		if numNamedVars > numVars {
			return &font, parser.NewError("found NumNamedVars > NumVars")
		}

		// advance NamedVarKeys and (NamedVarEndOffsets - 1)
		err = parser.AdvanceBytes(int(numNamedVars)*1 + int(numNamedVars - 1)*2)
		if err != nil { return &font, err }
		variableNamesLen, err := parser.ReadUint16()
		if err != nil { return &font, err }
		if variableNamesLen > uint16(numNamedVars)*32 {
			return &font, parser.NewError("VarNameEndOffsets declares VariableNames to end beyond allowed")
		}
		err = parser.AdvanceBytes(int(variableNamesLen))
		if err != nil { return &font, err }
	}

	font.data = parser.bytes // possible slice reallocs
	err = font.Vars().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }

	// --- mappings ---
	if traceParsing { fmt.Printf("parsing mappings... (index = %d)\n", parser.index) }
	font.offsetToMappingModes = uint32(parser.index)

	numMappingModes, err := parser.ReadUint8()
	if err != nil { return &font, err }
	if numMappingModes > 0 {
		if numMappingModes == 255 {
			return &font, parser.NewError("NumMappingModes can't be 255")
		}
		err = parser.AdvanceBytes(int(numMappingModes - 1)*2) // advance MappingModeRoutineEndOffsets - 1
		if err != nil { return &font, err }
		mappingModeRoutinesLen, err := parser.ReadUint16()
		if err != nil { return &font, err }
		err = parser.AdvanceBytes(int(mappingModeRoutinesLen))
		if err != nil { return &font, err }	
	}
	
	numFastMappingTables, err := parser.ReadUint8()
	if err != nil { return &font, err }

	fastMappingTableMem := 0 // used memory, we have to check against the limit
	for i := uint8(0); i < numFastMappingTables; i++ {
		font.offsetsToFastMapTables = append(font.offsetsToFastMapTables, uint32(parser.index))
		err = parser.AdvanceBytes(3) // advance TableCondition
		if err != nil { return &font, err }
		startCodePoint, err := parser.ReadInt32()
		if err != nil { return &font, err }
		endCodePoint, err := parser.ReadInt32()
		if err != nil { return &font, err }
		if endCodePoint <= startCodePoint {
			return &font, parser.NewError("fast mapping table declares a negative range")
		}
		tableLen := endCodePoint - startCodePoint
		if tableLen > maxFastMappingTableCodePoints {
			return &font, parser.NewError(
				"fast mapping table length can't exceed " + maxFastMappingTableCodePointsStr + " code points",
			)
		}
		fastMappingTableMem += (3 + 4 + 4 + int(tableLen) + int(tableLen)*2) // still missing CodePointModeIndices data
		if fastMappingTableMem > maxFastMappingTablesSize {
			return &font, parser.NewError("fast mapping tables exceed max memory usage limits")
		}
		
		// advance CodePointModes and CodePointMainIndices
		err = parser.AdvanceBytes(int(tableLen)*3)
		if err != nil { return &font, err }

		codePointModeIndicesLen, err := parser.ReadUint16()
		if err != nil { return &font, err }
		fastMappingTableMem += (2 + int(codePointModeIndicesLen))
		if fastMappingTableMem > maxFastMappingTablesSize { // second check for max fast tables memory usage
			return &font, parser.NewError("fast mapping tables exceed max memory usage limits")
		}

		err = parser.AdvanceBytes(int(codePointModeIndicesLen))
		if err != nil { return &font, err }
	}

	// main mapping table
	font.offsetToMainMappings = uint32(parser.index)
	numMappingEntries, err := parser.ReadUint16()
	if err != nil { return &font, err }

	// advance CodePointList, CodePointModes and CodePointMainIndices
	err = parser.AdvanceBytes(int(numMappingEntries)*7)
	if err != nil { return &font, err }
	codePointModeIndicesLen, err := parser.ReadUint16()
	if err != nil { return &font, err }
	err = parser.AdvanceBytes(int(codePointModeIndicesLen))
	if err != nil { return &font, err }

	font.data = parser.bytes // possible slice reallocs
	err = font.Mapping().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }

	// --- FSMs ---
	// ... (not designed yet)

	// --- kerning ---
	if traceParsing { fmt.Printf("parsing kernings... (index = %d)\n", parser.index) }
	font.offsetToHorzKernings = uint32(parser.index)
	maxKerningPairs := uint32(numGlyphs)*uint32(numGlyphs)

	numHorzKerningPairs, err := parser.ReadUint32()
	if numHorzKerningPairs > 0 {
		if numHorzKerningPairs > maxKerningPairs {
			return &font, parser.NewError("NumHorzKerningPairs can't exceed NumGlyphs^2")
		}

		// advance HorzKerningPairs and HorzKerningValues
		err = parser.AdvanceBytes(int(numHorzKerningPairs)*5)
		if err != nil { return &font, err }
	}

	font.offsetToVertKernings = uint32(parser.index)
	numVertKerningPairs, err := parser.ReadUint32()
	if numVertKerningPairs > 0 {
		if numVertKerningPairs > maxKerningPairs {
			return &font, parser.NewError("NumVertKerningPairs can't exceed NumGlyphs^2")
		}

		// advance VertKerningPairs and VertKerningValues
		err = parser.AdvanceBytes(int(numVertKerningPairs)*5)
		if err != nil { return &font, err }
	}

	font.data = parser.bytes // possible slice reallocs
	err = font.Kerning().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }

	// --- EOF ---
	if traceParsing { fmt.Printf("testing EOF... (index = %d)\n", parser.index) }
	// ensure we reach EOF exactly at the right time
	err = parser.EnsureEOF()
	if err != nil { return &font, parser.NewError(err.Error()) }

	// everything went well
	if traceParsing { fmt.Printf("parsing correct!\n") }
	return &font, nil
}

