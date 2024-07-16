# `ggfnt` font format specification

> Version 0x0000_0001

This is a bitmap font format created for indie game development and pixel art games.

Font files use the `.ggfnt` file extension.

File names should preferently follow the "name-type-size-version.ggfnt" convention, with `type` and `version` being the most optional parts. For example: `bouncy-mono-8d4-v2p14.ggfnt`, or `bouncy-8d4.ggfnt`. `8d4` indicates that the font's typical ascent is 8 pixels and the typical descent is 4 pixels[^1]. The extra ascent and extra descent, as described later in the metrics section, are not considered for this indication. None of this is mandatory.

[^1]: This is currently an open issue in the format regarding the ambiguity of "ascent" and "descent". For example, a font might have all uppercase letters take at most 5 pixels, but then have parentheses and some punctuation symbols go one pixel higher and one pixel lower. What should ascent be, then? For the font name, I feel like 5d0 would be more representative than 6d1, even though font creators would typically use ascent = 6 and descent = 1 (again, this remains too ambiguous). TODO: figure it out...

Maximum font file size is limited to 32MiB per spec, which is much more than enough for practical purposes and eliminates a whole class of problems during parsing, storage and others.

The binary format is designed to be reasonably compact, have safe limits and keep all critical data on easy to search structures (binary searches mostly), avoiding the need for many complex helper structures after parsing.

## Main limitations

- Max font file size is 32MiB.
- Max glyph advance is 255.
- Max number of glyphs is 56789. Some space is reserved for additional "control glyph indices".
- Glyph composition is not supported (creating a single glyph from the combination of other glyphs).

## Data types

Data in the spec can be described using the following types:
- `uint8`, `uint16`, `uint24`, `uint32`, `uint64`: little-endian encoded, taking 1, 2, 3, 4 and 8 bytes respectively.
- `int8`, `int16`, `int32`, `int64`: little-endian encoded, taking 1, 2, 4 and 8 bytes respectively.
- `nzuint8`, `nzint8`: non-zero `uint8` and `int8`. For `nzuint8`, we take the `uint8` value + 1. For `nzint8`, negative values are the same as `int8` and non-negative values are +1.
- `bool`: single byte, must be 0 (false) or 1 (true).
- `short string`: an `uint8` indicating string length in bytes, followed by the string data in utf8.
- `string`: an `uint16` indicating string length in bytes, followed by the string data in utf8.
- `short slice`: an `uint8` indicating the number of elements in the slice, followed by the list contents.
- `slice`: an `uint16` indicating the number of elements in the slice, followed by the list contents.
- `array`: a slice with an implicit length given by some other value already present in the font data. In other words: a slice without the initial size indication.
- `date`: a triplet of (uint16, uint8, uint8) (year, month, day). Zero can be used as undefined, but year can only be zero if month and day are also zero, and month can only be zero if day is also zero. In other words: providing a day without a month or a month without a year is invalid.
- `blob`: a data blob of variably-sized elements, usually indexed by a separate slice.

Misc.:
- basic-name-regexp: `[a-zA-Z](-?[A-Za-z0-9]+)*`, used in a couple places. The natural description of the regexp is the following: name uses only ascii, starts with a letter, and might use hyphens as separators. There's also an additional size limit of max 32 characters.

## Data sections

All data sections are defined consecutively and they are all non-skippable.

Many data blobs are indexed with "EndOffset" arrays. The first element would be defined by `blob[0 : EndOffset[0]]`, the second by `blob[EndOffset[0] : EndOffset[1]]`, and so on. The length of the data blob is `EndOffset[NumElements - 1]`.

## Signature

6 bytes:
```Golang
[6]byte{'t', 'g', 'g', 'f', 'n', 't'}
```

After the signature, all data is gzipped. The 32MiB size limit applies to both the compressed and uncompressed font data. In any realistic scenario, if the already compressed font data exceeds 32MiB, the uncompressed version will exceed that by even a much wider margin.

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

