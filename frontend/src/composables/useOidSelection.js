/**
 * @file Composable for managing the currently selected OID and SNMP operation.
 */
import { ref } from 'vue';
import { canonicalTreeOid } from '../utils/snmp';

// Global state for OID selection
const selectedOid = ref('');
const selectedOperation = ref('get');

/**
 * Composable for managing the shared state of OID selection.
 * @returns {object} Reactive state and methods for OID management.
 */
export function useOidSelection() {
  /**
   * Updates the selected OID in the application state.
   * @param {string} oid - The OID selected by the user, usually from the MIB tree.
   */
  const handleOidSelect = (oid) => {
    selectedOid.value = oid;
  };

  /**
   * Selects an OID from a log entry, converting it to the canonical tree format.
   * @param {{oid: string}} payload - Data related to the selected entry.
   */
  const handleLogEntrySelect = (payload) => {
    if (!payload?.oid) return;
    selectedOid.value = canonicalTreeOid(payload.oid);
  };

  return {
    selectedOid,
    selectedOperation,
    handleOidSelect,
    handleLogEntrySelect,
  };
}
