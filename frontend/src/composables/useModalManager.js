/**
 * @file Composable for managing the state and interactions of various modals.
 */
import { ref, reactive } from 'vue';
import { GetMIBNode } from '../../wailsjs/go/app/App';
import { useNotifications } from './useNotifications';
import { isWritableNode, createLogEntryId, canonicalTreeOid } from '../utils/snmp';

// Global state for modals
const showMibManager = ref(false);

const graphInstanceModal = reactive({
  visible: false,
  node: null,
  host: null,
});

const setOperationModal = reactive({
  visible: false,
  targetOid: '',
  node: null,
  loadingCurrent: false,
  loadError: '',
  currentValue: null,
});

const operationInstanceModal = reactive({
  visible: false,
  node: null,
  operation: null,
  setPayload: null,
});

/**
 * Composable for managing modal states and their related logic.
 * @param {object} snmpContext - The context from useSnmp, containing handleExecuteSnmpOperation.
 * @param {object} tabsContext - The context from useTabsManager, containing openGraphTab.
 * @returns {object} Reactive state and methods for modal management.
 */
export function useModalManager(snmpContext, tabsContext) {
  const { addNotification } = useNotifications();

  const resetGraphInstanceModal = () => {
    graphInstanceModal.visible = false;
    graphInstanceModal.node = null;
    graphInstanceModal.host = null;
  };

  const resetSetOperationModal = () => {
    setOperationModal.visible = false;
    setOperationModal.targetOid = '';
    setOperationModal.node = null;
    setOperationModal.loadingCurrent = false;
    setOperationModal.loadError = '';
    setOperationModal.currentValue = null;
  };

  const resetOperationInstanceModal = () => {
    operationInstanceModal.visible = false;
    operationInstanceModal.node = null;
    operationInstanceModal.operation = null;
    operationInstanceModal.setPayload = null;
  };

  const handleLoadMib = () => {
    showMibManager.value = true;
  };

  const handleGraphInstanceConfirm = (instanceId) => {
    const node = graphInstanceModal.node;
    const host = graphInstanceModal.host;
    resetGraphInstanceModal();
    if (!node) {
      return;
    }
    tabsContext.openGraphTab(node, instanceId, host);
  };

  const handleGraphInstanceCancel = () => {
    resetGraphInstanceModal();
  };

  const handleOperationInstanceConfirm = async (instanceId) => {
    const node = operationInstanceModal.node;
    const operation = operationInstanceModal.operation;
    const setPayload = operationInstanceModal.setPayload;
    resetOperationInstanceModal();

    if (!node || !operation) {
      return;
    }

    const { buildInstanceOid } = await import('../utils/snmp');
    const fullOid = buildInstanceOid(node.oid, instanceId);

    // Se è un'operazione SET, dobbiamo aprire la modale SET con l'OID completo
    if (operation === 'set') {
      await openSetModalForOid(fullOid, node);
      return;
    }

    // Per GET/GETNEXT, eseguiamo direttamente l'operazione
    await snmpContext.handleExecuteSnmpOperation({
      operation,
      oid: fullOid,
      skipSetModal: true,
      skipInstanceModal: true,
      setPayload,
      node,
    });
  };

  const handleOperationInstanceCancel = () => {
    resetOperationInstanceModal();
  };

  const handleSetModalCancel = () => {
    resetSetOperationModal();
  };

  const handleSetModalConfirm = async (payload) => {
    const targetOid = setOperationModal.targetOid;
    const node = setOperationModal.node;
    resetSetOperationModal();
    if (!targetOid) {
      return;
    }
    await snmpContext.handleExecuteSnmpOperation({
      operation: 'set',
      oid: targetOid,
      skipSetModal: true,
      setPayload: payload,
      node,
    });
  };

  const openSetModalForOid = async (oid, contextNode = null) => {
    resetSetOperationModal();
    const normalizedOid = canonicalTreeOid(oid) ?? oid;
    let node = contextNode;

    if (!node) {
      try {
        node = await GetMIBNode(normalizedOid);
      } catch (error) {
        console.error('Failed to load node metadata for SET:', error);
        addNotification({
          id: createLogEntryId(),
          message: `Unable to load metadata for ${normalizedOid}`,
          severity: 'error',
        });
        return;
      }
    }

    if (!node) {
      addNotification({
        id: createLogEntryId(),
        message: `Node ${normalizedOid} not found in database`,
        severity: 'error',
      });
      return;
    }

    if (!isWritableNode(node)) {
      addNotification({
        id: createLogEntryId(),
        message: `${node.name ?? node.oid} is not writable`,
        severity: 'warning',
        timeout: 4000,
      });
      return;
    }

    setOperationModal.visible = true;
    setOperationModal.targetOid = oid;
    setOperationModal.node = node;
    setOperationModal.loadingCurrent = true;
    setOperationModal.loadError = '';
    setOperationModal.currentValue = null;

    try {
      const response = await snmpContext.handleExecuteSnmpOperation({
        operation: 'get',
        oid,
        skipSetModal: true,
        skipInstanceModal: true,
        node,
        silent: true,
      });

      if (response?.result) {
        setOperationModal.currentValue = response.result;
      } else if (response?.error) {
        setOperationModal.loadError = response.error.message ?? String(response.error);
      }
    } catch (error) {
      setOperationModal.loadError = error.message ?? String(error);
    } finally {
      setOperationModal.loadingCurrent = false;
    }
  };

  const handleOpenGraphRequest = (node, hostLike = null) => {
    if (!node || !node.oid) {
      return;
    }

    const { getOriginalNodeType } = snmpContext; // Assuming this will be in snmpContext
    const nodeType = String(node.type || '').toLowerCase();
    const originalType = getOriginalNodeType(nodeType);

    if (originalType === 'column') {
      graphInstanceModal.node = node;
      graphInstanceModal.host = hostLike ? { ...hostLike } : null;
      graphInstanceModal.visible = true;
      return;
    }

    tabsContext.openGraphTab(node, '', hostLike);
  };

  /**
   * Apre la modale per richiedere l'instance ID quando si esegue GET/SET su una colonna di tabella.
   * @param {object} node - Il nodo MIB della colonna
   * @param {string} operation - L'operazione da eseguire (get o set)
   * @param {object} setPayload - Payload per operazione SET (opzionale)
   */
  const openOperationInstanceModal = (node, operation, setPayload = null) => {
    operationInstanceModal.visible = true;
    operationInstanceModal.node = node;
    operationInstanceModal.operation = operation;
    operationInstanceModal.setPayload = setPayload;
  };

  return {
    showMibManager,
    graphInstanceModal,
    setOperationModal,
    operationInstanceModal,
    handleLoadMib,
    handleGraphInstanceConfirm,
    handleGraphInstanceCancel,
    handleOperationInstanceConfirm,
    handleOperationInstanceCancel,
    handleSetModalConfirm,
    handleSetModalCancel,
    openSetModalForOid,
    openOperationInstanceModal,
    handleOpenGraphRequest,
  };
}
