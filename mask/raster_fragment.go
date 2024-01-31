package mask

import "image"

type diagType uint8
const (
	diagNone diagType = 0
	diagAsc  diagType = 1
	diagDesc diagType = 2
)

type rasterFragment struct {
	minX int // inclusive
	minY int // inclusive
	maxX int // inclusive
	maxY int // inclusive
	value uint8 // palette index
	diagonalType diagType // diagNone, diagAsc, diagDesc
}

func (self *rasterFragment) ClearFrom(img *image.Alpha) {
	switch self.diagonalType {
	case diagNone:
		for y := (self.minY - img.Rect.Min.Y); y <= (self.maxY - img.Rect.Min.Y); y++ {
			index := y*img.Stride + self.minX
			for x := (self.minX - img.Rect.Min.X); x <= (self.maxX - img.Rect.Min.X); x++ {
				img.Pix[y*img.Stride + x] = 0
				index += 1
			}
		}
	case diagAsc:
		index := (self.minY - img.Rect.Min.Y)*img.Stride + (self.maxX - img.Rect.Min.X)
		steps := self.maxX + 1 - self.minX
		for i := 0; i < steps; i++ {
			img.Pix[index] = 0
			index += img.Stride - 1
		}
	case diagDesc:
		index := (self.minY - img.Rect.Min.Y)*img.Stride + (self.minX - img.Rect.Min.X)
		steps := self.maxX + 1 - self.minX 
		for i := 0; i < steps; i++ {
			img.Pix[index] = 0
			index += img.Stride + 1
		}
	default:
		panic("invalid rasterFragment.diagonalType value")
	}
}

func (self *rasterFragment) GetDistsFrom(x, y int) (int, int) {
	return self.minX - x, self.minY - y
}

func (self *rasterFragment) GetDrawSizes() (int, int) {
	return (self.maxX + 1) - self.minX, (self.maxY + 1) - self.minY
}
