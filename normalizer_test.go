package main

import "testing"

func TestNumbersNormalizer(t *testing.T) {
	equivalences := map[string]string{
		"10":       "10",
		"10.0":     "10",
		"0010.000": "10",
		"05":       "5",
	}

	normalizer := numbersNormalizer{}
	for left, want := range equivalences {
		if got := normalizer.Normalize(left); got != want {
			t.Errorf("Got: %#v  --  Want: %#v\n", got, want)
		}
	}
}

func TestSizesNormalizer(t *testing.T) {
	equivalences := map[string]string{
		"1K":   "1024",
		"1M":   "1048576",
		"1G":   "1073741824",
		"1T":   "1099511627776",
		"2K":   "2048",
		"2k":   "2048",
		"2093": "2093",
		"3F":   "3F",
		"NaN":  "NaN",
		"12.0": "12.0",
	}

	normalizer := sizesNormalizer{}
	for left, want := range equivalences {
		if got := normalizer.Normalize(left); got != want {
			t.Errorf("Got: %#v  --  Want: %#v\n", got, want)
		}
	}
}
