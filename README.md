# ggfnt
[![Go Reference](https://pkg.go.dev/badge/tinne26/ggfnt.svg)](https://pkg.go.dev/github.com/tinne26/ggfnt)

A bitmap font format that no one asked for:
- Designed to be used primarily for indie game development.
- Can be used with [Ebitengine](https://github.com/hajimehoshi/ebiten) through [`tinne26/ptxt`](https://github.com/tinne26/ptxt).
- Some fonts with permissive licenses are available at [`tinne26/ggfnt-fonts`](https://github.com/tinne26/ggfnt-fonts).
- Font editor is available at [`tinne26/ggfnt-editor`](https://github.com/tinne26/ggfnt-editor) (TO BE DEVELOPED).

This is an opinionated format with many hard limits, a specific memory layout for the data and a fair amount of unconventional choices (when compared to more standard font formats). Compatibility with existing formats is a non-goal. Being "generally better" than existing formats is a non-goal. I do have some knowledge about text rendering and font formats, but I don't claim to be an authority nor having done extensive research. I simply built what I wanted to build based on my experience with [etxt](https://github.com/tinne26/etxt) and game development with Ebitengine.

## Status

The basics are functional, but many advanced features and checks and tests and everything are missing. Very WIP.

## Examples

You can see some fonts in action through [tinne26.github.io/ptxt-examples](https://tinne26.github.io/ptxt-examples).

## Specification

See the [specification document](https://github.com/tinne26/ggfnt/blob/main/specification.md).

Summary of features and technical properties:
- Single file spec under 400 lines (not simple, but accessible).
- Many hard limits to make life safer.
- Support for vertical text.
- Support for colored glyphs with up to 255 colors.
- Support for variables and conditional character mapping.
- Support for kerning and kerning classes during edition.
- Font data layout is directly usable once ungzipped (not many additional structures required).

In my opinion, though, the most important property is that ggfnt has been designed alongside the [ptxt](https://github.com/tinne26/ptxt) renderer, while also having experience from developing [etxt](https://github.com/tinne26/etxt). This results in a font format design that understands indie game usage needs and renderer implementation details. This leads to fairly good harmony between the format design and the tools that need to use it. At the same time, ggfnt doesn't shy away from adding more advanced features that might seem strange at first sight, like conditional character mapping and variables to control it. These add a non trivial amount of complexity to the format, but they also give a lot of power in the relevant context of game development (these features would be much more complex to shoehorn into the rendering tools separately).

Other points not directly related to spec features but worth mentioning:
- The fonts can be easily edited to add custom icons for your games, adding variables or adjusting mapping for custom effects and so on. This tends to be less relevant in other contexts, but in game development it's great to have the door open to hack your own things. And editing bitmaps (unlike vectorial fonts) is easy. You can even add custom glyphs at runtime.
- While the font format is not Golang-specific, all the related code is written in Golang, Golang-aware and Golang-friendly.
- The font object can be operated at a low-level for some advanced features without feeling like a hacker. For example, querying named font variables and playing with their values at runtime can be perfectly reasonable and should be accessible.
- I always put a lot of effort on documentation. The current state is still WIP and very spammy and possibly unclear in some cases, but I will get there.

## Differences with other popular formats

- `ttf` and `otf`: can't be compared at all as these are vectorial font formats. But they are very inaccessible monsters in comparison.
- `bmfont`: ggfnt is fairly more complex, but at the same time the binary format has explicit safety limits and tends to use smaller data types. With the extra complexity, ggfnt gains more detailed metrics, support for vertical text, optional names and labels for some glyphs and other elements of the font, conditional character mapping, variables, etc. During edition, ggfnt also supports kerning classes and category names. On the negative side, ggfnt doesn't care about compatibility or convertibility with other formats at all, and as already said, it's more complex.
- `PCF`: maybe the most similar format to ggfnt. Almost all the features of PCF are also provided by ggfnt, but ggfnt allows quite a few more things. PCF is a bit quirky with compression, ggfnt is quirky with data layout and binary-searchability. If you wanted to create a bitmap rendering library for an already existing format, then PCF and bmfont would probably be your main contenders.
- `BDF`: this thing includes the following quote on its spec: "The Adobe Systems glyph bitmaps are typically distributed on magnetic tape". Not to be ageist, the format is actually ok, but it's very limited and verbose... and practices, style and expectations have changed a lot since 1987.

## Transparency

While I'm pretty happy with ggfnt's features and general direction, I feel like it would be dishonest if I didn't also talk about the less shiny parts. In particular, the current implementation is rather hacky and far less modular and clean than it could be... but the spec is also quite quirky and could define the different sections in much more consistent and homogeneous ways, making implementation easier at the same time. The safety of the format is also quite hand-wavy, with me saying "yeah well we have size limits" as if that would hold as a formal proof of anything. And then we also have some features like rewrite rules that turned out to be way harder to implement in practice than I ever intended. At the end of the day, this is a rather rogue attempt at creating a font format, and it's quite unprofessional, quirky, overly exploratory and hacky in many ways.

In other words, while I really enjoy ggfnt's vision, a top-tier execution would require way more time, effort and discussion with knowledgeable individuals in order to reach consensus, polish the rough edges and iron out the quirky bits. It's not like I tried to make ggfnt anything beyond a font format that could be used for indie game development with Ebitengine, but yeah... let's remind everyone that that's all it is —if it ever looked any prettier—.
