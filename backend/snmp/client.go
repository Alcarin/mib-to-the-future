package snmp

import (
	"encoding/hex"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

// Config configurazione SNMP
type Config struct {
	Host             string `json:"host"`
	Port             int    `json:"port"`
	Community        string `json:"community"`
	WriteCommunity   string `json:"writeCommunity,omitempty"`
	Version          string `json:"version"`
	ContextName      string `json:"contextName,omitempty"`
	SecurityLevel    string `json:"securityLevel,omitempty"`
	SecurityUsername string `json:"securityUsername,omitempty"`
	AuthProtocol     string `json:"authProtocol,omitempty"`
	AuthPassword     string `json:"authPassword,omitempty"`
	PrivProtocol     string `json:"privProtocol,omitempty"`
	PrivPassword     string `json:"privPassword,omitempty"`
}

// Result risultato operazione SNMP
type Result struct {
	OID          string `json:"oid"`
	Value        string `json:"value"`
	Type         string `json:"type"`
	Status       string `json:"status"`
	ResponseTime int64  `json:"responseTime"`
	Timestamp    string `json:"timestamp"`
	ResolvedName string `json:"resolvedName"`
	RawValue     string `json:"rawValue,omitempty"`
	DisplayValue string `json:"displayValue,omitempty"`
	Syntax       string `json:"syntax,omitempty"`
}

// Client client SNMP
type Client struct {
	snmp *gosnmp.GoSNMP
	cfg  Config
}

// NewClient crea nuovo client SNMP
func NewClient(config Config) (*Client, error) {
	host := strings.TrimSpace(config.Host)

	port := config.Port
	if port <= 0 {
		port = 161
	}

	client := &gosnmp.GoSNMP{
		Target:  host,
		Port:    uint16(port),
		Timeout: 5 * time.Second,
		Retries: 2,
	}

	version := strings.ToLower(strings.TrimSpace(config.Version))
	switch version {
	case "", "v2c":
		version = "v2c"
	case "v1":
		version = "v1"
	case "v3":
		version = "v3"
	default:
		return nil, fmt.Errorf("versione SNMP non supportata: %s", config.Version)
	}

	community := strings.TrimSpace(config.Community)
	if community == "" && version != "v3" {
		community = "public"
	}

	switch version {
	case "v1":
		client.Version = gosnmp.Version1
		client.Community = community
	case "v2c":
		client.Version = gosnmp.Version2c
		client.Community = community
	case "v3":
		client.Version = gosnmp.Version3
		client.ContextName = strings.TrimSpace(config.ContextName)

		securityLevel, err := normalizeSecurityLevel(config.SecurityLevel)
		if err != nil {
			return nil, err
		}

		securityUsername := strings.TrimSpace(config.SecurityUsername)
		if securityUsername == "" {
			return nil, fmt.Errorf("username di sicurezza richiesto per SNMPv3")
		}

		params := &gosnmp.UsmSecurityParameters{
			UserName: securityUsername,
		}

		client.SecurityModel = gosnmp.UserSecurityModel

		switch securityLevel {
		case "noAuthNoPriv":
			client.MsgFlags = gosnmp.NoAuthNoPriv
		case "authNoPriv":
			client.MsgFlags = gosnmp.AuthNoPriv
		case "authPriv":
			client.MsgFlags = gosnmp.AuthPriv
		}

		if securityLevel == "authNoPriv" || securityLevel == "authPriv" {
			authProtocol, err := normalizeAuthProtocol(config.AuthProtocol)
			if err != nil {
				return nil, err
			}
			if authProtocol == "" {
				return nil, fmt.Errorf("protocollo di autenticazione richiesto per SNMPv3")
			}
			if strings.TrimSpace(config.AuthPassword) == "" {
				return nil, fmt.Errorf("password di autenticazione richiesta per SNMPv3")
			}

			if err := applyAuthProtocol(params, authProtocol); err != nil {
				return nil, err
			}
			params.AuthenticationPassphrase = config.AuthPassword
		}

		if securityLevel == "authPriv" {
			privProtocol, err := normalizePrivProtocol(config.PrivProtocol)
			if err != nil {
				return nil, err
			}
			if privProtocol == "" {
				return nil, fmt.Errorf("protocollo di privacy richiesto per SNMPv3")
			}
			if strings.TrimSpace(config.PrivPassword) == "" {
				return nil, fmt.Errorf("password di privacy richiesta per SNMPv3")
			}

			if err := applyPrivProtocol(params, privProtocol); err != nil {
				return nil, err
			}
			params.PrivacyPassphrase = config.PrivPassword
		}

		client.SecurityParameters = params
	default:
		client.Version = gosnmp.Version2c
		client.Community = community
	}

	cfg := config
	cfg.Host = host
	cfg.Port = port
	cfg.Version = version
	cfg.Community = community
	cfg.WriteCommunity = strings.TrimSpace(config.WriteCommunity)
	if cfg.WriteCommunity == "" {
		cfg.WriteCommunity = community
	}
	if version == "v3" {
		cfg.WriteCommunity = ""
	}

	return &Client{snmp: client, cfg: cfg}, nil
}

// Connect connette al target
func (c *Client) Connect() error {
	return c.snmp.Connect()
}

// Close chiude la connessione
func (c *Client) Close() error {
	return c.snmp.Conn.Close()
}

// Get esegue SNMP GET
func (c *Client) Get(oid string) (*Result, error) {
	start := time.Now()

	err := c.Connect()
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}
	defer c.Close()

	result, err := c.snmp.Get([]string{oid})
	if err != nil {
		return &Result{
			OID:          oid,
			Status:       "error",
			ResponseTime: time.Since(start).Milliseconds(),
			Timestamp:    time.Now().Format(time.RFC3339),
		}, err
	}

	if len(result.Variables) == 0 {
		return nil, fmt.Errorf("no data received")
	}

	variable := result.Variables[0]

	return &Result{
		OID:          variable.Name,
		Value:        formatPDUValue(variable),
		Type:         variable.Type.String(),
		Status:       "success",
		ResponseTime: time.Since(start).Milliseconds(),
		Timestamp:    time.Now().Format(time.RFC3339),
	}, nil
}

