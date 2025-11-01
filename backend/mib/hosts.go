package mib

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// HostConfig rappresenta i parametri di connessione per un host SNMP persistito nel database.
type HostConfig struct {
	Address          string `json:"address"`
	Port             int    `json:"port"`
	Community        string `json:"community"`
	WriteCommunity   string `json:"writeCommunity"`
	Version          string `json:"version"`
	LastUsedAt       string `json:"lastUsedAt"`
	CreatedAt        string `json:"createdAt"`
	ContextName      string `json:"contextName,omitempty"`
	SecurityLevel    string `json:"securityLevel,omitempty"`
	SecurityUsername string `json:"securityUsername,omitempty"`
	AuthProtocol     string `json:"authProtocol,omitempty"`
	AuthPassword     string `json:"authPassword,omitempty"`
	PrivProtocol     string `json:"privProtocol,omitempty"`
	PrivPassword     string `json:"privPassword,omitempty"`
}

// SaveHost salva o aggiorna la configurazione SNMP per un host.
// L'indirizzo viene utilizzato come chiave primaria e l'ora di ultimo utilizzo viene aggiornata ad ogni salvataggio.
func (d *Database) SaveHost(config HostConfig) (*HostConfig, error) {
	address := strings.TrimSpace(config.Address)
	if address == "" {
		return nil, fmt.Errorf("address is required")
	}

	port := config.Port
	if port <= 0 {
		port = 161
	}

	community := strings.TrimSpace(config.Community)
	version := strings.TrimSpace(config.Version)
	switch strings.ToLower(version) {
	case "", "v2c":
		version = "v2c"
	case "v1":
		version = "v1"
	case "v3":
		version = "v3"
	default:
		return nil, fmt.Errorf("versione SNMP non supportata: %s", config.Version)
	}

	if community == "" && version != "v3" {
		community = "public"
	}

	writeCommunity := strings.TrimSpace(config.WriteCommunity)
	if version == "v3" {
		community = strings.TrimSpace(config.Community)
		writeCommunity = ""
	} else {
		if writeCommunity == "" {
			writeCommunity = community
		}
	}

	contextName := ""
	securityLevel := ""
	securityUsername := ""
	authProtocol := ""
	authPassword := ""
	privProtocol := ""
	privPassword := ""

	if version == "v3" {
		var err error

		contextName = strings.TrimSpace(config.ContextName)

		securityLevel, err = normalizeSecurityLevel(config.SecurityLevel)
		if err != nil {
			return nil, err
		}

		securityUsername = strings.TrimSpace(config.SecurityUsername)
		if securityUsername == "" {
			return nil, fmt.Errorf("username di sicurezza richiesto per SNMPv3")
		}

		switch securityLevel {
		case "noAuthNoPriv":
			// Nessun parametro aggiuntivo richiesto
		case "authNoPriv":
			authProtocol, err = normalizeAuthProtocol(config.AuthProtocol)
			if err != nil {
				return nil, err
			}
			if authProtocol == "" {
				return nil, fmt.Errorf("protocollo di autenticazione richiesto per SNMPv3 livello authNoPriv")
			}
			authPassword = config.AuthPassword
			if strings.TrimSpace(authPassword) == "" {
				return nil, fmt.Errorf("password di autenticazione richiesta per SNMPv3 livello authNoPriv")
			}
		case "authPriv":
			authProtocol, err = normalizeAuthProtocol(config.AuthProtocol)
			if err != nil {
				return nil, err
			}
			if authProtocol == "" {
				return nil, fmt.Errorf("protocollo di autenticazione richiesto per SNMPv3 livello authPriv")
			}
			authPassword = config.AuthPassword
			if strings.TrimSpace(authPassword) == "" {
				return nil, fmt.Errorf("password di autenticazione richiesta per SNMPv3 livello authPriv")
			}

			privProtocol, err = normalizePrivProtocol(config.PrivProtocol)
			if err != nil {
				return nil, err
			}
			if privProtocol == "" {
				return nil, fmt.Errorf("protocollo di privacy richiesto per SNMPv3 livello authPriv")
			}
			privPassword = config.PrivPassword
			if strings.TrimSpace(privPassword) == "" {
				return nil, fmt.Errorf("password di privacy richiesta per SNMPv3 livello authPriv")
			}
		default:
			return nil, fmt.Errorf("livello di sicurezza SNMPv3 non valido: %s", securityLevel)
		}
	}

	_, err := d.db.Exec(`
		INSERT INTO host_configs (
			address, port, community, write_community, version, last_used_at,
			context_name, security_level, security_username, auth_protocol, auth_password, priv_protocol, priv_password
		)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(address) DO UPDATE SET
			port = excluded.port,
			community = excluded.community,
			write_community = excluded.write_community,
			version = excluded.version,
			last_used_at = CURRENT_TIMESTAMP,
			context_name = excluded.context_name,
			security_level = excluded.security_level,
			security_username = excluded.security_username,
			auth_protocol = excluded.auth_protocol,
			auth_password = excluded.auth_password,
			priv_protocol = excluded.priv_protocol,
			priv_password = excluded.priv_password
	`, address, port, community, writeCommunity, version,
		contextName, securityLevel, securityUsername,
		authProtocol, authPassword, privProtocol, privPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to persist host config: %w", err)
	}

	return d.GetHost(address)
}

