package mask

import "image"
import "math"
import "errors"
import "image/color"

const (
	ctrlFlagChangePaletteIndex = 0b0000_0001 // uint8 value in the data
	ctrlFlagPreHorzMove        = 0b0000_0010 // nzint8 value in the data
	ctrlFlagPreVertMove        = 0b0000_0100 // nzint8 value in the data
	ctrlFlagPreVertRowAdvance  = 0b0000_1000 // incompatible with ctrlFlagPreVertMove
	ctrlFlagDiagonalMode       = 0b0001_0000 // changes meaning of next flags
	ctrlFlagDiagOffHorzDrawLen = 0b0010_0000 // nzuint8 in the data
	ctrlFlagDiagOnDiagLen      = 0b0010_0000 // nzuint8 in the data
	ctrlFlagDiagOffVertDrawLen = 0b0100_0000 // nzuint8 in the data
	ctrlFlagDiagOnAscending    = 0b0100_0000 // no data
	ctrlFlagSinglePixDraw      = 0b1000_0000 // draw single pixel if set. incompatible with DrawLen flags
)

var ErrUnexpectedRasterOptsEnd = errors.New("unexpected raster data termination")
var ErrIncompatibleRowAdvance = errors.New("raster operation can't use single row pre vertical advance while also having set a vertical pre move")
var ErrIncompatibleSinglePixDraw = errors.New("raster operation can't use single pixel draw while also having set other draw flags")
var ErrSuperfluousDiagonal = errors.New("raster operation sets the diagonal drawing mode flag but ends up not drawing anything")
var ErrDiagonalSinglePixDraw = errors.New("raster operation can't use single pixel draw on diagonal mode")

// Given a set of raster operations in binary format, returns
// the corresponding glyph mask. Empty masks return nil.
func Rasterize(rasterOps []byte) (*image.Alpha, error) {
	// obtain mask bounds
	rect, err := computeRasterOpsRect(rasterOps)
	if err != nil { return nil, err }
	if rect.Empty() { return nil, nil }
	
	// create mask
	mask := image.NewAlpha(rect)

	var index int = 0
	var paletteIndex uint8 = 255
	var x, y int = 0, 0
	for {
		ctrl := rasterOps[index]
		if ctrl & ctrlFlagChangePaletteIndex != 0 {
			index += 1
			if index >= len(rasterOps) { return mask, ErrUnexpectedRasterOptsEnd }
			paletteIndex = rasterOps[index]
		}

		if ctrl & ctrlFlagPreHorzMove != 0 {
			index += 1
			if index >= len(rasterOps) { return mask, ErrUnexpectedRasterOptsEnd }
			x += nzint8AsInt(rasterOps[index])
		}

		if ctrl & ctrlFlagPreVertMove != 0 {
			index += 1
			if index >= len(rasterOps) { return mask, ErrUnexpectedRasterOptsEnd }
			y += nzint8AsInt(rasterOps[index])
		}

		if ctrl & ctrlFlagPreVertRowAdvance != 0 {
			if ctrl & ctrlFlagPreVertMove != 0 { return mask, ErrIncompatibleRowAdvance }
			y += 1
		}

		if ctrl & ctrlFlagDiagonalMode != 0 {
			// diagonal mode
			if ctrl & ctrlFlagSinglePixDraw != 0 {
				return mask, ErrDiagonalSinglePixDraw
			}
			if ctrl & ctrlFlagDiagOnDiagLen == 0 {
				return mask, ErrSuperfluousDiagonal
			}

			index += 1
			if index >= len(rasterOps) { return mask, ErrUnexpectedRasterOptsEnd }
			diagonalLen := int(rasterOps[index]) + 1

			// draw diagonal
			if ctrl & ctrlFlagDiagOnAscending != 0 {
				// ascending diagonal
				y += diagonalLen
				for i := 0; i < diagonalLen; i++ {
					y -= 1
					mask.SetAlpha(x, y, color.Alpha{paletteIndex})
					x += 1
				}
			} else {
				// descending diagonal
				for i := 0; i < diagonalLen; i++ {
					mask.SetAlpha(x, y, color.Alpha{paletteIndex})
					x += 1
					y += 1
				}
				y -= diagonalLen
			}
		} else {
			// rect mode
			var draws bool = false
			var xInc, yInc int
			if ctrl & ctrlFlagDiagOffHorzDrawLen != 0 {
				index += 1
				if index >= len(rasterOps) { return mask, ErrUnexpectedRasterOptsEnd }
				xInc = int(rasterOps[index]) + 1
				yInc = 1
				draws = true
			}

			if ctrl & ctrlFlagDiagOffVertDrawLen != 0 {
				index += 1
				if index >= len(rasterOps) { return mask, ErrUnexpectedRasterOptsEnd }
				if !draws { xInc = 1 }
				yInc = int(rasterOps[index]) + 1 // TODO: is this +1 or +2? hmmm...
				draws = true
			}

			if ctrl & ctrlFlagSinglePixDraw != 0 {
				if draws { return mask, ErrIncompatibleSinglePixDraw }
				xInc, yInc = 1, 1
				draws = true
			}

			if draws {
				for j := 0; j < yInc; j++ {
					for i := 0; i < xInc; i++ {
						mask.SetAlpha(x + i, y + j, color.Alpha{paletteIndex})
					}
				}
				x += xInc
			}
		}

		// check if done
		index += 1
		if index == len(rasterOps) {
			return mask, nil
		}
	}

	panic("unreachable")
}

