<script setup>
/**
 * @vue-component
 * @description A panel that displays and manages a collection of tabs.
 * It supports adding, closing, and reordering tabs via drag-and-drop.
 *
 * @vue-prop {Array} tabs - The array of tab objects to display.
 * @vue-prop {String} activeTabId - The ID of the currently active tab.
 *
 * @vue-event {Array} update:tabs - Emitted when the tabs array is updated (e.g., reordered). The payload is the new tabs array.
 * @vue-event {string} update:activeTabId - Emitted when the active tab changes. The payload is the ID of the new active tab.
 * @vue-event {void} add-tab - Emitted to request the creation of a new tab.
 * @vue-event {string} close-tab - Emitted to request the closing of a tab. The payload is the ID of the tab to close.
 * @vue-event {object} rename-tab - Emitted to request renaming a tab. The payload is an object with `id` and `title`.
 */
import { computed, ref } from 'vue'
import LogTab from './LogTab.vue'
import TableTab from './TableTab.vue'
import ChartTab from './ChartTab.vue'
import '@material/web/tabs/tabs.js'
import '@material/web/tabs/primary-tab.js'
import '@material/web/iconbutton/icon-button.js'

const props = defineProps({
  tabs: Array,
  activeTabId: String
})

const emit = defineEmits([
  'update:tabs',
  'update:activeTabId',
  'add-tab',
  'close-tab',
  'rename-tab',
  'log-entry-select',
  'table-data-updated',
  'chart-state-updated'
])

const activeTab = computed(() => {
  return props.tabs.find(t => t.id === props.activeTabId)
})

const activeComponent = computed(() => {
  const tab = activeTab.value
  if (!tab) {
    return LogTab
  }
  switch (tab.type) {
    case 'table':
      return TableTab
    case 'chart':
      return ChartTab
    default:
      return LogTab
  }
})

const tabComponentProps = computed(() => {
  const tab = activeTab.value
  const base = {
    data: tab?.data || [],
    tabInfo: tab || null
  }
  if (tab?.type === 'table' || tab?.type === 'chart') {
    const hostConfig = tab?.hostSnapshot ? { ...tab.hostSnapshot } : null
    return {
      ...base,
      hostConfig
    }
  }
  return base
})

const draggedTabId = ref(null)
const dragOverIndex = ref(null)
const isDragging = ref(false)
const activeTabIndex = computed(() => props.tabs.findIndex(t => t.id === props.activeTabId))

/**
 * @function closeTab
 * @description Emits an event to request closing a tab.
 * @param {string} tabId - The ID of the tab to close.
 */
const closeTab = (tabId) => {
  emit('close-tab', tabId)
}
/**
 * @function handleTabChange
 * @description Handles the active tab change.
 * @param {Event} event - The tab change event.
 */
const handleTabChange = (event) => {
  if (event.target.activeTabIndex === undefined) return;
  emit('update:activeTabId', props.tabs[event.target.activeTabIndex].id)
}

/**
 * @function addTab
 * @description Emits an event to request adding a new tab.
 */
const addTab = () => {
  emit('add-tab')
}

/**
 * @function handleDragStart
 * @description Handles the start of a tab drag operation.
 * @param {string} tabId - The ID of the dragged tab.
 */
const handleDragStart = (tabId, event) => {
  draggedTabId.value = tabId
  isDragging.value = true
  if (event?.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
  }
}

/**
 * @function handleDrop
 * @description Handles dropping a dragged tab to reorder it.
 * @param {number} targetIndex - The target index of the tab.
 */
const handleDrop = (targetIndex) => {
  if (draggedTabId.value === null) return

  const sourceIndex = props.tabs.findIndex(t => t.id === draggedTabId.value)
  if (sourceIndex === -1 || sourceIndex === targetIndex) return

  const newTabs = [...props.tabs]
  const [draggedTab] = newTabs.splice(sourceIndex, 1)
  newTabs.splice(targetIndex, 0, draggedTab)

  emit('update:tabs', newTabs)
  draggedTabId.value = null
  dragOverIndex.value = null
  isDragging.value = false
}

/**
 * @function handleDragEnd
 * @description Handles the end of a drag operation.
 */
const handleDragEnd = () => {
  draggedTabId.value = null
  dragOverIndex.value = null
  isDragging.value = false
}

/**
 * @function handleDragOver
 * @description Handles the dragover event to indicate the drop position.
 * @param {number} index - The index being dragged over.
 */
const handleDragOver = (index) => {
  dragOverIndex.value = index
}

/**
 * @function handleTabsContainerDragLeave
 * @description Resets the dragover index when leaving the tabs area.
 */
const handleTabsContainerDragLeave = () => {
  // This event fires when leaving the entire tabs container
  dragOverIndex.value = null
}

/**
 * @function handleLogEntrySelect
 * @description Propagates the log entry selection event to the parent component.
 * @param {object} payload - Selection details.
 */
const handleLogEntrySelect = (payload) => {
  emit('log-entry-select', payload)
}

const resolveTabIcon = (tab) => {
  if (!tab) {
    return 'list_alt'
  }
  switch (tab.type) {
    case 'table':
      return 'table_chart'
    case 'chart':
      return 'show_chart'
    default:
      return 'list_alt'
  }
}

const forwardChartStateUpdate = (payload) => {
  if (!payload) {
    return
  }
  const tabId = payload.tabId || payload.id || activeTab.value?.id || null
  if (!tabId) {
    return
  }
  // eslint-disable-next-line no-unused-vars
  const { tabId: _ignored, ...state } = payload
  emit('chart-state-updated', tabId, state)
}

