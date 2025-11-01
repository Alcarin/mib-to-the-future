<script setup>
/**
 * @vue-component
 * @description Dialog per la gestione dei moduli MIB: consente di caricare nuovi file,
 * visualizzare quelli presenti e rimuoverli.
 *
 * @vue-event {string} mib-loaded - Emette il nome del modulo MIB caricato o eliminato.
 * Durante un caricamento multiplo l'evento viene emesso una volta per ogni modulo.
 * @vue-event {void} close - Emette la richiesta di chiusura del dialog.
 */
import { ref, onMounted, computed } from 'vue'
import {
  LoadMIBFile,
  ListMIBModules,
  DeleteMIBModule,
  GetMIBStats,
  GetMIBModuleDetails
} from '../../wailsjs/go/app/App'
import { useNotifications } from '../composables/useNotifications'
import { useErrorHandler } from '../composables/useErrorHandler'
import ModuleTreeNode from './ModuleTreeNode.vue'
import '@material/web/button/filled-button.js'
import '@material/web/button/outlined-button.js'
import '@material/web/iconbutton/icon-button.js'
import '@material/web/labs/card/elevated-card.js'
import '@material/web/list/list.js'
import '@material/web/list/list-item.js'

const emit = defineEmits(['mib-loaded', 'close'])

const modules = ref([])
const stats = ref({})
const selectedModuleName = ref('')
const moduleTree = ref([])
const moduleStatsDetails = ref(null)
const moduleMissingImports = ref([])
const moduleDetailsLoading = ref(false)
const moduleDetailsError = ref(null)
const expandedNodes = ref(new Set())
const loading = ref(false)

const { addNotification } = useNotifications()
const { handleError } = useErrorHandler()

const selectedModuleSummary = computed(() => modules.value.find((module) => module.name === selectedModuleName.value) || null)
const moduleStatsComputed = computed(
  () =>
    moduleStatsDetails.value || {
      nodeCount: 0,
      scalarCount: 0,
      tableCount: 0,
      columnCount: 0,
      typeCount: 0,
      skippedNodes: 0,
      missingCount: 0
    }
)
const selectedModulePath = computed(() => selectedModuleSummary.value?.filePath || '')
const hasModuleWarnings = computed(
  () => (moduleStatsComputed.value.skippedNodes ?? 0) > 0 || moduleMissingImports.value.length > 0
)

const resetModuleDetails = () => {
  moduleTree.value = []
  moduleStatsDetails.value = null
  moduleMissingImports.value = []
  moduleDetailsError.value = null
  expandedNodes.value = new Set()
}

const updateModuleSummaryEntry = (name, statsSnapshot, missingImports) => {
  modules.value = modules.value.map((module) => {
    if (module.name !== name) {
      return module
    }
    return {
      ...module,
      nodeCount: statsSnapshot?.nodeCount ?? module.nodeCount,
      scalarCount: statsSnapshot?.scalarCount ?? module.scalarCount,
      tableCount: statsSnapshot?.tableCount ?? module.tableCount,
      columnCount: statsSnapshot?.columnCount ?? module.columnCount,
      typeCount: statsSnapshot?.typeCount ?? module.typeCount,
      skippedNodes: statsSnapshot?.skippedNodes ?? module.skippedNodes,
      missingImports: missingImports ?? module.missingImports
    }
  })
}

const loadModuleDetails = async (name) => {
  if (!name) {
    resetModuleDetails()
    return
  }

  moduleDetailsLoading.value = true
  moduleDetailsError.value = null

  try {
    const details = await GetMIBModuleDetails(name)
    moduleTree.value = Array.isArray(details?.tree) ? details.tree : []
    moduleStatsDetails.value = details?.stats ?? null
    moduleMissingImports.value = Array.isArray(details?.missingImports) ? details.missingImports : []
    expandedNodes.value = new Set(moduleTree.value.map((node) => node.oid))
    updateModuleSummaryEntry(name, details?.stats, moduleMissingImports.value)
  } catch (err) {
    moduleDetailsError.value = err instanceof Error ? err : new Error(String(err))
    resetModuleDetails()
  } finally {
    moduleDetailsLoading.value = false
  }
}

