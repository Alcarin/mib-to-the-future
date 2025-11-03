<script setup>
/**
 * @vue-component
 * @description A component that displays SNMP table data in a sortable and filterable table.
 *
 * @vue-prop {Object} tabInfo - Information about the current tab, including the table OID, data, columns, and lastUpdated timestamp.
 * @vue-prop {Object} hostConfig - Current SNMP host configuration used to query table data.
 *
 * @vue-event {object} data-updated - Fired when table data has been successfully fetched. Payload contains { columns, rows }.
 */
import { ref, computed, onMounted, watch } from 'vue'
import '@material/web/iconbutton/icon-button.js'
import '@material/web/button/filled-button.js'
import '@material/web/textfield/outlined-text-field.js'
import { FetchTableData, SaveCSVFile } from '../../wailsjs/go/app/App'
import { useErrorHandler } from '../composables/useErrorHandler'

const props = defineProps({
  tabInfo: {
    type: Object,
    default: () => ({})
  },
  hostConfig: {
    type: Object,
    default: () => null
  }
})

const emit = defineEmits(['data-updated'])

const { handleError, lastError } = useErrorHandler()

const NUMERIC_COLUMN_HINTS = [
  'number',
  'integer',
  'counter',
  'gauge',
  'timeticks',
  'unsigned',
  'float',
  'double',
  'numeric'
]

const loading = ref(false)
const sortColumn = ref('')
const sortDirection = ref('asc')
const searchQuery = ref('')
let requestToken = 0

const hostSnapshot = computed(() => props.tabInfo?.hostSnapshot || props.hostConfig || null)

const hostLabel = computed(() => {
  const snapshot = hostSnapshot.value
  if (!snapshot || !snapshot.address) {
    return ''
  }
  const port = snapshot.port ? `:${snapshot.port}` : ''
  const version = snapshot.version ? ` · ${snapshot.version}` : ''
  return `${snapshot.address}${port}${version}`
})

const tableName = computed(() => {
  if (props.tabInfo?.displayName) {
    return props.tabInfo.displayName
  }
  const title = props.tabInfo?.title || ''
  if (title.toLowerCase().startsWith('table:')) {
    const [, ...rest] = title.split(':')
    const trimmed = rest.join(':').trim()
    if (trimmed) {
      return trimmed
    }
  }
  if (title) {
    return title
  }
  return props.tabInfo?.oid || 'MIB Table'
})

const numericOid = computed(() => props.tabInfo?.oid || '')

const coercePort = (value) => {
  const parsed = Number.parseInt(value ?? '', 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 161
}

const isValueEmpty = (value) => value === null || value === undefined || value === ''

const parseFiniteNumber = (value) => {
  if (typeof value === 'number') {
    return Number.isFinite(value) ? value : null
  }
  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (trimmed === '') {
      return null
    }
    const parsed = Number(trimmed)
    return Number.isFinite(parsed) ? parsed : null
  }
  return null
}

const isLikelyNumericColumn = (column) => {
  if (!column) {
    return false
  }
  const type = String(column.type ?? '').toLowerCase()
  const syntax = String(column.syntax ?? '').toLowerCase()

  return NUMERIC_COLUMN_HINTS.some((hint) => type.includes(hint) || syntax.includes(hint))
}

const compareTableValues = (aVal, bVal, column) => {
  const aEmpty = isValueEmpty(aVal)
  const bEmpty = isValueEmpty(bVal)

  if (aEmpty && bEmpty) {
    return 0
  }
  if (aEmpty) {
    return 1
  }
  if (bEmpty) {
    return -1
  }

  const numericColumn = isLikelyNumericColumn(column)
  const aNum = parseFiniteNumber(aVal)
  const bNum = parseFiniteNumber(bVal)

  if (numericColumn || (aNum !== null && bNum !== null)) {
    if (aNum !== null && bNum !== null) {
      return aNum - bNum
    }
  }

  const aStr = String(aVal ?? '')
  const bStr = String(bVal ?? '')
  return aStr.localeCompare(bStr, undefined, { numeric: true, sensitivity: 'base' })
}