// GetNext esegue SNMP GETNEXT
func (c *Client) GetNext(oid string) (*Result, error) {
	start := time.Now()

	err := c.Connect()
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}
	defer c.Close()

	result, err := c.snmp.GetNext([]string{oid})
	if err != nil {
		return &Result{
			OID:          oid,
			Status:       "error",
			ResponseTime: time.Since(start).Milliseconds(),
			Timestamp:    time.Now().Format(time.RFC3339),
		}, err
	}

	if len(result.Variables) == 0 {
		return nil, fmt.Errorf("no data received")
	}

	variable := result.Variables[0]

	return &Result{
		OID:          variable.Name,
		Value:        formatPDUValue(variable),
		Type:         variable.Type.String(),
		Status:       "success",
		ResponseTime: time.Since(start).Milliseconds(),
		Timestamp:    time.Now().Format(time.RFC3339),
	}, nil
}

// Walk esegue SNMP WALK
func (c *Client) Walk(oid string) ([]Result, error) {
	start := time.Now()

	err := c.Connect()
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}
	defer c.Close()

	results := []Result{}

	err = c.snmp.Walk(oid, func(variable gosnmp.SnmpPDU) error {
		results = append(results, Result{
			OID:          variable.Name,
			Value:        formatPDUValue(variable),
			Type:         variable.Type.String(),
			Status:       "success",
			ResponseTime: time.Since(start).Milliseconds(),
			Timestamp:    time.Now().Format(time.RFC3339),
		})
		return nil
	})

	if err != nil {
		return results, err
	}

	return results, nil
}

