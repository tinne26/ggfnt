package builder

const brokenCode = "broken code"

func startsWith(definition string, start, end int, expr string) bool {
	if start < 0 || end + 1 > len(definition) { panic("broken code usage") }
	if len(expr) > (end + 1 - start) { return false }
	for i := 0; i < len(expr); i++ {
		if definition[start + i] != expr[i] { return false }
	}
	return true
}

// end is inclusive
func trimSpaces(definition string, start, end int) (int, int) {
	var changed bool = true
	for start <= end && changed {
		changed = false

		// trim spaces
		for start <= end && definition[start] == ' ' {
			start += 1
			changed = true
		}
		for end >= 0 && definition[end] == ' ' {
			end -= 1
			changed = true
		}
	}

	return start, end
}

type integer interface { uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64 }
func isContinuousSlice[T integer](slice []T) bool {
	if len(slice) == 0 { return true }
	for i := 1; i < len(slice); i++ {
		if slice[i] != slice[i - 1] + 1 {
			return false
		}
	}
	return true
}
