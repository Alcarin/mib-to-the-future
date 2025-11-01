/**
 * @file Utility functions for handling SNMP-related data, such as OIDs and log entries.
 */

/**
 * Removes the leading dot from OIDs returned by gosnmp to align them with the database.
 * @param {string} oid - The OID to normalize.
 * @returns {string|null} The OID without the leading dot, or null if invalid.
 */
export const normalizeOid = (oid) => {
  if (!oid || typeof oid !== 'string') {
    return oid ?? null;
  }
  return oid.startsWith('.') ? oid.slice(1) : oid;
};

/**
 * Returns the OID used by the MIB tree (without instance prefix and suffix).
 * @param {string|null} oid - The OID to convert.
 * @returns {string|null} The canonical OID for the tree.
 */
export const canonicalTreeOid = (oid) => {
  if (!oid) return oid;
  let result = normalizeOid(oid);
  if (!result) return result;
  if (result.endsWith('.0')) {
    result = result.slice(0, -2);
  }
  return result;
};

let logEntryCounter = 0;

/**
 * Generates a unique identifier for a log entry.
 * @returns {string} The log entry identifier.
 */
export const createLogEntryId = () => {
  logEntryCounter += 1;
  return `log-${Date.now()}-${logEntryCounter}`;
};

/**
 * Normalizes the OID and applies the resolved name from the backend (if available).
 * @param {object} baseEntry - Starting data for the log entry.
 * @returns {object} Enriched log entry with name and normalized OID.
 */
export const buildLogEntry = (baseEntry) => {
  const normalizedOid = normalizeOid(baseEntry.oid);
  const resolvedName = baseEntry.resolvedName ?? baseEntry.oidName ?? null;
  return {
    ...baseEntry,
    oid: normalizedOid,
    resolvedName,
    oidName: resolvedName || normalizedOid,
  };
};

/**
 * Extracts the display value from an SNMP result.
 * @param {object} result - The SNMP result object.
 * @returns {any}
 */
export const getResultDisplayValue = (result) => {
  if (!result) return null;
  if (result.displayValue !== undefined && result.displayValue !== null && result.displayValue !== '') {
    return result.displayValue;
  }
  return result.value ?? null;
};

/**
 * Extracts the raw value from an SNMP result.
 * @param {object} result - The SNMP result object.
 * @returns {any}
 */
export const getResultRawValue = (result) => {
  if (!result) return null;
  if (result.rawValue !== undefined && result.rawValue !== null) {
    return result.rawValue;
  }
  return result.value ?? null;
};

/**
 * Extracts the syntax from an SNMP result.
 * @param {object} result - The SNMP result object.
 * @returns {string|null}
 */
export const getResultSyntax = (result) => result?.syntax ?? null;

/**
 * Checks if a MIB node is writable.
 * @param {object} node - The MIB node.
 * @returns {boolean}
 */
export const isWritableNode = (node) => {
  const access = (node?.access || '').toLowerCase();
  return access.includes('write');
};

/**
 * Extracts the original type from a bookmark node type.
 * @param {string} type - The node type (e.g., 'bookmark-column', 'column', 'scalar')
 * @returns {string} The original type without the 'bookmark-' prefix
 */
export const getOriginalNodeType = (type) => {
  if (!type) return 'scalar';
  if (type.startsWith('bookmark-')) {
    return type.substring('bookmark-'.length);
  }
  return type === 'bookmark' ? 'scalar' : type;
};

/**
 * Sanitizes an instance ID by trimming whitespace and removing leading/trailing dots.
 * @param {string | null | undefined} value - The instance ID.
 * @returns {string}
 */
export const sanitizeInstanceId = (value) => {
  if (value === null || value === undefined) {
    return '';
  }
  return String(value).trim().replace(/\s+/g, '').replace(/^\.+/, '').replace(/\.+$/, '');
};

/**
 * Builds a full OID from a base OID and an instance ID.
 * @param {string} baseOid - The base OID.
 * @param {string} instanceId - The instance ID.
 * @returns {string}
 */
export const buildInstanceOid = (baseOid, instanceId) => {
  const normalizedBase = (baseOid || '').trim().replace(/\.+$/, '');
  if (!instanceId) {
    return normalizedBase;
  }
  const trimmed = String(instanceId).trim().replace(/\s+/g, '');
  if (trimmed.startsWith(normalizedBase)) {
    return trimmed;
  }
  const cleanedInstance = trimmed.replace(/^\.+/, '');
  return cleanedInstance ? `${normalizedBase}.${cleanedInstance}` : normalizedBase;
};
