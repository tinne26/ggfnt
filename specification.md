# `ggfnt` font format specification

> Version 0x0000_0001

This is a bitmap font format created for indie game development and pixel art games.

Font files use the `.ggfnt` file extension.

File names should preferently follow the "name-type-size-version.ggfnt" convention, with `type` and `version` being the most optional parts. For example: `bouncy-mono-8d4-v2p14.ggfnt`. `8d4` indicates that the font's ascent is 8 pixels and the descent is 4 pixels. The extra ascent and extra descent, as described later in the metrics section, are not considered for this indication. None of this is mandatory. 

Maximum font file size is limited to 32MiB per spec, which is much more than enough for practical purposes and eliminates a whole class of problems during parsing, storage and others.

The binary format is designed to be reasonably compact, have safe limits and keep all critical data on easy to search structures (binary searches mostly), avoiding the need for many complex helper structures after parsing.

## Main limitations

- Max font file size is 32MiB.
- Max number of glyphs is 56789. Some space is reserved for additional "control glyph indices".
- Glyph composition is not supported (creating a single glyph from the combination of other glyphs).
- FSMs section hasn't been designed yet. This would enable complex script shaping assistance and ligatures, which I personally consider fairly important features (even if advanced and not critical for most users of this format).

## Data types

Data in the spec can be described using the following types:
- `uint8`, `uint16`, `uint32`, `uint64`: little-endian encoded, taking 1, 2, 4 and 8 bytes respectively.
- `int8`, `int16`, `int32`, `int64`: little-endian encoded, taking 1, 2, 4 and 8 bytes respectively.
- `nzuint8`, `nzint8`: non-zero `uint8` and `int8`. For `nzuint8`, we take the `uint8` value + 1. For `nzint8`, negative values are the same as `int8` and non-negative values are +1.
- `bool`: single byte, must be 0 (false) or 1 (true).
- `short string`: an `uint8` indicating string length in bytes, followed by the string data in utf8.
- `string`: an `uint16` indicating string length in bytes, followed by the string data in utf8.
- `short slice`: an `uint8` indicating the number of elements in the slice, followed by the list contents.
- `slice`: an `uint16` indicating the number of elements in the slice, followed by the list contents.
- `array`: a slice with an implicit length given by some other value already present in the font data. In other words: a slice without the initial size indication.
- `date`: a triplet of (uint16, uint8, uint8) (year, month, day). Zero can be used as undefined, but year can only be zero if month and day are also zero, and month can only be zero if day is also zero. In other words: providing a day without a month or a month without a year is invalid.
- `blob`: a data blob of variably-sized elements, indexed by a separate slice.

Misc.:
- basic-name-regexp: `[a-zA-Z](-?[A-Za-z0-9]+)*`, used in a couple places. The natural description of the regexp is the following: name uses only ascii, starts with a letter, and might use hyphens as separators. There's also an additional size limit of max 32 characters.

## Data sections

All data sections are defined consecutively and they are all non-skippable.

## Signature

6 bytes:
```Golang
[6]byte{'t', 'g', 'g', 'f', 'n', 't'}
```

After the signature, all data is gzipped. The 32MiB size limit applies to the *uncompressed font data*.

### Header

```Golang
FormatVersion uint32 // ggfnt format version (only 0x0000_0001 allowed at the moment)
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

Everything should be fairly self-explanatory. Family is for related groups of fonts or font faces, like bold and italic versions, sans/serif variants and so on. Therefore, names would be "Swaggy Sans", "Swaggy Serif" and "Swaggy Bold", and the family would be only "Swaggy". In most cases, the name and family name will be the same.

For major and minor versions, any incompatible change (removing glyphs or tags, changing their meaning, etc) should happen on major versions only, with the only exception of version 0, which is considered alpha/unstable.

### Metrics

```Golang
NumGlyphs uint16 // max is 56789, then 3211 reserved for control codes, then 5530 for custom glyphs
HasVertLayout bool // 0 for no vert layout, 1 for yes // TODO: maybe 2 for yes with mono height?
MonoWidth uint8 // set to 0 if the font is not monospaced