Regarding major and minor versions, any incompatible change (removing glyphs or tags, changing their meaning, etc) should happen only on major versions. Version 0 is an exception which should always be considered alpha/unstable.

TODO: what about adding a field for "LICENSE"?

### Metrics

```Golang
NumGlyphs uint16 // max is 56789, then 3211 reserved for control codes, then 5530 for custom glyphs
HasVertLayout bool // if a font has a vert layout, it will have to indicate how to place glyphs vertically
MonoWidth uint8 // set to 0 if the font is not monospaced

Ascent uint8 // font ascent without accounting for diacritics or decorations. can't be zero.
ExtraAscent uint8 // extra ascent for diacritics or decorations, if any required. must be < Ascent
Descent uint8 // font descent without accounting for diacritics or decorations.
ExtraDescent uint8 // extra descent for diacritics or decorations, if any is required
UppercaseAscent uint8 // a.k.a cap height. usually the same value as 'Ascent'
MidlineAscent uint8 // a.k.a xheight. set to 0 if no lowercase letters exist
HorzInterspacing uint8 // default horz spacing between glyphs. typically one or zero
VertInterspacing uint8 // must be zero if HasVertLayout is false
LineGap uint8 // suggested line gap for the font
VertLineWidth uint8 // must be zero if HasVertLayout is false. doesn't include VertLineGap
VertLineGap uint8 // must be zero if HasVertLayout is false
```

`NumGlyphs` includes only the number of glyphs with actual graphical data (no control glyph indices).

MonoWidth must be allowed to be 0 even if in practice all glyphs have the same width, as this field also expresses intent and future compatibility. If 1, the parser might check that the monospaced width is respected.

Regarding the glyph range, this font format has multiple areas for control glyph indices:
- 56789 - 56899 (inclusive): predefined control codes that everyone must understand. The currently defined control indices are the following:
	- 56789: missing glyph. Used when a unicode code point doesn't have a corresponding glyph in the font. Notice that this is not the same as the "notdef" glyph. It's recommended that a font has a "notdef" named glyph as the very first glyph, but it's up to the programmer and renderer code to replace 56789 with notdef or panic or return an error or replace with something else.
	- 56790: the zilch glyph. This can be used when manipulating glyph buffers for padding and optimization purposes, to replace missing glyphs and avoid panicking or other things. Renderers should simply skip this glyph as if nothing happened, not even resetting the previous glyph for kerning, and ignoring them when it comes to rewrite rules.
	- 56791: the new line glyph. This allows working with glyph index slices without renouncing to line breaks. This should reset the previous glyph for kerning.
	- TODO: due to unicode-like sequences with variants and zero width spaces, maybe I should have something to omit glyphs too. Zilch glyph can be used to some extent, but the rules are not clearly defined.
	- ... the rest is undefined. Other potential control indices (most would be implemented at the renderer level): `text-start`, useful for rewrite rules; `wrap-enabler-mark`, to signal line wrapping being possible at custom points; `rewrite-breaker`, to interrupt rewrite rules but be treated as the zilch glyph otherwise; etc.
- 56900 - 56999: reserved for font-specific control codes. E.g., for use in rewrite rules. Some of these control codes can be named (font-level customization).
- 57000 - 57999: reserved for renderer-specific control codes (library-level customization).
- 58000 - 59999: reserved for user-specific control codes (application-level customization).

Custom glyphs can be added at runtime with indices in the 60k - 62k range (inclusive). The rest is undefined.

The font's basic size is considered to be `Ascent + Descent`. The `LineGap` must be applied between lines of this size, without also adding the extra ascents and descents. This could lead to overlaps in some extreme cases. A font without extra ascents and extra descents will be referred to as a "tightly sized" font.

