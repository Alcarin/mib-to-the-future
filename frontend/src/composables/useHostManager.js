/**
 * @file Composable for managing host state, including loading, saving, and deleting hosts.
 */
import { ref } from 'vue';
import { ListHosts, DeleteHost } from '../../wailsjs/go/app/App';
import { toTimestamp, coercePort } from '../utils/formatters';

// Default host state factory
const createDefaultHost = () => ({
  address: '127.0.0.1',
  port: 161,
  community: 'public',
  writeCommunity: 'private',
  version: 'v2c',
  contextName: '',
  securityLevel: '',
  securityUsername: '',
  authProtocol: '',
  authPassword: '',
  privProtocol: '',
  privPassword: '',
  lastUsedAt: '',
  createdAt: ''
});

const resolveWriteCommunity = (raw = {}) => {
  const candidate = raw?.writeCommunity;
  if (typeof candidate === 'string' && candidate.trim() === '') {
    return raw?.community ?? 'public';
  }
  if (candidate === undefined || candidate === null) {
    return raw?.community ?? 'public';
  }
  return candidate;
};

// Private helper functions
const normalizeHostRecord = (raw = {}) => ({
  address: raw?.address ?? '',
  port: coercePort(raw?.port, 161),
  community: raw?.community ?? 'public',
  writeCommunity: resolveWriteCommunity(raw),
  version: raw?.version ?? 'v2c',
  contextName: raw?.contextName ?? '',
  securityLevel: raw?.securityLevel ?? '',
  securityUsername: raw?.securityUsername ?? '',
  authProtocol: raw?.authProtocol ?? '',
  authPassword: raw?.authPassword ?? '',
  privProtocol: raw?.privProtocol ?? '',
  privPassword: raw?.privPassword ?? '',
  lastUsedAt: raw?.lastUsedAt ?? '',
  createdAt: raw?.createdAt ?? ''
});

const sortHostsByRecency = (hosts) =>
  [...hosts].sort((a, b) => toTimestamp(b.lastUsedAt) - toTimestamp(a.lastUsedAt));

// Global state for hosts, shared across the app
const host = ref(createDefaultHost());
const savedHosts = ref([]);
let hasInitializedHost = false;

/**
 * Composable for managing host state and interactions.
 * @returns {object} Reactive state and methods for host management.
 */
export function useHostManager() {
  const loadSavedHosts = async () => {
    try {
      const hosts = await ListHosts();
      const normalized = Array.isArray(hosts) ? hosts.map(normalizeHostRecord) : [];
      savedHosts.value = sortHostsByRecency(normalized);

      if (savedHosts.value.length > 0) {
        if (!hasInitializedHost) {
          const target = savedHosts.value[0];
          host.value = {
            ...createDefaultHost(),
            ...target
          };
          hasInitializedHost = true;
        }
      } else {
        host.value = createDefaultHost();
        hasInitializedHost = false;
      }
    } catch (error) {
      console.error('Failed to load saved hosts:', error);
      savedHosts.value = [];
      host.value = createDefaultHost();
      hasInitializedHost = false;
    }
  };

  const handleDeleteHost = async (address) => {
    const trimmed = address?.trim?.();
    if (!trimmed) {
      return;
    }

    try {
      await DeleteHost(trimmed);
    } catch (error) {
      console.error('Failed to delete saved host:', error);
    }

    await loadSavedHosts();
  };

  return {
    host,
    savedHosts,
    loadSavedHosts,
    handleDeleteHost,
    createDefaultHost, // Espone anche la factory per riuso esterno
  };
}
