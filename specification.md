# `tgbf` font format specification

This is a bitmap font format created for indie game development and pixel art games.

Font files use the `.tgbf` file extension.

File names should preferently follow the "name-type-size-version.tgbf" convention, with `type` and `version` being the most optional parts. For example: `bouncy-mono-8d4-v2p14.tgbf`. `8d4` indicates that the font's ascent is 8 pixels and the descent is 4 pixels. The extra ascent and extra descent, as described later in the metrics section, are not considered for this indication. None of this is mandatory. 

Maximum font file size is limited to 64MiB per spec, which is much more than enough for practical purposes and eliminates a whole class of problems during parsing, storage and others.

The binary format is designed to be reasonably compact, have safe limits and keep all critical data on easy to search structures (binary searches mostly), avoiding the need for many complex helper structures after parsing.

## Main limitations

- Glyph max size is 256x256. This could theoretically get problematic with ligatures in complex scripts, but this is not a primary concern at the moment.
- Max font file size is 64MiB.
- Max number of glyphs is 56789. Some space is reserved for additional "control glyph indices".
- Glyph composition is not supported (creating a single glyph from the combination of other glyphs).
- FSMs section hasn't been designed yet. This would enable complex script shaping assistance and ligatures, which I personally consider fairly important features (even if advanced and not critical for most users of this format).

## Data types

Data in the spec can be described using the following types:
- `uint8`, `uint16`, `uint32`, `uint64`: little-endian encoded, taking 1, 2, 4 and 8 bytes respectively.
- `int8`, `int16`, `int32`, `int64`: little-endian encoded, taking 1, 2, 4 and 8 bytes respectively.
- `bool`: single byte, must be 0 (false) or 1 (true).
- `short string`: an `uint8` indicating string length in bytes, followed by the string data in utf8.
- `string`: an `uint16` indicating string length in bytes, followed by the string data in utf8.
- `short slice`: an `uint16` indicating the number of elements in the slice, followed by the list contents.
- `slice`: an `uint32` indicating the number of elements in the slice, followed by list contents.
- `array`: a slice with an implicit length given by some other value already present in the font data. In other words: a slice without the initial size indication.
- `date`: a triplet of (uint16, uint8, uint8) (year, month, day).

Misc.:
- basic-name-regexp: `[a-zA-Z](-?[A-Za-z0-9]+)*`, used in a couple places. The natural description of the regxp is the following: name uses only ascii, starts with a letter, and might use hyphens as separators.

## Data sections

All data sections are defined consecutively and they are all non-skippable.

## Signature

6 bytes:
```Golang
[]byte{'t', '2', '6', 'g', 'b', 'f'}
```

After the signature, all data is gzipped. The 64MiB size limit applies to the *uncompressed font data*.

### Header

```Golang
FormatVersion uint32 // tgbf format version (only 0x0000_0001 allowed at the moment)
FontID uint64 // font unique ID, generated with crypto/rand or similar
VersionMajor uint16 // starts at 0, raise when releasing major font changes
VersionMinor uint16 // starts at 0, raise when releasing minor font changes
FirstVersionDate date // year, month, day
MajorVersionDate date // year, month, day
MinorVersionDate date // year, month, day

Name shortString // font name (must have at least length = 1) [recommendation: keep it ASCII]
Family shortString // font family, may have length 0 [recommendation: keep it ASCII]
Author shortString // author name(s), may have length 0
About string // other info about the font, may have length 0. <255 chars recommended
```

Everything should be fairly self-explanatory. Family is for related groups of fonts or font faces, like bold and italic versions, sans/serif variants and so on.

### Metrics

```Golang
NumGlyphs uint16 // max is 56789, then 3211 reserved for control codes, then 5530 for custom glyphs
VertLayout bool // if true, additional data must be provided for vertical drawing metrics
MonoWidth uint8 // set to 0 if the font is not monospaced
MonoHeight uint8 // only relevant if VertLayout is true. like MonoWidth but for height.

Ascent uint8 // font ascent without accounting for diacritics. can't be zero.
ExtraAscent uint8 // extra ascent for diacritics, if any required. must be < Ascent
Descent uint8 // font descent without accounting for diacritics.
ExtraDescent uint8 // extra descent for diacritics, if any is required
LowercaseAscent uint8 // a.k.a xheight. set to 0 if no lowercase letters exist
HorzInterspacing uint8 // default horz spacing between glyphs. typically one or zero
VertInterspacing uint8 // if no VertLayout, just leave to 0
LineGap uint8 // suggested line gap for the font
```