// GetBulk esegue SNMP GETBULK
func (c *Client) GetBulk(oid string, maxRepetitions uint8) ([]Result, error) {
	start := time.Now()

	err := c.Connect()
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}
	defer c.Close()

	c.snmp.MaxRepetitions = uint32(maxRepetitions)

	result, err := c.snmp.GetBulk([]string{oid}, 0, uint32(maxRepetitions))
	if err != nil {
		return nil, err
	}

	results := []Result{}
	for _, variable := range result.Variables {
		results = append(results, Result{
			OID:          variable.Name,
			Value:        formatPDUValue(variable),
			Type:         variable.Type.String(),
			Status:       "success",
			ResponseTime: time.Since(start).Milliseconds(),
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}

	return results, nil
}

// Set esegue SNMP SET
func (c *Client) Set(oid string, valueType string, value interface{}) (*Result, error) {
	pdu, err := buildSetPDU(oid, valueType, value)
	if err != nil {
		return nil, err
	}

	originalCommunity := c.snmp.Community
	if c.snmp.Version != gosnmp.Version3 {
		writeCommunity := strings.TrimSpace(c.cfg.WriteCommunity)
		if writeCommunity != "" {
			c.snmp.Community = writeCommunity
		}
	}

	start := time.Now()

	if err := c.Connect(); err != nil {
		c.snmp.Community = originalCommunity
		return nil, fmt.Errorf("connection failed: %v", err)
	}
	defer func() {
		c.snmp.Community = originalCommunity
		_ = c.Close()
	}()

	packet, err := c.snmp.Set([]gosnmp.SnmpPDU{pdu})
	if err != nil {
		return &Result{
			OID:          oid,
			Status:       "error",
			ResponseTime: time.Since(start).Milliseconds(),
			Timestamp:    time.Now().Format(time.RFC3339),
		}, err
	}

	if packet == nil || len(packet.Variables) == 0 {
		return nil, fmt.Errorf("no data received")
	}

	if packet.Error != gosnmp.NoError {
		return &Result{
			OID:          oid,
			Status:       "error",
			ResponseTime: time.Since(start).Milliseconds(),
			Timestamp:    time.Now().Format(time.RFC3339),
		}, fmt.Errorf("SNMP error: %s (index %d)", packet.Error, packet.ErrorIndex)
	}

	variable := packet.Variables[0]

	return &Result{
		OID:          variable.Name,
		Value:        formatPDUValue(variable),
		Type:         variable.Type.String(),
		Status:       "success",
		ResponseTime: time.Since(start).Milliseconds(),
		Timestamp:    time.Now().Format(time.RFC3339),
	}, nil
}

// formatPDUValue restituisce una rappresentazione testuale leggibile del valore SNMP.
func formatPDUValue(pdu gosnmp.SnmpPDU) string {
	switch pdu.Type {
	case gosnmp.OctetString, gosnmp.BitString:
		if data, ok := toByteSlice(pdu.Value); ok {
			if isPrintableASCII(data) {
				return string(data)
			}
			return "0x" + hex.EncodeToString(data)
		}
	case gosnmp.IPAddress:
		if str, ok := pdu.Value.(string); ok && str != "" {
			return str
		}
		if data, ok := toByteSlice(pdu.Value); ok {
			ip := net.IP(data)
			if ip.To4() != nil || ip.To16() != nil {
				return ip.String()
			}
		}
	}

	return fmt.Sprintf("%v", pdu.Value)
}

// toByteSlice prova a convertire un valore generico in slice di byte.
func toByteSlice(value interface{}) ([]byte, bool) {
	if value == nil {
		return nil, false
	}

	switch v := value.(type) {
	case []byte:
		return v, true
	default:
		return nil, false
	}
}

// isPrintableASCII verifica se tutti i byte rientrano nel range ASCII stampabile.
func isPrintableASCII(data []byte) bool {
	if len(data) == 0 {
		return true
	}

	for _, b := range data {
		if b < 32 || b > 126 {
			return false
		}
	}

	return true
}

func buildSetPDU(oid string, valueType string, raw interface{}) (gosnmp.SnmpPDU, error) {
	vt := strings.ToLower(strings.TrimSpace(valueType))
	switch vt {
	case "integer", "int", "enum", "enumerated":
		value, err := coerceInt64(raw)
		if err != nil {
			return gosnmp.SnmpPDU{}, err
		}
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Integer, Value: int(value)}, nil
	case "unsigned32", "uinteger32":
		value, err := coerceUint64(raw)
		if err != nil {
			return gosnmp.SnmpPDU{}, err
		}
		if value > math.MaxUint32 {
			return gosnmp.SnmpPDU{}, fmt.Errorf("value %d exceeds Unsigned32 range", value)
		}
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Uinteger32, Value: uint32(value)}, nil
	case "counter32":
		value, err := coerceUint64(raw)
		if err != nil {
			return gosnmp.SnmpPDU{}, err
		}
		if value > math.MaxUint32 {
			return gosnmp.SnmpPDU{}, fmt.Errorf("value %d exceeds Counter32 range", value)
		}
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Counter32, Value: uint32(value)}, nil
	case "counter64":
		value, err := coerceUint64(raw)
		if err != nil {
			return gosnmp.SnmpPDU{}, err
		}
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Counter64, Value: value}, nil
	case "gauge32":
		value, err := coerceUint64(raw)
		if err != nil {
			return gosnmp.SnmpPDU{}, err
		}
		if value > math.MaxUint32 {
			return gosnmp.SnmpPDU{}, fmt.Errorf("value %d exceeds Gauge32 range", value)
		}
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Gauge32, Value: uint32(value)}, nil
	case "timeticks":
		value, err := coerceUint64(raw)
		if err != nil {
			return gosnmp.SnmpPDU{}, err
		}
		if value > math.MaxUint32 {
			return gosnmp.SnmpPDU{}, fmt.Errorf("value %d exceeds TimeTicks range", value)
		}
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.TimeTicks, Value: uint32(value)}, nil
	case "ipaddress":
		addr, err := coerceIPAddress(raw)
		if err != nil {
			return gosnmp.SnmpPDU{}, err
		}
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.IPAddress, Value: addr}, nil
	case "objectidentifier", "object_identifier":
		oidValue, err := coerceObjectIdentifier(raw)
		if err != nil {
			return gosnmp.SnmpPDU{}, err
		}
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.ObjectIdentifier, Value: oidValue}, nil
	case "bits", "bitstring":
		bytes, err := coerceByteArray(raw)
		if err != nil {
			return gosnmp.SnmpPDU{}, err
		}
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.BitString, Value: bytes}, nil
	case "opaque":
		bytes, err := coerceOctetString(raw)
		if err != nil {
			return gosnmp.SnmpPDU{}, err
		}
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.Opaque, Value: bytes}, nil
	case "octetstring", "string", "displaystring":
		fallthrough
	default:
		bytes, err := coerceOctetString(raw)
		if err != nil {
			return gosnmp.SnmpPDU{}, err
		}
		return gosnmp.SnmpPDU{Name: oid, Type: gosnmp.OctetString, Value: bytes}, nil
	}
}

