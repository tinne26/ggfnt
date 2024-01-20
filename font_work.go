package ggfnt

// An editable version of [Font]. It allows modifying and exporting
// ggfnt fonts. It can also store and edit glyph category names, kerning
// classes and a few other elements not present in regular .ggfnt files.
// See .ggwkfnt in the spec document for more details.
//
// This object should never replace a [Font] outside the edition context,
// as memory usage and performance can't be optimized in the same way when
// the font needs to be easy to modify.
type WorkFont struct {
	// ...
}

// TODO: the pain is that we need basically all the getter methods we
// already had in font + all the setter methods we didn't have there.
