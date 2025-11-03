package app

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"mib-to-the-future/backend/mib"
)

// formatTimeTicks converte un valore TimeTicks in formato leggibile.
func formatTimeTicks(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}

	ticks, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return "", false
	}
	if ticks < 0 {
		ticks = -ticks
	}

	totalSeconds := ticks / 100
	hundredths := ticks % 100

	days := totalSeconds / 86400
	remaining := totalSeconds % 86400
	hours := remaining / 3600
	remaining %= 3600
	minutes := remaining / 60
	seconds := remaining % 60

	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if days > 0 || hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if days > 0 || hours > 0 || minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if hundredths > 0 {
		secs := float64(seconds) + float64(hundredths)/100.0
		parts = append(parts, fmt.Sprintf("%.2fs", secs))
	} else {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	return strings.Join(parts, " "), true
}

// parseHexLikeString parsifica una stringa esadecimale in vari formati.
func parseHexLikeString(raw string) ([]byte, bool) {
	if raw == "" {
		return nil, false
	}
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}
	replacer := strings.NewReplacer(":", "", "-", "", " ", "")
	s = replacer.Replace(s)
	if len(s)%2 != 0 || len(s) == 0 {
		return nil, false
	}
	data, err := hex.DecodeString(s)
	if err != nil {
		return nil, false
	}
	return data, true
}

// formatMacAddress formatta un indirizzo MAC in formato standard.
func formatMacAddress(raw string) (string, bool) {
	if raw == "" {
		return "", false
	}

	if strings.Count(raw, ":") == 5 && len(raw) >= 17 {
		parts := strings.Split(raw, ":")
		allHex := true
		for _, p := range parts {
			if len(p) != 2 {
				allHex = false
				break
			}
			if _, err := strconv.ParseUint(p, 16, 8); err != nil {
				allHex = false
				break
			}
		}
		if allHex {
			for i := range parts {
				parts[i] = strings.ToUpper(parts[i])
			}
			return strings.Join(parts, ":"), true
		}
	}

	data, ok := parseHexLikeString(raw)
	if !ok {
		// Last attempt: raw ASCII of length 6 (rare)
		if len(raw) == 6 {
			data = []byte(raw)
			ok = true
		} else {
			return "", false
		}
	}

	if len(data) == 0 {
		return "", false
	}

	parts := make([]string, len(data))
	for i, b := range data {
		parts[i] = fmt.Sprintf("%02X", b)
	}
	return strings.Join(parts, ":"), true
}

// formatDateAndTime formatta un timestamp DateAndTime SNMP.
func formatDateAndTime(raw string) (string, bool) {
	data, ok := parseHexLikeString(raw)
	if !ok {
		return "", false
	}
	if len(data) != 8 && len(data) != 11 {
		return "", false
	}

	year := int(data[0])<<8 | int(data[1])
	month := int(data[2])
	day := int(data[3])
	hour := int(data[4])
	minute := int(data[5])
	second := int(data[6])
	deci := int(data[7])

	timePart := fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d.%02d", year, month, day, hour, minute, second, deci)

	if len(data) == 8 {
		return timePart + " Z", true
	}

	sign := '+'
	if data[8] == '-' {
		sign = '-'
	}
	tzHour := int(data[9])
	tzMinute := int(data[10])

	return fmt.Sprintf("%s %c%02d:%02d", timePart, sign, tzHour, tzMinute), true
}

// formatBits formatta un valore BITS usando il mapping dal MIB.
func formatBits(raw string, mapping map[string]string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) == 0 {
		return "", false
	}
	if !strings.HasPrefix(strings.ToLower(trimmed), "0x") {
		return "", false
	}

	data, ok := parseHexLikeString(raw)
	if !ok || len(data) == 0 || len(mapping) == 0 {
		return "", false
	}

	type bitLabel struct {
		index int
		label string
	}

	seen := make(map[int]struct{})
	var pairs []bitLabel
	for key, label := range mapping {
		idx, err := strconv.Atoi(key)
		if err != nil {
			continue
		}
		if _, exists := seen[idx]; exists {
			continue
		}
		seen[idx] = struct{}{}
		pairs = append(pairs, bitLabel{index: idx, label: label})
	}

	if len(pairs) == 0 {
		return "", false
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].index < pairs[j].index
	})

	var labels []string
	for _, pair := range pairs {
		byteIndex := pair.index / 8
		if byteIndex >= len(data) {
			continue
		}
		bitIndex := pair.index % 8
		mask := byte(1 << (7 - bitIndex))
		if data[byteIndex]&mask != 0 {
			labels = append(labels, pair.label)
		}
	}

	if len(labels) == 0 {
		return "", false
	}

	return strings.Join(labels, ", "), true
}

