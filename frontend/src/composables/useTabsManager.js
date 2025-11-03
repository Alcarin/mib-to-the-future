/**
 * @file Composable for managing the tab system, including different tab types like log, table, and chart.
 */
import { ref, nextTick } from 'vue';
import { useOidSelection } from './useOidSelection';
import { sanitizeInstanceId, buildInstanceOid } from '../utils/snmp';

// Global state for tabs
const tabs = ref([
  { id: 'log-default', title: 'Request Log', type: 'log', data: [] }
]);
const activeTabId = ref('log-default');

const coercePort = (value, fallback = 161) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
};

const unwrapHost = (maybeRef) => {
  if (!maybeRef || typeof maybeRef !== 'object') {
    return null;
  }
  if ('value' in maybeRef && typeof maybeRef.value === 'object' && maybeRef.value !== null) {
    return maybeRef.value;
  }
  return maybeRef;
};

const snapshotHostConfig = (rawHost) => {
  const source = unwrapHost(rawHost);
  if (!source || typeof source !== 'object') {
    return null;
  }

  const address = typeof source.address === 'string'
    ? source.address.trim()
    : typeof source.host === 'string'
      ? source.host.trim()
      : '';

  if (!address) {
    return null;
  }

  const version = typeof source.version === 'string' && source.version
    ? source.version
    : 'v2c';

  const community = typeof source.community === 'string' && source.community
    ? source.community
    : 'public';

  const writeCommunity = typeof source.writeCommunity === 'string' && source.writeCommunity
    ? source.writeCommunity
    : community;

  return {
    address,
    port: coercePort(source.port, 161),
    community,
    writeCommunity,
    version,
    contextName: source.contextName ?? '',
    securityLevel: source.securityLevel ?? '',
    securityUsername: source.securityUsername ?? '',
    authProtocol: source.authProtocol ?? '',
    authPassword: source.authPassword ?? '',
    privProtocol: source.privProtocol ?? '',
    privPassword: source.privPassword ?? ''
  };
};

const buildHostKey = (snapshot) => {
  if (!snapshot) {
    return 'host:unknown';
  }
  const version = (snapshot.version ?? 'v2c').toLowerCase();
  return `host:${snapshot.address}:${snapshot.port}:${version}`;
};

const formatHostLabel = (snapshot) => {
  if (!snapshot) {
    return 'Host sconosciuto';
  }
  const suffix = snapshot.port ? `:${snapshot.port}` : '';
  const version = snapshot.version ? ` · ${snapshot.version}` : '';
  return `${snapshot.address}${suffix}${version}`;
};

/**
 * Composable for managing tab state and interactions.
 * @returns {object} Reactive state and methods for tab management.
 */