const selectModule = async (name, options = {}) => {
  const forceReload = options.forceReload ?? false
  const normalized = (name || '').trim()

  if (!normalized) {
    selectedModuleName.value = ''
    resetModuleDetails()
    return
  }

  const shouldReload = forceReload || normalized !== selectedModuleName.value
  selectedModuleName.value = normalized

  if (shouldReload) {
    await loadModuleDetails(normalized)
  }
}

const loadModules = async () => {
  try {
    const [moduleList, globalStats] = await Promise.all([ListMIBModules(), GetMIBStats()])
    modules.value = Array.isArray(moduleList) ? moduleList : []
    stats.value = globalStats || {}

    if (modules.value.length === 0) {
      await selectModule('')
      return
    }

    const current = selectedModuleName.value
    const match = modules.value.find((module) => module.name === current)
    const targetName = match ? match.name : modules.value[0].name
    await selectModule(targetName, { forceReload: true })
  } catch (err) {
    handleError(err, 'Failed to load MIB modules')
  }
}

const handleLoadMIB = async () => {
  loading.value = true

  try {
    const moduleNames = await LoadMIBFile()

    if (Array.isArray(moduleNames) && moduleNames.length > 0) {
      await loadModules()
      moduleNames.forEach((moduleName) => emit('mib-loaded', moduleName))

      const successMessage =
        moduleNames.length === 1
          ? `MIB module "${moduleNames[0]}" loaded successfully!`
          : `${moduleNames.length} MIB modules loaded successfully: ${moduleNames.join(', ')}`
      addNotification({ message: successMessage, type: 'success' })
    }
  } catch (err) {
    handleError(err, 'Failed to load MIB file')
  } finally {
    loading.value = false
  }
}

const handleDeleteModule = async (moduleName) => {
  if (!confirm(`Are you sure you want to delete module "${moduleName}"?`)) {
    return
  }

  try {
    await DeleteMIBModule(moduleName)
    await loadModules()
    emit('mib-loaded')
    addNotification({ message: `Module "${moduleName}" deleted successfully!`, type: 'success' })
  } catch (err) {
    handleError(err, `Failed to delete module "${moduleName}"`)
  }
}

const toggleExpandedNode = (oid) => {
  if (!oid) return
  const next = new Set(expandedNodes.value)
  if (next.has(oid)) {
    next.delete(oid)
  } else {
    next.add(oid)
  }
  expandedNodes.value = next
}

onMounted(() => {
  loadModules()
})
</script>