Ascent uint8 // font ascent without accounting for diacritics. can't be zero.
ExtraAscent uint8 // extra ascent for diacritics, if any required. must be < Ascent
Descent uint8 // font descent without accounting for diacritics.
ExtraDescent uint8 // extra descent for diacritics, if any is required
LowercaseAscent uint8 // a.k.a xheight. set to 0 if no lowercase letters exist
HorzInterspacing uint8 // default horz spacing between glyphs. typically one or zero
VertInterspacing uint8 // if no VertLayout, just leave to 0
LineGap uint8 // suggested line gap for the font
VertLineWidth uint8 // must be zero if no VertLayout is used
VertLineGap uint8 // must be zero if no VertLayout is used
```

NumGlyphs includes only the number of glyphs with actual graphical data (no control glyph indices).

MonoWidth must be allowed to be 0 even if in practice all glyphs have the same width, as this field also expresses intent and future compatibility. If 1, the parser might check that the monospaced width is respected.

Regarding the glyph range, this font format has multiple areas for control glyph indices:
- 56789 - 56899 (inclusive): predefined control codes that everyone must understand. The currently defined control indices are the following:
	- 56789: missing glyph. Used when a unicode code point doesn't have a corresponding glyph in the font. Notice that this is not the same as the "notdef" glyph. It's recommended that a font has a "notdef" named glyph, but it's up to the programmer and renderer code to replace 56789 with notdef or panic or return an error or replace with something else.
	- 56790: the zilch glyph. This can be used when manipulating glyph buffers for padding and optimization purposes, to replace mising glyphs and avoid panicking or other things. Renderers should simply skip this glyph as if nothing happened, not even resetting the previous glyph for kerning.
	- 56791: the new line glyph. This allows working with glyph index slices without renouncing to line breaks. This should reset the previous glyph for kerning.
	- ... the rest are undefined
- 56900 - 56999: reserved for font-specific control codes. E.g., for use in FSMs. Some of these control codes can be named (font-level customization).
- 57000 - 57999: reserved for renderer-specific control codes (library-level customization).
- 58000 - 59999: reserved for user-specific control codes (application-level customization).

Custom glyphs can be added at runtime in the 60k - 62k range (inclusive). The rest is undefined.

The font's basic size is considered to be `Ascent + Descent`. The `LineGap` must be applied between lines of this size, without also adding the extra ascents and descents. This could lead to overlaps in some extreme cases. A font without extra ascents and extra descents will be referred to as a "tightly sized" font.

Having `ExtraDescent` is very uncommon among languages using the latin alphabet, as even the languages that use hooks like "ᶏ" and "ꞔ" rarely descend more than "g", "q" and other common letters. An exception is if only "capital" letters are used (e.g. "Ƒ", "Ģ" or even "Ç").

The `ExtraAscent` is quite common unless the font actively "squeezes" capital letters to accommodate accents and diacritic marks like "Ä", "É", "Û" and similar. Some fonts may use a feature flag to handle this in different ways. Squeezing letters is not aesthetic and should be avoided, but on some low-res games it might be necessary. If you only want to do it for the retro feel... please seek help.

The `LowercaseAscent` should be set to 0 when no lowercase version of the letters exist. Having unicode mappings from lowercase letters to their capitalized glyphs is discouraged; being strict is better in the font definition context. Diacritic marks on lowecase letters must not be considered for this ascent value. Finally, if lowercase letters are shaped as uppercase letters (changing only the size of lowercase and uppercase letters, known as "small-caps"), this metric still applies and must be set. This can also be done conditionally with a feature flag.

If any of the fields is proven to be incorrect during parsing (ascent, extra ascent, etc.), parsing should immediately return an error. That being said, not all verifications are cheap, so font parsers might offer options or additional methods to check the correctness of the font with different degrees of strictness.

TODO: are we sure any error should be reported? If we do glyph animations, in some cases we might break some rules, and it's not so clear that the parser should complain about it. Having explicit exceptions doesn't sound super clean, but might be better than nothing..?

### Glyphs data

```Golang
NumNamedGlyphs uint16
NamedGlyphIDs [NumNamedGlyphs]uint16 // references GlyphNames in order (custom control codes can be included)
GlyphNameEndOffsets [NumNamedGlyphs]uint32 // indexes GlyphNames in order
GlyphNames blob[noLenString] // in lexicographical order. names can't repeat

