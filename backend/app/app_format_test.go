package app

import (
	"testing"

	"mib-to-the-future/backend/mib"
)

func TestFormatValueWithSyntax_IntegerEnumsDontTriggerBits(t *testing.T) {
	node := &mib.Node{Syntax: "INTEGER { other(1), regular1822(2), hdh1822(3) }"}
	formatted, ok := formatValueWithSyntax("71", "integer", node)
	if ok {
		t.Fatalf("expected no specialized formatting, got ok=true with %q", formatted)
	}
	if formatted != "71" {
		t.Fatalf("expected raw value '71', got %q", formatted)
	}
}

func TestFormatValueWithSyntax_BitsRequiresHexPayload(t *testing.T) {
	node := &mib.Node{Syntax: "BITS { up(0), down(1) }"}

	if formatted, ok := formatValueWithSyntax("0x80", "bits", node); !ok || formatted != "up" {
		t.Fatalf("expected bit label 'up', got %q (ok=%v)", formatted, ok)
	}

	if formatted, ok := formatValueWithSyntax("128", "bits", node); ok || formatted != "128" {
		t.Fatalf("expected raw decimal value '128', got %q (ok=%v)", formatted, ok)
	}
}

func TestFormatValueWithSyntax_DisplayStringDecoding(t *testing.T) {
	node := &mib.Node{Syntax: "DisplayString"}

	if formatted, ok := formatValueWithSyntax("0x5265616c74656b", "octetstring", node); !ok || formatted != "Realtek" {
		t.Fatalf("expected ASCII decoding to Realtek, got %q (ok=%v)", formatted, ok)
	}

	utf16Raw := "0x53006f00660074007700610072006500" // "Software" in UTF-16 LE
	if val := decodeTextBytes([]byte{0x53, 0x00, 0x6f, 0x00, 0x66, 0x00, 0x74, 0x00, 0x77, 0x00, 0x61, 0x00, 0x72, 0x00, 0x65, 0x00}); val != "Software" {
		t.Fatalf("decodeTextBytes expected Software, got %q", val)
	}

	if data, ok := parseHexLikeString(utf16Raw); !ok {
		t.Fatalf("parseHexLikeString failed")
	} else {
		t.Logf("parsed bytes: %v", data)
	}

	if val, ok := formatDisplayString(utf16Raw); !ok || val != "Software" {
		t.Fatalf("formatDisplayString expected Software, got %q (ok=%v)", val, ok)
	}

	if formatted, ok := formatValueWithSyntax(utf16Raw, "octetstring", node); !ok || formatted != "Software" {
		t.Fatalf("expected UTF16 decoding to Software, got %q (ok=%v)", formatted, ok)
	}
}