Having `ExtraDescent` is very uncommon among languages using the latin alphabet, as even the languages that use hooks like "ᶏ" and "ꞔ" rarely descend more than "g", "q" and other common letters. An exception is if only "capital" letters are used (since you can have characters like "Ƒ", "Ģ" or "Ç").
TODO: ascent and descent are too ambiguous. Sometimes certain punctuation symbols or parentheses can exceed the ascent and descent of the upper and lowercase letters. I need to either change fields to be less ambiguous or define clearer policies around what we should include in ascent / descent.

 (TODO: decide about this; while centering can get weird, it's unclear what should be the policy in deciding what belongs or does not belong to the descent. One idea would be to have an extra field for it, but it gets kinda weird too. is line height ascent + descent + punctDescent then, or what? it's tricky... but unlike in vectorial fonts, this is a real problem because parens and so on often exceed the basic line dimensions... but even if you make the distinction in the metrics, dealing with it in practical code gets annoying. and then, what about file names? 6d0 might refer to punctuation or uppercase... false advertising, yikes. DECIDE, THIS IS IMPORTANT)

The `ExtraAscent` is quite common unless the font actively "squeezes" capital letters to accommodate accents and diacritic marks like "Ä", "É", "Û" and similar. Some fonts may use a feature flag to make squeezing optional. Squeezing letters is not aesthetic and should be avoided, but on some low-res games it might be necessary. If you only want to do it for the retro feel... please seek help.

`ExtraAscent` and `ExtraDescent` might also be needed in some fonts with animated or decorated glyphs in order to account for the required extra space.

The `MidlineAscent` should be set to 0 when no lowercase version of the letters exist. Having unicode mappings from lowercase letters to their capitalized glyphs is discouraged; being strict is better in the font definition context. Diacritic marks on lowecase letters must not be considered for this ascent value. Finally, if lowercase letters are shaped as uppercase letters (changing only the size of lowercase and uppercase letters, known as "small-caps"), this metric still applies and must be set. This can also be done conditionally with a feature flag.

If any of the fields is proven to be incorrect during parsing (ascent, extra ascent, etc.), strict parsing should immediately return an error. That being said, verifications can be expensive, so default parsing methods might omit some of these checks that are linear on the amount of font glyphs.

### Color

The font format supports up to 255 colors, divided in two types of colors: dyes and palettes.

Dyes are colors that can be configured on the renderer. Each dye can include mutiple alpha values. Almost all fonts have a `"main"` dye, typically with 255 alpha. Sometimes the main dye will include additional alpha values, like 128. In other words: a dye is an alpha scale to which we can apply a custom color.

Palettes are sets of "static" colors. A renderer might allow swapping the colors in a palette, but this requires context awareness and understanding what the palette colors are being used for. Typically, they are only used for emojis and icons.

```Golang
NumDyes uint8
DyeEndIndices [NumDyes]uint8 // end indices are exclusive
DyeAlphas blob[...]
DyeNameEndOffsets [NumDyes]uint16
DyeNames blob[...]
NumPalettes uint8 // NumDyes + NumPalettes can't exceed 255
PaletteEndIndices [NumPalettes]uint8 // this doesn't directly index PaletteColors, that's *4
PaletteColors blob[...] // 4 uint8 per color (RGBA format)
PaletteNameEndOffsets [NumPalettes]uint16
PaletteNames blob[...]
```

Internally, colors are stored by index. The index zero is reserved for transparent. Then come dyes, and after that the palettes. This means the total number of color indices can't exceed 256 (255 if you don't count the built-in transparent).

The DyeEndIndices don't refer to these internal indices, though, but rather the relative dye indices. For example, if we have two dyes with a single alpha each, we would have `DyeEndIndices = [2]uint8{1, 2}`. Palette indices operate in the same way (they don't continue from 3, they restart).

> Note for GPU renderer implementers: main dye should be optimized using vertex attributes. Others will need explicit uniform changes, but that's expected.

### Glyphs data

```Golang
NumNamedGlyphs uint16
NamedGlyphIDs [NumNamedGlyphs]uint16 // references GlyphNames in order (custom control codes can be included)
GlyphNameEndOffsets [NumNamedGlyphs]uint24 // indexes GlyphNames in order
GlyphNames blob[noLenString] // in lexicographical order. names can't repeat

GlyphMaskEndOffsets [NumGlyphs]uint24 // indexes GlyphMasks
GlyphMasks blob[Placement, [...]RasterOperation] // (move to, line to, etc.)
```

