package ggfnt

const MaxFontDataSize = (32 << 20) // check both total file size and after uncompressing without signature
const FormatVersion = 0x0000_00001
const MaxGlyphs = 56789

const maxFastMappingTablesSize = (32 << 10)
const maxFastMappingTableCodePoints = 1000
const maxFastMappingTableCodePointsStr = "1000"
const maxGlyphsPerCodePoint = 64
const minEntropyID = 0.26
