package internal

const MaxFontDataSize = (32 << 20) // check both total file size and after uncompressing without signature
const FormatVersion = 0x0000_00001
const MaxGlyphs = 56789

// some of these could be exposed to the public
const MaxFastMappingTablesSize = (32 << 10)
const MaxFastMappingTableCodePoints = 1000
const MaxFastMappingTableCodePointsStr = "1000"
const MaxGlyphsPerCodePoint = 64
const MinEntropyID = 0.26

// these must definitely not be exposed to the public
const MappingConditionOpEqual = 0b0000_0000
const MappingConditionOpDiff  = 0b0001_0000
const MappingConditionOpLT    = 0b0010_0000
const MappingConditionOpLE    = 0b0011_0000

const MappingConditionArg1Var   = 0b0000_0000
const MappingConditionArg1RNG   = 0b0000_0100
const MappingConditionArg1Const = 0b0000_1000

const MappingConditionArg2Var   = 0b0000_0000
const MappingConditionArg2RNG   = 0b0000_0100
const MappingConditionArg2Const = 0b0000_1000