export function useTabsManager() {
  const { selectedOid } = useOidSelection();

  const handleAddTab = () => {
    const newTabId = `tab-${Date.now()}`;
    tabs.value.push({
      id: newTabId,
      title: `Log ${tabs.value.length}`,
      type: 'log',
      data: []
    });
    nextTick(() => {
      activeTabId.value = newTabId;
    });
  };

  const handleTabsUpdate = (newTabs) => {
    tabs.value = newTabs;
  };

  const handleCloseTab = (tabId) => {
    if (tabs.value.length === 1) return; // Do not close the last tab

    const index = tabs.value.findIndex(t => t.id === tabId);
    if (index !== -1) {
      tabs.value.splice(index, 1);
      if (activeTabId.value === tabId) {
        activeTabId.value = tabs.value[0].id;
      }
    }
  };

  const handleTableDataUpdate = (tabId, { columns, rows }) => {
    const tab = tabs.value.find(t => t.id === tabId);
    if (tab) {
      tab.data = rows;
      tab.columns = columns;
      tab.lastUpdated = new Date().toISOString();
    }
  };

  const handleChartStateUpdate = (tabId, state) => {
    if (!state) return;
    const tab = tabs.value.find(t => t.id === tabId);
    if (!tab || tab.type !== 'chart') return;

    const samples = Array.isArray(state.samples)
      ? state.samples.map(sample => {
          const cloned = { ...sample };
          if (cloned.derivative === undefined && cloned.perSecond !== undefined) {
            cloned.derivative = cloned.perSecond;
          }
          if (cloned.difference === undefined && cloned.delta !== undefined) {
            cloned.difference = cloned.delta;
          }
          return cloned;
        })
      : [];

    tab.chartState = {
      pollingInterval: Number(state.pollingInterval) > 0 ? Number(state.pollingInterval) : 5,
      isPolling: Boolean(state.isPolling),
      useLogScale: Boolean(state.useLogScale),
      useDifference: Boolean(state.useDifference),
      useDerivative: Boolean(state.useDerivative),
      enforceNonNegative: Boolean(state.enforceNonNegative),
      samples,
      lastRawValue: state.lastRawValue ?? null,
      lastSampleTimestamp: state.lastSampleTimestamp ?? null,
      error: state.error ?? null,
      oid: state.oid ?? tab.oid
    };

    if (tab.instanceId) {
      tab.chartState.instanceId = tab.instanceId;
    }
  };

  const handleOpenTableTab = (oidData, hostLike = null) => {
    const hostSnapshot = snapshotHostConfig(hostLike);
    const hostKey = buildHostKey(hostSnapshot);
    const displayName = oidData?.name || oidData?.label || oidData?.title || oidData?.oid || 'SNMP Table';
    const targetOid = oidData?.oid || '';

    const existingTab = tabs.value.find(
      t => t.type === 'table' && t.oid === targetOid && t.hostKey === hostKey
    );
    if (existingTab) {
      activeTabId.value = existingTab.id;
      return;
    }

    const newTabId = `table-${Date.now()}`;
    tabs.value.push({
      id: newTabId,
      title: `Table: ${displayName}`,
      displayName,
      type: 'table',
      oid: targetOid,
      data: [],
      columns: [],
      lastUpdated: null,
      hostSnapshot,
      hostKey,
      hostLabel: formatHostLabel(hostSnapshot),
      createdAt: new Date().toISOString()
    });
    nextTick(() => {
      activeTabId.value = newTabId;
    });
  };

  const openGraphTab = (node, instanceId = '', hostLike = null) => {
    if (!node || !node.oid) return;

    const hostSnapshot = snapshotHostConfig(hostLike);
    const hostKey = buildHostKey(hostSnapshot);
    const sanitizedInstance = instanceId ? sanitizeInstanceId(instanceId) : '';
    const targetOid = buildInstanceOid(node.oid, sanitizedInstance);
    selectedOid.value = targetOid;

    const existing = tabs.value.find(
      tab => tab.type === 'chart' && tab.oid === targetOid && tab.hostKey === hostKey
    );
    if (existing) {
      activeTabId.value = existing.id;
      return;
    }

    const chartTabId = `chart-${Date.now()}`;
    const displayName = node.name || node.oid;
    const suffix = sanitizedInstance ? ` [${sanitizedInstance}]` : '';

    tabs.value.push({
      id: chartTabId,
      title: `Graph: ${displayName}${suffix}`,
      displayName,
      type: 'chart',
      oid: targetOid,
      baseOid: node.oid,
      instanceId: sanitizedInstance || null,
      syntax: node.syntax || '',
      hostSnapshot,
      hostKey,
      hostLabel: formatHostLabel(hostSnapshot),
      createdAt: new Date().toISOString(),
      chartState: {
        pollingInterval: 5,
        isPolling: false,
        useLogScale: false,
        useDifference: false,
        useDerivative: false,
        enforceNonNegative: false,
        samples: [],
        lastRawValue: null,
        lastSampleTimestamp: null,
        error: null,
        oid: targetOid
      }
    });

    nextTick(() => {
      activeTabId.value = chartTabId;
    });
  };

  const findTargetLogTab = () => {
    const activeLogTab = tabs.value.find(
      tab => tab.id === activeTabId.value && tab.type === 'log'
    );
    if (activeLogTab) {
      return activeLogTab;
    }
    return tabs.value.find(tab => tab.type === 'log') || null;
  };

  const ensureLogArray = (tab) => {
    if (!tab.data || !Array.isArray(tab.data)) {
      tab.data = [];
    }
    return tab.data;
  };

  const prependLogEntry = (tab, entry) => {
    const current = ensureLogArray(tab);
    tab.data = [entry, ...current];
    return entry;
  };

  const replaceLogEntry = (tab, target, patch) => {
    const current = ensureLogArray(tab);
    let index = -1;

    if (target?.id !== undefined) {
      index = current.findIndex(entry => entry?.id === target.id);
    }

    if (index === -1) {
      index = current.indexOf(target);
    }

    if (index === -1) {
      return target;
    }

    const updated = { ...current[index], ...patch };
    const next = [...current];
    next.splice(index, 1, updated);
    tab.data = next;
    return updated;
  };

  const handleRenameTab = (tabId, newTitle) => {
    const tab = tabs.value.find(t => t.id === tabId);
    if (tab && newTitle.trim() !== '') {
      tab.title = newTitle.trim();
    }
  };

  return {
    tabs,
    activeTabId,
    handleAddTab,
    handleTabsUpdate,
    handleCloseTab,
    handleTableDataUpdate,
    handleChartStateUpdate,
    handleOpenTableTab,
    openGraphTab,
    findTargetLogTab,
    prependLogEntry,
    replaceLogEntry,
    handleRenameTab,
  };
}
