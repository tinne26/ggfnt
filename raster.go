package ggfnt

import "image"

// Appends the raster operations necessary to reproduce the given
// mask to the buffer.
func AppendMaskRasterOps(buffer []byte, mask *image.Alpha) ([]byte, error) {
	panic("unimplemented")
}

// Given a set of raster operations in binary format, returns
// the corresponding glyph mask.
// TODO: allow passing a reusable image? or alpha buffer?
func RasterizeMask(width, height uint8, rasterData []byte) (*image.Alpha, error) {
	panic("unimplemented")
}
