package mask

import "image"
import "image/color"
import "testing"

func TestCountNeighbours(t *testing.T) {
	// test #1
	mask := image.NewAlpha(image.Rect(-2, -2, 2, 2))
	mask.SetAlpha(0, 0, color.Alpha{200})
	
	ncount := countNeighboursAutoIndex(mask, 0, 0, 200)
	expected := 0
	if ncount != expected {
		t.Fatalf("expected %d neighbours, got %d", expected, ncount)
	}

	// test #2
	mask.SetAlpha(1, 1, color.Alpha{200})
	ncount = countNeighboursAutoIndex(mask, 0, 0, 200)
	expected = 0
	if ncount != expected {
		t.Fatalf("expected %d neighbours, got %d", expected, ncount)
	}

	// test #3
	mask.SetAlpha(1, 0, color.Alpha{200})
	ncount = countNeighboursAutoIndex(mask, 1, 0, 200)
	expected = 2
	if ncount != expected {
		t.Fatalf("expected %d neighbours, got %d", expected, ncount)
	}

	// test #4
	ncount = countNeighboursAutoIndex(mask, 1, -1, 200)
	expected = 1
	if ncount != expected {
		t.Fatalf("expected %d neighbours, got %d", expected, ncount)
	}

	// test #5
	ncount = countNeighboursAutoIndex(mask, 1, 0, 199)
	expected = 0
	if ncount != expected {
		t.Fatalf("expected %d neighbours, got %d", expected, ncount)
	}

	// test #6
	ncount = countNeighboursAutoIndex(mask, 0, 0, 255)
	expected = 0
	if ncount != expected {
		t.Fatalf("expected %d neighbours, got %d", expected, ncount)
	}

	// test #7
	mask.SetAlpha(1, -1, color.Alpha{200})
	ncount = countNeighboursAutoIndex(mask, 1, 0, 200)
	expected = 3
	if ncount != expected {
		t.Fatalf("expected %d neighbours, got %d", expected, ncount)
	}
}

func TestFindLine(t *testing.T) {
	// test #1
	mask := image.NewAlpha(image.Rect(-2, -2, 2, 2))
	mask.SetAlpha(-2, -1, color.Alpha{1})
	mask.SetAlpha(-1, -1, color.Alpha{1})
	mask.SetAlpha( 0, -1, color.Alpha{1})
	mask.SetAlpha(-1, -2, color.Alpha{1})
	mask.SetAlpha(-1,  0, color.Alpha{1})
	
	fragment1 := findLineAutoIndex(mask, -2, -1, 1) // ltr expansion
	expected := rasterFragment{ minX: -2, minY: -1, maxX: 0, maxY: -1, value: 1 }
	if fragment1 != expected {
		t.Fatalf("expected %v fragment, got %v", expected, fragment1)
	}

	// test #2
	fragment2 := findLineAutoIndex(mask, -1, -2, 1) // ttb expansion
	expected = rasterFragment{ minX: -1, minY: -2, maxX: -1, maxY: 0, value: 1 }
	if fragment2 != expected {
		t.Fatalf("expected %v fragment, got %v", expected, fragment2)
	}

	// test #3 (clears)
	fragment1.ClearFrom(mask)
	fragment2.ClearFrom(mask)
	if !ComputeRect(mask).Empty() {
		t.Fatalf("expected fragment clears to work, but rect is not empty %s", ComputeRect(mask))
	}

	// test #4 (q shape)
	mask.SetAlpha(-2, -2, color.Alpha{1})
	mask.SetAlpha(-1, -2, color.Alpha{1})
	mask.SetAlpha(-2, -1, color.Alpha{1})
	mask.SetAlpha(-1, -1, color.Alpha{1})
	mask.SetAlpha(-1,  0, color.Alpha{1})
	fragment := findLineAutoIndex(mask, -1, 0, 1)
	expected  = rasterFragment{ minX: -1, minY: -2, maxX: -1, maxY: 0, value: 1 }
	if fragment != expected {
		t.Fatalf("expected %v fragment, got %v", expected, fragment)
	}
}

