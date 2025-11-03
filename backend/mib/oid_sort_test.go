package mib

import (
	"reflect"
	"sort"
	"testing"
)

func TestCompareOIDsOrdersNaturally(t *testing.T) {
	values := []string{
		"1.3.6.1.2.1.2.2.1.10",
		"1.3.6.1.2.1.2.2.1.2",
		"1.3.6.1.2.1.2.2.1.1",
	}

	sort.Slice(values, func(i, j int) bool {
		return CompareOIDs(values[i], values[j]) < 0
	})

	expected := []string{
		"1.3.6.1.2.1.2.2.1.1",
		"1.3.6.1.2.1.2.2.1.2",
		"1.3.6.1.2.1.2.2.1.10",
	}

	if !reflect.DeepEqual(values, expected) {
		t.Fatalf("sorted = %v, expected %v", values, expected)
	}
}

func TestCompareOIDsIgnoresFormattingNoise(t *testing.T) {
	values := []string{
		" .1.3.6.1.4.1.9.1.10",
		".1.3.6.1.4.1.9.1.2",
		"1.3.6.1.4.1.9.1.1",
	}

	sort.Slice(values, func(i, j int) bool {
		return CompareOIDs(values[i], values[j]) < 0
	})

	expected := []string{
		"1.3.6.1.4.1.9.1.1",
		".1.3.6.1.4.1.9.1.2",
		" .1.3.6.1.4.1.9.1.10",
	}

	if !reflect.DeepEqual(values, expected) {
		t.Fatalf("sorted = %v, expected %v", values, expected)
	}
}
