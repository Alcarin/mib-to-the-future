<script setup>
/**
 * @vue-component
 * @description A component that displays SNMP operation logs in a filterable and searchable table.
 *
 * @vue-prop {Array} data - The array of log entries to display.
 * @vue-prop {Object} tabInfo - Information about the current tab.
 */
import { ref, computed, watch } from 'vue'
import '@material/web/chips/filter-chip.js'
import '@material/web/iconbutton/icon-button.js'
import '@material/web/textfield/outlined-text-field.js'
import { SaveCSVFile } from '../../wailsjs/go/app/App'
import { useErrorHandler } from '../composables/useErrorHandler'

const props = defineProps({
  data: Array,
  tabInfo: Object
})

const emit = defineEmits(['entry-select'])

const { handleError } = useErrorHandler()

const filterStatus = ref('all')
const searchQuery = ref('')
const selectedEntryId = ref(null)

const filteredData = computed(() => {
  const entries = props.data ?? []
  const byStatus = filterStatus.value
  const query = searchQuery.value?.toLowerCase()

  return entries.filter(item => {
    if (byStatus !== 'all' && item.status !== byStatus) {
      return false
    }

    if (query) {
      const oid = (item.oid || '').toLowerCase()
      const operation = (item.operation || '').toLowerCase()
      const host = (item.host || '').toLowerCase()
      const name = (item.oidName || item.resolvedName || '').toLowerCase()
      return oid.includes(query) || operation.includes(query) || host.includes(query) || name.includes(query)
    }

    return true
  })
})

const statusCount = computed(() => {
  const entries = props.data ?? []
  const counts = {
    all: entries.length,
    success: 0,
    error: 0,
    pending: 0,
    timeout: 0
  }

  entries.forEach(item => {
    const statusKey = item?.status || 'unknown'
    counts[statusKey] = (counts[statusKey] || 0) + 1
  })

  return counts
})

watch(
  () => props.data,
  (entries) => {
    if (!entries?.some(item => item?.id === selectedEntryId.value)) {
      selectedEntryId.value = null
    }
  },
  { deep: true }
)

/**
 * @function handleRowSelect
 * @description Gestisce la selezione di una voce di log e notifica il componente padre.
 * @param {object} entry - La voce selezionata.
 */
const handleRowSelect = (entry) => {
  if (!entry?.id) return
  selectedEntryId.value = entry.id
  emit('entry-select', {
    oid: entry.oid,
    name: entry.oidName || entry.oid,
    entry
  })
}

/**
 * @function getStatusColor
 * @description Restituisce il colore CSS associato a un determinato stato di log.
 * @param {string} status - Lo stato del log (es. 'success', 'error').
 * @returns {string} La variabile CSS del colore corrispondente.
 */
const getStatusColor = (status) => {
  const colors = {
    success: 'var(--md-sys-color-tertiary)',
    error: 'var(--md-sys-color-error)',
    pending: 'var(--md-sys-color-secondary)',
    timeout: 'var(--md-sys-color-outline)'
  }
  return colors[status] || 'var(--md-sys-color-outline)'
}

/**
 * @function getStatusIcon
 * @description Restituisce il nome dell'icona Material Symbols associata a un determinato stato di log.
 * @param {string} status - Lo stato del log (es. 'success', 'error').
 * @returns {string} Il nome dell'icona.
 */
const getStatusIcon = (status) => {
  const icons = {
    success: 'check_circle',
    error: 'error',
    pending: 'pending',
    timeout: 'schedule'
  }
  return icons[status] || 'help'
}

/**
 * @function formatTimestamp
 * @description Formatta una stringa timestamp ISO in un formato orario leggibile.
 * @param {string} timestamp - La stringa timestamp in formato ISO.
 * @returns {string} L'orario formattato (HH:mm:ss).
 */
