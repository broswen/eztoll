package toll

import (
	"testing"
)

func TestPlateNumberCompact(t *testing.T) {
	tests := []struct {
		Input string
		Want  string
	}{
		{Input: "abc 123", Want: "ABC123"},
		{Input: "aBC 123", Want: "ABC123"},
		{Input: "Abc-123", Want: "ABC123"},
		{Input: "abc-123", Want: "ABC123"},
	}
	for _, v := range tests {
		got := NormalizeLicensePlate(v.Input)
		if got != v.Want {
			t.Errorf("wanted %s but got %s", v.Want, got)
		}
	}
}