const buildSnmpConfig = () => {
  const host = hostSnapshot.value ?? {}
  const address = host.address?.trim?.()
  if (!address) {
    return null
  }
  const config = {
    host: address,
    port: coercePort(host.port),
    community: host.community || 'public',
    version: host.version || 'v2c',
    contextName: host.contextName ?? '',
    securityLevel: host.securityLevel ?? '',
    securityUsername: host.securityUsername ?? '',
    authProtocol: host.authProtocol ?? '',
    authPassword: host.authPassword ?? '',
    privProtocol: host.privProtocol ?? '',
    privPassword: host.privPassword ?? ''
  }

  if ((config.version ?? '').toLowerCase() !== 'v3') {
    config.contextName = ''
    config.securityLevel = ''
    config.securityUsername = ''
    config.authProtocol = ''
    config.authPassword = ''
    config.privProtocol = ''
    config.privPassword = ''
  }

  return config
}

const filteredData = computed(() => {
  let result = props.tabInfo?.data || []
  const cols = props.tabInfo?.columns || []

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(row =>
      Object.values(row).some(val =>
        String(val ?? '').toLowerCase().includes(query)
      )
    )
  }

  if (sortColumn.value) {
    const columnDef = cols.find(column => column?.key === sortColumn.value)
    result = [...result].sort((a, b) => {
      const aVal = a?.[sortColumn.value]
      const bVal = b?.[sortColumn.value]
      const comparison = compareTableValues(aVal, bVal, columnDef)
      return sortDirection.value === 'asc' ? comparison : -comparison
    })
  }

  return result
})

const hasExportableData = computed(() => {
  const columns = Array.isArray(props.tabInfo?.columns) ? props.tabInfo.columns : []
  return columns.length > 0 && filteredData.value.length > 0
})

/**
 * @function escapeCsvValue
 * @description Converte un valore in una cella CSV rispettando le regole di escaping.
 * @param {*} value - Il valore da convertire.
 * @returns {string} La rappresentazione sicura per il CSV.
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
 * @description Restituisce un nome file valido rimuovendo caratteri non permessi.
 * @param {string} input - La base del nome file.
 * @returns {string} Il nome file sanificato.
 */
const sanitizeFileName = (input) => {
  if (!input) {
    return 'table'
  }
  return input.replace(/[^a-z0-9-_]+/gi, '-').replace(/^-+|-+$/g, '').toLowerCase() || 'table'
}

/**
 * @function sortBy
 * @description Imposta la colonna e la direzione di ordinamento per la tabella.
 * @param {string} column - La chiave della colonna da ordinare.
 */
const sortBy = (column) => {
  if (!column) {
    return
  }
  if (sortColumn.value === column) {
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortColumn.value = column
    sortDirection.value = 'asc'
  }
}

/**
 * @function loadTableData
 * @description Carica i dati della tabella SNMP richiedendoli al backend tramite Wails e emette un evento con i dati.
 * @async
 */
const loadTableData = async () => {
  const oid = props.tabInfo?.oid
  if (!oid) {
    handleError('Invalid table OID', 'Configuration Error')
    emit('data-updated', { columns: [], rows: [] })
    return
  }

  const config = buildSnmpConfig()
  if (!config) {
    handleError('Invalid SNMP configuration: host, port, community, and version must be specified.', 'Configuration Error')
    emit('data-updated', { columns: [], rows: [] })
    return
  }

  const currentToken = ++requestToken
  loading.value = true

  try {
    const response = await FetchTableData(config, oid)
    if (currentToken !== requestToken) {
      return
    }

    const receivedColumns = Array.isArray(response?.columns) ? response.columns : []
    const receivedRows = Array.isArray(response?.rows) ? response.rows : []

    emit('data-updated', { columns: receivedColumns, rows: receivedRows })

    if (!receivedColumns.some(column => column.key === sortColumn.value)) {
      sortColumn.value = ''
      sortDirection.value = 'asc'
    }
  } catch (err) {
    if (currentToken === requestToken) {
      handleError(err, 'Failed to fetch table data')
      emit('data-updated', { columns: [], rows: [] })
    }
  } finally {
    if (currentToken === requestToken) {
      loading.value = false
    }
  }
}

/**
 * @function refreshTable
 * @description Ricarica i dati della tabella.
 */
const refreshTable = () => {
  loadTableData()
}