<template>
  <div class="mib-manager-dialog">
    <div class="dialog-header">
      <h2 class="dialog-title">
        <span class="material-symbols-outlined">folder_open</span>
        MIB Manager
      </h2>
      <md-icon-button @click="emit('close')" title="Close">
        <span class="material-symbols-outlined">close</span>
      </md-icon-button>
    </div>

    <div class="dialog-content">
      <div class="stats-section">
        <md-elevated-card class="stat-card">
          <div class="stat-value">{{ stats.modules || 0 }}</div>
          <div class="stat-label">Modules</div>
        </md-elevated-card>
        <md-elevated-card class="stat-card">
          <div class="stat-value">{{ stats.total_nodes || 0 }}</div>
          <div class="stat-label">Total Nodes</div>
        </md-elevated-card>
        <md-elevated-card class="stat-card">
          <div class="stat-value">{{ stats.scalar || 0 }}</div>
          <div class="stat-label">Scalars</div>
        </md-elevated-card>
        <md-elevated-card class="stat-card">
          <div class="stat-value">{{ stats.table || 0 }}</div>
          <div class="stat-label">Tables</div>
        </md-elevated-card>
      </div>

      <div class="actions-section">
        <md-filled-button @click="handleLoadMIB" :disabled="loading">
          <span slot="icon" class="material-symbols-outlined">add</span>
          Load MIB Files
        </md-filled-button>
        <md-outlined-button @click="loadModules">
          <span slot="icon" class="material-symbols-outlined">refresh</span>
          Refresh
        </md-outlined-button>
      </div>

      <div class="dialog-body">
        <md-elevated-card class="module-tree-panel">
          <div class="module-tree-header">
            <div class="module-tree-title">
              <span class="material-symbols-outlined">account_tree</span>
              <div>
                <p class="module-tree-heading">{{ selectedModuleName || 'Select a module' }}</p>
                <p v-if="selectedModulePath" class="module-tree-subheading">{{ selectedModulePath }}</p>
              </div>
            </div>
            <div v-if="selectedModuleName" class="module-metrics">
              <div class="module-metric">
                <span class="metric-label">Nodes</span>
                <span class="metric-value">{{ moduleStatsComputed.nodeCount }}</span>
              </div>
              <div class="module-metric">
                <span class="metric-label">Scalars</span>
                <span class="metric-value">{{ moduleStatsComputed.scalarCount }}</span>
              </div>
              <div class="module-metric">
                <span class="metric-label">Tables</span>
                <span class="metric-value">{{ moduleStatsComputed.tableCount }}</span>
              </div>
              <div class="module-metric">
                <span class="metric-label">Columns</span>
                <span class="metric-value">{{ moduleStatsComputed.columnCount }}</span>
              </div>
              <div class="module-metric">
                <span class="metric-label">Types</span>
                <span class="metric-value">{{ moduleStatsComputed.typeCount }}</span>
              </div>
              <md-outlined-button
                v-if="selectedModuleSummary && selectedModuleSummary.name !== 'BASE'"
                class="module-delete-btn"
                @click="handleDeleteModule(selectedModuleSummary.name)"
              >
                <span slot="icon" class="material-symbols-outlined">delete</span>
                Module
              </md-outlined-button>
            </div>
          </div>

          <div v-if="hasModuleWarnings" class="module-alert">
            <span class="material-symbols-outlined module-alert-icon">warning</span>
            <div class="module-alert-content">
              <p v-if="moduleStatsComputed.skippedNodes">
                {{ moduleStatsComputed.skippedNodes }} nodes skipped due to unresolved dependencies.
              </p>
              <p v-if="moduleMissingImports.length">
                Missing modules: {{ moduleMissingImports.join(', ') }}
              </p>
            </div>
          </div>

          <div class="module-tree-body">
            <div v-if="moduleDetailsLoading" class="module-tree-status">
              <span class="material-symbols-outlined spinner">autorenew</span>
              <p>Loading module tree...</p>
            </div>
            <div v-else-if="moduleDetailsError" class="module-tree-status module-tree-status--error">
              <span class="material-symbols-outlined">error</span>
              <p>{{ moduleDetailsError.message }}</p>
            </div>
            <div v-else-if="!selectedModuleName" class="module-tree-status">
              <span class="material-symbols-outlined">info</span>
              <p>Select a module to inspect its MIB tree.</p>
            </div>
            <div v-else-if="moduleTree.length === 0" class="module-tree-status">
              <span class="material-symbols-outlined">park</span>
              <p>No nodes stored for this module.</p>
            </div>
            <ul v-else class="module-tree-list">
              <ModuleTreeNode
                v-for="node in moduleTree"
                :key="node.oid"
                :node="node"
                :expanded="expandedNodes"
                @toggle="toggleExpandedNode"
              />
            </ul>
          </div>
        </md-elevated-card>

        <div class="modules-section">
          <div class="modules-header">
            <span class="modules-title">Loaded Modules</span>
          </div>
          <div v-if="modules.length === 0" class="empty-state">
            <span class="material-symbols-outlined">inbox</span>
            <p>No MIB modules loaded</p>
            <p class="empty-subtext">Click "Load MIB Files" to add modules</p>
          </div>

          <div v-else class="modules-list" aria-label="Loaded MIB modules">
            <md-elevated-card
              v-for="module in modules"
              :key="module.name"
              class="module-item-card"
              :class="{ 'module-item-card--active': module.name === selectedModuleName }"
              @click="selectModule(module.name)"
            >
              <span class="module-list-headline">{{ module.name }}</span>
            </md-elevated-card>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mib-manager-dialog {
  display: flex;
  flex-direction: column;
  background-color: var(--md-sys-color-surface-container);
  border-radius: var(--border-radius-xl);
  box-shadow: var(--md-sys-elevation-3);
  width: 900px;
  max-width: 90vw;
  max-height: 85vh;
  overflow: hidden;
}

.dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-lg);
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
  background-color: var(--md-sys-color-surface);
}

.dialog-title {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  font-size: var(--md-sys-typescale-headline-small-size);
  font-weight: 500;
  margin: 0;
  color: var(--md-sys-color-on-surface);
}

.dialog-title .material-symbols-outlined {
  color: var(--md-sys-color-primary);
  font-size: 28px;
}

.dialog-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
  padding: var(--spacing-lg);
  overflow: hidden;
}

.stats-section {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: var(--spacing-md);
}

.stat-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: var(--spacing-md);
}

.stat-value {
  font-size: 32px;
  font-weight: 600;
  color: var(--md-sys-color-primary);
}

.stat-label {
  font-size: 12px;
  color: var(--md-sys-color-on-surface-variant);
}

.actions-section {
  display: flex;
  gap: var(--spacing-md);
  flex-wrap: wrap;
}

.dialog-body {
  display: grid;
  grid-template-columns: minmax(360px, 1fr) minmax(280px, 320px);
  gap: var(--spacing-lg);
  flex: 1;
  min-height: 0;
}

.module-tree-panel {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
  padding: var(--spacing-md);
  min-height: 0;
}

.module-tree-header {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.module-tree-title {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-sm);
  color: var(--md-sys-color-on-surface);
}

.module-tree-heading {
  margin: 0;
  font-size: var(--md-sys-typescale-title-medium-size);
  font-weight: 600;
}

.module-tree-subheading {
  margin: 0;
  font-size: 12px;
  color: var(--md-sys-color-on-surface-variant);
  word-break: break-all;
}

.module-metrics {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-sm);
}

.module-metric {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 12px;
  border-radius: 12px;
  background-color: var(--md-sys-color-surface-container-high);
  min-width: 110px;
}

.metric-label {
  font-size: 12px;
  color: var(--md-sys-color-on-surface-variant);
}

.metric-value {
  font-size: 18px;
  font-weight: 600;
  color: var(--md-sys-color-on-surface);
}

.module-alert {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-sm);
  padding: 12px;
  border-radius: 12px;
  background-color: var(--md-sys-color-error-container);
  color: var(--md-sys-color-on-error-container);
}

.module-alert-icon {
  font-size: 20px;
}

.module-tree-body {
  flex: 1;
  min-height: 0;
  border-radius: 12px;
  border: 1px solid var(--md-sys-color-outline-variant);
  background-color: var(--md-sys-color-surface);
  overflow-y: auto;
  padding: 8px 0;
}

.module-tree-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.module-tree-status {
  display: flex;
  flex-direction: column;
  align.items: center;
  justify-content: center;
  gap: 8px;
  color: var(--md-sys-color-on-surface-variant);
  padding: 24px;
  text-align: center;
}

.module-tree-status .spinner {
  font-size: 28px;
  animation: rotate 1.4s linear infinite;
}

.module-tree-status--error {
  color: var(--md-sys-color-error);
}

.modules-section {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.modules-header {
  display: flex;
  justify-content: center;
  flex-shrink: 0;
}

.modules-title {
  font-weight: 600;
  font-size: var(--md-sys-typescale-title-medium-size);
  color: var(--md-sys-color-on-surface);
}

.modules-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
  flex: 1;
  overflow-y: auto;
  padding: 0 2px;
}

.module-item-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: var(--spacing-md);
  cursor: pointer;
  transition: all 0.2s ease;
  min-height: 60px;
  flex-shrink: 0;
}

.module-item-card:hover {
  background-color: var(--md-sys-color-surface-container-high);
}

.module-item-card--active {
  background-color: var(--md-sys-color-primary-container);
}

.module-item-card--active .module-list-headline {
  color: var(--md-sys-color-on-primary-container);
}

.module-list-headline {
  font-weight: 600;
  font-size: 14px;
  text-align: center;
  color: var(--md-sys-color-primary);
  word-break: break-word;
}

.empty-state {
  text-align: center;
  color: var(--md-sys-color-on-surface-variant);
  padding: 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.empty-state .material-symbols-outlined {
  font-size: 32px;
}

.empty-subtext {
  font-size: 12px;
}

.module-delete-btn {
  align-self: flex-start;
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