GlyphMaskEndOffsets [NumGlyphs]uint32 // indexes GlyphMasks
GlyphMasks blob[Placement, [...]RasterOperation] // (move to, line to, etc.)
```

Minor technical note: while using uint16 instead of uint32 for NameGlyphOffsets would almost always be reasonable, I want to avoid tricky implicit restrictions, and names themselves also tend to have much higher overhead than those extra 2 bytes per entry.

Glyph names must match basic-name-regexp.

The glyph placement is:
```Golang
advance uint8
topAdvance uint8 // omitted if VertLayout is false
bottomAdvance uint8 // omitted if VertLayout is false
horzCenter uint8 // omitted if VertLayout is false
```

Data is encoded using raster operations, which are sequences of "control codes" and data:
- Control codes are based on bit flags:
	- 0b0000_0001 : change palette index. Defaults to 255. Changes persist through multiple commands (only reset on new glyph mask definition). Will have uint8 value in the data (which can't be zero).
	- 0b0000_0010 : pre horizontal move, will have nzint8 value in the data.
	- 0b0000_0100 : pre vertical move, will have nzint8 value in the data.
	- 0b0000_1000 : single row pre vertical advance (pre vertical move flag must be unset).
	- 0b0001_0000 : flag for diagonal mode. If set, some of the next flags are interpreted differently.
	- 0b0010_0000 : draw horizontally; on diagonal mode, diagonal length. Will have `nzuint8` value in the data.
	- 0b0100_0000 : draw vertically (will have `nzuint8` value in the data); on diagonal mode, flag for ascending (set) or descending (unset) diagonal.
	- 0b1000_0000 : single pixel draw flag (both draw horz and draw vert flags must be unset).

Important:
- The horizontal draw width is always advanced automatically during operations. The vertical draw height is not applied.
- Overflows in movements or draw sizes might cause an error or panic (this could be configurable at the renderer level).
- The initial position (0, 0) corresponds to the first left pixel below the baseline. To go right, we use positive horizontal values. To up, we use negative vertical values (though in general operations are given from top-left to bottom right, so negative vertical advances are not naturally found except on the initial movement).

There are many ways to encode any single mask. This might not be a trivial problem to solve optimally, but that's not particularly important at the moment, and many simple solutions are enough.

Named glyphs are useful to look up "unique" glyphs from within the code, like, "ico-heart-half" or others that may be relevant in your game but you can't or don't want connected to standard unicode code points. These would be mapped to a unicode private area like (U+E000, U+F8FF). Recommended standard named glyphs:
- `notdef`: rectangle representing missing glyph.
Names must conform to `basic-name-regexp`. Names aren't meant to support naming *all* glyphs in the font, only glyphs that have to be referenceable in particular due to *specific game needs* or general operation (e.g. notdef).

### Color

Actual color table:
```Golang
NumColorSections uint8 // must be at least 1
ColorSectionModes [NumColorSections]uint8 // if 1, palette, if 0, alpha scale
ColorSectionStarts [NumColorSections]uint8 // inclusive, can't be zero, in descending order
ColorSectionEndOffsets [NumColorSection]uint16 // section length must be checked at parsing time
ColorSections blob[[...]byte] // if palette, sectionDataLen = sectionLen*4, otherwise, sectionDataLen = sectionLen
ColorSectionNameEndOffsets [NumColorSections]uint16
ColorSectionNames blob[ColorSectionDefinition]
```

Note for renderer implementers: main dye should be optimized using vertex attributes. Others will need explicit uniform changes, but that's expected.

### Variables

```Golang
NumVariables uint8
Variables [NumVariables](uint8, uint8, uint8) // (init value, min value, max value)