func coerceInt64(raw interface{}) (int64, error) {
	switch v := raw.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		if v > math.MaxInt64 {
			return 0, fmt.Errorf("value %d exceeds int64 range", v)
		}
		return int64(v), nil
	case float32:
		return coerceFloatToInt64(float64(v))
	case float64:
		return coerceFloatToInt64(v)
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, fmt.Errorf("empty string")
		}
		parsed, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid integer %q: %w", trimmed, err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported integer type %T", raw)
	}
}

func coerceUint64(raw interface{}) (uint64, error) {
	switch v := raw.(type) {
	case int:
		if v < 0 {
			return 0, fmt.Errorf("negative value %d", v)
		}
		return uint64(v), nil
	case int8:
		if v < 0 {
			return 0, fmt.Errorf("negative value %d", v)
		}
		return uint64(v), nil
	case int16:
		if v < 0 {
			return 0, fmt.Errorf("negative value %d", v)
		}
		return uint64(v), nil
	case int32:
		if v < 0 {
			return 0, fmt.Errorf("negative value %d", v)
		}
		return uint64(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("negative value %d", v)
		}
		return uint64(v), nil
	case uint:
		return uint64(v), nil
	case uint8:
		return uint64(v), nil
	case uint16:
		return uint64(v), nil
	case uint32:
		return uint64(v), nil
	case uint64:
		return v, nil
	case float32:
		return coerceFloatToUint64(float64(v))
	case float64:
		return coerceFloatToUint64(v)
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, fmt.Errorf("empty string")
		}
		parsed, err := strconv.ParseUint(trimmed, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid unsigned integer %q: %w", trimmed, err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported unsigned integer type %T", raw)
	}
}

func coerceFloatToInt64(v float64) (int64, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("invalid numeric value %v", v)
	}
	rounded := math.Trunc(v)
	if rounded != v {
		return 0, fmt.Errorf("value %v is not an integer", v)
	}
	if rounded > math.MaxInt64 || rounded < math.MinInt64 {
		return 0, fmt.Errorf("value %v exceeds int64 range", v)
	}
	return int64(rounded), nil
}

func coerceFloatToUint64(v float64) (uint64, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("invalid numeric value %v", v)
	}
	if v < 0 {
		return 0, fmt.Errorf("negative value %v", v)
	}
	rounded := math.Trunc(v)
	if rounded != v {
		return 0, fmt.Errorf("value %v is not an integer", v)
	}
	if rounded > math.MaxUint64 {
		return 0, fmt.Errorf("value %v exceeds uint64 range", v)
	}
	return uint64(rounded), nil
}

func coerceOctetString(raw interface{}) ([]byte, error) {
	if bytes, err := coerceByteArray(raw); err == nil {
		return bytes, nil
	}

	switch v := raw.(type) {
	case string:
		return []byte(v), nil
	case []rune:
		return []byte(string(v)), nil
	default:
		return []byte(fmt.Sprintf("%v", raw)), nil
	}
}