Glyph names must match `basic-name-regexp`.

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
	- 0b1000_0000 : single pixel draw flag (both draw horz and draw vert flags must be unset). on diagonal mode, undefined (panic).

Important:
- The horizontal draw width is always advanced automatically during operations. The vertical draw height is not applied.
- Overflows in movements or draw sizes might cause an error or panic (this could be configurable at the renderer level).
- The initial position (0, 0) corresponds to the first left pixel below the baseline. To go right, we use positive horizontal values. To up, we use negative vertical values (though in general operations are given from top-left to bottom right, so negative vertical advances are not naturally found except on the initial movement).

There are many ways to encode any single mask. This might not be a trivial problem to solve optimally, but that's not particularly important at the moment, and many simple solutions are enough.

Named glyphs are useful to look up "unique" glyphs from within the code, like, "ico-heart-half" or others that may be relevant in your game but you can't or don't want connected to standard unicode code points. These would be mapped to a unicode private area like (U+E000, U+F8FF). Recommended standard named glyphs:
- `notdef`: rectangle representing missing glyph. This should also be the first glyph in the font.
Names must conform to `basic-name-regexp`. Names aren't meant to support naming *all* glyphs in the font, only glyphs that have to be referenceable due to *specific game needs* or general operation (e.g. "notdef").

### Settings

Public, named font settings that can be configured from the user side.

```Golang
NumWords uint8 // many words are already predefined if not overridden
WordEndOffsets [NumWords]uint16
Words blob[noLenString] // must conform to basic-name-regexp

NumSettings uint8
SettingNameEndOffsets [NumSettings]uint16
SettingNames blob[noLenString]
SettingEndOffsets [NumSettings]uint16 // references Settings
Settings blob[...]
```

Each setting has a list of options, written as bytes referencing the available words. The value 255 is not valid.

Setting names must conform to `basic-name-regexp`. The max number of settings is 255.

Settings are used for conditional mappings and rewrite rules, which allow supporting stylistic glyph alternates, feature flags, animations and ligatures, among others.

To see the worlds already defined, check [the source code]().

### Mapping

Every font needs a mapping section in order to indicate which glyph indices correspond to each unicode code point.

```Golang
NumMappingSwitches uint8 // can't exceed 254
MappingSwitchEndOffsets [NumMappingSwitches]uint16
MappingSwitches blob[...]

NumMappingEntries uint16
CodePointsIndex [NumMappingEntries]int32 // we do the binary search here
MappingEndOffsets [NumMappingEntries]uint24
Mappings blob[...]
```

Each entry of mapping data consists of a switch. The first byte is the switch type (255 if inconditional mapping is used). If any other switch type is used, we have a sequence of glyph groups with one or more glyph indices. 254 is reserved for a single group without settings. Glyph groups are defined as follows:
- Total group size as `uint8`. If the top bit is 1, the group is a range, otherwise it's a list of individual glyphs. The remaining 0x0XXX_XXXX bits indicate the group size - 1 (so, 0 is 1, 1 is 2, etc).
- If group size > 1, we have one byte indicating the group's animation flags and characteristics:
	- 0b0000_0001: loopable. The animation can wrap back to the start after reaching the end.
	- 0b0000_0010: sequential. The animation should always be played sequentially from start to end or backwards.
	- 0b0000_0100: terminal. The animation represents a vanishing or destructive sequence that shouldn't be automatically rewinded or replayed.
	- 0b0000_1000: split. The animation is composed of independent fragments that the animation can be stopped at. A split animation should never be sequential, but a non-sequential animation is not always necessarily split.
	- 0bXXXX_0000: the animation class index. Still undefined. (TODO). discretionary use by the font creator? Maybe leave as custom class flags instead. Like UI vs Char and so on.

