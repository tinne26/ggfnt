package mask

import "testing"
import "image"
import "image/color"
import "math/rand"

// Notice: these tests rely on encoder tests passing too.
// Otherwise, the results can be anything, gibberish.

func TestComputeRasterOpsRect(t *testing.T) {
	var encoder Encoder

	// -5  X X
	// -4  X X
	// -3  XXX
	// -2    X
	// -1    X
	mask := image.NewAlpha(image.Rect(0, -5, 3, 0))
	mask.SetAlpha(0, -5, color.Alpha{255})
	mask.SetAlpha(2, -5, color.Alpha{255})
	mask.SetAlpha(0, -4, color.Alpha{255})
	mask.SetAlpha(2, -4, color.Alpha{255})
	mask.SetAlpha(0, -3, color.Alpha{255})
	mask.SetAlpha(1, -3, color.Alpha{255})
	mask.SetAlpha(2, -3, color.Alpha{255})
	mask.SetAlpha(2, -2, color.Alpha{255})
	mask.SetAlpha(2, -1, color.Alpha{255})
	data := encoder.AppendRasterOps(nil, mask)

	rect, err := computeRasterOpsRect(data)
	if err != nil { t.Fatal(err) }
	expRect := image.Rect(0, -5, 3, 0)
	if !rect.Eq(expRect) {
		t.Fatalf("expected rect '%s', got '%s'", expRect, rect)
	}

	// q test
	mask = image.NewAlpha(image.Rect(-6, -6, 6, 6))
	mask.SetAlpha(-2, -2, color.Alpha{1})
	mask.SetAlpha(-1, -2, color.Alpha{1})
	mask.SetAlpha(-2, -1, color.Alpha{1})
	mask.SetAlpha(-1, -1, color.Alpha{1})
	mask.SetAlpha(-1,  0, color.Alpha{1})
	data = data[ : 0]
	data = encoder.AppendRasterOps(data, mask)

	rect, err = computeRasterOpsRect(data)
	if err != nil { t.Fatal(err) }
	expRect = image.Rect(-2, -2, 0, 1)
	if !rect.Eq(expRect) {
		t.Fatalf("expected rect '%s', got '%s'", expRect, rect)
	}
}

func TestRngRasterize(t *testing.T) {
	var encoder Encoder
	rasterOps := make([]uint8, 0, 256)
	img := image.NewAlpha(image.Rect(-2, -2, 3, 3))

	// run test N times with random values
	for i := 0; i < 222; i++ {
		// generate mask
		for j := 0; j < len(img.Pix); j++ {
			if rand.Float64() < 0.3 { img.Pix[j] = 255 }
		}

		// encode to raster commands
		rasterOps = rasterOps[ : 0]
		rasterOps := encoder.AppendRasterOps(rasterOps, img)

		// rasterize mask
		mask, err := Rasterize(rasterOps)
		if err != nil { t.Fatal(err) }

		// compare masks
		if mask.Rect.Min.Y < img.Rect.Min.Y || mask.Rect.Max.Y > img.Rect.Max.Y ||
		   mask.Rect.Min.X < img.Rect.Min.X || mask.Rect.Max.X > img.Rect.Max.X {
				t.Fatalf("expected mask: %v\nfound mask: %v\n(found mask can't be bigger than expected mask)\n", img, mask)
		}

		for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
			for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
				oob := (y < mask.Rect.Min.Y || y >= mask.Rect.Max.Y || x < mask.Rect.Min.X || x >= mask.Rect.Max.X)
				if (img.AlphaAt(x, y).A != 0 && oob) || (!oob && (img.AlphaAt(x, y).A != mask.AlphaAt(x, y).A)) {
					t.Fatalf("expected mask: %v\nfound mask: %v\n", img, mask)
				}
			}
		}
	}
}
