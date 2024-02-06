package ggfnt

// Dyes are part of the font coloring. See [FontColor] for
// more details.
//
// Dye keys go from 0 to [FontColor.NumDyes]() - 1, so they can be
// used directly for slice indexing if necessary.
type DyeKey uint8

// Palettes are part of the font coloring. See [FontColor] for
// more details.
//
// Palette keys go from 0 to [FontColor.NumPalettes]() - 1, so they can
// be used directly for slice indexing if necessary.
type PaletteKey uint8