NumNamedVars uint8
NamedVarKeys [NumNamedVars]uint8 // references VariableNames in order
VarNameEndOffsets [NumNamedVars]uint16 // references VariableNames in order
VariableNames blob[noLenString] // in lexicographical order
```

Variable names must conform to `basic-name-regexp`. The max number of variables is 255.

Variables are used for conditional or variable `code point -> rune` mappings and for custom FSMs. This is all explored throughout the next sections.

We also have some special variables that renderers might detect and adjust automatically if they are named:
- "vert-mode-on": set to 1 when rendering in vertical mode, 0 otherwise.
- TODO: decide if this is really a good idea. It's helpful for vert mode, but if that's the only thing, it might be better to do it manually or something. leading or first glyph could be a thing too. maybe a special variable to allow metrical transgressions, though that seems like a terrible way to go about it.

There are also other variables that have semi-standardized names but are not automatically adjusted:
- "glyph-rotation": if 0, glyph rotation is determined by "vert-mode-on". If 1, glyphs are always rotated. If 2, glyphs are never rotated.

### Mapping

Every font needs a mapping section in order to indicate which glyph indices correspond to each unicode code point.

```Golang
NumMappingModes uint8 // typically 0. default mode is 255. max is 254
MappingModeRoutineEndOffsets [NumMappingModes]uint16
MappingModeRoutines blob[[...]uint8]

FastMappingTables short[]FastMappingTable // see FastMappingTable section, prioritized over the general mapping table

