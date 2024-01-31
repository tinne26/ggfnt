package mask

import "sort"
import "image"

type Encoder struct {
	fragments []rasterFragment
}

func (self *Encoder) AppendRasterOps(data []byte, mask *image.Alpha) []byte {
	// reset self memory
	self.fragments = self.fragments[ : 0]

	// compute mask fragments
	self.computeMaskFragments(mask)

	// empty mask case
	if len(self.fragments) == 0 { return data }

	// sort by palette index and (y, x)
	sort.Slice(self.fragments, func(i, j int) bool {
		if self.fragments[i].value > self.fragments[j].value { return true  } // higher value goes first
		if self.fragments[i].value < self.fragments[j].value { return false }

		// equal value, sort by coordinates (from top-left to bottom-right)
		if self.fragments[i].minY < self.fragments[j].minY { return true  }
		if self.fragments[i].minY > self.fragments[j].minY { return false }
		return self.fragments[i].minX <= self.fragments[j].minX
	})

	x, y := 0, 0
	value := uint8(255) // default palette index value
	var payload [8]byte
	for i := 0; i < len(self.fragments); i++ {
		// reset payload control code and payload size
		var payloadSize int = 1
		payload[0] = 0

		// palette index value change case
		if self.fragments[i].value != value {
			newValue := self.fragments[i].value
			if newValue == 0 { panic("broken code") }
			payload[0] |= 0b0000_0001
			payload[payloadSize] = newValue
			payloadSize += 1
			value = newValue
		}

		// encode pre-moves to reach the target
		xDist, yDist := self.fragments[i].GetDistsFrom(x, y)
		if xDist != 0 {
			if xDist < -128 || xDist > 128 { panic("distances > 128 unimplemented") } // not too hard to do
			payload[0] |= 0b0000_0010
			if xDist > 0 {
				payload[payloadSize] = uint8(int8(xDist - 1)) // nzint8 conversion
			} else {
				payload[payloadSize] = uint8(int8(xDist))
			}			
			payloadSize += 1
		}
		if yDist != 0 {
			if yDist == 1 {
				payload[0] |= 0b0000_1000
			} else {
				payload[0] |= 0b0000_0100
				if yDist < -128 || yDist > 128 { panic("distances > 128 unimplemented") } // not too hard to do
				if yDist > 0 {
					payload[payloadSize] = uint8(int8(yDist - 1)) // nzint8 conversion
				} else {
					payload[payloadSize] = uint8(int8(yDist))
				}
				payloadSize += 1
			}
		}

		// encode draw sizes
		xDraw, yDraw := self.fragments[i].GetDrawSizes()
		if (xDraw == 1) && (yDraw == 1) { // single pixel draw shortcut case
			payload[0] |= 0b1000_0000
		} else if self.fragments[i].diagonalType != diagNone { // diagonal case
			payload[0] |= 0b0001_0000 // diagonal marker
			if xDraw > 0 {
				payload[0] |= 0b0010_0000
				if xDraw > 256 { panic("draw distances > 256 unimplemented") } // not too hard to do
				payload[payloadSize] = uint8(xDraw - 1)
				payloadSize += 1
			}

			if self.fragments[i].diagonalType == diagAsc {
				payload[0] |= 0b0100_0000 // ascending diagonal flag
			}
		} else { // general rect case
			if xDraw > 0 && (xDraw > 1 || yDraw <= 1) {
				payload[0] |= 0b0010_0000
				if yDraw == 1 { yDraw = 0 }
				if xDraw > 256 { panic("draw distances > 256 unimplemented") } // not too hard to do
				payload[payloadSize] = uint8(xDraw - 1)
				payloadSize += 1
			}

			if yDraw > 0 {
				payload[0] |= 0b0100_0000
				if yDraw > 256 { panic("draw distances > 256 unimplemented") } // not too hard to do
				payload[payloadSize] = uint8(yDraw - 1)
				payloadSize += 1
			}
		}

		// advance position
		x, y = x + xDist, y + yDist
		x += xDraw // x is automatically advanced for xDraw

		// append command to data
		data = append(data, payload[ : payloadSize]...)
	}

	return data
}

func (self *Encoder) computeMaskFragments(mask *image.Alpha) {
	maskCopy := image.NewAlpha(mask.Rect)
	copy(maskCopy.Pix, mask.Pix)
	if mask.Stride != maskCopy.Stride {
		panic("mask.Stride != maskCopy.Stride")
	}

	// find isolated pixels and diagonals (only down-left/down-right)
	var index int
	var x, y int = 0, maskCopy.Rect.Min.Y
	var minX, minY, maxX, maxY int = 9999, 9999, 0, 0
	for index < len(maskCopy.Pix) {
		for x := maskCopy.Rect.Min.X; x < maskCopy.Rect.Max.X; x++ {
			value := maskCopy.Pix[index]
			if value != 0 {
				n := countNeighbours(maskCopy, x, y, index, value)
				switch n {
				case 0:
					diag, found := findDiagonal(maskCopy, x, y, index, value)
					if found {
						diag.ClearFrom(maskCopy)
						self.fragments = append(self.fragments, diag)
					} else {
						point := rasterFragment{ minX: x, minY: y, maxX: x, maxY: y, value: value }
						point.ClearFrom(maskCopy)
						self.fragments = append(self.fragments, point)
					}
				case 1:
					line := findLine(maskCopy, x, y, index, value)
					line.ClearFrom(maskCopy)
					self.fragments = append(self.fragments, line)
				case 2, 3: // register for later
					if y  < minY { minX, minY = x, y } // since we go LTR, x is already min if y == minY
					if y >= maxY { maxX, maxY = x, y } // since we go LTR, current x is max if y == maxY
				default:
					panic("unexpected case") // 4 can't happen if we are cleaning up properly
				}
			}
			index += 1
		}
		y += 1
	}

	// find remaining rects
	index = (minY - maskCopy.Rect.Min.Y)*maskCopy.Stride + (minX - maskCopy.Rect.Min.X)
	endIndex := (maxY - maskCopy.Rect.Min.Y)*maskCopy.Stride + (maxX - maskCopy.Rect.Min.X)
	x, y = minX, minY
	for index < endIndex {
		for x < maskCopy.Rect.Max.X {
			value := maskCopy.Pix[index]
			if value != 0 {
				if countNeighbours(maskCopy, x, y, index, value) == 0 {
					point := rasterFragment{ minX: x, minY: y, maxX: x, maxY: y, value: value }
					point.ClearFrom(maskCopy)
					self.fragments = append(self.fragments, point)
				} else {
					rect := findRect(maskCopy, x, y, value)
					rect.ClearFrom(maskCopy)
					self.fragments = append(self.fragments, rect)	
				}
			}
			x += 1
			index += 1
		}
	}
}
