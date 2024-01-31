package mask

import "image"

func ComputeRect(mask *image.Alpha) image.Rectangle {
	minX := mask.Rect.Max.X + 1
	maxX := mask.Rect.Min.X - 1
	minY := mask.Rect.Max.Y + 1
	maxY := mask.Rect.Min.Y - 1

	empty := true
	for y := mask.Rect.Min.Y; y < mask.Rect.Max.Y; y++ {
		index := (y - mask.Rect.Min.Y)*mask.Stride 
		activeValueInRow := false
		for x := mask.Rect.Min.X; x < mask.Rect.Max.X; x++ {
			value := mask.Pix[index]
			if value != 0 {
				activeValueInRow = true
				if x < minX { minX = x }
				if x > maxX { maxX = x }
			}
			index += 1
		}

		if activeValueInRow {
			empty = false
			if y < minY { minY = y }
			if y > maxY { maxY = y }
		}
	}

	if empty { return image.Rectangle{} }
	return image.Rect(minX, minY, maxX + 1, maxY + 1)
}

