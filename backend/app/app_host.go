package app

import (
	"fmt"
	"strings"

	"mib-to-the-future/backend/mib"
	"mib-to-the-future/backend/snmp"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ListHosts restituisce l'elenco degli host SNMP salvati, ordinati per ultimo utilizzo.
func (a *App) ListHosts() ([]mib.HostConfig, error) {
	if a.mibDB == nil {
		return nil, a.mibNotInitializedErr()
	}

	hosts, err := a.mibDB.ListHosts(0)
	if err != nil {
		return nil, fmt.Errorf("failed to list host configs: %w", err)
	}
	return hosts, nil
}

// SaveHost salva o aggiorna la configurazione SNMP di un host e restituisce la versione persistita.
func (a *App) SaveHost(config mib.HostConfig) (*mib.HostConfig, error) {
	if a.mibDB == nil {
		return nil, a.mibNotInitializedErr()
	}

	saved, err := a.mibDB.SaveHost(config)
	if err != nil {
		return nil, fmt.Errorf("failed to save host config: %w", err)
	}
	return saved, nil
}

// TouchHost aggiorna la data dell'ultimo utilizzo per un host salvato.
func (a *App) TouchHost(address string) error {
	if a.mibDB == nil {
		return a.mibNotInitializedErr()
	}
	if strings.TrimSpace(address) == "" {
		return fmt.Errorf("address is required")
	}

	if err := a.mibDB.TouchHost(address); err != nil {
		return fmt.Errorf("failed to register host usage: %w", err)
	}
	return nil
}

// DeleteHost rimuove definitivamente la configurazione di un host salvato.
func (a *App) DeleteHost(address string) error {
	if a.mibDB == nil {
		return a.mibNotInitializedErr()
	}
	if strings.TrimSpace(address) == "" {
		return fmt.Errorf("address is required")
	}

	if err := a.mibDB.DeleteHost(address); err != nil {
		return fmt.Errorf("failed to delete host config: %w", err)
	}
	return nil
}

// persistHostUsage salva automaticamente la configurazione di un host quando viene utilizzato.
func (a *App) persistHostUsage(config snmp.Config) {
	if a.mibDB == nil {
		return
	}

	address := strings.TrimSpace(config.Host)
	if address == "" {
		return
	}

	hostConfig := mib.HostConfig{
		Address:          address,
		Port:             config.Port,
		Community:        config.Community,
		WriteCommunity:   config.WriteCommunity,
		Version:          config.Version,
		ContextName:      config.ContextName,
		SecurityLevel:    config.SecurityLevel,
		SecurityUsername: config.SecurityUsername,
		AuthProtocol:     config.AuthProtocol,
		AuthPassword:     config.AuthPassword,
		PrivProtocol:     config.PrivProtocol,
		PrivPassword:     config.PrivPassword,
	}

	if _, err := a.mibDB.SaveHost(hostConfig); err != nil {
		if a.ctx != nil {
			runtime.LogError(a.ctx, fmt.Sprintf("Failed to persist host usage: %v", err))
		}
	}
}
