package app

import (
	"fmt"
	"strings"

	"mib-to-the-future/backend/mib"
	"mib-to-the-future/backend/snmp"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// normalizeOIDKey normalizza un OID rimuovendo i punti iniziali e gli spazi.
func normalizeOIDKey(oid string) string {
	key := strings.TrimSpace(oid)
	for strings.HasPrefix(key, ".") {
		key = strings.TrimPrefix(key, ".")
	}
	return key
}

// splitSegments divide un OID nei suoi segmenti numerici.
func splitSegments(oid string) []string {
	norm := normalizeOIDKey(oid)
	if norm == "" {
		return nil
	}
	return strings.Split(norm, ".")
}

// lookupNodeForOID cerca il nodo MIB corrispondente a un OID, usando la cache.
func (a *App) lookupNodeForOID(oid string) *mib.Node {
	if a.mibDB == nil {
		return nil
	}
	normalized := normalizeOIDKey(oid)
	if normalized == "" {
		return nil
	}

	a.oidNameCacheM.RLock()
	if cached, ok := a.oidNodeCache[normalized]; ok {
		a.oidNameCacheM.RUnlock()
		return cached
	}
	a.oidNameCacheM.RUnlock()

	segments := splitSegments(normalized)
	for len(segments) > 0 {
		candidate := strings.Join(segments, ".")
		if node, err := a.mibDB.GetNode(candidate); err == nil && node != nil {
			a.oidNameCacheM.Lock()
			a.oidNodeCache[normalized] = node
			a.oidNameCacheM.Unlock()
			return node
		}
		segments = segments[:len(segments)-1]
	}

	a.oidNameCacheM.Lock()
	a.oidNodeCache[normalized] = nil
	a.oidNameCacheM.Unlock()
	return nil
}

// cacheBaseName memorizza nella cache il nome base di un OID.
func (a *App) cacheBaseName(key, name string) {
	if key == "" || name == "" {
		return
	}
	normalized := normalizeOIDKey(key)
	if normalized == "" {
		return
	}
	a.oidNameCacheM.Lock()
	a.oidBaseCache[normalized] = name
	a.oidNameCacheM.Unlock()
}

// cacheResolvedName memorizza nella cache il nome risolto per uno o più OID.
func (a *App) cacheResolvedName(label string, keys ...string) {
	if label == "" && len(keys) == 0 {
		return
	}
	a.oidNameCacheM.Lock()
	for _, key := range keys {
		normalized := normalizeOIDKey(key)
		if normalized == "" {
			continue
		}
		a.oidNameCache[normalized] = label
	}
	a.oidNameCacheM.Unlock()
}

// getBaseName recupera il nome base dalla cache.
func (a *App) getBaseName(key string) (string, bool) {
	normalized := normalizeOIDKey(key)
	if normalized == "" {
		return "", false
	}
	a.oidNameCacheM.RLock()
	name, ok := a.oidBaseCache[normalized]
	a.oidNameCacheM.RUnlock()
	return name, ok
}

// getResolvedName recupera il nome risolto dalla cache.
func (a *App) getResolvedName(key string) (string, bool) {
	normalized := normalizeOIDKey(key)
	if normalized == "" {
		return "", false
	}
	a.oidNameCacheM.RLock()
	name, ok := a.oidNameCache[normalized]
	a.oidNameCacheM.RUnlock()
	return name, ok
}

// resolveOIDName risolve un OID numerico nel suo nome simbolico (es. 1.3.6.1.2.1.1.5 -> sysName).
func (a *App) resolveOIDName(oid string) string {
	if oid == "" || a.mibDB == nil {
		return ""
	}

	primaryKey := normalizeOIDKey(oid)
	if primaryKey == "" {
		return ""
	}

	if name, ok := a.getResolvedName(primaryKey); ok {
		return name
	}

	segments := strings.Split(primaryKey, ".")
	if len(segments) == 0 {
		return ""
	}

	type candidate struct {
		oid    string
		normal string
		suffix []string
	}

	seen := make(map[string]struct{})
	candidates := []candidate{}
	addCandidate := func(target string, suffix []string) {
		norm := normalizeOIDKey(target)
		if norm == "" {
			return
		}
		if _, exists := seen[norm]; exists {
			return
		}
		seen[norm] = struct{}{}
		candidates = append(candidates, candidate{
			oid:    norm,
			normal: norm,
			suffix: suffix,
		})
	}

	addCandidate(primaryKey, []string{})
	if strings.HasPrefix(oid, ".") {
		addCandidate(strings.TrimPrefix(oid, "."), []string{})
	}

	for length := len(segments); length > 0; length-- {
		prefixSegments := segments[:length]
		suffixSegments := segments[length:]
		prefix := strings.Join(prefixSegments, ".")
		addCandidate(prefix, suffixSegments)
		addCandidate("."+prefix, suffixSegments)
	}

	formatLabel := func(base string, suffix []string) string {
		if base == "" {
			return ""
		}
		if len(suffix) == 0 {
			return base
		}
		if len(suffix) == 1 && suffix[0] == "0" {
			return base
		}
		return fmt.Sprintf("%s[%s]", base, strings.Join(suffix, "."))
	}

	for _, cand := range candidates {
		if base, ok := a.getBaseName(cand.normal); ok && base != "" {
			label := formatLabel(base, cand.suffix)
			a.cacheResolvedName(label, primaryKey)
			return label
		}
	}

	var lastErr error

	for _, cand := range candidates {
		node, err := a.mibDB.GetNode(cand.oid)
		if err != nil {
			lastErr = err
			continue
		}
		if node == nil || node.Name == "" {
			continue
		}
		base := node.Name
		a.cacheBaseName(cand.normal, base)
		label := formatLabel(base, cand.suffix)
		a.cacheResolvedName(label, primaryKey)
		return label
	}

	ancestors, err := a.mibDB.GetNodeAncestors(primaryKey)
	if err != nil {
		lastErr = err
	} else {
		for _, ancestor := range ancestors {
			if ancestor == nil || ancestor.Name == "" {
				continue
			}
			ancestorKey := normalizeOIDKey(ancestor.OID)
			a.cacheBaseName(ancestorKey, ancestor.Name)
			ancestorSegments := splitSegments(ancestorKey)
			suffix := []string{}
			if len(ancestorSegments) < len(segments) {
				suffix = segments[len(ancestorSegments):]
			}
			label := formatLabel(ancestor.Name, suffix)
			a.cacheResolvedName(label, primaryKey)
			return label
		}
	}

	if lastErr != nil {
		runtime.LogDebug(a.ctx, fmt.Sprintf("resolveOIDName fallback for %s: %v", primaryKey, lastErr))
	}
	a.cacheResolvedName("", primaryKey)
	return ""
}

// enrichResult arricchisce un risultato SNMP con il nome risolto dell'OID.
func (a *App) enrichResult(result *snmp.Result) {
	if result == nil {
		return
	}
	name := a.resolveOIDName(result.OID)
	result.ResolvedName = name
	a.decorateResultValue(result)
}

// decorateResultValue formatta il valore di un risultato SNMP usando le informazioni MIB.
func (a *App) decorateResultValue(result *snmp.Result) {
	if result == nil {
		return
	}

	raw := result.Value
	result.RawValue = raw
	result.DisplayValue = raw

	node := a.lookupNodeForOID(result.OID)
	if node != nil {
		if node.Syntax != "" {
			result.Syntax = node.Syntax
		}
		if formatted, ok := formatValueWithSyntax(raw, result.Type, node); ok {
			result.DisplayValue = formatted
		}
	}
}
