<script setup>
/**
 * @vue-component
 * @description The main top bar of the application, containing controls for host configuration,
 * OID input, operation selection, and main actions like executing SNMP queries.
 *
 * @vue-prop {Object} host - The current host configuration object.
 * @vue-prop {String} selectedOid - The currently selected or entered OID.
 * @vue-prop {String} operation - The currently selected SNMP operation.
 *
 * @vue-event {Object} update:host - Emitted when the host configuration is updated.
 * @vue-event {string} update:selectedOid - Emitted when the OID is updated.
 * @vue-event {string} update:operation - Emitted when the SNMP operation is changed.
 * @vue-event {void} execute - Emitted to trigger the execution of the selected SNMP operation.
 * @vue-event {void} load-mib - Emitted to open the MIB manager dialog for loading MIBs.
 */
import { computed, ref, watch, onBeforeUnmount } from 'vue'
import HostSettingsModal from './HostSettingsModal.vue'
import PreferencesModal from './PreferencesModal.vue'
import '@material/web/button/outlined-button.js'
import '@material/web/button/filled-button.js'
import '@material/web/iconbutton/icon-button.js'
import '@material/web/textfield/outlined-text-field.js'
import '@material/web/select/outlined-select.js'
import '@material/web/select/select-option.js'
import '@material/web/menu/menu.js'
import '@material/web/menu/menu-item.js'
import '@material/web/divider/divider.js'

