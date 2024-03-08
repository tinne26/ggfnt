package ggfnt

import "fmt"
import "io"
import "io/fs"
import "slices"
import "errors"

import "github.com/tinne26/ggfnt/internal"

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
	var parser internal.ParsingBuffer
	parser.InitBuffers()
	parser.FileType = "ggfnt"

	if traceParsing { fmt.Printf("starting parsing...\n") }

	// read signature first (this is not gzipped, so it's important)
	n, err := reader.Read(parser.TempBuff[0 : 6])
	if err != nil || n != 6 {
		return &font, parser.NewError("failed to read file signature")
		// if debug is required: return font, err
	}
	if !slices.Equal(parser.TempBuff[0 : 6], []byte{'t', 'g', 'g', 'f', 'n', 't'}) {
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
	
	font.Data = parser.Bytes // initial assignation (required before validation)
	err = font.Header().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }

	// --- metrics ---
	if traceParsing { fmt.Printf("parsing metrics... (index = %d)\n", parser.Index) }
	font.OffsetToMetrics = uint32(parser.Index)
	err = parser.AdvanceBytes(15)
	if err != nil { return &font, err }

	font.Data = parser.Bytes // possible slice reallocs
	err = font.Metrics().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }

	// --- color sections ---
	if traceParsing { fmt.Printf("parsing color sections... (index = %d)\n", parser.Index) }
	font.OffsetToColorSections = uint32(parser.Index)
	numDyes, err := parser.ReadUint8()
	if err != nil { return &font, err }
	numPalettes, err := parser.ReadUint8()
	if err != nil { return &font, err }
	if numDyes == 0 && numPalettes == 0 {
		return &font, parser.NewError("NumDyes + NumPalettes must be at least 1")
	}
	if int(numDyes) + int(numPalettes) > 255 {
		return &font, parser.NewError("NumDyes + NumPalettes can't exceed 255")
	}
	numColorSections := numDyes + numPalettes // we know this can't overflow
	
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
	if colorSectionsLen < uint16(numPalettes)*4 + uint16(numDyes) {
		return &font, parser.NewError("ColorSectionEndOffsets declares ColorSections to end before allowed")
	}
	err = parser.AdvanceBytes(int(colorSectionsLen)) // skip color sections
	if err != nil { return &font, err }

	// (color section names)
	font.OffsetToColorSectionNames = uint32(parser.Index)
	err = parser.AdvanceBytes(int(numColorSections - 1)*2) // advance ColorSectionNameEndOffsets
	if err != nil { return &font, err }
	sectionNamesLen, err := parser.ReadUint16()
	if err != nil { return &font, err }
	if sectionNamesLen > uint16(numColorSections)*32 {
		return &font, parser.NewError("ColorSectionNameEndOffsets declares ColorSectionNames to end beyond allowed")
	}
	if sectionNamesLen < uint16(numColorSections) {
		return &font, parser.NewError("ColorSectionNameEndOffsets declares ColorSectionNames to end before allowed")
	}
	err = parser.AdvanceBytes(int(sectionNamesLen)) // advance ColorSectionNames
	if err != nil { return &font, err }
	
	font.Data = parser.Bytes // possible slice reallocs
	err = font.Color().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }	

	// --- glyphs ---
	if traceParsing { fmt.Printf("parsing glyphs... (index = %d)\n", parser.Index) }
	font.OffsetToGlyphNames = uint32(parser.Index)
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
		err = parser.AdvanceBytes(int(numNamedGlyphs - 1)*3) // advance NamedGlyphEndOffsets except last
		if err != nil { return &font, err }
		glyphNamesLen, err := parser.ReadUint24()
		if err != nil { return &font, err }
		if glyphNamesLen > uint32(numNamedGlyphs)*32 {
			return &font, parser.NewError("GlyphNameEndOffsets declares GlyphNames to end beyond allowed")
		}

		// skip glyph names based on last index
		err = parser.AdvanceBytes(int(glyphNamesLen))
		if err != nil { return &font, err }
	}
	
	// (glyph masks)
	font.OffsetToGlyphMasks = uint32(parser.Index)
	err = parser.AdvanceBytes(int(numGlyphs - 1)*3) // advance GlyphMaskEndOffsets
	if err != nil { return &font, err }
	glyphMasksLen, err := parser.ReadUint24()
	if err != nil { return &font, err }
	placementSize := 1
	if hasVertLayout { placementSize += 3 }
	if glyphMasksLen < (uint32(placementSize)*uint32(numGlyphs)) {
		return &font, parser.NewError("GlyphMaskEndOffsets declares GlyphMasks to end before allowed")
	}
	if glyphMasksLen > (uint32(placementSize) + 255)*uint32(numGlyphs) {
		return &font, parser.NewError("GlyphMaskEndOffsets declares GlyphMasks to end beyond allowed")
	}
	err = parser.AdvanceBytes(int(glyphMasksLen))
	if err != nil { return &font, err }

	// (glyphs validation)
	font.Data = parser.Bytes // possible slice reallocs
	err = font.Glyphs().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }

	// --- settings ---
	if traceParsing { fmt.Printf("parsing settings... (index = %d)\n", parser.Index) }
	font.OffsetToWords = uint32(parser.Index)
	numWords, err := parser.ReadUint8()
	if err != nil { return &font, err }
	if numWords > 0 {
		// advance WordEndOffsets - 1
		err = parser.AdvanceBytes(int(numWords - 1)*2)
		if err != nil { return &font, err }
		wordsLen, err := parser.ReadUint16()
		if err != nil { return &font, err }

		if wordsLen < uint16(numWords) {
			return &font, parser.NewError("WordEndOffsets declares Words to end before allowed")
		}
		if int(wordsLen) > int(numWords)*32 {
			return &font, parser.NewError("WordEndOffsets declares Words to end beyond allowed")
		}
	}
	
	font.OffsetToSettingNames = uint32(parser.Index)
	font.OffsetToSettingDefinitions = uint32(parser.Index) + 1
	numSettings, err := parser.ReadUint8()
	if err != nil { return &font, err }	
	if numSettings > 0 {
		// advance SettingNameEndOffsets - 1
		err = parser.AdvanceBytes(int(numSettings - 1)*2)
		if err != nil { return &font, err }
		settingNamesLen, err := parser.ReadUint16()
		if err != nil { return &font, err }
		if settingNamesLen < uint16(numSettings) {
			return &font, parser.NewError("SettingNameEndOffsets declares SettingNames to end before allowed")
		}
		if settingNamesLen > uint16(numSettings)*32 {
			return &font, parser.NewError("SettingNameEndOffsets declares SettingNames to end beyond allowed")
		}
		err = parser.AdvanceBytes(int(settingNamesLen))
		if err != nil { return &font, err }

		// advance SettingEndOffsets
		font.OffsetToSettingDefinitions = uint32(parser.Index)
		err = parser.AdvanceBytes(int(numSettings - 1)*2)
		if err != nil { return &font, err }
		settingsLen, err := parser.ReadUint16()
		if err != nil { return &font, err }
		if settingsLen < uint16(numSettings) {
			return &font, parser.NewError("SettingEndOffsets declares Settings to end before allowed")
		}
		err = parser.AdvanceBytes(int(settingsLen))
		if err != nil { return &font, err }
	}

	font.Data = parser.Bytes // possible slice reallocs
	err = font.Settings().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }

	// --- mappings ---
	if traceParsing { fmt.Printf("parsing mappings... (index = %d)\n", parser.Index) }
	
	// mapping switches
	font.OffsetToMappingSwitches = uint32(parser.Index)
	numMappingSwitches, err := parser.ReadUint8()
	if err != nil { return &font, err }
	if numMappingSwitches > 0 {
		// advance MappingSwitchEndOffsets - 1
		err = parser.AdvanceBytes(int(numMappingSwitches - 1)*2)
		if err != nil { return &font, err }

		mappingSwitchesLen, err := parser.ReadUint16()
		if err != nil { return &font, err }
		if mappingSwitchesLen < uint16(numMappingSwitches) {
			return &font, parser.NewError("MappingSwitchEndOffsets declares MappingSwitches to end before allowed")
		} else if uint32(mappingSwitchesLen) > uint32(numMappingSwitches)*uint32(numSettings) {
			return &font, parser.NewError("MappingSwitchEndOffsets declares MappingSwitches to end beyond allowed")
		}

		// advance MappingSwitches
		err = parser.AdvanceBytes(int(mappingSwitchesLen))
		if err != nil { return &font, err }
	}

	// mapping table
	font.OffsetToMapping = uint32(parser.Index)
	numMappingEntries, err := parser.ReadUint16()
	if err != nil { return &font, err }
	if numMappingEntries > 0 {
		// advance CodePointsIndex
		err = parser.AdvanceBytes(int(numMappingEntries)*4)
		if err != nil { return &font, err }

		// advance MappingEndOffsets - 1
		err = parser.AdvanceBytes(int(numMappingEntries - 1)*3)
		if err != nil { return &font, err }
		mappingsLen, err := parser.ReadUint24()
		if err != nil { return &font, err }
		if mappingsLen < uint32(numMappingEntries)*3 {
			return &font, parser.NewError("MappingEndOffsets declares Mappings to end before allowed")
		}
		// note: no upper bound check, we gotta trust max font size here
		err = parser.AdvanceBytes(int(mappingsLen))
		if err != nil { return &font, err }
	}

	font.Data = parser.Bytes // possible slice reallocs
	err = font.Mapping().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }

	// --- rewrite rules ---
	if traceParsing { fmt.Printf("parsing rewrite rules... (index = %d)\n", parser.Index) }
	font.OffsetToRewriteConditions = uint32(parser.Index)
	numConditions, err := parser.ReadUint8()
	if err != nil { return &font, err }
	if numConditions > 0 {
		// advance ConditionEndOffsets - 1
		err := parser.AdvanceBytes(int(numConditions - 1)*2)
		if err != nil { return &font, err }
		conditionsLen, err := parser.ReadUint16()
		if err != nil { return &font, err }
		if int(conditionsLen) < int(numConditions)*2 {
			return &font, parser.NewError("MappingEndOffsets declares Mappings to end before allowed")
		}

		// advance Conditions
		err = parser.AdvanceBytes(int(conditionsLen))
		if err != nil { return &font, err }
	}

	font.OffsetToUtf8Rewrites = uint32(parser.Index)
	numUtf8Rules, err := parser.ReadUint16()
	if err != nil { return &font, err }
	if numUtf8Rules > 0 {
		// advance UTF8RuleEndOffsets - 1
		err := parser.AdvanceBytes(int(numUtf8Rules - 1)*3)
		if err != nil { return &font, err }
		utf8RulesLen, err := parser.ReadUint24()
		if int(utf8RulesLen) < int(numUtf8Rules)*8 {
			return &font, parser.NewError("UTF8RuleEndOffsets declares UTF8Rules to end before allowed")
		}
		if int(utf8RulesLen) > int(numUtf8Rules)*(6 + 255*4) {
			return &font, parser.NewError("UTF8RuleEndOffsets declares UTF8Rules to end beyond allowed")
		}

		// advance UTF8Rules
		err = parser.AdvanceBytes(int(utf8RulesLen))
		if err != nil { return &font, err }
	}

	font.OffsetToGlyphRewrites = uint32(parser.Index)
	numGlyphRules, err := parser.ReadUint16()
	if err != nil { return &font, err }
	if numGlyphRules > 0 {
		// advance GlyphRuleEndOffsets - 1
		err := parser.AdvanceBytes(int(numGlyphRules - 1)*3)
		if err != nil { return &font, err }
		glyphRulesLen, err := parser.ReadUint24()
		if int(glyphRulesLen) < int(numGlyphRules)*8 {
			return &font, parser.NewError("GlyphRuleEndOffsets declares GlyphRules to end before allowed")
		}
		if int(glyphRulesLen) > int(numGlyphRules)*514 {
			return &font, parser.NewError("GlyphRuleEndOffsets declares GlyphRules to end beyond allowed")
		}

		// advance GlyphRules
		err = parser.AdvanceBytes(int(glyphRulesLen))
		if err != nil { return &font, err }
	}

	font.Data = parser.Bytes // possible slice reallocs
	err = font.Rewrites().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }

	// --- kerning ---
	if traceParsing { fmt.Printf("parsing kernings... (index = %d)\n", parser.Index) }
	font.OffsetToHorzKernings = uint32(parser.Index)
	maxKerningPairs := uint32(numGlyphs)*uint32(numGlyphs)

	numHorzKerningPairs, err := parser.ReadUint24()
	if numHorzKerningPairs > 0 {
		if numHorzKerningPairs > maxKerningPairs {
			return &font, parser.NewError("NumHorzKerningPairs can't exceed NumGlyphs^2")
		}

		// advance HorzKerningPairs and HorzKerningValues
		err = parser.AdvanceBytes(int(numHorzKerningPairs)*5)
		if err != nil { return &font, err }
	}

	font.OffsetToVertKernings = uint32(parser.Index)
	numVertKerningPairs, err := parser.ReadUint24()
	if numVertKerningPairs > 0 {
		if numVertKerningPairs > maxKerningPairs {
			return &font, parser.NewError("NumVertKerningPairs can't exceed NumGlyphs^2")
		}

		// advance VertKerningPairs and VertKerningValues
		err = parser.AdvanceBytes(int(numVertKerningPairs)*5)
		if err != nil { return &font, err }
	}

	font.Data = parser.Bytes // possible slice reallocs
	err = font.Kerning().Validate(FmtDefault)
	if err != nil { return &font, parser.NewError(err.Error()) }

	// --- EOF ---
	if traceParsing { fmt.Printf("testing EOF... (index = %d)\n", parser.Index) }
	// ensure we reach EOF exactly at the right time
	err = parser.EnsureEOF()
	if err != nil { return &font, parser.NewError(err.Error()) }

	// everything went well
	if traceParsing { fmt.Printf("parsing correct!\n") }
	return &font, nil
}