Switches are defined as lists of settings. Based on the amount of combinations and their possible values, we have an exhaustive switch, indexed from 0...N, where N can't be higher than 253. Implementers should cache recent 'code point to glyph index set' results. Going through switch cases is slow, as we need to advance linearly; this is why caching is considered important in this context.

Mapping to a set of glyph indices and using switches can be useful for a number of features:
- Stylistic alternates. This can include randomized variations, dialectal glyph variations (for game flavor mostly), conditional font glyph stylizations, etc.
- Animated glyphs. Games can often benefit from textual cursors, small input icons and other small animated elements that are part of the text. For letters themselves it's more uncommon, but falling blood, letters breaking, rotations and many others are also totally possible.
- Feature flags:
	- Small caps (lowercase displayed as smaller uppercase) (should be named "small-caps").
	- Squeezing capital letters when they have accents.
	- Numeric style (lining figures, oldstyle figures, proportional figures, tabular figures) (TODO: this sounds like needing enums).
	- Slashed zero.
	- Superscript and subscript.

### Rewrite rules

Rewrite rules allow fonts to support ligatures, emoji codes or other user-defined special sequences.

```Golang
NumConditions uint8 // auxiliar conditions used on some rewrite rules
ConditionEndOffsets [NumConditions]uint16
Conditions blob[...]

NumUTF8Sets uint8
UTF8SetEndOffsets [NumUTF8Set]uint16
UTF8Sets blob[...]
NumGlyphSets uint8
GlyphSetEndOffsets [NumGlyphSet]uint16
GlyphSets blob[...]

NumUTF8Rules uint16
UTF8RuleEndOffsets []uint24
UTF8Rules blob[...]
NumGlyphRules uint16
GlyphRuleEndOffsets []uint24
GlyphRules blob[...]
```

UTF8 and glyph sets are defined in two parts, a list of element ranges and a list of individual elements:
- The first byte indicates the number of element ranges we have. Each range is defined by the glyph or code point, plus a uint8 afterwards indicating the range size. For example, {'0', 9} would capture the {'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'} range.
- After that, we get a byte indicating the length of the list of elements, followed by the elements themselves.

