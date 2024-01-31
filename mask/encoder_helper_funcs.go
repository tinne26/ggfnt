package mask

import "image"

const applySafetyChecks = false
const safetyCheckViolation = "safety check violation"
const unexpected = "unexpected"

func getPixelIndex(img *image.Alpha, x, y int) int {
	return (y - img.Rect.Min.Y)*img.Stride + (x - img.Rect.Min.X)
}

func countNeighboursAutoIndex(img *image.Alpha, x, y int, value uint8) int {
	return countNeighbours(img, x, y, getPixelIndex(img, x, y), value)
}

func countNeighbours(img *image.Alpha, x, y int, index int, value uint8) int {
	if applySafetyChecks {
		if index != getPixelIndex(img, x, y) { panic(safetyCheckViolation) }
	}

	var count int
	if y > img.Rect.Min.Y && img.Pix[index - img.Stride] == value { count += 1 } // pixel above
	if y + 1 < img.Rect.Max.Y && img.Pix[index + img.Stride] == value { count += 1 } // pixel below
	if x > img.Rect.Min.X && img.Pix[index - 1] == value { count += 1 } // pixel to the left
	if x + 1 < img.Rect.Max.X && img.Pix[index + 1] == value { count += 1 } // pixel to the right
	return count
}

func findLineAutoIndex(img *image.Alpha, x, y int, value uint8) rasterFragment {
	return findLine(img, x, y, getPixelIndex(img, x, y), value)
}

// precondition: this must only be used during a top-left to bottom-right pass,
// executed while clearing any previous lines found. among others, this means left
// lines can't happen. it must also be called only on indices where the neighbours
// count is exactly 1
func findLine(img *image.Alpha, x, y int, index int, value uint8) rasterFragment {
	if applySafetyChecks {
		if index != getPixelIndex(img, x, y) { panic(safetyCheckViolation) }
		if img.Pix[index] != value { panic(safetyCheckViolation) }
		if countNeighbours(img, x, y, index, value) != 1 { panic(safetyCheckViolation) }
	}

	// line can't go left or it would violate preconds
	if x > img.Rect.Min.X && img.Pix[index - 1] == value { panic(unexpected) } // line going left

	rect := rasterFragment{ minX: x, minY: y, maxX: x, maxY: y, value: value }
	if y > img.Rect.Min.Y && img.Pix[index - img.Stride] == value { // line going up
		if y + 1 < img.Rect.Max.Y && img.Pix[index + img.Stride] == value { panic(unexpected) } // line also going down?
		if x + 1 < img.Rect.Max.X && img.Pix[index + 1] == value { panic(unexpected) } // line also going right?
		
		// (this can happen with a p shape, for example)
		rect.minY -= 1
		index -= img.Stride*2
		for rect.minY > img.Rect.Min.Y && img.Pix[index] == value {
			rect.minY -= 1
			index -= img.Stride
		}
		return rect
	} 

	if y + 1 < img.Rect.Max.Y && img.Pix[index + img.Stride] == value { // line going down
		if x + 1 < img.Rect.Max.X && img.Pix[index + 1] == value { panic(unexpected) } // line also going right?
		rect.maxY += 1
		index += img.Stride*2
		for rect.maxY + 1 < img.Rect.Max.Y && img.Pix[index] == value {
			rect.maxY += 1
			index += img.Stride
		}
		return rect
	}
	if x + 1 < img.Rect.Max.X && img.Pix[index + 1] == value { // line going right
		rect.maxX += 1
		index += 2
		for rect.maxX + 1 < img.Rect.Max.X && img.Pix[index] == value {
			rect.maxX += 1
			index += 1
		}
		return rect
	} else {
		panic(unexpected) // no line to be found
	}
}