NumGlyphs includes only the number of glyphs with actual graphical data (no control glyph indices).

MonoWidth must be allowed to be 0 even if in practice all glyphs have the same width, as this field also expresses intent.

Regarding the glyph range, this font format has multiple areas for control glyph indices:
- 56789 - 56899 (inclusive): predefined control codes that everyone must understand. The currently defined control indices are the following:
	- 56789: missing glyph. Used when a unicode code point doesn't have a corresponding glyph in the font. Notice that this is not the same as the "notdef" glyph. It's recommended that a font has a "notdef" named glyph, but it's up to the programmer and renderer code to replace 56789 with notdef or panic or return an error or replace with something else.
	- 56790: the zilch glyph. This can be used when manipulating glyph buffers for padding and optimization purposes, to replace mising glyphs and avoid panicking or other things. Renderers should simply skip this glyph as if nothing happened, not even resetting the previous glyph for kerning.
	- 56791: the new line glyph. This allows working with glyph index slices without renouncing to line breaks. This should reset the previous glyph for kerning.
	- ... the rest are undefined
- 56900 - 56999: reserved for font-specific control codes. E.g., for use in FSMs. Some of these control codes can be named.
- 57000 - 57999: reserved for renderer-specific control codes (library-level customization).
- 58000 - 59999: reserved for user-specific control codes (library user customization).

Custom glyphs can be added at runtime in the 60k - 62k range (inclusive). The rest is undefined.

The font's basic size is considered to be `Ascent + Descent`. The `LineGap` must be applied between lines of this size, without also adding the extra ascents and descents. This could lead to overlaps in some extreme cases. A font without extra ascents and extra descents will be referred to as a "tightly sized" font.

Having `ExtraDescent` is very uncommon among languages using the latin alphabet, as even the languages that use hooks like "ᶏ" and "ꞔ" rarely descend more than "g", "q" and other common letters. An exception is if only "capital" letters are used (e.g. "Ƒ", "Ģ" or even "Ç").

The `ExtraAscent` is quite common unless the font actively "squeezes" capital letters to accommodate accents and diacritic marks like "Ä", "É", "Û" and similar. Some fonts may use a feature flag to handle this in different ways. Squeezing letters is not aesthetic and should be avoided, but on some low-res games it might be necessary. If you only want to do it for the retro feel... please seek help.

The `LowercaseAscent` should be set to 0 when no lowercase version of the letters exist. Having unicode mappings from lowercase letters to their capitalized glyphs is discouraged; being strict is better in the font definition context. Diacritic marks on lowecase letters must not be considered for this ascent value. Finally, if lowercase letters are shaped as uppercase letters (changing only the size of lowercase and uppercase letters, known as "small-caps"), this metric still applies and must be set. This can also be done conditionally with a feature flag.

If any of the fields is proven to be incorrect during parsing (ascent, extra ascent, etc.), parsing should immediately return an error. That being said, not all verifications are cheap, so font parsers should offer options or additional methods to check the correctness of the font with different degrees of strictness.

### Glyphs data

```Golang
NumNamedGlyphs uint16
NamedGlyphIDs [NumNamedGlyphs]uint16 // references GlyphNames in order (custom control codes can be included)
NamedGlyphOffsets [NumNamedGlyphs]uint16 // references GlyphNames in order
GlyphNames [NumNamedGlyphs]shortString // in lexicographical order. names can't repeat

GlyphMaskOffsets [NumGlyphs]uint32 // offsets to glyph data for the given glyph index
GlyphMasks [NumGlyphs](Bounds, short[]RasterOperation) // (move to, line to, etc.)
```

The glyph bounds are:
```Golang
maskWidth, maskHeight uint8
leftOffset, rightOffset int8 // I prefer "offsets" to "side bearings" terminology in this context
topOffset, bottomOffset int8 // omitted if VertLayout is false.
vertDrawHorzOffset int8 // omitted if VertLayout is false. leftOffset is *NOT APPLIED* on vert draws
```

Glyph data is easy to look up thanks to the `GlyphMaskOffsets` index. Data is encoded using raster commands are sequences of "control codes" and data:
- Control codes are based on bit flags:
	- 0b0000_0001 : change lightness. Defaults to 255 (max). Changes persist through multiple commands (lightness is only reset on new glyph mask definition). Will have uint8 value in the data.
	- 0b0000_0010 : move horz, will have int8 value in the data.
	- 0b0000_0100 : move vert, will have int8 value in the data.
	- 0b0000_1000 : draw horz, will have int8 value in the data.
	- 0b0001_0000 : draw vert, will have int8 value in the data.
	- (note: if draw horz + vert are combined, we draw a rect)
	- 0b0010_0000 : post horz move (can be used even if not drawing).
	- 0b0100_0000 : post vert move (can be used even if not drawing).
	- 0b1000_0000 : reset alpha to previous value.