/**
 * @function exportTable
 * @description Esporta i dati della tabella visualizzata in un file CSV.
 */
const exportTable = async () => {
  if (!hasExportableData.value) {
    return
  }

  const cols = props.tabInfo.columns
  const headers = cols.map(col => escapeCsvValue(col.label ?? col.key ?? '')).join(',')
  const rows = filteredData.value.map(row =>
    cols.map(col => escapeCsvValue(row[col.key])).join(',')
  )

  const csv = [headers, ...rows].join('\n')
  const baseName = sanitizeFileName(props.tabInfo?.title || props.tabInfo?.oid || 'table')

  try {
    await SaveCSVFile(`${baseName}-${Date.now()}.csv`, csv)
  } catch (err) {
    handleError(err, 'Failed to save CSV file')
  }
}

/**
 * @function formatValue
 * @description Restituisce un valore pronto per la UI senza applicare euristiche arbitrarie.
 * Quando la sintassi (`syntax`) indica TimeTicks viene eseguita una conversione leggibile,
 * altrimenti il valore originale viene mostrato così com'è.
 * @param {*} value - Il valore da formattare.
 * @param {object} column - La colonna associata, usata per leggere metadati come la syntax.
 * @returns {string} Il valore formattato o la stringa originale.
 */
const formatValue = (value, column) => {
  if (value === null || value === undefined || value === '') {
    return '—'
  }

  if (column?.syntax && column.syntax.toLowerCase().includes('timeticks')) {
    const numericValue = typeof value === 'number' ? value : Number(value)
    if (Number.isFinite(numericValue)) {
      const millis = numericValue * 10
      const seconds = Math.floor(millis / 1000) % 60
      const minutes = Math.floor(millis / (1000 * 60)) % 60
      const hours = Math.floor(millis / (1000 * 60 * 60)) % 24
      const days = Math.floor(millis / (1000 * 60 * 60 * 24))
      const parts = [
        days > 0 ? `${days}d` : null,
        hours > 0 ? `${hours}h` : null,
        minutes > 0 ? `${minutes}m` : null,
        `${seconds}s`,
      ].filter(Boolean)
      return parts.join(' ')
    }
  }

  return String(value)
}

watch(
  () => props.tabInfo?.oid,
  (newOid, oldOid) => {
    if (newOid && newOid !== oldOid) {
      // OID changed, always reload
      loadTableData()
    }
  }
)

onMounted(() => {
  // Load data only if it's not already present in the tab state
  if (!props.tabInfo?.data || props.tabInfo.data.length === 0) {
    loadTableData()
  }
})
</script>

<template>
  <div class="table-tab">
    <!-- Toolbar -->
    <div class="table-toolbar">
      <div class="toolbar-left">
        <div class="title-group">
          <h3 class="table-title">{{ tableName }}</h3>
          <span v-if="numericOid" class="table-oid">{{ numericOid }}</span>
        </div>
        <span v-if="hostLabel" class="table-host">Host: {{ hostLabel }}</span>
      </div>
      <div class="toolbar-center">
        <span v-if="tabInfo?.lastUpdated" class="last-updated">
          Last updated: {{ new Date(tabInfo.lastUpdated).toLocaleString() }}
        </span>
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
        <md-icon-button @click="refreshTable" title="Refresh table">
          <span class="material-symbols-outlined">refresh</span>
        </md-icon-button>
        <md-icon-button
          @click="exportTable"
          :disabled="!hasExportableData"
          title="Export to CSV"
        >
          <span class="material-symbols-outlined">download</span>
        </md-icon-button>
      </div>
    </div>

    <!-- Table Content -->
    <div class="table-content">
      <!-- Loading State -->
      <div v-if="loading" class="loading-state">
        <div class="spinner"></div>
        <p>Loading table data...</p>
      </div>

      <!-- Error State -->
      <div v-else-if="lastError && lastError.message" class="error-state">
        <span class="material-symbols-outlined error-icon">error</span>
        <p class="error-text">{{ lastError.message }}</p>
        <md-filled-button @click="refreshTable">Retry</md-filled-button>
      </div>

      <!-- Data Table -->
      <table v-else-if="filteredData.length > 0" class="data-table">
        <thead>
          <tr>
            <th
              v-for="column in tabInfo.columns"
              :key="column.key"
              @click="sortBy(column.key)"
              class="sortable-header"
            >
              <div class="header-content">
                <span>{{ column.label }}</span>
                <span v-if="sortColumn === column.key" class="material-symbols-outlined sort-icon">
                  {{ sortDirection === 'asc' ? 'arrow_upward' : 'arrow_downward' }}
                </span>
              </div>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, index) in filteredData" :key="row.__instance || index" class="data-row">
            <td v-for="column in tabInfo.columns" :key="column.key">
              <span>{{ formatValue(row[column.key], column) }}</span>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Empty State -->
      <div v-else class="empty-state">
        <span class="material-symbols-outlined empty-icon">table_chart</span>
        <p class="empty-text">No data available</p>
        <p class="empty-subtext">Click the refresh button to load data</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.table-tab {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.table-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-md);
  padding: var(--spacing-sm) var(--spacing-md);
  background-color: var(--md-sys-color-surface-container-low);
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: var(--spacing-lg);
  flex-wrap: wrap;
  flex-shrink: 0;
}

