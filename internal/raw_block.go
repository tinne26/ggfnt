package internal

// Internal base type for [ggfnt.GlyphRewriteRule], [ggfnt.Utf8RewriteRule],
// [ggfnt.GlyphRewriteSet], [ggfnt.Utf8RewriteSet], etc.
type RawBlock struct { Data []uint8 }