func coerceByteArray(raw interface{}) ([]byte, error) {
	switch v := raw.(type) {
	case []byte:
		return v, nil
	case []int:
		out := make([]byte, len(v))
		for i, item := range v {
			if item < 0 || item > 255 {
				return nil, fmt.Errorf("byte value %d out of range", item)
			}
			out[i] = byte(item)
		}
		return out, nil
	case []interface{}:
		out := make([]byte, len(v))
		for i, item := range v {
			num, err := coerceUint64(item)
			if err != nil {
				return nil, err
			}
			if num > 255 {
				return nil, fmt.Errorf("byte value %d out of range", num)
			}
			out[i] = byte(num)
		}
		return out, nil
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return []byte{}, nil
		}
		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			s = s[2:]
		}
		if len(s)%2 == 1 {
			s = "0" + s
		}
		bytes, err := hex.DecodeString(s)
		if err != nil {
			return nil, fmt.Errorf("invalid hex string %q: %w", v, err)
		}
		return bytes, nil
	default:
		return nil, fmt.Errorf("unsupported byte slice type %T", raw)
	}
}

func coerceIPAddress(raw interface{}) (string, error) {
	str, err := coerceString(raw)
	if err != nil {
		return "", err
	}
	if str == "" {
		return "", fmt.Errorf("IP address cannot be empty")
	}
	ip := net.ParseIP(str)
	if ip == nil {
		return "", fmt.Errorf("invalid IP address: %s", str)
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return "", fmt.Errorf("only IPv4 addresses are supported: %s", str)
	}
	return ip4.String(), nil
}

func coerceObjectIdentifier(raw interface{}) (string, error) {
	str, err := coerceString(raw)
	if err != nil {
		return "", err
	}
	if str == "" {
		return "", fmt.Errorf("OID cannot be empty")
	}
	parts := strings.Split(str, ".")
	for _, part := range parts {
		if part == "" {
			return "", fmt.Errorf("invalid OID %q", str)
		}
		if _, err := strconv.ParseUint(part, 10, 32); err != nil {
			return "", fmt.Errorf("invalid OID %q: %w", str, err)
		}
	}
	return str, nil
}

func coerceString(raw interface{}) (string, error) {
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v), nil
	case fmt.Stringer:
		return strings.TrimSpace(v.String()), nil
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", raw)), nil
	}
}

func normalizeSecurityLevel(level string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "noauthnopriv":
		return "noAuthNoPriv", nil
	case "authnopriv":
		return "authNoPriv", nil
	case "authpriv":
		return "authPriv", nil
	default:
		return "", fmt.Errorf("livello di sicurezza non valido: %s", level)
	}
}

func normalizeAuthProtocol(protocol string) (string, error) {
	value := strings.ToUpper(strings.TrimSpace(protocol))
	if value == "" {
		return "", nil
	}

	switch value {
	case "MD5", "SHA", "SHA224", "SHA256", "SHA384", "SHA512":
		return value, nil
	default:
		return "", fmt.Errorf("protocollo di autenticazione non supportato: %s", protocol)
	}
}

func normalizePrivProtocol(protocol string) (string, error) {
	value := strings.ToUpper(strings.TrimSpace(protocol))
	if value == "" {
		return "", nil
	}

	switch value {
	case "DES", "AES", "AES192", "AES192C", "AES256", "AES256C":
		return value, nil
	default:
		return "", fmt.Errorf("protocollo di privacy non supportato: %s", protocol)
	}
}

func applyAuthProtocol(params *gosnmp.UsmSecurityParameters, protocol string) error {
	switch protocol {
	case "MD5":
		params.AuthenticationProtocol = gosnmp.MD5
	case "SHA":
		params.AuthenticationProtocol = gosnmp.SHA
	case "SHA224":
		params.AuthenticationProtocol = gosnmp.SHA224
	case "SHA256":
		params.AuthenticationProtocol = gosnmp.SHA256
	case "SHA384":
		params.AuthenticationProtocol = gosnmp.SHA384
	case "SHA512":
		params.AuthenticationProtocol = gosnmp.SHA512
	default:
		return fmt.Errorf("protocollo di autenticazione non supportato: %s", protocol)
	}
	return nil
}

func applyPrivProtocol(params *gosnmp.UsmSecurityParameters, protocol string) error {
	switch protocol {
	case "DES":
		params.PrivacyProtocol = gosnmp.DES
	case "AES":
		params.PrivacyProtocol = gosnmp.AES
	case "AES192":
		params.PrivacyProtocol = gosnmp.AES192
	case "AES192C":
		params.PrivacyProtocol = gosnmp.AES192C
	case "AES256":
		params.PrivacyProtocol = gosnmp.AES256
	case "AES256C":
		params.PrivacyProtocol = gosnmp.AES256C
	default:
		return fmt.Errorf("protocollo di privacy non supportato: %s", protocol)
	}
	return nil
}
