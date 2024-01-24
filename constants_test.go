package ggfnt

import "testing"
import "strconv"

func TestMaxFastMappingTableCodePointsStr(t *testing.T) {
	str := strconv.Itoa(maxFastMappingTableCodePoints)
	if str != maxFastMappingTableCodePointsStr {
		t.Fatalf(
			"maxFastMappingTableCodePointsStr should be \"%s\", but have \"%s\" instead",
			str, maxFastMappingTableCodePointsStr,
		)
	}
}
