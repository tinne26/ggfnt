package mask

import "image"
import "image/color"
import "testing"

func TestComputeMask(t *testing.T) {
	// mask test #1
	mask := image.NewAlpha(image.Rect(-1, -1, 2, 2))
	mask.SetAlpha(-1, -1, color.Alpha{255})
	expected := image.Rect(-1, -1, 0, 0)

	rect := ComputeRect(mask)
	if !rect.Eq(expected) {
		t.Fatalf("expected rect %s, got %d", expected, rect)
	}

	// mask test #2
	mask.SetAlpha(1, 1, color.Alpha{255})
	expected = image.Rect(-1, -1, 2, 2)

	rect = ComputeRect(mask)
	if !rect.Eq(expected) {
		t.Fatalf("expected rect %s, got %d", expected, rect)
	}

	// mask test #3
	mask.SetAlpha(-1, -1, color.Alpha{0}) // clear
	mask.SetAlpha(1, 0, color.Alpha{255})
	expected = image.Rect(1, 0, 2, 2)

	rect = ComputeRect(mask)
	if !rect.Eq(expected) {
		t.Fatalf("expected rect %s, got %d", expected, rect)
	}
}
