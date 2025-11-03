package snmp

import (
	"testing"

	"github.com/gosnmp/gosnmp"
)

func TestFormatPDUValue_OctetStringVariants(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "ASCII string represented as hex",
			value:    []byte("eth0"),
			expected: "0x65746830",
		},
		{
			name:     "UTF16 little endian without BOM",
			value:    []byte{'E', 0x00, 't', 0x00, 'h', 0x00, '0', 0x00},
			expected: "0x4500740068003000",
		},
		{
			name:     "UTF16 big endian with BOM",
			value:    []byte{0xFE, 0xFF, 0x00, 'L', 0x00, 'A', 0x00, 'N'},
			expected: "0xfeff004c0041004e",
		},
		{
			name:     "Latin1 bytes preserved as hex",
			value:    []byte{0x53, 0xF1, 0x6F}, // "Sño"
			expected: "0x53f16f",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatPDUValue(gosnmp.SnmpPDU{
				Type:  gosnmp.OctetString,
				Value: tc.value,
			})
			if result != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestFormatPDUValue_OctetStringFallbackToHex(t *testing.T) {
	t.Run("binary data becomes hex", func(t *testing.T) {
		raw := []byte{0x00, 0xFF, 0x10}
		result := formatPDUValue(gosnmp.SnmpPDU{
			Type:  gosnmp.OctetString,
			Value: raw,
		})
		if result != "0x00ff10" {
			t.Fatalf("expected hex fallback, got %q", result)
		}
	})

	t.Run("mac address stays hex formatted", func(t *testing.T) {
		raw := []byte{0x00, 0x1a, 0x2b, 0x3c, 0x4d, 0x5e}
		result := formatPDUValue(gosnmp.SnmpPDU{
			Type:  gosnmp.OctetString,
			Value: raw,
		})
		if result != "0x001a2b3c4d5e" {
			t.Fatalf("expected MAC hex fallback, got %q", result)
		}
	})
}