// You should generally check if the rect is empty afterwards.
// It can be in many cases.
func computeRasterOpsRect(rasterOps []byte) (image.Rectangle, error) {
	rect := image.Rectangle{ image.Pt(math.MaxInt, math.MaxInt), image.Pt(math.MinInt, math.MinInt) }
	if len(rasterOps) == 0 { return rect, nil }

	var index int = 0
	var x, y int = 0, 0
	for {
		ctrl := rasterOps[index]
		if ctrl & ctrlFlagChangePaletteIndex != 0 {
			index += 1
			if index >= len(rasterOps) { return rect, ErrUnexpectedRasterOptsEnd }
			// actual new palette value can be ignored here
		}

		if ctrl & ctrlFlagPreHorzMove != 0 {
			index += 1
			if index >= len(rasterOps) { return rect, ErrUnexpectedRasterOptsEnd }
			x += nzint8AsInt(rasterOps[index])
		}

		if ctrl & ctrlFlagPreVertMove != 0 {
			index += 1
			if index >= len(rasterOps) { return rect, ErrUnexpectedRasterOptsEnd }
			y += nzint8AsInt(rasterOps[index])
		}

		if ctrl & ctrlFlagPreVertRowAdvance != 0 {
			if ctrl & ctrlFlagPreVertMove != 0 { return rect, ErrIncompatibleRowAdvance }
			y += 1
		}

		if ctrl & ctrlFlagDiagonalMode != 0 {
			// diagonal mode
			if ctrl & ctrlFlagSinglePixDraw != 0 {
				return rect, ErrDiagonalSinglePixDraw
			}
			if ctrl & ctrlFlagDiagOnDiagLen == 0 {
				return rect, ErrSuperfluousDiagonal
			}

			// we can ignore the diagonal direction because it doesn't change
			// the boundaries nor adds extra data on the raster command	
			index += 1
			if index >= len(rasterOps) { return rect, ErrUnexpectedRasterOptsEnd }
			diagonalLen := int(rasterOps[index]) + 1

			// update rect bounds
			if x < rect.Min.X { rect.Min.X = x }
			if y < rect.Min.Y { rect.Min.Y = y }
			x += diagonalLen
			if x > rect.Max.X { rect.Max.X = x }
			if y + diagonalLen > rect.Max.Y { rect.Max.Y = y + diagonalLen }
		} else {
			// rect mode
			var draws bool = false
			var xInc, yInc int
			if ctrl & ctrlFlagDiagOffHorzDrawLen != 0 {
				index += 1
				if index >= len(rasterOps) { return rect, ErrUnexpectedRasterOptsEnd }
				xInc = int(rasterOps[index]) + 1
				yInc = 1
				draws = true
			}

			if ctrl & ctrlFlagDiagOffVertDrawLen != 0 {
				index += 1
				if index >= len(rasterOps) { return rect, ErrUnexpectedRasterOptsEnd }
				if !draws { xInc = 1 }
				yInc = int(rasterOps[index]) + 1 // TODO: is this +1 or +2? hmmm...
				draws = true
			}

			if ctrl & ctrlFlagSinglePixDraw != 0 {
				if draws { return rect, ErrIncompatibleSinglePixDraw }
				xInc, yInc = 1, 1
				draws = true
			}

			if draws {
				// update rect bounds
				if x < rect.Min.X { rect.Min.X = x }
				if y < rect.Min.Y { rect.Min.Y = y }
				x += xInc
				if x > rect.Max.X { rect.Max.X = x }
				if y + yInc > rect.Max.Y { rect.Max.Y = y + yInc }
			}
		}

		// check if done
		index += 1
		if index == len(rasterOps) {
			return rect, nil
		}
	}

	panic("unreachable")
}

// --- helper methods ---

func nzint8AsInt(value uint8) int {
	if value >= 128 {
		return int(int8(value))
	} else {
		return int(value) + 1
	}
}
