package ggfnt

import "io"
import "io/fs"
import "slices"
import "errors"

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
	font.offsetToMetrics = uint32(parser.index)
	err = parser.AdvanceBytes(13)
	if err != nil { return &font, err }

	font.data = parser.bytes // possible slice reallocs
	err = font.Metrics().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }

	// --- glyphs ---
	font.offsetToGlyphNames = uint32(parser.index)
	numNamedGlyphs, err := parser.ReadUint16()
	if err != nil { return &font, err }
	
	numGlyphs := font.Header().NumGlyphs()
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
	boundsSize := 4
	if hasVertLayout { boundsSize += 3 }
	if glyphMasksLen > (uint32(boundsSize) + 255)*uint32(numGlyphs) {
		return &font, parser.NewError("GlyphMaskEndOffsets declares GlyphMasks to end beyond allowed")
	}
	err = parser.AdvanceBytes(int(glyphMasksLen))
	if err != nil { return &font, err }

	// (glyphs validation)
	font.data = parser.bytes // possible slice reallocs
	err = font.Glyphs().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }

	// --- coloring ---
	font.offsetToColoring = uint32(parser.index)
	numDyes, err := parser.ReadUint8()
	if err != nil { return &font, err }
	if numDyes > 0 {
		err = parser.AdvanceBytes(int(numDyes - 1)*2) // advance DyeNameEndOffsets
		if err != nil { return &font, err }
		dyeNamesLen, err := parser.ReadUint16()
		if err != nil { return &font, err }
		if dyeNamesLen > uint16(numDyes)*32 {
			return &font, parser.NewError("DyeNameEndOffsets declares DyeNames to end beyond allowed")
		}

		err = parser.AdvanceBytes(int(dyeNamesLen))
		if err != nil { return &font, err }
	}

	// predefined palettes
	font.offsetToColoringPalettes = uint32(parser.index)
	numPalettes, err := parser.ReadUint8()
	if err != nil { return &font, err }
	if numPalettes > 0 {
		err = parser.AdvanceBytes(int(numPalettes - 1)*2) // advance PaletteEndOffsets
		if err != nil { return &font, err }
		palettesLen, err := parser.ReadUint16()
		if err != nil { return &font, err }
		if uint32(palettesLen) > uint32(numPalettes)*255*4 {
			return &font, parser.NewError("PaletteEndOffsets declares Palettes to end beyond allowed")
		}
		err = parser.AdvanceBytes(int(palettesLen)) // advance Palettes
		if err != nil { return &font, err }
		
		font.offsetToColoringPaletteNames = uint32(parser.index)
		err = parser.AdvanceBytes(int(numPalettes - 1)*2) // advance PaletteNameEndOffsets
		if err != nil { return &font, err }
		paletteNamesLen, err := parser.ReadUint16()
		if err != nil { return &font, err }
		if paletteNamesLen > uint16(numPalettes)*32 {
			return &font, parser.NewError("PaletteNameEndOffsets declares PaletteNames to end beyond allowed")
		}
		err = parser.AdvanceBytes(int(paletteNamesLen)) // advance PaletteNames
		if err != nil { return &font, err }
	} else { // numPalettes == 0
		font.offsetToColoringPaletteNames = uint32(parser.index) // this is actually meaningless
	}

	// coloring sections
	font.offsetToColoringSections = uint32(parser.index)
	numColoringSections, err := parser.ReadUint8()
	if err != nil { return &font, err }
	if numColoringSections > 0 {
		// advance SectionStarts, SectionsEnd and SectionsNameEndOffsets - 1
		advance := int(numColoringSections) + 1 + int(numColoringSections - 1)*2
		err = parser.AdvanceBytes(advance)
		if err != nil { return &font, err }
		sectionNamesLen, err := parser.ReadUint16()
		if err != nil { return &font, err }
		if sectionNamesLen > uint16(numColoringSections)*32 {
			return &font, parser.NewError("SectionNameEndOffsets declares SectionNames to end beyond allowed")
		}
		err = parser.AdvanceBytes(int(sectionNamesLen))
		if err != nil { return &font, err }
		
		font.offsetToColoringSectionOptions = uint32(parser.index)
		err = parser.AdvanceBytes(int(numColoringSections - 1)*2) // advance SectionOptionEndOffsets - 1
		if err != nil { return &font, err }
		sectionOptionsLen, err := parser.ReadUint16()
		if err != nil { return &font, err }
		if sectionOptionsLen > uint16(numColoringSections)*16 {
			return &font, parser.NewError("SectionOptionEndOffsets declares SectionOptions to end beyond allowed")
		}
		err = parser.AdvanceBytes(int(sectionOptionsLen)) // advance SectionOptions
	} else { // numColoringSections == 0
		font.offsetToColoringSectionOptions = uint32(parser.index) // this is actually meaningless
	}
	
	font.data = parser.bytes // possible slice reallocs
	err = font.Coloring().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }
	

	// --- variables ---
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

	// advance CodePointList, CodePointMods and CodePointMainIndices
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
	// ensure we reach EOF exactly at the right time
	err = parser.EnsureEOF()
	if err != nil { return &font, parser.NewError(err.Error()) }

	// everything went well
	return &font, nil
}