const props = defineProps({
  host: Object,
  selectedOid: String,
  operation: String,
  hostSuggestions: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:host', 'update:selectedOid', 'update:operation', 'execute', 'load-mib', 'delete-host'])

const hostModel = computed(() => props.host ?? {
  address: '',
  port: 161,
  community: 'public',
  version: 'v2c',
  contextName: '',
  securityLevel: '',
  securityUsername: '',
  authProtocol: '',
  authPassword: '',
  privProtocol: '',
  privPassword: ''
})

const sanitizeSuggestion = (entry) => ({
  address: entry?.address ?? '',
  port: Number.parseInt(entry?.port, 10) > 0 ? Number.parseInt(entry.port, 10) : 161,
  community: entry?.community ?? 'public',
  writeCommunity: entry?.writeCommunity ?? entry?.community ?? 'public',
  version: entry?.version ?? 'v2c',
  contextName: entry?.contextName ?? '',
  securityLevel: entry?.securityLevel ?? '',
  securityUsername: entry?.securityUsername ?? '',
  authProtocol: entry?.authProtocol ?? '',
  authPassword: entry?.authPassword ?? '',
  privProtocol: entry?.privProtocol ?? '',
  privPassword: entry?.privPassword ?? '',
  lastUsedAt: entry?.lastUsedAt ?? ''
})

const hostSuggestionsModel = computed(() => {
  if (!Array.isArray(props.hostSuggestions)) {
    return []
  }
  const seen = new Set()
  const result = []
  for (const entry of props.hostSuggestions) {
    const normalized = sanitizeSuggestion(entry)
    if (!normalized.address || seen.has(normalized.address)) {
      continue
    }
    seen.add(normalized.address)
    result.push(normalized)
  }
  return result
})

const showHostSuggestions = ref(false)
const highlightedSuggestionIndex = ref(-1)
let hideSuggestionsTimer = null

const filteredHostSuggestions = computed(() => {
  const base = hostSuggestionsModel.value
  if (base.length === 0) {
    return []
  }

  const query = hostModel.value.address?.toLowerCase?.().trim() ?? ''
  if (!query) {
    return base
  }

  return base.filter((item) => item.address.toLowerCase().includes(query))
})

const shouldShowHostSuggestions = computed(
  () => showHostSuggestions.value && filteredHostSuggestions.value.length > 0
)

const emitHostUpdate = (patch) => {
  emit('update:host', {
    ...hostModel.value,
    ...patch
  })
}

const openHostSuggestions = () => {
  if (filteredHostSuggestions.value.length === 0) {
    showHostSuggestions.value = false
    highlightedSuggestionIndex.value = -1
    return
  }
  showHostSuggestions.value = true
}

const closeHostSuggestions = () => {
  showHostSuggestions.value = false
  highlightedSuggestionIndex.value = -1
}

const cancelHideSuggestions = () => {
  if (hideSuggestionsTimer) {
    clearTimeout(hideSuggestionsTimer)
    hideSuggestionsTimer = null
  }
}

const scheduleHideSuggestions = () => {
  cancelHideSuggestions()
  hideSuggestionsTimer = setTimeout(() => {
    closeHostSuggestions()
  }, 120)
}

const handleHostInput = (event) => {
  emitHostUpdate({ address: event.target.value })
  cancelHideSuggestions()
  openHostSuggestions()
}

const handleHostFocus = () => {
  cancelHideSuggestions()
  openHostSuggestions()
}

const handleHostBlur = () => {
  scheduleHideSuggestions()
}

const selectHostSuggestion = (suggestion) => {
  if (!suggestion?.address) {
    return
  }
  // eslint-disable-next-line no-unused-vars
  const { lastUsedAt, createdAt, ...rest } = suggestion
  emit('update:host', {
    ...hostModel.value,
    ...rest
  })
  closeHostSuggestions()
}

const handleHostKeydown = (event) => {
  if (!shouldShowHostSuggestions.value) {
    if (event.key === 'Escape') {
      closeHostSuggestions()
    }
    return
  }

  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      highlightedSuggestionIndex.value =
        (highlightedSuggestionIndex.value + 1) % filteredHostSuggestions.value.length
      break
    case 'ArrowUp':
      event.preventDefault()
      highlightedSuggestionIndex.value =
        highlightedSuggestionIndex.value <= 0
          ? filteredHostSuggestions.value.length - 1
          : highlightedSuggestionIndex.value - 1
      break
    case 'Enter':
      if (highlightedSuggestionIndex.value >= 0) {
        event.preventDefault()
        selectHostSuggestion(filteredHostSuggestions.value[highlightedSuggestionIndex.value])
      } else {
        closeHostSuggestions()
      }
      break
    case 'Escape':
      event.preventDefault()
      closeHostSuggestions()
      break
    default:
      break
  }
}

const formatHostSuggestionMeta = (suggestion) => {
  const version = suggestion?.version ? suggestion.version.toUpperCase() : 'SNMP'
  const port = suggestion?.port ?? 161
  if ((suggestion?.version ?? '').toLowerCase() === 'v3') {
    const level = suggestion?.securityLevel || 'noAuthNoPriv'
    return `${level} • ${version} • :${port}`
  }
  const community = suggestion?.community ?? 'public'
  const writeCommunity = suggestion?.writeCommunity ?? community
  const label = writeCommunity && writeCommunity !== community
    ? `${community}→${writeCommunity}`
    : community
  return `${label} • ${version} • :${port}`
}

watch(filteredHostSuggestions, (next) => {
  if (next.length === 0) {
    highlightedSuggestionIndex.value = -1
    return
  }
  if (highlightedSuggestionIndex.value >= next.length) {
    highlightedSuggestionIndex.value = -1
  }
})

watch(showHostSuggestions, (isVisible) => {
  if (!isVisible) {
    highlightedSuggestionIndex.value = -1
  }
})

onBeforeUnmount(() => {
  cancelHideSuggestions()
})

const oidModel = computed({
  get: () => props.selectedOid,
  set: (val) => emit('update:selectedOid', val)
})

const operationModel = computed({
  get: () => props.operation,
  set: (val) => emit('update:operation', val)
})

const isSettingsModalOpen = ref(false)
const isPreferencesModalOpen = ref(false)

const operations = [
  { value: 'get', label: 'GET' },
  { value: 'getnext', label: 'GET NEXT' },
  { value: 'getbulk', label: 'GET BULK' },
  { value: 'walk', label: 'WALK' },
  { value: 'set', label: 'SET' }
]

const snmpVersions = [
  { value: 'v1', label: 'SNMPv1' },
  { value: 'v2c', label: 'SNMPv2c' },
  { value: 'v3', label: 'SNMPv3' }
]
</script>

<template>
  <header class="top-bar">
    <div class="top-bar-section top-bar-controls">
      <!-- MIB Management -->
      <div class="control-group">
        <div class="menu-container">
          <md-icon-button id="file-menu-anchor" @click="() => ($refs.fileMenu.open = !$refs.fileMenu.open)" title="File Menu">
            <span class="material-symbols-outlined">menu</span>
          </md-icon-button>
          <md-menu id="file-menu" anchor="file-menu-anchor" ref="fileMenu">
            <md-menu-item @click="emit('load-mib')">
              <div slot="headline">Load MIB</div>
            </md-menu-item>
            <md-divider></md-divider>
            <md-menu-item @click="isPreferencesModalOpen = true">
              <div slot="headline">Preferences</div>
            </md-menu-item>
          </md-menu>
        </div>
      </div>

      <!-- Host Configuration -->
      <div class="control-group host-control">
        <div class="host-input-wrapper">
          <md-outlined-text-field
            label="Host"
            :value="hostModel.address"
            autocomplete="off"
            @input="handleHostInput"
            @focusin="handleHostFocus"
            @focusout="handleHostBlur"
            @keydown="handleHostKeydown"
          >
            <md-icon-button slot="trailing-icon" @click="isSettingsModalOpen = true" title="Host Settings">
              <span class="material-symbols-outlined">settings</span>
            </md-icon-button>
          </md-outlined-text-field>

          <div
            v-if="shouldShowHostSuggestions"
            class="host-suggestions"
            role="listbox"
          >
            <div
              v-for="(suggestion, index) in filteredHostSuggestions"
              :key="suggestion.address"
              class="host-suggestion"
              :class="{ 'is-active': index === highlightedSuggestionIndex }"
              role="option"
              :aria-selected="index === highlightedSuggestionIndex"
              @mouseenter="() => { highlightedSuggestionIndex = index; cancelHideSuggestions() }"
            >
              <button
                type="button"
                class="host-suggestion__select"
                @mousedown.prevent="selectHostSuggestion(suggestion)"
                @focus="highlightedSuggestionIndex = index"
              >
                <span class="host-suggestion__address">{{ suggestion.address }}</span>
                <span class="host-suggestion__meta">{{ formatHostSuggestionMeta(suggestion) }}</span>
              </button>
              <md-icon-button
                class="host-suggestion__delete"
                title="Remove host"
                @click.stop="emit('delete-host', suggestion.address)"
              >
                <span class="material-symbols-outlined">delete</span>
              </md-icon-button>
            </div>
          </div>
        </div>
      </div>

      <!-- OID and Operation -->
      <div class="control-group flex-grow">
        <md-outlined-text-field
          class="flex-grow"
          label="OID"
          :value="oidModel"
          @input="(e) => (oidModel = e.target.value)"
          placeholder="1.3.6.1.2.1.1.1.0"
        ></md-outlined-text-field>

        <md-outlined-select label="Operation" :value="operationModel" @change="(e) => (operationModel = e.target.value)">
            <md-select-option v-for="op in operations" :key="op.value" :value="op.value">
              {{ op.label }}
            </md-select-option>
        </md-outlined-select>

        <md-filled-button class="execute-btn" @click="emit('execute')">
          <span slot="icon" class="material-symbols-outlined">play_arrow</span>
          Execute
        </md-filled-button>
      </div>
    </div>
  </header>

  <HostSettingsModal
    v-model:show="isSettingsModalOpen"
    :host="host"
    @update:host="emit('update:host', $event)"
    :snmp-versions="snmpVersions"
  />

  <PreferencesModal v-model:show="isPreferencesModalOpen" />
</template>

<style scoped>
.top-bar {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-md);
  padding: var(--spacing-sm) var(--spacing-md);
  background-color: var(--md-sys-color-surface-container);
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
  flex-wrap: wrap;
  z-index: 10;
}

