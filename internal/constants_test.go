package internal

import "testing"
import "strconv"

func TestMaxFastMappingTableCodePointsStr(t *testing.T) {
	str := strconv.Itoa(MaxFastMappingTableCodePoints)
	if str != MaxFastMappingTableCodePointsStr {
		t.Fatalf(
			"MaxFastMappingTableCodePointsStr should be \"%s\", but have \"%s\" instead",
			str, MaxFastMappingTableCodePointsStr,
		)
	}
}
