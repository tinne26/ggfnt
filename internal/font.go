package internal

type Font struct {
	Data []byte // already ungzipped, starting from HEADER (signature is ignored)

	// offsets to specific points at which critical data appears
	// (offsetToHeader is always zero)
	OffsetToMetrics uint32
	OffsetToColorSections uint32
	OffsetToColorSectionNames uint32
	OffsetToGlyphNames uint32
	OffsetToGlyphMasks uint32
	OffsetToWords uint32
	OffsetToSettingNames uint32
	OffsetToSettingDefinitions uint32
	OffsetToMappingSwitches uint32
	OffsetToMapping uint32
	OffsetToRewriteConditions uint32
	OffsetToGlyphRewrites uint32
	OffsetToUtf8Rewrites uint32
	OffsetToHorzKernings uint32
	OffsetToVertKernings uint32
}