- Data sequences are determined by the control codes.
- The sequences end based on the `short[]RasterOperation` length.

There are many ways to encode any single mask. This might not be a trivial problem to solve optimally, but that's not particularly important. Any decent algorithm will do (e.g. find big rectangular areas, split them, sort by proximity, encode them one by one (for hard edge cases take the simplest path, they are not common)).

Named glyphs are useful to look up "unique" glyphs from within the code, like, "half heart icon" or others that may be relevant in your game but you can't or don't want connected to standard unicode code points. These would be mapped to a unicode private area like (U+E000, U+F8FF). Recommended standard named glyphs:
- `notdef`: rectangle representing missing glyph.
Names must conform to `basic-name-regexp`. Names aren't meant to support naming *all* glyphs in the font, only glyphs that have to be referenceable in particular due to *specific game needs* or general operation (e.g. notdef).

### Parametrization

```Golang
NumVariables uint8
Variables [NumVariables](uint8, uint8, uint8, uint8) // (init value, min value, max value)

NamedVarKeys []uint8 // references VariableNames in order
NamedVarOffsets []uint16 // references VariableNames in order
VariableNames []shortString // in lexicographical order
```

Variable names must conform to `basic-name-regexp`. The number of variables is limited to 255.

Variables are used for conditional or variable `code point -> rune` mappings, and for custom FSMs. This is all explored throughout the next sections.

### Mapping table

Every font needs a mapping table in order to indicate which glyph indices correspond to each unicode code point.

```Golang
NumMappingEntries uint32
NumMappingModes uint8 // typically 0. can't be 256, max is 255. default mode not counted
MappingModeRoutineOffsets [NumMappingModes]uint16
MappingModeRoutines short[]uint8

FastMappingTables short[]FastMappingTable // see FastMappingTable section, prioritized over the general mapping table

CodePointList [NumMappingEntries]uint32
CodePointModes [NumMappingEntries]uint8 // we do the binary search here
CodePointMainIndices [NumMappingEntries]uint16 // glyph indices if mode == 255, or offsets to CodePointModeIndices
CodePointModeIndices short[]uint16
```

At the most basic level, mapping can be done by assigning mode 255 to every entry, which is a special mode that indicates that a code point maps unconditionally to a single glyph index.

The remaining modes, from 0 to 254, can be customly defined to enable a variety of features:
- Stylistic alternates. This can include randomized variations, dialectal glyph variations (for game flavor mostly), conditional font glyph stylizations, etc.
- Animated glyphs. This is uncommon for regular letters, but games can often benefit from textual cursors, small input icons and other small animated elements that can often be part of the text.
- Feature flags:
	- Small caps (lowercase displayed as smaller uppercase) (should be named "small-caps").
	- Squeezing capital letters when they have accents.
	- Numeric style (lining figures, oldstyle figures, proportional figures, tabular figures).
	- Slashed zero.
	- Superscript and subscript.

For the custom modes, each mapping mode routine is defined byte by byte:
- The first byte is an uint8 indicating the number of possible results. This must be at least 2.
- The `ResultIndex` (chosen glyph among the listed choices) defaults to 0 at the start of each routine execution.
- We get one byte indicating a command:
	- The top 2 bits (0bXX00_0000) indicate the command type:
		- 00 = operate: increase / decrease / set the "result" index or a variable.
		- 01 = if/stop: a.k.a filter. Can also be used as a single byte "terminate" (0 == 1).
		- 10 = if/else: has two bytes of offsets (num `if` bytes, num `else` bytes)
		- 11 = undefined: ...maybe quick conditional operation?
	- The remaining 6 bits are the parametrization, which depends on the command type itself. This is described in more detail later.
- After the command byte, we get zero to many bytes with the data for the command. This amount of bytes depends on the command type and its parametrization.
- Each mode routine can have at most 196 bytes. Mode routines are evaluated each time we have to map a character, so going beyond that would result in an unacceptable level of complexity and overhead.

> Notice: a renderer can't really cache mode results. While this would work well in some cases, in others dropping the cache would be too expensive. Random rolls also can't be cached. Animations are not caching-friendly. Trying to develop heuristics will often result in more overhead than eexcuting the mode routine itself.

