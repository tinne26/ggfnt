# ggfnt

A bitmap font format that no one asked for:
- Designed to be used primarily for indie game development.
- Can be used with [*Ebitengine*](github.com/hajimehoshi/ebiten) through *tinne26/ptxt* (TO BE DEVELOPED).
- Some fonts with permissive licenses are available at *tinne26/fonts/ggfnt/* (TO BE ADDED).
- Font editor is available at *tinne26/ggfnt-editor* (TO BE DEVELOPED).

This is an opinionated format with many hard limits, a specific memory layout for the data and a fair amount of unconventional choices (when compared to more standard font formats). Compatibility with existing formats is a non-goal. Being "generally better" than existing formats is a non-goal. I do have some knowledge about text rendering and font formats, but I don't claim to be an authority nor having done extensive research. I simply built what I wanted to build based on my experience with *etxt* and game development with Ebitengine.

TODO: add some examples from ptxt-examples.

# Specification

See the [specification document](https://github.com/tinne26/ggfnt/blob/main/specification.md).

Summary of features and technical properties:
- Single file spec under 400 lines (not easy, but easily accessible).
- Many hard limits to make life safer.
- Support for vertical text.
- Support for colored glyphs with up to 255 colors.
- Support for variables and conditional character mapping.
- Support for kerning and kerning classes during edition.
- Font data layout is directly usable once ungzipped (not many additional structures required, straightforward implementation).

In my opinion, though, the most important property is that ggfnt has been designed alongside the ptxt renderer, while also having experience from developing etxt. This results in a font format design that understands indie game usage needs and renderer implementation details. This leads to fairly good harmony between the format design and the tools that need to use it. At the same time, ggfnt doesn't shy away from adding some features that might seem strange at first sight, like conditional character mapping and variables to control it. These add a non trivial amount of complexity to the format, but they also give a lot of power in the relevant context of game development (and they would still be much more complex to shoehorn into the rendering tools separately).

Other points not directly related to spec features but worth mentioning:
- The fonts can be easily edited to add custom icons for your games, adding variables or adjusting mapping for custom effects and so on. This tends to be less relevant in other contexts, but in game development it's great to have the door open to hack your own things. And editing bitmaps (unlike vectorial fonts) is easy. You can even add custom glyphs at runtime.
- While the font format is not Golang-specific, all the related code is written in Golang, Golang-aware and Golang-friendly.
- The font object can be operated at a low-level for some advanced features without feeling like a hacker. For example, querying named font variables and playing with their values at runtime can be perfectly reasonable and should be pleasant to do.
- I always put a lot of effort on documentation. The current state is still WIP and very spammy and possibly unclear in some cases, but it will get better.

## Differences with other popular formats

- `ttf` and `otf`: can't be compared at all as these are vectorial font formats. But they are very inaccessible monsters in comparison.
- `bmfont`: ggfnt is fairly more complex, but at the same time the binary format has explicit safety limits and tends to use smaller data types. With the extra complexity, ggfnt gains more detailed metrics, support for vertical text, optional names and labels for some glyphs and other elements of the font, conditional character mapping, variables, etc. During edition, ggfnt also supports kerning classes and category names. On the negative side, ggfnt doesn't care about compatibility or convertibility with other formats at all, and as already said, it's more complex.
- `PCF`: maybe the most similar format to ggfnt. Almost all the features of PCF are also provided by ggfnt, but ggfnt allows quite a few more things. PCF is a bit quirky with compression, ggfnt is quirky with data layout and binary-searchability. If you wanted to create a bitmap rendering library for an already existing format, then PCF and bmfont would probably be your main contenders.
- `BDF`: this thing includes the following quote on its spec: "The Adobe Systems glyph bitmaps are typically distributed on magnetic tape". The format is actually ok, but it's very basic and verbose. Many things have changed since 1987.