const formatTimestamp = (timestamp) => {
  const date = new Date(timestamp)
  return date.toLocaleTimeString('it-IT', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

/**
 * @function clearLog
 * @description Cancella tutte le voci di log dopo aver chiesto conferma all'utente.
 */
const clearLog = () => {
  if (confirm('Clear all log entries?')) {
    props.data.splice(0, props.data.length)
    selectedEntryId.value = null
  }
}

/**
 * @function exportLog
 * @description Esporta le voci di log correnti in un file CSV.
 */
/**
 * @function escapeCsvValue
 * @description Applica l'escaping necessario per inserire il valore in una cella CSV.
 * @param {*} value - Il valore da convertire.
 * @returns {string} Il valore convertito.
 */
const escapeCsvValue = (value) => {
  if (value === null || value === undefined) {
    return ''
  }
  let stringValue = String(value)
  if (stringValue.includes('"')) {
    stringValue = stringValue.replace(/"/g, '""')
  }
  if (/[",\n]/.test(stringValue)) {
    return `"${stringValue}"`
  }
  return stringValue
}

/**
 * @function sanitizeFileName
 * @description Rimuove caratteri non ammessi per generare un nome file valido.
 * @param {string} input - La stringa di partenza.
 * @returns {string} Il nome file sanificato.
 */
const sanitizeFileName = (input) => {
  if (!input) {
    return 'log'
  }
  return input.replace(/[^a-z0-9-_]+/gi, '-').replace(/^-+|-+$/g, '').toLowerCase() || 'log'
}

/**
 * @function exportLog
 * @description Esporta le voci di log correnti in un file CSV utilizzando il bridge Wails.
 */
const exportLog = async () => {
  const entries = props.data || []
  if (!entries.length) {
    return
  }

  const header = 'Timestamp,Operation,OID,Host,Status,Value,Response Time'
  const rows = entries.map(item =>
    [
      escapeCsvValue(item.timestamp),
      escapeCsvValue(item.operation),
      escapeCsvValue(item.oid),
      escapeCsvValue(item.host),
      escapeCsvValue(item.status),
      escapeCsvValue(item.value || ''),
      escapeCsvValue(item.responseTime || '')
    ].join(',')
  )

  const csv = [header, ...rows].join('\n')
  const baseName = sanitizeFileName(props.tabInfo?.title || 'snmp-log')

  try {
    await SaveCSVFile(`${baseName}-${Date.now()}.csv`, csv)
  } catch (err) {
    handleError(err, 'Impossibile salvare il file CSV')
  }
}
</script>

<template>
  <div class="log-tab">
    <!-- Toolbar -->
    <div class="log-toolbar">
      <div class="filter-chips">
        <md-filter-chip
          :label="`All (${statusCount.all})`"
          :selected="filterStatus === 'all'"
          @click="filterStatus = 'all'"
        ></md-filter-chip>
        <md-filter-chip
          v-if="statusCount.success"
          :label="`Success (${statusCount.success})`"
          :selected="filterStatus === 'success'"
          @click="filterStatus = 'success'"
        ></md-filter-chip>
        <md-filter-chip
          v-if="statusCount.error"
          :label="`Error (${statusCount.error})`"
          :selected="filterStatus === 'error'"
          @click="filterStatus = 'error'"
        ></md-filter-chip>
        <md-filter-chip
          v-if="statusCount.pending"
          :label="`Pending (${statusCount.pending})`"
          :selected="filterStatus === 'pending'"
          @click="filterStatus = 'pending'"
        ></md-filter-chip>
      </div>

      <div class="toolbar-actions">
        <md-outlined-text-field
          label="Search"
          v-model="searchQuery"
          type="search"
          class="search-input"
        >
          <span slot="leading-icon" class="material-symbols-outlined">search</span>
        </md-outlined-text-field>
        <md-icon-button @click="exportLog" title="Export to CSV">
          <span class="material-symbols-outlined">download</span>
        </md-icon-button>
        <md-icon-button @click="clearLog" title="Clear log">
          <span class="material-symbols-outlined">delete</span>
        </md-icon-button>
      </div>
    </div>

    <!-- Log Table -->
    <div class="log-content">
      <table v-if="filteredData.length > 0" class="log-table">
        <thead>
          <tr>
            <th class="col-time">Time</th>
            <th class="col-status">Status</th>
            <th class="col-operation">Operation</th>
            <th class="col-name">Name</th>
            <th class="col-host">Host</th>
            <th class="col-value">Value</th>
            <th class="col-response-time">Time (ms)</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="entry in filteredData"
            :key="entry.id || entry.timestamp"
            class="log-row"
            :data-oid="entry.oid"
            :class="{ selected: entry.id === selectedEntryId }"
            :title="entry.oid"
            @click="handleRowSelect(entry)"
          >
            <td class="time-cell">{{ formatTimestamp(entry.timestamp) }}</td>
            <td class="status-cell">
              <span 
                class="status-badge"
                :style="{ backgroundColor: getStatusColor(entry.status) }"
              >
                <span class="material-symbols-outlined">
                  {{ getStatusIcon(entry.status) }}
                </span>
              </span>
            </td>
            <td class="operation-cell">
              <span class="operation-badge">{{ entry.operation.toUpperCase() }}</span>
            </td>
            <td class="name-cell">
              <span class="name-primary">{{ entry.oidName || entry.oid }}</span>
            </td>
            <td class="host-cell">{{ entry.host }}</td>
            <td class="value-cell">
              <span v-if="entry.value">{{ entry.value }}</span>
              <span v-else class="empty-value">—</span>
            </td>
            <td class="time-cell">
              <span v-if="entry.responseTime">{{ entry.responseTime }}</span>
              <span v-else class="empty-value">—</span>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-else class="empty-state">
        <span class="material-symbols-outlined empty-icon">inbox</span>
        <p class="empty-text">No log entries</p>
        <p class="empty-subtext">Execute SNMP operations to see logs here</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.log-tab {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.log-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-md);
  padding: var(--spacing-sm) var(--spacing-md);
  background-color: var(--md-sys-color-surface-container-low);
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
}

.filter-chips {
  display: flex;
  gap: var(--spacing-xs);
  flex-wrap: wrap;
}
.filter-chips md-filter-chip {
  --md-filter-chip-icon-size: 0;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.search-input {
  width: 200px;
}

.log-content {
  flex: 1;
  overflow: auto;
}

.log-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
  table-layout: auto;
}

.log-table thead {
  position: sticky;
  top: 0;
  background-color: var(--md-sys-color-surface-container-low);
  z-index: 1;
}

.log-table th {
  padding: 12px 16px;
  text-align: left;
  font-weight: 600;
  color: var(--md-sys-color-on-surface-variant);
  border-bottom: 2px solid var(--md-sys-color-outline-variant);
  text-transform: uppercase;
  font-size: 11px;
  letter-spacing: 0.5px;
}

/* Colonne adattive al contenuto */
.col-time {
  white-space: nowrap;
}

.col-status {
  width: 1%;
  white-space: nowrap;
  text-align: center;
}

.col-operation {
  white-space: nowrap;
}

.col-name {
  /* Consente di espandersi ma con ellipsis se troppo lunga */
  max-width: 300px;
}

.col-host {
  white-space: nowrap;
}

.col-value {
  /* La colonna value prende tutto lo spazio rimanente disponibile */
  width: 100%;
  min-width: 200px;
}

.col-response-time {
  white-space: nowrap;
  text-align: right;
}

.log-row {
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
  transition: background-color 0.15s ease;
  cursor: pointer;
}

.log-row:hover {
  background-color: var(--md-sys-color-surface-container-highest);
}

.log-row.selected {
  background-color: var(--md-sys-color-secondary-container);
}

.log-table td {
  padding: 12px 16px;
  color: var(--md-sys-color-on-surface);
}

.time-cell {
  font-family: 'Courier New', monospace;
  font-size: 12px;
  color: var(--md-sys-color-on-surface-variant);
}

.status-cell {
  text-align: center;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
}

.status-badge .material-symbols-outlined {
  font-size: 18px;
  color: white;
}

.operation-cell {
  font-weight: 600;
}

.operation-badge {
  display: inline-block;
  padding: 4px 8px;
  background-color: var(--md-sys-color-primary-container);
  color: var(--md-sys-color-on-primary-container);
  border-radius: var(--border-radius-sm);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.5px;
}

.name-cell {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.name-primary {
  font-weight: 500;
  color: var(--md-sys-color-on-surface);
}

.host-cell {
  font-family: 'Courier New', monospace;
  font-size: 12px;
}

.value-cell {
  /* Permette al testo di andare a capo */
  white-space: pre-wrap;
  word-wrap: break-word;
  word-break: break-word;
  max-width: 600px;
  line-height: 1.5;
}

.empty-value {
  color: var(--md-sys-color-outline);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: var(--spacing-xl);
  color: var(--md-sys-color-on-surface-variant);
}

.empty-icon {
  font-size: 64px;
  color: var(--md-sys-color-outline);
  margin-bottom: var(--spacing-md);
}

.empty-text {
  font-size: 18px;
  font-weight: 500;
  margin-bottom: var(--spacing-xs);
}

.empty-subtext {
  font-size: 14px;
  color: var(--md-sys-color-outline);
}

/* Scrollbar styling */
.log-content::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.log-content::-webkit-scrollbar-track {
  background: var(--md-sys-color-surface-container);
}

.log-content::-webkit-scrollbar-thumb {
  background: var(--md-sys-color-outline-variant);
  border-radius: 4px;
}

.log-content::-webkit-scrollbar-thumb:hover {
  background: var(--md-sys-color-outline);
}
</style>
>