// GetHost recupera la configurazione associata a un indirizzo host.
func (d *Database) GetHost(address string) (*HostConfig, error) {
	row := d.db.QueryRow(`
		SELECT address, port, community, COALESCE(write_community, '') AS write_community, version, last_used_at, created_at,
		       COALESCE(context_name, '') AS context_name,
		       COALESCE(security_level, '') AS security_level,
		       COALESCE(security_username, '') AS security_username,
		       COALESCE(auth_protocol, '') AS auth_protocol,
		       COALESCE(auth_password, '') AS auth_password,
		       COALESCE(priv_protocol, '') AS priv_protocol,
		       COALESCE(priv_password, '') AS priv_password
		FROM host_configs
		WHERE address = ?
	`, strings.TrimSpace(address))

	host := &HostConfig{}
	err := row.Scan(
		&host.Address, &host.Port, &host.Community, &host.WriteCommunity, &host.Version, &host.LastUsedAt, &host.CreatedAt,
		&host.ContextName, &host.SecurityLevel, &host.SecurityUsername, &host.AuthProtocol, &host.AuthPassword,
		&host.PrivProtocol, &host.PrivPassword,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load host config: %w", err)
	}
	if parsed, err := parseTimestamp(host.LastUsedAt); err == nil && parsed != "" {
		host.LastUsedAt = parsed
	}
	if parsed, err := parseTimestamp(host.CreatedAt); err == nil && parsed != "" {
		host.CreatedAt = parsed
	}
	if host.WriteCommunity == "" && host.Community != "" {
		host.WriteCommunity = host.Community
	}
	return host, nil
}

// ListHosts restituisce le configurazioni host ordinate per ultimo utilizzo decrescente.
// Il parametro limit permette di limitare il numero di risultati (0 per nessun limite).
func (d *Database) ListHosts(limit int) ([]HostConfig, error) {
	query := `
		SELECT address, port, community, COALESCE(write_community, '') AS write_community, version, last_used_at, created_at,
		       COALESCE(context_name, '') AS context_name,
		       COALESCE(security_level, '') AS security_level,
		       COALESCE(security_username, '') AS security_username,
		       COALESCE(auth_protocol, '') AS auth_protocol,
		       COALESCE(auth_password, '') AS auth_password,
		       COALESCE(priv_protocol, '') AS priv_protocol,
		       COALESCE(priv_password, '') AS priv_password
		FROM host_configs
		ORDER BY datetime(last_used_at) DESC, address ASC
	`

	args := []interface{}{}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := d.db.Query(query, args...)

	if err != nil {
		return nil, fmt.Errorf("failed to list host configs: %w", err)
	}
	defer rows.Close()

	hosts := []HostConfig{}
	for rows.Next() {
		var host HostConfig
		err := rows.Scan(
			&host.Address, &host.Port, &host.Community, &host.WriteCommunity, &host.Version, &host.LastUsedAt, &host.CreatedAt,
			&host.ContextName, &host.SecurityLevel, &host.SecurityUsername, &host.AuthProtocol, &host.AuthPassword,
			&host.PrivProtocol, &host.PrivPassword,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan host config: %w", err)
		}
		if parsed, err := parseTimestamp(host.LastUsedAt); err == nil && parsed != "" {
			host.LastUsedAt = parsed
		}
		if parsed, err := parseTimestamp(host.CreatedAt); err == nil && parsed != "" {
			host.CreatedAt = parsed
		}
		if host.WriteCommunity == "" && host.Community != "" {
			host.WriteCommunity = host.Community
		}
		hosts = append(hosts, host)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed during host config iteration: %w", err)
	}

	return hosts, nil
}

// TouchHost aggiorna l'istante dell'ultimo utilizzo senza modificare gli altri parametri.
func (d *Database) TouchHost(address string) error {
	res, err := d.db.Exec(`
		UPDATE host_configs
		SET last_used_at = CURRENT_TIMESTAMP
		WHERE address = ?
	`, strings.TrimSpace(address))
	if err != nil {
		return fmt.Errorf("failed to touch host config: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to inspect touch result: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("host config not found")
	}
	return nil
}

// DeleteHost rimuove definitivamente la configurazione di un host dal database.
func (d *Database) DeleteHost(address string) error {
	trimmed := strings.TrimSpace(address)
	if trimmed == "" {
		return fmt.Errorf("address is required")
	}

	if _, err := d.db.Exec(`DELETE FROM host_configs WHERE address = ?`, trimmed); err != nil {
		return fmt.Errorf("failed to delete host config: %w", err)
	}
	return nil
}

func parseTimestamp(ts string) (string, error) {
	if strings.TrimSpace(ts) == "" {
		return "", nil
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.000000000-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05",
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, ts); err == nil {
			return parsed.Format(time.RFC3339), nil
		}
	}

	return "", fmt.Errorf("unsupported timestamp format: %s", ts)
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