// formatInetAddress formatta un indirizzo IP (IPv4 o IPv6).
func formatInetAddress(raw string) (string, bool) {
	if parsed := net.ParseIP(raw); parsed != nil {
		return parsed.String(), true
	}

	data, ok := parseHexLikeString(raw)
	if !ok {
		return "", false
	}

	switch len(data) {
	case net.IPv4len, net.IPv6len:
		ip := net.IP(data)
		return ip.String(), true
	default:
		return "", false
	}
}

// formatDisplayString formatta una DisplayString verificando che sia ASCII stampabile.
func formatDisplayString(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}

	if data, ok := parseHexLikeString(trimmed); ok && len(data) > 0 {
		if decoded := decodeTextBytes(data); decoded != "" {
			return decoded, true
		}
		return "", false
	}

	// raw non è esadecimale: prova a interpretarlo come UTF-8
	if utf8.ValidString(trimmed) && sanitizedString(trimmed) != "" {
		return sanitizedString(trimmed), true
	}

	return "", false
}

func decodeTextBytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	if isPrintableASCIIBytes(data) {
		if str := sanitizedString(string(data)); str != "" {
			return str
		}
	}

	if utf8.Valid(data) {
		if str := sanitizedString(string(data)); str != "" {
			return str
		}
	}

	if looksLikeUTF16(data) {
		if str, ok := decodeUTF16Bytes(data); ok {
			if s := sanitizedString(str); s != "" {
				return s
			}
		}
	}

	if looksMostlyLatin1(data) {
		if str := sanitizedString(decodeLatin1Bytes(data)); str != "" {
			return str
		}
	}

	return ""
}

func isPrintableASCIIBytes(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return false
		}
		if b < 32 && b != 9 && b != 10 && b != 13 {
			return false
		}
		if b > 126 && b < 160 {
			return false
		}
	}
	return true
}

func looksLikeUTF16(data []byte) bool {
	if len(data) < 2 || len(data)%2 != 0 {
		return false
	}

	prefix := binary.BigEndian.Uint16(data[:2])
	if prefix == 0xFEFF || prefix == 0xFFFE {
		return true
	}

	zeroPairs := 0
	for i := 0; i+1 < len(data); i += 2 {
		if data[i] == 0 || data[i+1] == 0 {
			zeroPairs++
		}
	}

	return zeroPairs*4 >= len(data)
}

func decodeUTF16Bytes(data []byte) (string, bool) {
	if len(data) == 0 || len(data)%2 != 0 {
		return "", false
	}

	var (
		order binary.ByteOrder
		start int
	)

	if len(data) >= 2 {
		switch binary.BigEndian.Uint16(data[:2]) {
		case 0xFEFF:
			order = binary.BigEndian
			start = 2
		case 0xFFFE:
			order = binary.LittleEndian
			start = 2
		}
	}

	if order == nil {
		leScore := 0
		beScore := 0
		for i := 0; i < len(data); i += 2 {
			if data[i] == 0 && data[i+1] != 0 {
				beScore++
			}
			if data[i+1] == 0 && data[i] != 0 {
				leScore++
			}
		}

		if leScore == 0 && beScore == 0 {
			order = binary.LittleEndian
		} else if leScore >= beScore {
			order = binary.LittleEndian
		} else {
			order = binary.BigEndian
		}
	}

	u16Len := (len(data) - start) / 2
	u16 := make([]uint16, u16Len)
	for i, j := start, 0; j < u16Len; i, j = i+2, j+1 {
		u16[j] = order.Uint16(data[i : i+2])
	}

	runes := utf16.Decode(u16)
	return string(runes), true
}

func looksMostlyLatin1(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	printable := 0
	for _, b := range data {
		if b >= 32 || b == '\n' || b == '\r' || b == '\t' {
			printable++
		}
	}

	return printable*100 >= len(data)*70
}

