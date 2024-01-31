package mask

import "testing"
import "fmt"
import "image"
import "image/color"
import "slices"
import "strings"

func fmtBinarySlice(data []byte) string {
	var builder strings.Builder
	builder.WriteRune('[')
	for i, value := range data {
		builder.WriteRune(' ')
		builder.WriteString(fmt.Sprintf("%08b", value))
		if i != len(data) - 1 { builder.WriteRune(',') }
	}
	builder.WriteRune(' ')
	builder.WriteRune(']')
	return builder.String()
}

func TestEncoder(t *testing.T) {
	var encoder Encoder
	
	// test #1
	mask := image.NewAlpha(image.Rect(0, -3, 4, 2))
	mask.SetAlpha(0, 0, color.Alpha{255})
	mask.SetAlpha(1, 0, color.Alpha{255})
	mask.SetAlpha(2, 0, color.Alpha{255})
	mask.SetAlpha(3, 0, color.Alpha{255})

	data := encoder.AppendRasterOps(nil, mask)
	expected := []byte{
		0b0010_0000, // draw horizontally
		0b0000_0011, // payload: draw horizontally 4 pixels (4 = 3 + 1 due to non-zero)
	}
	if !slices.Equal(data, expected) {
		t.Fatalf("expected mask encoder to return %s, got %s", fmtBinarySlice(expected), fmtBinarySlice(data))
	}

	// test #2
	mask = image.NewAlpha(image.Rect(0, -3, 4, 2))
	mask.SetAlpha(0, -1, color.Alpha{255})

	data = encoder.AppendRasterOps(nil, mask)
	expected = []byte{
		0b1000_0100, // pre move vert and draw single pixel
		0b1111_1111, // payload: pre move vert -1
	}
	if !slices.Equal(data, expected) {
		t.Fatalf("expected mask encoder to return %s, got %s", fmtBinarySlice(expected), fmtBinarySlice(data))
	}

	// test #3
	// -3
	// -2   X
	// -1  X X
	//  0 X   X X
	//  1      X
	mask = image.NewAlpha(image.Rect(0, -3, 7, 2))
	mask.SetAlpha(0,  0, color.Alpha{2})
	mask.SetAlpha(1, -1, color.Alpha{2})
	mask.SetAlpha(2, -2, color.Alpha{2})
	mask.SetAlpha(3, -1, color.Alpha{2})
	mask.SetAlpha(4,  0, color.Alpha{2})
	mask.SetAlpha(5,  1, color.Alpha{2})
	mask.SetAlpha(6,  0, color.Alpha{1})

	data = encoder.AppendRasterOps(nil, mask)
	expected = []byte{
		0b0011_0111, // set palette value, move to top and diagonal down
		0b0000_0010, // palette value
		0b0000_0001, // payload: pre move horz + 2
		0b1111_1110, // payload: pre move vert - 2
		0b0000_0011, // payload: diagonal length = 4

		0b0111_1010, // move horz -6, advance y by one, ascending diagonal
		0b1111_1010, // payload: pre horz move - 6
		0b0000_0001, // payload: diagonal length = 2

		0b1000_1011, // set palette value, move horz, single advance y, single pix
		0b0000_0001, // palette value
		0b0000_0011, // payload: pre move horz + 4 (due to auto-advance)
	}
	if !slices.Equal(data, expected) {
		t.Fatalf("expected mask encoder to return %s, got %s", fmtBinarySlice(expected), fmtBinarySlice(data))
	}

	// test #4
	// -5  X X
	// -4  X X
	// -3  XXX
	// -2    X
	// -1    X
	mask = image.NewAlpha(image.Rect(0, -5, 3, 0))
	mask.SetAlpha(0, -5, color.Alpha{255})
	mask.SetAlpha(2, -5, color.Alpha{255})
	mask.SetAlpha(0, -4, color.Alpha{255})
	mask.SetAlpha(2, -4, color.Alpha{255})
	mask.SetAlpha(0, -3, color.Alpha{255})
	mask.SetAlpha(1, -3, color.Alpha{255})
	mask.SetAlpha(2, -3, color.Alpha{255})
	mask.SetAlpha(2, -2, color.Alpha{255})
	mask.SetAlpha(2, -1, color.Alpha{255})
	data = encoder.AppendRasterOps(nil, mask)
	expected = []byte{
		0b0100_0100, // move to top and vertical line
		0b1111_1011, // payload: pre move vert - 5
		0b0000_0010, // payload: vert draw len = 3

		0b0100_0010, // move horz 1, vert line
		0b0000_0000, // payload: pre move horz + 1
		0b0000_0100, // payload: vert line len = 5

		0b1000_0110, // move horz - 2, move vert + 2, single pix
		0b1111_1110, // payload: pre move horz - 2
		0b0000_0001, // payload: pre move vert + 2
	}
	if !slices.Equal(data, expected) {
		t.Fatalf("expected mask encoder to return %s, got %s", fmtBinarySlice(expected), fmtBinarySlice(data))
	}
}
