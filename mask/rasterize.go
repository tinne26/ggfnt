package mask

import "image"
import "math"

// Given a set of raster operations in binary format, returns
// the corresponding glyph mask.
func Rasterize(rasterOps []byte) (*image.Alpha, error) {
	panic("unimplemented")
}

// The returned bool is true if the mask is empty.
func computeRasterOpsRect(ops []byte) (image.Rectangle, bool) {
	rect := image.Rect(math.MaxInt, math.MaxInt, math.MinInt, math.MinInt)
	_ = rect
	panic("unimplemented")
}