const renamingTabId = ref(null)
const editedTitle = ref('')

const startRenaming = (tab) => {
  renamingTabId.value = tab.id
  editedTitle.value = tab.title
}

const finishRenaming = (tabId) => {
  if (renamingTabId.value !== null) {
    emit('rename-tab', { id: tabId, title: editedTitle.value })
    renamingTabId.value = null
    editedTitle.value = ''
  }
}

const vFocus = {
  mounted: (el) => {
    el.focus();
    el.select();
  }
}
</script>

<template>
  <div class="tabs-panel">
    <!-- Tab Headers -->
    <div class="tabs-header">
      <md-tabs
        @change="handleTabChange"
        :class="{ 'is-dragging-tab': isDragging }"
        :active-tab-index="activeTabIndex"
        @dragend="handleDragEnd"
        @dragleave="handleTabsContainerDragLeave"
      >
        <md-primary-tab
          v-for="(tab, index) in tabs"
          :key="tab.id"
          :data-testid="`tab-${tab.id}`"
          :class="{
            'dragging': tab.id === draggedTabId,
            'drag-over': index === dragOverIndex,
            'polling-active': tab.type === 'chart' && tab.chartState?.isPolling
          }"
          :aria-label="tab.title"
          draggable="true"
          @dragstart.stop="handleDragStart(tab.id, $event)"
          @dragover.prevent="handleDragOver(index)"
          @drop="handleDrop(index)"
          @dblclick="startRenaming(tab)"
        >
          <div class="tab-content-wrapper">
            <span 
              class="material-symbols-outlined tab-icon"
              slot="icon"
            >
              {{ resolveTabIcon(tab) }}
            </span>
            <input
              v-if="renamingTabId === tab.id"
              type="text"
              class="tab-title-input"
              v-model="editedTitle"
              @blur.stop="finishRenaming(tab.id)"
              @keydown.enter.stop="finishRenaming(tab.id)"
              @keydown.esc="renamingTabId = null"
              v-focus
            />
            <span v-else class="tab-title">{{ tab.title }}</span>
            <md-icon-button
              v-if="tabs.length > 1"
              class="close-tab-btn"
              @click.stop="closeTab(tab.id)"
            >
              <span class="material-symbols-outlined">close</span>
            </md-icon-button>
          </div>
        </md-primary-tab>
      </md-tabs>
      
      <md-icon-button class="add-tab-btn" @click="addTab" title="Add new log tab">
        <span class="material-symbols-outlined">add</span>
      </md-icon-button>
    </div>

    <!-- Tab Content -->
    <div class="tab-content">
      <KeepAlive>
        <component
          :is="activeComponent"
          :key="activeTab?.id"
          v-bind="tabComponentProps"
          @entry-select="handleLogEntrySelect"
          @data-updated="$emit('table-data-updated', activeTab.id, $event)"
          @state-update="forwardChartStateUpdate"
        />
      </KeepAlive>
    </div>
  </div>
</template>

<style scoped>
.tabs-panel {
  display: flex;
  flex-direction: column;
  background-color: var(--md-sys-color-surface-container);
  height: 100%; /* Occupies the full height of its grid container */
}

.tabs-header {
  display: flex;
  align-items: center;
  background-color: var(--md-sys-color-surface-container-low);
  padding-right: var(--spacing-sm);
}

md-tabs {
  flex-grow: 1;
  --md-tabs-container-color: var(--md-sys-color-surface-container-low);
  --md-primary-tab-container-color: var(--md-sys-color-surface-container-low);
}

md-primary-tab {
  max-width: 250px;
  --md-primary-tab-container-height: 48px;
}

.is-dragging-tab md-primary-tab * {
  /* Prevent children from stealing drag events */
  pointer-events: none;
}

md-primary-tab[draggable="true"] {
  cursor: grab;
}

md-primary-tab.dragging {
  opacity: 0.5;
}

md-primary-tab.drag-over {
  border-left: 2px solid var(--md-sys-color-primary);
  background-color: var(--md-sys-color-surface-container-high);
}


.tab-content-wrapper {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.tab-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tab-title-input {
  background-color: var(--md-sys-color-surface-container-highest);
  color: var(--md-sys-color-on-surface);
  border: 1px solid var(--md-sys-color-primary);
  border-radius: 4px;
  padding: 2px 6px;
  margin: -2px -6px;
  font-family: inherit;
  font-size: inherit;
  width: 100%;
  outline: none;
}

.close-tab-btn {
  opacity: 0;
  transition: opacity 0.15s ease;
  --md-icon-button-icon-size: 18px;
  margin-left: var(--spacing-sm);
  margin-right: -8px; /* Compensate for button padding */
}

md-primary-tab:hover .close-tab-btn {
  opacity: 1;
}

.close-tab-btn:hover {
  --md-icon-button-hover-state-layer-color: var(--md-sys-color-error);
  --md-icon-button-pressed-state-layer-color: var(--md-sys-color-error);
}

.add-tab-btn {
  flex-shrink: 0;
}

.tab-content {
  flex: 1;
  overflow: hidden;
  background-color: var(--md-sys-color-surface-container);
}

.polling-active .tab-icon {
  animation: bounce 0.7s ease-in-out infinite;
}

@keyframes bounce {
  0%, 100% {
    transform: translateY(+6px);
  }
  50% {
    transform: translateY(-6px);
  }
}
</style>