.title-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.toolbar-center {
  display: flex;
  justify-content: center;
  flex-grow: 1;
  min-width: 0;
}

.last-updated {
  font-size: 12px;
  color: var(--md-sys-color-on-surface-variant);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.table-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--md-sys-color-on-surface);
  margin: 0;
}

.table-oid {
  font-size: 12px;
  font-family: 'Courier New', monospace;
  color: var(--md-sys-color-on-surface-variant);
  background-color: var(--md-sys-color-surface-container-highest);
  padding: 2px 8px;
  border-radius: var(--border-radius-sm);
}

.table-host {
  font-size: 12px;
  color: var(--md-sys-color-on-surface-variant);
  background-color: var(--md-sys-color-surface-container);
  padding: 2px 10px;
  border-radius: var(--border-radius-sm);
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  flex-shrink: 0;
}

.search-input {
  width: 200px;
}
.table-content {
  flex: 1;
  overflow: auto;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.data-table thead {
  position: sticky;
  top: 0;
  background-color: var(--md-sys-color-surface-container-low);
  z-index: 1;
}

.sortable-header {
  padding: 12px 16px;
  text-align: left;
  font-weight: 600;
  color: var(--md-sys-color-on-surface-variant);
  border-bottom: 2px solid var(--md-sys-color-outline-variant);
  cursor: pointer;
  user-select: none;
}

.sortable-header:hover {
  background-color: var(--md-sys-color-surface-container-highest);
}

.header-content {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  text-transform: uppercase;
  font-size: 11px;
  letter-spacing: 0.5px;
}

.sort-icon {
  font-size: 16px;
  color: var(--md-sys-color-primary);
}

.data-row {
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
  transition: background-color 0.15s ease;
}

.data-row:hover {
  background-color: var(--md-sys-color-surface-container-highest);
}

.data-table td {
  padding: 12px 16px;
  color: var(--md-sys-color-on-surface);
}

.data-table td span {
  display: inline-block;
  white-space: nowrap;
}

.loading-state,
.error-state,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: var(--spacing-xl);
  color: var(--md-sys-color-on-surface-variant);
}

.spinner {
  width: 48px;
  height: 48px;
  border: 4px solid var(--md-sys-color-outline-variant);
  border-top-color: var(--md-sys-color-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: var(--spacing-md);
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error-icon,
.empty-icon {
  font-size: 64px;
  color: var(--md-sys-color-outline);
  margin-bottom: var(--spacing-md);
}

.error-text,
.empty-text {
  font-size: 18px;
  font-weight: 500;
  margin-bottom: var(--spacing-xs);
}

.empty-subtext {
  font-size: 14px;
  color: var(--md-sys-color-outline);
}

.md-btn-filled {
  margin-top: var(--spacing-md);
}

/* Scrollbar styling */
.table-content::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.table-content::-webkit-scrollbar-track {
  background: var(--md-sys-color-surface-container);
}

.table-content::-webkit-scrollbar-thumb {
  background: var(--md-sys-color-outline-variant);
  border-radius: 4px;
}

.table-content::-webkit-scrollbar-thumb:hover {
  background: var(--md-sys-color-outline);
}
</style>