func TestFindRect(t *testing.T) {
	// test #1
	mask := image.NewAlpha(image.Rect(-2, -2, 2, 2))
	mask.SetAlpha(-1, -1, color.Alpha{77})
	
	fragment := findRect(mask, -1, -1, 77)
	expected := rasterFragment{ minX: -1, minY: -1, maxX: -1, maxY: -1, value: 77 }
	if fragment != expected {
		t.Fatalf("expected %v fragment, got %v", expected, fragment)
	}

	// test #2
	mask.SetAlpha(0, -1, color.Alpha{77})
	fragment = findRect(mask, -1, -1, 77)
	expected = rasterFragment{ minX: -1, minY: -1, maxX: 0, maxY: -1, value: 77 }
	if fragment != expected {
		t.Fatalf("expected %v fragment, got %v", expected, fragment)
	}

	// test #3 (test that we expand down before right)
	mask.SetAlpha(-1, 0, color.Alpha{77})
	fragment = findRect(mask, -1, -1, 77)
	expected = rasterFragment{ minX: -1, minY: -1, maxX: -1, maxY: 0, value: 77 }
	if fragment != expected {
		t.Fatalf("expected %v fragment, got %v", expected, fragment)
	}

	// test #4 (test small rect expansion)
	mask.SetAlpha(0,  0, color.Alpha{77})
	fragment = findRect(mask, -1, -1, 77)
	expected = rasterFragment{ minX: -1, minY: -1, maxX: 0, maxY: 0, value: 77 }
	if fragment != expected {
		t.Fatalf("expected %v fragment, got %v", expected, fragment)
	}

	// test #4 (test larger rect expansion)
	mask.SetAlpha(1, -1, color.Alpha{77})
	mask.SetAlpha(1,  0, color.Alpha{77})
	fragment = findRect(mask, -1, -1, 77)
	expected = rasterFragment{ minX: -1, minY: -1, maxX: 1, maxY: 0, value: 77 }
	if fragment != expected {
		t.Fatalf("expected %v fragment, got %v", expected, fragment)
	}

	// test #5
	fragment.ClearFrom(mask)
	rect := ComputeRect(mask)
	if !rect.Empty() {
		t.Fatalf("expected fragment clears to work, but rect is not empty %s", rect)
	}
}

func TestFindDiag(t *testing.T) {
	// test #1
	mask := image.NewAlpha(image.Rect(-2, -2, 2, 2))
	mask.SetAlpha(-1, -1, color.Alpha{3})
	
	diag, found := findDiagonalAutoIndex(mask, -1, -1, 3)
	if found {
		t.Fatalf("expected no diagonal found, but got %v", diag)
	}

	// test #2
	mask.SetAlpha(0, 0, color.Alpha{3})
	diag, found = findDiagonalAutoIndex(mask, -1, -1, 3)
	expected := rasterFragment{ minX: -1, minY: -1, maxX: 0, maxY: 0, value: 3, diagonalType: diagDesc }
	if !found {
		t.Fatalf("expected diagonal %v, but found nothing", expected)
	}
	if diag != expected {
		t.Fatalf("expected %v diagonal, got %v", expected, diag)
	}

	// test #3
	diag.ClearFrom(mask)
	rect := ComputeRect(mask)
	if !rect.Empty() {
		t.Fatal("expected diagonal clear to work, but mask is not blank")
	}

	// test #4
	mask.SetAlpha(-1, 1, color.Alpha{3})
	mask.SetAlpha( 0, 0, color.Alpha{3})
	diag, found = findDiagonalAutoIndex(mask, 0, 0, 3)
	expected = rasterFragment{ minX: -1, minY: 0, maxX: 0, maxY: 1, value: 3, diagonalType: diagAsc }
	if !found {
		t.Fatalf("expected diagonal %v, but found nothing", expected)
	}
	if diag != expected {
		t.Fatalf("expected %v diagonal, got %v", expected, diag)
	}

	// test #5
	diag.ClearFrom(mask)
	rect = ComputeRect(mask)
	if !rect.Empty() {
		t.Fatal("expected diagonal clear to work, but mask is not blank")
	}
}