func findRect(img *image.Alpha, x, y int, value uint8) rasterFragment {
	if applySafetyChecks {
		if img.Pix[getPixelIndex(img, x, y)] != value { panic(safetyCheckViolation) }
	}
	
	rect := rasterFragment{ minX: x, minY: y, maxX: x, maxY: y, value: value }

	// alternate expanding down and right
	width := 1
	height := 1
	stoppedExpandingDown  := (rect.maxY + 1 >= img.Rect.Max.Y)
	stoppedExpandingRight := (rect.maxX + 1 >= img.Rect.Max.X)
	for !(stoppedExpandingDown && stoppedExpandingRight) {
		if !stoppedExpandingDown { // try expand down
			index := ((rect.maxY + 1) - img.Rect.Min.Y)*img.Stride + (rect.minX - img.Rect.Min.X)
			allPixelsExpandable := true
			for i := 0; i < width; i++ {
				if img.Pix[index] == value {
					index += 1
					continue
				}
				allPixelsExpandable = false
				break
			}
			if allPixelsExpandable {
				rect.maxY += 1
				height += 1
				stoppedExpandingDown = (rect.maxY + 1 >= img.Rect.Max.Y)
			} else {
				stoppedExpandingDown = true
			}
		}

		if !stoppedExpandingRight { // try expand right
			index := (rect.minY - img.Rect.Min.Y)*img.Stride + (rect.maxX - img.Rect.Min.X) + 1
			allPixelsExpandable := true
			for i := 0; i < height; i++ {
				if img.Pix[index] == value {
					index += img.Stride
					continue
				}
				allPixelsExpandable = false
				break
			}
			if allPixelsExpandable {
				rect.maxX += 1
				width += 1
				stoppedExpandingRight = (rect.maxX + 1 >= img.Rect.Max.X)
			} else {
				stoppedExpandingRight = true
			}
		}
	}
	
	return rect
}

func findDiagonalAutoIndex(img *image.Alpha, x, y int, value uint8) (rasterFragment, bool) {
	return findDiagonal(img, x, y, getPixelIndex(img, x, y), value)
}

func findDiagonal(img *image.Alpha, x, y int, index int, value uint8) (rasterFragment, bool) {
	if applySafetyChecks {
		if index != getPixelIndex(img, x, y) { panic(safetyCheckViolation) }
		if img.Pix[index] != value { panic(safetyCheckViolation) }
		if countNeighbours(img, x, y, index, value) != 0 { panic(safetyCheckViolation) }
	}

	var diag rasterFragment
	diag.value = value

	// check diagonal going bottom right
	maxSteps := min((img.Rect.Max.X - 1) - x, (img.Rect.Max.Y - 1) - y)
	if maxSteps > 0 {
		diagIndex := index + img.Stride + 1
		if img.Pix[diagIndex] == value && countNeighbours(img, x + 1, y + 1, diagIndex, value) == 0 {
			diag.minX, diag.minY = x, y
			diag.maxX, diag.maxY = diag.minX + 1, diag.minY + 1
			maxSteps -= 1
			for maxSteps > 0 {
				diagIndex += img.Stride + 1
				if img.Pix[diagIndex] != value { break }
				if countNeighbours(img, diag.maxX + 1, diag.maxY + 1, diagIndex, value) != 0 {
					break
				}
				diag.maxX += 1
				diag.maxY += 1
				maxSteps -= 1
			}
			diag.diagonalType = diagDesc
			return diag, true
		}
	}

	// check diagonal going bottom left
	maxSteps = min(x - img.Rect.Min.X, (img.Rect.Max.Y - 1) - y)
	if maxSteps > 0 {
		diagIndex := index + img.Stride - 1
		if img.Pix[diagIndex] == value  && countNeighbours(img, x - 1, y + 1, diagIndex, value) == 0 {
			diag.maxX, diag.minY = x, y
			diag.minX, diag.maxY = diag.maxX - 1, diag.minY + 1
			maxSteps -= 1
			for maxSteps > 0 {
				diagIndex += img.Stride - 1
				if img.Pix[diagIndex] != value { break }
				if countNeighbours(img, diag.minX - 1, diag.maxY + 1, diagIndex, value) != 0 {
					break
				}
				diag.minX -= 1
				diag.maxY += 1
				maxSteps -= 1
			}
			diag.diagonalType = diagAsc
			return diag, true
		}
	}

	return diag, false
}
