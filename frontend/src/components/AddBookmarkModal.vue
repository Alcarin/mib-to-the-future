<script setup>
import { ref, watch } from 'vue'
import '@material/web/button/filled-button.js'
import '@material/web/button/outlined-button.js'
import '@material/web/iconbutton/icon-button.js'

const props = defineProps({
  show: { type: Boolean, default: false },
  nodeName: { type: String, default: '' },
  nodeOid: { type: String, default: '' },
  folders: { type: Array, default: () => [] },
  selectedKey: { type: String, default: 'bookmarks' }
})

const emit = defineEmits(['update:show', 'confirm', 'create-folder'])

const selection = ref(props.selectedKey || 'bookmarks')

watch(() => props.selectedKey, (value) => {
  selection.value = value || 'bookmarks'
})

watch(() => props.show, (visible) => {
  if (visible) {
    selection.value = props.selectedKey || 'bookmarks'
  }
})

const closeModal = () => {
  emit('update:show', false)
}

const confirm = () => {
  emit('confirm', selection.value || 'bookmarks')
}

const createFolder = () => {
  emit('create-folder', selection.value || 'bookmarks')
}
</script>

<template>
  <div v-if="show" class="modal-overlay" @click.self="closeModal">
    <div class="modal-content">
      <div class="modal-header">
        <h2>Add bookmark</h2>
        <md-icon-button @click="closeModal" title="Close">
          <span class="material-symbols-outlined">close</span>
        </md-icon-button>
      </div>
      <div class="modal-body">
        <div class="node-summary">
          <div class="node-name" :title="nodeName">{{ nodeName || 'Unnamed OID' }}</div>
          <div v-if="nodeOid" class="node-oid">{{ nodeOid }}</div>
        </div>
        <div class="folder-selector">
          <label for="bookmark-folder-select">Folder</label>
          <select
            id="bookmark-folder-select"
            v-model="selection"
            class="folder-select"
          >
            <option v-for="option in folders" :key="option.key" :value="option.key">
              {{ option.label }}
            </option>
          </select>
          <md-outlined-button class="new-folder-btn" @click="createFolder">
            <span class="material-symbols-outlined">create_new_folder</span>
            New folder
          </md-outlined-button>
        </div>
      </div>
      <div class="modal-footer">
        <md-outlined-button @click="closeModal">Cancel</md-outlined-button>
        <md-filled-button @click="confirm">Add</md-filled-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
@import '../assets/styles/modal.css';

.modal-body {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
}

.node-summary {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.node-name {
  font-size: var(--md-sys-typescale-title-medium-size);
  font-weight: 600;
  color: var(--md-sys-color-on-surface);
}

.node-oid {
  font-size: 12px;
  color: var(--md-sys-color-on-surface-variant);
  word-break: break-all;
}

.folder-selector {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.folder-select {
  width: 100%;
  padding: 10px 12px;
  border-radius: var(--border-radius-sm);
  border: 1px solid var(--md-sys-color-outline-variant);
  background: var(--md-sys-color-surface);
  color: var(--md-sys-color-on-surface);
  font: inherit;
}

.folder-select:focus {
  outline: 2px solid var(--md-sys-color-primary);
  outline-offset: 1px;
}

.new-folder-btn {
  align-self: flex-start;
  display: inline-flex;
  gap: 4px;
}
</style>