func decodeLatin1Bytes(data []byte) string {
	runes := make([]rune, len(data))
	for i, b := range data {
		runes[i] = rune(b)
	}
	return string(runes)
}

func sanitizedString(value string) string {
	if value == "" {
		return ""
	}

	stripped := strings.Trim(value, "\x00")
	stripped = strings.TrimSpace(stripped)
	if stripped == "" {
		return ""
	}

	total := 0
	printable := 0

	for _, r := range stripped {
		total++
		if isPrintableRune(r) {
			printable++
		}
	}

	if total == 0 {
		return ""
	}

	if printable*100 < total*85 {
		return ""
	}

	return stripped
}

func isPrintableRune(r rune) bool {
	if r == 0 {
		return false
	}
	if r == '\n' || r == '\r' || r == '\t' {
		return true
	}
	return unicode.IsGraphic(r) && !unicode.IsControl(r)
}

// parseEnumMapping estrae il mapping dei valori enumerati dalla sintassi MIB.
func parseEnumMapping(syntax string) map[string]string {
	start := strings.Index(syntax, "{")
	if start == -1 {
		return nil
	}
	end := strings.Index(syntax[start:], "}")
	if end == -1 {
		return nil
	}
	content := syntax[start+1 : start+end]
	if strings.TrimSpace(content) == "" {
		return nil
	}

	items := strings.Split(content, ",")
	mapping := make(map[string]string)
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		open := strings.Index(item, "(")
		close := strings.LastIndex(item, ")")
		if open == -1 || close == -1 || close <= open+1 {
			continue
		}
		label := strings.TrimSpace(item[:open])
		value := strings.TrimSpace(item[open+1 : close])
		if label == "" || value == "" {
			continue
		}
		mapping[value] = label
		if num, err := strconv.ParseInt(value, 10, 64); err == nil {
			mapping[strconv.FormatInt(num, 10)] = label
		}
	}
	if len(mapping) == 0 {
		return nil
	}
	return mapping
}

// formatValueWithSyntax formatta un valore SNMP usando le informazioni della sintassi MIB.
func formatValueWithSyntax(rawValue string, valueType string, node *mib.Node) (string, bool) {
	if node == nil {
		return rawValue, false
	}

	syntax := strings.TrimSpace(node.Syntax)
	normalizedRaw := strings.TrimSpace(rawValue)
	if normalizedRaw == "" {
		return rawValue, false
	}

	loweredSyntax := strings.ToLower(syntax)
	normalizedType := strings.ToLower(strings.TrimSpace(valueType))

	if strings.Contains(loweredSyntax, "timeticks") || strings.Contains(loweredSyntax, "timestamp") ||
		strings.Contains(loweredSyntax, "timeinterval") || normalizedType == "timeticks" {
		if formatted, ok := formatTimeTicks(normalizedRaw); ok {
			return formatted, true
		}
	}

	if strings.Contains(loweredSyntax, "dateandtime") {
		if formatted, ok := formatDateAndTime(normalizedRaw); ok {
			return formatted, true
		}
	}

	if strings.Contains(loweredSyntax, "macaddress") || strings.Contains(loweredSyntax, "physaddress") {
		if formatted, ok := formatMacAddress(normalizedRaw); ok {
			return formatted, true
		}
	}

	if strings.Contains(loweredSyntax, "inetaddress") || strings.Contains(loweredSyntax, "ipaddress") {
		if formatted, ok := formatInetAddress(normalizedRaw); ok {
			return formatted, true
		}
	}

	mapping := parseEnumMapping(syntax)
	if strings.Contains(loweredSyntax, "bits") && mapping != nil {
		if formatted, ok := formatBits(normalizedRaw, mapping); ok {
			return formatted, true
		}
	}

	if mapping != nil {
		if label, ok := mapping[normalizedRaw]; ok {
			if label == "" {
				return rawValue, false
			}
			if strings.EqualFold(label, normalizedRaw) {
				return label, true
			}
			return fmt.Sprintf("%s (%s)", label, normalizedRaw), true
		}
	}

	if strings.Contains(loweredSyntax, "displaystring") || strings.Contains(loweredSyntax, "snmpadminstring") {
		if formatted, ok := formatDisplayString(normalizedRaw); ok {
			return formatted, true
		}
	}

	return rawValue, false
}
