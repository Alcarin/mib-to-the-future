package mib

import (
	"strconv"
	"strings"
)

// CompareOIDs confronta due OID garantendo un ordinamento naturale sulle componenti numeriche.
func CompareOIDs(a, b string) int {
	if a == b {
		return 0
	}

	partsA := splitOIDParts(normalizeOID(a))
	partsB := splitOIDParts(normalizeOID(b))

	limit := len(partsA)
	if len(partsB) < limit {
		limit = len(partsB)
	}

	for i := 0; i < limit; i++ {
		segmentA := partsA[i]
		segmentB := partsB[i]

		intA, errA := strconv.Atoi(segmentA)
		intB, errB := strconv.Atoi(segmentB)

		if errA == nil && errB == nil {
			switch {
			case intA < intB:
				return -1
			case intA > intB:
				return 1
			default:
				continue
			}
		}

		if segmentA < segmentB {
			return -1
		}
		if segmentA > segmentB {
			return 1
		}
	}

	switch {
	case len(partsA) < len(partsB):
		return -1
	case len(partsA) > len(partsB):
		return 1
	default:
		return 0
	}
}

func normalizeOID(oid string) string {
	trimmed := strings.TrimSpace(oid)
	for strings.HasPrefix(trimmed, ".") {
		trimmed = trimmed[1:]
	}
	return trimmed
}

func splitOIDParts(oid string) []string {
	if oid == "" {
		return nil
	}

	rawParts := strings.Split(oid, ".")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parts = append(parts, part)
	}
	return parts
}
