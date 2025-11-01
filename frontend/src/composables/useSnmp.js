/**
 * @file Composable for handling all SNMP operations and their side effects.
 */
import { nextTick } from 'vue';
import {
  SNMPGet, SNMPGetNext, SNMPWalk, SNMPGetBulk, SNMPSet
} from '../../wailsjs/go/app/App';
import { useHostManager } from './useHostManager';
import { useOidSelection } from './useOidSelection';
import { useTabsManager } from './useTabsManager';
import { useNotifications } from './useNotifications';
import {
  buildLogEntry, createLogEntryId, getResultDisplayValue, getResultRawValue, getResultSyntax, getOriginalNodeType
} from '../utils/snmp';

/**
 * Composable for managing SNMP operations.
 * @param {object} options - Configuration options.
 * @param {Function} options.openSetModalForOid - Function to open the SET modal.
 * @param {Function} options.openOperationInstanceModal - Function to open the instance selection modal for column operations.
 * @returns {object} Methods for executing SNMP operations.
 */
export function useSnmp({ openSetModalForOid, openOperationInstanceModal }) {
  const { host, loadSavedHosts } = useHostManager();
  const { selectedOid, selectedOperation } = useOidSelection();
  const { findTargetLogTab, prependLogEntry, replaceLogEntry } = useTabsManager();
  const { addNotification } = useNotifications();

  const buildSnmpConfig = (useWriteCommunity = false) => {
    const version = host.value.version ?? 'v2c';
    const readCommunity = host.value.community ?? 'public';
    const preferredWrite = host.value.writeCommunity ?? readCommunity;

    const config = {
      host: host.value.address,
      port: host.value.port,
      community: useWriteCommunity && preferredWrite ? preferredWrite : readCommunity,
      writeCommunity: preferredWrite,
      version,
      contextName: host.value.contextName ?? '',
      securityLevel: host.value.securityLevel ?? '',
      securityUsername: host.value.securityUsername ?? '',
      authProtocol: host.value.authProtocol ?? '',
      authPassword: host.value.authPassword ?? '',
      privProtocol: host.value.privProtocol ?? '',
      privPassword: host.value.privPassword ?? ''
    };

    if ((version ?? '').toLowerCase() !== 'v3') {
      config.contextName = ''
      config.securityLevel = ''
      config.securityUsername = ''
      config.authProtocol = ''
      config.authPassword = ''
      config.privProtocol = ''
      config.privPassword = ''
    } else {
      config.writeCommunity = ''
    }

    return config;
  };

  async function handleExecuteSnmpOperation(options = {}) {
    const operation = String(options.operation ?? selectedOperation.value ?? 'get').toLowerCase();
    const targetOid = options.oid ?? selectedOid.value;
    const skipSetModal = options.skipSetModal === true;
    const skipInstanceModal = options.skipInstanceModal === true;
    const fromNode = options.node ?? null;
    const hostLabel = `${host.value.address}:${host.value.port}`;

    if (!targetOid) {
      if (!options.silent) {
        alert('Please select or enter an OID');
      }
      return { error: new Error('OID is required') };
    }

    // Controlla se il nodo è una colonna di tabella e l'operazione è GET o GETNEXT
    const nodeType = fromNode?.type ? String(fromNode.type).toLowerCase() : '';
    const originalType = getOriginalNodeType(nodeType);
    const isColumnOperation = originalType === 'column' && (operation === 'get' || operation === 'getnext');

    if (isColumnOperation && !skipInstanceModal && openOperationInstanceModal) {
      openOperationInstanceModal(fromNode, operation);
      return { modalOpened: true };
    }

    if (operation === 'set' && !skipSetModal) {
      // Se è una colonna, prima chiediamo l'istanza, poi il valore
      if (originalType === 'column' && !skipInstanceModal && openOperationInstanceModal) {
        openOperationInstanceModal(fromNode, operation, options.setPayload);
        return { modalOpened: true };
      }
      await openSetModalForOid(targetOid, fromNode);
      return { modalOpened: true };
    }

    const logTab = findTargetLogTab();
    if (!logTab) {
      console.warn('No log tab available to record the SNMP request');
      return { error: new Error('No log tab available') };
    }

    const payload = options.setPayload ?? null;
    const initialRawValue = operation === 'set' ? (payload?.value ?? null) : null;
    const logEntry = buildLogEntry({
      id: createLogEntryId(),
      timestamp: new Date().toISOString(),
      operation,
      oid: targetOid,
      host: hostLabel,
      status: 'pending',
      value: operation === 'set' ? (payload?.displayValue ?? null) : null,
      rawValue: initialRawValue,
      responseTime: null,
      resolvedName: fromNode?.name ?? null
    });

    let trackedEntry = prependLogEntry(logTab, logEntry);

    try {
      const config = buildSnmpConfig(operation === 'set');
      let result = null;

      switch (operation) {
        case 'get':
          result = await SNMPGet(config, targetOid);
          break;
        case 'getnext':
          result = await SNMPGetNext(config, targetOid);
          break;
        case 'walk': {
          const walkResults = await SNMPWalk(config, targetOid);
          for (let idx = 0; idx < walkResults.length; idx += 1) {
            const item = walkResults[idx];
            const displayValue = getResultDisplayValue(item);
            const rawValue = getResultRawValue(item);
            const enriched = buildLogEntry({
              id: idx === 0 ? trackedEntry.id : createLogEntryId(),
              timestamp: item.timestamp,
              operation: 'walk',
              oid: item.oid,
              host: hostLabel,
              status: 'success',
              value: displayValue,
              rawValue,
              syntax: getResultSyntax(item),
              responseTime: item.responseTime,
              resolvedName: item.resolvedName
            });

            if (idx === 0) {
              trackedEntry = replaceLogEntry(logTab, trackedEntry, enriched);
            } else {
              prependLogEntry(logTab, enriched);
            }
          }
          return { result: walkResults, entry: trackedEntry };
        }
        case 'getbulk': {
          const bulkResults = await SNMPGetBulk(config, targetOid, 10);
          for (let idx = 0; idx < bulkResults.length; idx += 1) {
            const item = bulkResults[idx];
            const displayValue = getResultDisplayValue(item);
            const rawValue = getResultRawValue(item);
            const enriched = buildLogEntry({
              id: idx === 0 ? trackedEntry.id : createLogEntryId(),
              timestamp: item.timestamp,
              operation: 'getbulk',
              oid: item.oid,
              host: hostLabel,
              status: 'success',
              value: displayValue,
              rawValue,
              syntax: getResultSyntax(item),
              responseTime: item.responseTime,
              resolvedName: item.resolvedName
            });

            if (idx === 0) {
              trackedEntry = replaceLogEntry(logTab, trackedEntry, enriched);
            } else {
              prependLogEntry(logTab, enriched);
            }
          }
          return { result: bulkResults, entry: trackedEntry };
        }
        case 'set': {
          if (!payload || !payload.valueType) {
            throw new Error('Missing payload for SNMP SET');
          }
          result = await SNMPSet(config, targetOid, payload.valueType, payload.value);
          break;
        }
        default:
          throw new Error(`Unsupported SNMP operation: ${operation}`);
      }

      if (result) {
        const displayValue = operation === 'set'
          ? (payload?.displayValue ?? getResultDisplayValue(result))
          : getResultDisplayValue(result);
        const rawValue = operation === 'set'
          ? (payload?.value ?? getResultRawValue(result))
          : getResultRawValue(result);
        const enrichedResult = buildLogEntry({
          ...trackedEntry,
          status: result.status ?? 'success',
          value: displayValue,
          rawValue,
          syntax: getResultSyntax(result) ?? fromNode?.syntax ?? trackedEntry.syntax,
          responseTime: result.responseTime,
          oid: result.oid,
          resolvedName: result.resolvedName ?? fromNode?.name ?? trackedEntry.oidName
        });
        trackedEntry = replaceLogEntry(logTab, trackedEntry, enrichedResult);
      } else if (operation === 'set') {
        trackedEntry = replaceLogEntry(logTab, trackedEntry, {
          status: 'success',
          value: payload?.displayValue ?? '',
          rawValue: payload?.value ?? null,
          syntax: fromNode?.syntax ?? trackedEntry.syntax
        });
      }

      if (operation === 'set') {
        addNotification({
          id: createLogEntryId(),
          message: `SET ${targetOid} completed successfully`,
          severity: 'success',
          timeout: 3000
        });
      }

      return { result, entry: trackedEntry };
    } catch (error) {
      trackedEntry = replaceLogEntry(logTab, trackedEntry, {
        status: 'error',
        value: error.toString()
      });
      if (operation === 'set') {
        addNotification({
          id: createLogEntryId(),
          message: `SET ${targetOid} failed: ${error}`,
          severity: 'error'
        });
      }
      console.error('SNMP operation failed:', error);
      return { error };
    } finally {
      await loadSavedHosts();
    }
  }

  const handleContextOperation = async ({ node, operation }) => {
    if (!node || !operation) {
      return;
    }

    selectedOid.value = node.oid;
    selectedOperation.value = operation;

    await nextTick();
    await handleExecuteSnmpOperation({ operation, oid: node.oid, node });
  };

  return {
    handleExecuteSnmpOperation,
    handleContextOperation,
    getOriginalNodeType, // Expose utility for modal manager
  };
}