Both sequences of glyph indices and code points are allowed. Each rewrite rule is defined in the following format:
- For glyph rules: `uint8` for the condition that must be satisfied for the rewrite rule to be applicable (255 if no condition). Then three bytes with the sizes of head block, body block and tail block. Then a `uint8` indicating the number of elements in the output sequence. Then the output sequence glyph indices, as `uint16`s. The length of the body block must be at least equal to the number of characters in the output sequence. The length of the three blocks together can't exceed 255.
- After that, we get the definitions of the three blocks: the head block (can be empty), the body block (can't be empty) and the tail block (can be empty). For each one, we have zero (if empty) to many fragments. In the first byte of a fragment, 0bXXXX_0000 defines the number of consecutive character sets, which constitute the first part of the fragment data, and 0b0000_XXXX defines the number of subsequent direct glyphs. Both can be combined if sets and glyphs follow one after another. If not enough elements are defined (based on the block size), there will be another fragment of the same block afterwards. Even if the block is empty, a "zero fragment" must be present to make it easier to parse. The character sets are represented with single `uint8`s. NOTE: this representation might be changed in the future, it can be made considerably shorter for runes.
- For code points: same as glyph rules, but with `int32`s for code points instead of `uint16`s for glyph indices.

Rewrite rules can't exceed 4096 bytes for their definitions.

> The application order for UTF8 and glyph rewrite rules can lead to very different results. There would be three main approaches: (1) process UTF8 and then have a second pass for glyphs processing everything again, (2) process UTF8 and glyphs at the same time, feeding glyphs derived from the UTF8 to the glyph rewrite rules too, and (3) process UTF8 and glyphs at the same time, but each time we change sets we flush the previous process. Both (1) and (2) have the problem that we don't always know the exact position of the code point in the text when we have to map it to a glyph, because later glyph rewriting might move the glyph to an earlier position or completely replace it with something else. Approach (3), instead, has the problem of making glyph rewrite rules fundamentally useless when working with strings if utf8 rewrite rules that can collide with them also exist. All the approaches have advantages and disadvantages, but I recommend approach (2). Internally, (3) is the one with the most predictable behavior, but it's too limiting in many cases.

Implementers should compile rules into decision trees, search tables or FSMs to deal with rewrite rule application.

The conditions can be cached, but have to be marked as dirty any time a setting is modified, so they can be recomputed next time. Conditions must be allowed to change throughout the text.

Conditions are defined in a binary format. The first term is always a control code indicating what follows:
- 0b000X_XXXX: `OR` condition group. The X's indicate the number of terms in the expression (can't be < 2). The terms can be nested OR and AND groups.
- 0b001X_XXXX: `AND` condition group. The X's indicate the number of terms in the expression (can't be < 2). The terms can be nested OR and AND groups.
- 0b010X_YYYY: comparison. If X is 0, we compare two settings. If X is 1, we compare a setting value with a constant. YYYY indicates the operator (000 is `==`, 001 is `!=`, 010 is `<`, 011 is `>`, 100 is `<=`, `101` is `>=`). Followed by two bytes with the settings/constants afterwards.
- 0b011X_XXXX: quick comparison operator `setting == const`. Constant is encoded in X's. Followed by one byte with the setting index.
- 0b100X_XXXX: quick comparison `setting != const`. Constant is encoded in X's.
- 0b101X_XXXX: quick comparison `setting < const`. Constant is encoded in X's.
- 0b110X_XXXX: quick comparison `setting > const`. Constant is encoded in X's.
- 0b111X_XXXX: undefined. Maybe consider adding `bool expr cmp bool expr`?

> INTERNAL NOTE: rewrite rules create many more practical problems than I anticipated. Rule heads and tails can overlap and force you to set somewhat arbitrary policies, create complex and unintuitive cases that are user-unfriendly, etc (the glyph that you initially see could possibly change later). Conditions are also problematic when we have to do rollbacks and re-evaluate them (or force us to evaluate them all in advance _just in case_, or develop complex policies for lazy/delayed evaluations). All in all, it gets far too complex for comfort, maintenance and ease of access to the source code.

### Kernings

```Golang
NumHorzKerningPairs uint24
HorzKerningPairs [NumHorzKerningPairs]uint32 // for binary search (the uint32 is uint16|uint16 glyph indices)
HorzKerningValues [NumHorzKerningPairs]int8
NumVertKerningPairs uint24 // must be 0 if HasVertLayout is false
VertKerningPairs [NumVertKerningPairs]uint32
VertKerningValues [NumVertKerningPairs]int8
```

Kerning encoding is simplistic and relies on binary searches. Kerning classes are supported, but only on the editor, through a separate file explained in the next section.

### End

Since data is gzipped, we expect the EOF here, which will also verify the checksum.

### Edition data

Edition data is stored in a separate file, the `.ggwkfnt` file. Preferently, the file name should be shared with the main font file so we can get the two easily when loading the files, but it's not strictly required. The data is gzipped right after the signature.

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

NumHorzKerningPairsWithClasses uint24
HorzKerningPairs [NumHorzKerningPairsWithClasses](first, second uint16, class uint16)

NumVertKerningPairsWithClasses uint24
VertKerningPairs [NumVertKerningPairsWithClasses](first, second uint16, class uint16)

ConditionNames short[]shortString // must conform to basic-name-regex (+ possible spaces)
```

### Unsolved issues

- Should I explicitly say in the spec that mapping different code points to the same offset is valid? Because I can imagine people still wanting to map different code points to the same thing. And it might even be reasonable in some cases.
- End offsets are not explicitly explained anywhere, and it's not clear if the end offset is inclusive or exclusive. They must be exclusive, but I'm not sure I did it right everywhere, or that there aren't range issues somewhere.
