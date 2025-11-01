package app

import (
	"fmt"
	"strings"

	"mib-to-the-future/backend/snmp"
)

// SNMPGet esegue un'operazione SNMP GET su un singolo OID, aggiungendo automaticamente l'istanza `.0` per gli scalar.
// Parametri:
//   - config: la configurazione per la connessione SNMP (host, porta, community, versione).
//   - oid: l'Object Identifier da interrogare.
//
// Ritorna un puntatore a snmp.Result in caso di successo, o un errore.
func (a *App) SNMPGet(config snmp.Config, oid string) (*snmp.Result, error) {
	normalizedOID := a.normalizeScalarOID(oid)

	client, err := snmp.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create SNMP client: %v", err)
	}

	a.persistHostUsage(config)

	result, err := client.Get(normalizedOID)
	if err != nil {
		return result, fmt.Errorf("SNMP GET failed: %v", err)
	}

	a.enrichResult(result)

	return result, nil
}

// SNMPGetNext esegue un'operazione SNMP GETNEXT.
// Questa operazione richiede l'OID successivo a quello specificato.
// Parametri:
//   - config: la configurazione per la connessione SNMP.
//   - oid: l'Object Identifier da cui partire per trovare il successivo.
//
// Ritorna un puntatore a snmp.Result in caso di successo, o un errore.
func (a *App) SNMPGetNext(config snmp.Config, oid string) (*snmp.Result, error) {
	client, err := snmp.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create SNMP client: %v", err)
	}

	a.persistHostUsage(config)

	result, err := client.GetNext(oid)
	if err != nil {
		return result, fmt.Errorf("SNMP GETNEXT failed: %v", err)
	}

	a.enrichResult(result)

	return result, nil
}

// SNMPWalk esegue un'operazione SNMP WALK a partire da un OID radice.
// Recupera ricorsivamente tutti gli OID all'interno del sottoalbero specificato.
// Parametri:
//   - config: la configurazione per la connessione SNMP.
//   - oid: l'Object Identifier radice del sottoalbero da "camminare".
//
// Ritorna una slice di snmp.Result in caso di successo, o un errore.
func (a *App) SNMPWalk(config snmp.Config, oid string) ([]snmp.Result, error) {
	client, err := snmp.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create SNMP client: %v", err)
	}

	a.persistHostUsage(config)

	results, err := client.Walk(oid)
	if err != nil {
		return results, fmt.Errorf("SNMP WALK failed: %v", err)
	}

	for i := range results {
		a.enrichResult(&results[i])
	}

	return results, nil
}

// SNMPGetBulk esegue un'operazione SNMP GETBULK, una versione ottimizzata di GETNEXT.
// Recupera un blocco di dati SNMP in una singola richiesta.
// Parametri:
//   - config: la configurazione per la connessione SNMP.
//   - oid: l'Object Identifier da cui iniziare a recuperare i dati.
//   - maxRepetitions: il numero massimo di OID successivi da recuperare.
//
// Ritorna una slice di snmp.Result in caso di successo, o un errore.
func (a *App) SNMPGetBulk(config snmp.Config, oid string, maxRepetitions uint8) ([]snmp.Result, error) {
	client, err := snmp.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create SNMP client: %v", err)
	}

	a.persistHostUsage(config)

	results, err := client.GetBulk(oid, maxRepetitions)
	if err != nil {
		return results, fmt.Errorf("SNMP GETBULK failed: %v", err)
	}

	for i := range results {
		a.enrichResult(&results[i])
	}

	return results, nil
}

// SNMPSet esegue un'operazione SNMP SET per modificare il valore di un OID, normalizzando gli scalar con l'istanza `.0`.
// Parametri:
//   - config: la configurazione per la connessione SNMP.
//   - oid: l'Object Identifier da modificare.
//   - valueType: il tipo di dato del valore da impostare (es. "integer", "string").
//   - value: il valore da impostare.
//
// Ritorna un puntatore a snmp.Result con il nuovo valore in caso di successo, o un errore.
func (a *App) SNMPSet(config snmp.Config, oid string, valueType string, value interface{}) (*snmp.Result, error) {
	normalizedOID := a.normalizeScalarOID(oid)

	if strings.EqualFold(config.Version, "v3") {
		config.WriteCommunity = ""
	} else {
		if strings.TrimSpace(config.WriteCommunity) == "" {
			config.WriteCommunity = config.Community
		}
	}

	client, err := snmp.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create SNMP client: %v", err)
	}

	a.persistHostUsage(config)

	result, err := client.Set(normalizedOID, valueType, value)
	if err != nil {
		return result, fmt.Errorf("SNMP SET failed: %v", err)
	}

	a.enrichResult(result)

	return result, nil
}

// normalizeScalarOID garantisce che gli OID relativi a scalar includano l'istanza `.0`.
// Per gli altri tipi restituisce l'OID ripulito (trim degli spazi) senza modifiche.
func (a *App) normalizeScalarOID(oid string) string {
	trimmed := strings.TrimSpace(oid)
	if trimmed == "" {
		return trimmed
	}

	if a.mibDB == nil {
		return trimmed
	}

	// Se abbiamo già il suffisso `.0`, verifichiamo che corrisponda a uno scalar
	if strings.HasSuffix(trimmed, ".0") {
		base := strings.TrimSuffix(trimmed, ".0")
		if node, err := a.mibDB.GetNode(base); err == nil && node != nil && strings.EqualFold(node.Type, "scalar") {
			return trimmed
		}
		return trimmed
	}

	node, err := a.mibDB.GetNode(trimmed)
	if err != nil || node == nil {
		return trimmed
	}

	if strings.EqualFold(node.Type, "scalar") {
		return appendInstanceSuffix(node.OID)
	}

	return trimmed
}

// appendInstanceSuffix aggiunge `.0` ad un OID, gestendo eventuali punti finali.
func appendInstanceSuffix(oid string) string {
	cleaned := strings.TrimSpace(oid)
	if cleaned == "" {
		return cleaned
	}

	if strings.HasSuffix(cleaned, ".0") {
		return cleaned
	}

	if strings.HasSuffix(cleaned, ".") {
		return cleaned + "0"
	}

	return cleaned + ".0"
}
