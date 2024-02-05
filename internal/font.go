package internal

type Font struct {
	Data []byte // already ungzipped, starting from HEADER (signature is ignored)

	// offsets to specific points at which critical data appears
	// (offsetToHeader is always zero)
	OffsetToMetrics uint32
	OffsetToGlyphNames uint32
	OffsetToGlyphMasks uint32
	OffsetToColorSections uint32
	OffsetToColorSectionNames uint32
	OffsetToVariables uint32
	OffsetToMappingModes uint32
	OffsetsToFastMapTables []uint32
	OffsetToMainMappings uint32 // part of mappings table
	OffsetToHorzKernings uint32
	OffsetToVertKernings uint32
}
