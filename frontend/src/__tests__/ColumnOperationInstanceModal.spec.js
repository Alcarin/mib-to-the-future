/**
 * @file Test funzionale per la selezione dell'indice quando si eseguono operazioni GET/SET su colonne di tabella
 *
 * Questo test verifica che:
 * 1. Le operazioni GET/GETNEXT su colonne aprono la modale di selezione indice
 * 2. L'operazione SET su colonne apre prima la modale indice, poi la modale valore
 * 3. Le operazioni su scalari non aprono la modale indice
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { nextTick } from 'vue';
import { useModalManager } from '../composables/useModalManager';
import { useSnmp } from '../composables/useSnmp';

// Mock delle dipendenze
vi.mock('../../wailsjs/go/app/App', () => ({
  SNMPGet: vi.fn(),
  SNMPGetNext: vi.fn(),
  SNMPSet: vi.fn(),
  GetMIBNode: vi.fn(),
}));

vi.mock('../composables/useHostManager', () => ({
  useHostManager: () => ({
    host: {
      value: {
        address: '192.168.1.1',
        port: 161,
        community: 'public',
        version: 'v2c',
      },
    },
    loadSavedHosts: vi.fn(),
  }),
}));

vi.mock('../composables/useOidSelection', () => ({
  useOidSelection: () => ({
    selectedOid: { value: null },
    selectedOperation: { value: null },
  }),
}));

vi.mock('../composables/useTabsManager', () => ({
  useTabsManager: () => ({
    findTargetLogTab: () => ({
      id: 'log-default',
      type: 'log',
      data: [],
    }),
    prependLogEntry: vi.fn((tab, entry) => entry),
    replaceLogEntry: vi.fn((tab, old, entry) => entry),
    openGraphTab: vi.fn(),
  }),
}));

vi.mock('../composables/useNotifications', () => ({
  useNotifications: () => ({
    addNotification: vi.fn(),
  }),
}));

describe('Column Operation Instance Modal', () => {
  let modalManager;
  let snmpManager;
  let tabsContext;
  let openSetModalFn;
  let openOperationInstanceModalFn;

  beforeEach(() => {
    // Reset dei mock
    vi.clearAllMocks();

    tabsContext = {
      findTargetLogTab: () => ({
        id: 'log-default',
        type: 'log',
        data: [],
      }),
      prependLogEntry: vi.fn((tab, entry) => entry),
      replaceLogEntry: vi.fn((tab, old, entry) => entry),
      openGraphTab: vi.fn(),
    };

    // Creiamo prima snmpManager con placeholder
    openSetModalFn = vi.fn();
    openOperationInstanceModalFn = vi.fn();

    snmpManager = useSnmp({
      openSetModalForOid: (...args) => openSetModalFn(...args),
      openOperationInstanceModal: (...args) => openOperationInstanceModalFn(...args),
    });

    // Creiamo modalManager
    modalManager = useModalManager(snmpManager, tabsContext);

    // Aggiorniamo i riferimenti per chiamare le funzioni reali del modal manager
    openSetModalFn.mockImplementation((...args) => modalManager.openSetModalForOid(...args));
    openOperationInstanceModalFn.mockImplementation((...args) => modalManager.openOperationInstanceModal(...args));
  });

  describe('Operazione GET su colonna di tabella', () => {
    it('dovrebbe aprire la modale per richiedere l\'instance ID', async () => {
      const columnNode = {
        oid: '1.3.6.1.2.1.2.2.1.10',
        name: 'ifInOctets',
        type: 'column',
        syntax: 'Counter32',
      };

      const result = await snmpManager.handleExecuteSnmpOperation({
        operation: 'get',
        oid: columnNode.oid,
        node: columnNode,
      });

      expect(result.modalOpened).toBe(true);
      expect(modalManager.operationInstanceModal.visible).toBe(true);
      expect(modalManager.operationInstanceModal.node).toStrictEqual(columnNode);
      expect(modalManager.operationInstanceModal.operation).toBe('get');
    });

    it('dovrebbe eseguire GET con l\'OID completo dopo aver confermato l\'instance ID', async () => {
      const { SNMPGet } = await import('../../wailsjs/go/app/App');
      SNMPGet.mockResolvedValue({
        oid: '1.3.6.1.2.1.2.2.1.10.2',
        value: '123456',
        status: 'success',
        responseTime: 10,
      });

      const columnNode = {
        oid: '1.3.6.1.2.1.2.2.1.10',
        name: 'ifInOctets',
        type: 'column',
        syntax: 'Counter32',
      };

      // Prima apriamo la modale
      await snmpManager.handleExecuteSnmpOperation({
        operation: 'get',
        oid: columnNode.oid,
        node: columnNode,
      });

      expect(modalManager.operationInstanceModal.visible).toBe(true);

      // Confermiamo con instance ID "2"
      await modalManager.handleOperationInstanceConfirm('2');

      // Verifica che la modale sia chiusa
      expect(modalManager.operationInstanceModal.visible).toBe(false);

      // Verifica che SNMPGet sia stato chiamato con l'OID completo
      expect(SNMPGet).toHaveBeenCalledWith(
        expect.objectContaining({
          host: '192.168.1.1',
          port: 161,
        }),
        '1.3.6.1.2.1.2.2.1.10.2'
      );
    });
  });

  describe('Operazione GETNEXT su colonna di tabella', () => {
    it('dovrebbe aprire la modale per richiedere l\'instance ID', async () => {
      const columnNode = {
        oid: '1.3.6.1.2.1.2.2.1.10',
        name: 'ifInOctets',
        type: 'column',
        syntax: 'Counter32',
      };

      const result = await snmpManager.handleExecuteSnmpOperation({
        operation: 'getnext',
        oid: columnNode.oid,
        node: columnNode,
      });

      expect(result.modalOpened).toBe(true);
      expect(modalManager.operationInstanceModal.visible).toBe(true);
      expect(modalManager.operationInstanceModal.operation).toBe('getnext');
    });
  });

  describe('Operazione SET su colonna di tabella', () => {
    it('dovrebbe aprire la modale per richiedere l\'instance ID', async () => {
      const columnNode = {
        oid: '1.3.6.1.2.1.2.2.1.7',
        name: 'ifAdminStatus',
        type: 'column',
        syntax: 'INTEGER',
        access: 'read-write',
      };

      const result = await snmpManager.handleExecuteSnmpOperation({
        operation: 'set',
        oid: columnNode.oid,
        node: columnNode,
      });

      expect(result.modalOpened).toBe(true);
      expect(modalManager.operationInstanceModal.visible).toBe(true);
      expect(modalManager.operationInstanceModal.node).toStrictEqual(columnNode);
      expect(modalManager.operationInstanceModal.operation).toBe('set');
    });

    it('dovrebbe aprire la modale SET dopo aver confermato l\'instance ID', async () => {
      const { GetMIBNode, SNMPGet } = await import('../../wailsjs/go/app/App');

      const columnNode = {
        oid: '1.3.6.1.2.1.2.2.1.7',
        name: 'ifAdminStatus',
        type: 'column',
        syntax: 'INTEGER',
        access: 'read-write',
      };

      GetMIBNode.mockResolvedValue(columnNode);
      SNMPGet.mockResolvedValue({
        oid: '1.3.6.1.2.1.2.2.1.7.2',
        value: '1',
        status: 'success',
        responseTime: 10,
      });

      // Prima apriamo la modale instance
      await snmpManager.handleExecuteSnmpOperation({
        operation: 'set',
        oid: columnNode.oid,
        node: columnNode,
      });

      const initiallyVisible = modalManager.operationInstanceModal.visible;
      expect(initiallyVisible).toBe(true);

      // Confermiamo con instance ID "2"
      const confirmPromise = modalManager.handleOperationInstanceConfirm('2');

      // La modale instance deve essere chiusa immediatamente dopo il reset (sincrono)
      expect(modalManager.operationInstanceModal.visible).toBe(false);

      // Aspettiamo che la modale SET venga aperta (asincrono)
      await confirmPromise;
      await nextTick();

      // Verifica che la modale SET sia aperta con l'OID completo
      expect(modalManager.setOperationModal.visible).toBe(true);
      expect(modalManager.setOperationModal.targetOid).toBe('1.3.6.1.2.1.2.2.1.7.2');
      expect(modalManager.setOperationModal.node).toStrictEqual(columnNode);
    });

    it('dovrebbe eseguire SET con l\'OID completo dopo entrambe le modali', async () => {
      const { GetMIBNode, SNMPGet, SNMPSet } = await import('../../wailsjs/go/app/App');

      const columnNode = {
        oid: '1.3.6.1.2.1.2.2.1.7',
        name: 'ifAdminStatus',
        type: 'column',
        syntax: 'INTEGER',
        access: 'read-write',
      };

      GetMIBNode.mockResolvedValue(columnNode);
      SNMPGet.mockResolvedValue({
        oid: '1.3.6.1.2.1.2.2.1.7.2',
        value: '1',
        status: 'success',
        responseTime: 10,
      });
      SNMPSet.mockResolvedValue({
        oid: '1.3.6.1.2.1.2.2.1.7.2',
        value: '2',
        status: 'success',
        responseTime: 15,
      });

      // Step 1: Apriamo la modale instance
      await snmpManager.handleExecuteSnmpOperation({
        operation: 'set',
        oid: columnNode.oid,
        node: columnNode,
      });

      // Step 2: Confermiamo l'instance ID
      await modalManager.handleOperationInstanceConfirm('2');

      // Step 3: Confermiamo il valore nella modale SET
      await modalManager.handleSetModalConfirm({
        value: '2',
        valueType: 'integer',
        displayValue: '2 (up)',
      });

      // Verifica che SNMPSet sia stato chiamato con l'OID completo
      expect(SNMPSet).toHaveBeenCalledWith(
        expect.objectContaining({
          host: '192.168.1.1',
          port: 161,
        }),
        '1.3.6.1.2.1.2.2.1.7.2',
        'integer',
        '2'
      );
    });
  });

  describe('Operazioni su nodi scalari', () => {
    it('NON dovrebbe aprire la modale instance per GET su scalare', async () => {
      // Reset manuale dello stato delle modali
      modalManager.operationInstanceModal.visible = false;
      modalManager.operationInstanceModal.node = null;
      modalManager.operationInstanceModal.operation = null;

      const scalarNode = {
        oid: '1.3.6.1.2.1.1.3.0',
        name: 'sysUpTime',
        type: 'scalar',
        syntax: 'TimeTicks',
      };

      const { SNMPGet } = await import('../../wailsjs/go/app/App');
      SNMPGet.mockResolvedValue({
        oid: '1.3.6.1.2.1.1.3.0',
        value: '123456',
        status: 'success',
        responseTime: 10,
      });

      const result = await snmpManager.handleExecuteSnmpOperation({
        operation: 'get',
        oid: scalarNode.oid,
        node: scalarNode,
      });

      // Verifica che la modale NON sia stata aperta
      expect(result.modalOpened).toBeUndefined();
      expect(modalManager.operationInstanceModal.visible).toBe(false);

      // Verifica che l'operazione sia stata eseguita direttamente
      expect(SNMPGet).toHaveBeenCalled();
    });

    it('dovrebbe aprire solo la modale SET per operazione SET su scalare', async () => {
      // Reset manuale dello stato delle modali
      modalManager.operationInstanceModal.visible = false;
      modalManager.operationInstanceModal.node = null;
      modalManager.operationInstanceModal.operation = null;

      const { GetMIBNode, SNMPGet } = await import('../../wailsjs/go/app/App');

      const scalarNode = {
        oid: '1.3.6.1.2.1.1.5.0',
        name: 'sysName',
        type: 'scalar',
        syntax: 'OCTET STRING',
        access: 'read-write',
      };

      GetMIBNode.mockResolvedValue(scalarNode);
      SNMPGet.mockResolvedValue({
        oid: '1.3.6.1.2.1.1.5.0',
        value: 'router1',
        status: 'success',
        responseTime: 10,
      });

      const result = await snmpManager.handleExecuteSnmpOperation({
        operation: 'set',
        oid: scalarNode.oid,
        node: scalarNode,
      });

      await nextTick();

      // Verifica che sia stata aperta solo la modale SET, non quella instance
      expect(result.modalOpened).toBe(true);
      expect(modalManager.operationInstanceModal.visible).toBe(false);
      expect(modalManager.setOperationModal.visible).toBe(true);
    });
  });

  describe('Gestione bookmark di colonne', () => {
    it('dovrebbe riconoscere un bookmark di colonna e aprire la modale instance', async () => {
      const bookmarkedColumnNode = {
        oid: '1.3.6.1.2.1.2.2.1.10',
        name: 'ifInOctets (Bookmark)',
        type: 'bookmark-column', // Tipo con prefisso bookmark
        syntax: 'Counter32',
      };

      const result = await snmpManager.handleExecuteSnmpOperation({
        operation: 'get',
        oid: bookmarkedColumnNode.oid,
        node: bookmarkedColumnNode,
      });

      expect(result.modalOpened).toBe(true);
      expect(modalManager.operationInstanceModal.visible).toBe(true);
      expect(modalManager.operationInstanceModal.operation).toBe('get');
    });
  });

  describe('Regressione: modale instance non deve riaprirsi', () => {
    it('NON dovrebbe riaprire la modale instance quando openSetModalForOid fa il GET interno', async () => {
      // Reset dello stato
      modalManager.operationInstanceModal.visible = false;
      modalManager.operationInstanceModal.node = null;
      modalManager.operationInstanceModal.operation = null;

      const { GetMIBNode, SNMPGet } = await import('../../wailsjs/go/app/App');

      const columnNode = {
        oid: '1.3.6.1.2.1.2.2.1.7',
        name: 'ifAdminStatus',
        type: 'column',
        syntax: 'INTEGER',
        access: 'read-write',
      };

      GetMIBNode.mockResolvedValue(columnNode);
      SNMPGet.mockResolvedValue({
        oid: '1.3.6.1.2.1.2.2.1.7.2',
        value: '1',
        status: 'success',
        responseTime: 10,
      });

      // Apriamo direttamente la modale SET con un OID completo (come farebbe handleOperationInstanceConfirm)
      await modalManager.openSetModalForOid('1.3.6.1.2.1.2.2.1.7.2', columnNode);
      await nextTick();

      // Verifica che la modale SET sia aperta
      expect(modalManager.setOperationModal.visible).toBe(true);

      // Verifica che la modale instance NON sia stata aperta
      expect(modalManager.operationInstanceModal.visible).toBe(false);

      // Verifica che il GET interno sia stato eseguito con skipInstanceModal
      expect(SNMPGet).toHaveBeenCalledWith(
        expect.objectContaining({
          host: '192.168.1.1',
          port: 161,
        }),
        '1.3.6.1.2.1.2.2.1.7.2'
      );
    });
  });

  describe('Flusso completo con cancellazione', () => {
    it('dovrebbe chiudere la modale instance quando l\'utente annulla', async () => {
      const columnNode = {
        oid: '1.3.6.1.2.1.2.2.1.10',
        name: 'ifInOctets',
        type: 'column',
        syntax: 'Counter32',
      };

      // Apriamo la modale
      await snmpManager.handleExecuteSnmpOperation({
        operation: 'get',
        oid: columnNode.oid,
        node: columnNode,
      });

      expect(modalManager.operationInstanceModal.visible).toBe(true);

      // Annulliamo
      modalManager.handleOperationInstanceCancel();

      // Verifica che sia chiusa
      expect(modalManager.operationInstanceModal.visible).toBe(false);
      expect(modalManager.operationInstanceModal.node).toBe(null);
      expect(modalManager.operationInstanceModal.operation).toBe(null);
    });

    it('dovrebbe gestire correttamente l\'annullamento della modale SET dopo aver confermato l\'instance', async () => {
      const { GetMIBNode, SNMPGet } = await import('../../wailsjs/go/app/App');

      const columnNode = {
        oid: '1.3.6.1.2.1.2.2.1.7',
        name: 'ifAdminStatus',
        type: 'column',
        syntax: 'INTEGER',
        access: 'read-write',
      };

      GetMIBNode.mockResolvedValue(columnNode);
      SNMPGet.mockResolvedValue({
        oid: '1.3.6.1.2.1.2.2.1.7.2',
        value: '1',
        status: 'success',
        responseTime: 10,
      });

      // Apriamo la modale instance
      await snmpManager.handleExecuteSnmpOperation({
        operation: 'set',
        oid: columnNode.oid,
        node: columnNode,
      });

      // Confermiamo l'instance
      await modalManager.handleOperationInstanceConfirm('2');

      expect(modalManager.setOperationModal.visible).toBe(true);

      // Annulliamo la modale SET
      modalManager.handleSetModalCancel();

      expect(modalManager.setOperationModal.visible).toBe(false);
    });
  });
});
