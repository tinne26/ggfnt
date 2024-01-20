package ggfnt

import "errors"
import "crypto/rand"

func growSliceByN[T any](buffer []T, increase int) []T {
	newSize := len(buffer) + increase
	if cap(buffer) >= newSize {
		return buffer[ : newSize]
	} else {
		newBuffer := make([]T, newSize)
		copy(newBuffer, buffer)
		return newBuffer
	}
}

func boolErrCheck(value uint8) error {
	if (value == 0 || value == 1) { return nil }
	return errors.New("bool value must be 0 or 1")
}

// Values <0.3 are fairly low entropy. anything <0.2 tends to be visibly low entropy
func lazyEntropyUint64(value uint64) float64 {
	var patterns [4]uint8
	for i := 0; i < 64; i += 2 {
		patterns[value & 0b11] += 1
		value >>= 1
	}
	
	var dist uint8
	for _, count := range patterns {
		if count <= 8 {
			dist += (8 - count)
		} else {
			dist += (count - 8)
		}
	}
	
	return 1.0 - float64(dist)/48.0
}

func cryptoRandUint64() (uint64, error) {
   randBytes := make([]byte, 8)
   _, err := rand.Read(randBytes)
   if err != nil { return 0, err }

   var id uint64
   for i := 0; i < 8; i++ {
      id |= (uint64(randBytes[i]) << (i << 3))
   }
   return id, nil
}
