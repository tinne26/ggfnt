package internal

import "errors"
import "crypto/rand"

func GrowSliceByN[T any](buffer []T, increase int) []T {
	newSize := len(buffer) + increase
	if cap(buffer) >= newSize {
		return buffer[ : newSize]
	} else {
		newBuffer := make([]T, newSize)
		copy(newBuffer, buffer)
		return newBuffer
	}
}

func SetSliceSize[T any](buffer []T, size int) []T {
	if cap(buffer) >= size {
		return buffer[ : size]
	} else {
		return make([]T, size)
	}
}

func BoolErrCheck(value uint8) error {
	if (value == 0 || value == 1) { return nil }
	return errors.New("bool value must be 0 or 1")
}

func BoolToUint8(truthy bool) uint8 {
	if truthy { return 1 }
	return 0
}

// Values <0.3 are fairly low entropy. anything <0.2 tends to be visibly low entropy
func LazyEntropyUint64(value uint64) float64 {
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

func CryptoRandUint64() (uint64, error) {
   randBytes := make([]byte, 8)
   _, err := rand.Read(randBytes)
   if err != nil { return 0, err }

   var id uint64
   for i := 0; i < 8; i++ {
      id |= (uint64(randBytes[i]) << (i << 3))
   }
   return id, nil
}

// LE stands for "little endian"

func DecodeUint16LE(buffer []byte) uint16 {
	if len(buffer) < 2 { panic(len(buffer)) }
	return uint16(buffer[0]) | (uint16(buffer[1]) << 8)
}

func DecodeUint24LE(buffer []byte) uint32 {
	if len(buffer) < 3 { panic(len(buffer)) }
	return (uint32(buffer[0]) <<  0) | (uint32(buffer[1]) <<  8) | (uint32(buffer[2]) << 16)
}

func DecodeUint32LE(buffer []byte) uint32 {
	if len(buffer) < 4 { panic(len(buffer)) }
	return (uint32(buffer[0]) <<  0) | (uint32(buffer[1]) <<  8) |
	       (uint32(buffer[2]) << 16) | (uint32(buffer[3]) << 24)
}

func DecodeDate(buffer []byte) (uint16, uint8, uint8) {
	if len(buffer) < 4 { panic(len(buffer)) }
	return DecodeUint16LE(buffer[0 : 2]), uint8(buffer[2]), uint8(buffer[3])
}

func DecodeUint64LE(buffer []byte) uint64 {
	if len(buffer) < 8 { panic(len(buffer)) }
	return (uint64(buffer[0]) <<  0) | (uint64(buffer[1]) <<  8) | (uint64(buffer[2]) << 16) |
	       (uint64(buffer[3]) << 24) | (uint64(buffer[4]) << 32) | (uint64(buffer[5]) << 40) |
	       (uint64(buffer[6]) << 48) | (uint64(buffer[7]) << 56)
}

func AppendUint8(buffer []byte, value byte) []byte {
	return append(buffer, value)
}

func AppendUint16LE(buffer []byte, value uint16) []byte {
	return append(buffer, byte(value), byte(value >> 8))
}

func EncodeUint16LE(buffer []byte, value uint16) {
	if len(buffer) < 2 { panic("invalid usage of encodeUint16LE") }
	buffer[0] = byte(value)
	buffer[1] = byte(value >> 8)
}

func AppendUint24LE(buffer []byte, value uint32) []byte {
	return append(buffer, byte(value), byte(value >> 8), byte(value >> 16))
}

func EncodeUint24LE(buffer []byte, value uint32) {
	if len(buffer) < 3 { panic("invalid usage of encodeUint24LE") }
	buffer[0] = byte(value)
	buffer[1] = byte(value >>  8)
	buffer[2] = byte(value >> 16)
}

func AppendUint32LE(buffer []byte, value uint32) []byte {
	return append(buffer, byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24))
}

func EncodeUint32LE(buffer []byte, value uint32) {
	if len(buffer) < 4 { panic("invalid usage of encodeUint32LE") }
	buffer[0] = byte(value)
	buffer[1] = byte(value >>  8)
	buffer[2] = byte(value >> 16)
	buffer[3] = byte(value >> 24)
}

func AppendUint64LE(buffer []byte, value uint64) []byte {
	return append(
		buffer, 
		byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24),
		byte(value >> 32), byte(value >> 40), byte(value >> 48), byte(value >> 56),
	)
}

func AppendShortString(buffer []byte, str string) []byte {
	if len(str) > 255 { panic("appendShortString() can't append string with len() > 255") }
	return append(append(buffer, uint8(len(str))), str...)
}

func AppendString(buffer []byte, str string) []byte {
	if len(str) > 65535 { panic("appendString() can't append string with len() > 65535") }
	return append(AppendUint16LE(buffer, uint16(len(str))), str...)
}

func AppendByteDigits(value byte, str []byte) []byte {
	if value > 99 {
		digit := value/100
		str = append(str, digit + '0')
	}
	if value > 9 {
		digit := (value/10) % 10
		str = append(str, digit + '0')
	}
	return append(str, (value % 10) + '0')
}