Parametrization bits for `operate`:
```
0b0000_00XX: 00 = set, 01 = increase by one, 10 = increase, 11 = decrease
0b0000_XX00: operand type (00 = var, 01 = const, 10 = rng, 11 = rng(NumModePossibleResults)
0b000X_0000: result type (0 = result index, 1 = variable (ID given explicitly in data))
0b00X0_0000: quick exit; if 1, we stop after this command
```

Parametrization bits for conditionals:
```
0b0000_00XX: first comparison argument (00 = var, 01 = rng, 10 = 1, 11 = ResultIndex)
0b0000_XX00: second argument (00 = var, 01 = rng, 10 = 0, 11 = const)
0b00XX_0000: comparison operator (00 = "==", 01 = "!=", 10 = "<", 11 = "<=")
```

Parametrization bits for quick conditional operation (prototype, undecided):
```
0b0000_000X: operation (0 = set, 1 = increase)
0b0000_00X0: first comparison argument (0 = var, 1 = const)
0b0000_XX00: second argument (00 = var, 01 = rng, 10 = 1, 11 = ResultIndex)
0b00XX_0000: comparison operator (00 = "==", 01 = "!=", 10 = "<", 11 = "<=")
```

##### FastMappingTable

Fast mapping tables are designed to avoid binary searches on common contiguous regions of unicode code points. The most common case is ASCII range 32 - 126. There can be some unused code points in the middle, and they should have 56789 ("missing" control index) assigned to them.

// TODO: wouldn't it be smart to have a whole table be able to use a "mode" condition? This way, simple feature flags like "small-caps" could be encoded directly at the table level.

```Golang
TableCondition MappingTableCondition
FirstCodePoint int32 // inclusive
LastCodePoint  int32 // inclusive
CodePointModes [TableLength]uint8
CodePointMainIndices [TableLength]uint16
CodePointModeIndices []uint16
```

Mapping table condition:
```Golang
Condition     uint8 // 0b00CC_BBAA; AA and BB are: 00 = var, 01 = rng, 10 = const; CC is comparison operator
FirstArgData  uint8
SecondArgData uint8
```

Parsers must limit the *total* fast mapping tables memory usage to 4KiB. Table lengths must also be strictly limited to less than 4096. Anything above a few hundred contiguous code points tends to be suspicious anyway.

### FSMs

(Not designed yet; finite state machines to assist complex script shaping and creation of ligatures).

### Kernings

```Golang
NumKerningPairs uint32
KerningPairs [NumKerningPairs]uint32 // for binary search (the uint32 is uint16|uint16 glyph indices)
KerningValues [NumKerningPairs]int8
NumVertKerningPairs uint32 // must be 0 unless VertLayout is true
VertKerningPairs [NumVertKerningPairs]uint32 // only relevant if 
VertKerningValues [NumVertKerningPairs]int8
```

Kerning encoding is simplistic and relies on binary searches. Kerning classes are supported, but only on the editor, through a separate file explained in the next setion.

### Edition data

Edition data is stored in a separate file, the `.tgbfwork` file. Preferently, the file name should be shared with the main font file so we can get the two easily when loading the files, but it's not strictly required. Unlike regular `.tgbf` files, the data is not gzipped.

Signature:
```Golang
[]byte{'t', 'w', 'k', 'g', 'b', 'f'}
```

```Golang
NumCategories uint8
CategoryNames [NumCategories]shortString // max 60 chars for the name
CategorySizes [NumCategories]uint16 // number of glyphs per category. total sum must equal NumGlyphs
NumKernClasses uint8
KerningClassNames [NumKernClasses]shortString // max 60 chars for the name
KerningClassValues [NumKernClasses]int8
NumClassedKernPairs uint32
KerningClassPairs [NumClassedKernPairs]uint32 // glyphPair binary search slice
KerningClassIDs [NumClassedKernPairs]uint8
NumClassedVertKernPairs uint32
VertKerningClassPairs [NumClassedVertKernPairs]uint32
VertKerningClassIDs [NumClassedVertKernPairs]uint8
MappingModeNames []shortString // max 60 chars for the name
```

Misc. notes:
- Offer options on export for `Overwrite version`, `New minor version`, `New major version`.
- I'll need some quality checker or stats / health program, to check fast tables and others. Like, report on actual avoidable inefficiencies, fast table usage rates, etc. Maybe integrate into the editor itself.
- ...