.top-bar-section {
  display: flex;
  align-items: center;
}

.top-bar-controls {
  width: 100%;
  gap: var(--spacing-md);
}

.control-group {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.host-control {
  position: relative;
}

.host-input-wrapper {
  position: relative;
  min-width: 220px;
}

.host-suggestions {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  background-color: var(--md-sys-color-surface);
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18);
  padding: 4px 0;
  max-height: 240px;
  overflow-y: auto;
  overflow-x: hidden;
  z-index: 40;
}

.host-suggestion {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 4px;
  border-radius: 10px;
  min-width: 0;
}

.host-suggestion__select {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  padding: 8px 12px;
  background: none;
  border: none;
  cursor: pointer;
  text-align: left;
  font: inherit;
  color: inherit;
  border-radius: 8px;
  min-width: 0;
  overflow: hidden;
}

.host-suggestion__select:focus {
  outline: none;
}

.host-suggestion:hover .host-suggestion__select,
.host-suggestion.is-active .host-suggestion__select {
  background-color: var(--md-sys-color-surface-container-highest);
}

.host-suggestion__delete {
  --md-icon-button-state-layer-size: 34px;
  color: var(--md-sys-color-on-surface-variant);
}

.host-suggestion__delete:hover {
  color: var(--md-sys-color-error);
}

.host-suggestion__address {
  font-weight: 600;
  letter-spacing: 0.2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

.host-suggestion__meta {
  font-size: 0.85rem;
  color: var(--md-sys-color-on-surface-variant);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

.control-group.flex-grow {
  flex: 1;
  min-width: 400px;
}

.flex-grow {
  flex: 1;
}

md-outlined-text-field,
md-outlined-select {
  min-width: 150px;
}

.execute-btn {
  padding-inline: 14px;
}

.menu-container {
  position: relative;
  display: flex;
  align-items: center;
  height: 56px; /* Align with text fields */
}
</style>