NumMappingEntries uint16
CodePointList [NumMappingEntries]int32 // we do the binary search here if no fast table was relevant
CodePointModes [NumMappingEntries]uint8
CodePointMainIndices [NumMappingEntries]uint16 // mode == 255 ? glyphIndex : CodePointModeIndices end index (exclusive)
CodePointModeIndices []uint16 // max 64 indices per code point
```

At the most basic level, mapping can be done by assigning mode 255 to every entry, which is a special mode that indicates that a code point maps unconditionally to a single glyph index.

The remaining modes, from 0 to 254, can be customly defined to enable a variety of features:
- Stylistic alternates. This can include randomized variations, dialectal glyph variations (for game flavor mostly), conditional font glyph stylizations, etc.
- Animated glyphs. Games can often benefit from textual cursors, small input icons and other small animated elements that are part of the text. For letters themselves it's more uncommon, but falling blood, letters breaking, rotations and many others are also totally possible.
- Feature flags:
	- Small caps (lowercase displayed as smaller uppercase) (should be named "small-caps").
	- Squeezing capital letters when they have accents.
	- Numeric style (lining figures, oldstyle figures, proportional figures, tabular figures) (TODO: this sounds like needing enums).
	- Slashed zero.
	- Superscript and subscript.

For the custom modes, each mapping mode routine is defined byte by byte:
- The first byte is an uint8 indicating the number of possible results. This must be at least 2.
- The `ResultIndex` (chosen glyph among the listed choices) defaults to 0 at the start of each routine execution.
- We get one byte indicating a command:
	- The top 2 bits (0bXX00_0000) indicate the command type:
		- 00 = operate: increase / decrease / set the "result" index or a variable.
		- 01 = if/stop: a.k.a filter. Can also be used as a single byte termination (0 == 1).
		- 10 = if/else: has two bytes of offsets (num `if` bytes, num `else` bytes)
		- 11 = reserved for future versions (...maybe quick conditional operation?).
	- The remaining 6 bits are the parametrization, which depends on the command type itself. This is described in more detail later.
- After the command byte, we get zero to many bytes with the data for the command. This amount of bytes depends on the command type and its parametrization.
- Each mode routine can have at most 228 bytes. Mode routines are evaluated each time we have to map a character, so going beyond that would result in an unacceptable level of complexity and overhead.

> Notice: a renderer can't really cache mode results. While this would work well in some cases, in others dropping the cache would be too expensive. Random rolls also can't be cached. Animations are not caching-friendly. Trying to develop heuristics will often result in more overhead than executing the mode routine itself.

Parametrization bits for `operate`:
```
0b0000_00XX: 00 = set, 01 = increase by one, 10 = increase, 11 = decrease
0b0000_XX00: operand type (00 = var, 01 = const, 10 = rng, 11 = rng(NumModePossibleResults)
0b000X_0000: result type (0 = ResultIndex, 1 = variable (ID given explicitly in data))
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

Notice that when `CodePointMainIndices` is indexing `CodePointModeIndices` in `mode != 255`, the offset is not in bytes like in most other tables throughout the spec, but elements (`uint16`).

Misc. technical notes:
- While glyph index overlap would be possible while indexing `CodePointModeIndices`, it leads to all kinds of unnecessary complications, so don't think about it.
- Technically the max 64 glyphs per code point can be bypassed by making the glyph appear in one or more fast mapping tables and/or the final main mapping table. This is ok, we don't enforce a *total* max 64 glyphs per code point.
- NumMappingEntries was initially uint32, but then I reasoned that while it's true that you can have multiple code points mapped to a single glyph (e.g. lowercase and uppercase, different ideographic symbols like triangles mapped to a single triangle glyph, etc), these would generally derive from bad practices and seem quite unrealistic. It's much more reasonable to hit other limits first.

##### FastMappingTable

Fast mapping tables are designed to avoid binary searches on common contiguous regions of unicode code points. The most common case is ASCII range 32 - 126. There can be some unused code points in the middle, and they should have 56789 ("missing" control index) assigned to them.

```Golang
TableCondition MappingTableCondition
StartCodePoint int32 // inclusive (int32 = rune)
EndCodePoint   int32 // exclusive (int32 = rune)
CodePointModes [TableLength]uint8
CodePointMainIndices [TableLength]uint16 // mode == 255 ? glyphIndex : CodePointModeIndices end index (exclusive)
CodePointModeIndices []uint16 // max 64 indices per code point
```

Mapping table condition:
```Golang
Condition     uint8 // 0b00CC_BBAA; AA and BB are: 00 = var, 01 = rng, 10 = const; CC is comparison operator
FirstArgData  uint8
SecondArgData uint8
```

Parsers must limit the *total* fast mapping tables memory usage to 32KiB. Table lengths must also be strictly limited to less or equal than 1000. Anything above a few hundred contiguous code points tends to be suspicious anyway.

### FSMs

(Not designed yet; finite state machines to assist complex script shaping and creation of ligatures).

### Kernings

```Golang
NumHorzKerningPairs uint32
HorzKerningPairs [NumHorzKerningPairs]uint32 // for binary search (the uint32 is uint16|uint16 glyph indices)
HorzKerningValues [NumHorzKerningPairs]int8
NumVertKerningPairs uint32 // must be 0 if VertLayout is false
VertKerningPairs [NumVertKerningPairs]uint32
VertKerningValues [NumVertKerningPairs]int8
```

Kerning encoding is simplistic and relies on binary searches. Kerning classes are supported, but only on the editor, through a separate file explained in the next section.

### End

Since data is gzipped, we expect the EOF here, which will also verify the checksum.

### Edition data

Edition data is stored in a separate file, the `.ggwkfnt` file. Preferently, the file name should be shared with the main font file so we can get the two easily when loading the files, but it's not strictly required. The data is gzipped right after the signature.

There's also a 

Signature:
```Golang
[]byte{'w', 'k', 'g', 'f', 'n', 't'}
```

```Golang
FontID uint64
NumCategories uint8
CategoryNames [NumCategories]shortString // max 60 chars for the name
CategorySizes [NumCategories]uint16 // number of glyphs per category. total sum must equal NumGlyphs

NumKerningClasses uint16 // classes are one-indexed
KerningClassNames [NumKerningClasses]shortString // must conform to basic-name-regex (+ possible spaces)
KerningClassValues [NumKerningClasses]int8

NumHorzKerningPairsWithClasses uint32
HorzKerningPairs [NumHorzKerningPairsWithClasses](first, second uint16, class uint16)

NumVertKerningPairsWithClasses uint32
VertKerningPairs [NumVertKerningPairsWithClasses](first, second uint16, class uint16)

MappingModeNames short[]shortString // must conform to basic-name-regex (+ possible spaces)
```
