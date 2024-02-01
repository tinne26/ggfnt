package ggfnt

import "testing"

func TestLeapYears(t *testing.T) {
	tests := []struct{ Year uint16; IsLeap bool }{
		{2020, true}, {2021, false}, {2022, false}, {2023, false}, {2024, true},
		{1700, false}, {1800, false}, {1900, false},
		{1600, true}, {2000, true},
	}
	for _, test := range tests {
		if isLeapYear(test.Year) == test.IsLeap { continue }
		if test.IsLeap {
			t.Fatalf("expected year %d to be a leap year", test.Year)
		} else {
			t.Fatalf("expected year %d to NOT be a leap year", test.Year)
		}
	}
}
