<!-- eslint-disable vue/no-unused-vars -->
<script setup>
/**
 * @vue-component
 * @description The main application component. It orchestrates the different parts of the UI,
 * including the top bar, MIB tree sidebar, and the tabs panel. It also handles the global state
 * and the execution of SNMP operations.
 */
import { ref, onMounted } from 'vue';
import MibTreeSidebar from './components/MibTreeSidebar.vue';
import TopBar from './components/TopBar.vue';
import TabsPanel from './components/TabsPanel.vue';
import MibManagerDialog from './components/MibManagerDialog.vue';
import { Splitpanes, Pane } from 'splitpanes';
import './assets/styles/splitpanes-theme.css';
import { useTheme } from './composables/useTheme';
import { useNotifications } from './composables/useNotifications';
import ToastNotification from './components/ToastNotification.vue';
import GraphInstanceModal from './components/GraphInstanceModal.vue';
import SetValueModal from './components/SetValueModal.vue';

import { useHostManager } from './composables/useHostManager';
import { useOidSelection } from './composables/useOidSelection';
import { useTabsManager } from './composables/useTabsManager';
import { useModalManager } from './composables/useModalManager';
import { useSnmp } from './composables/useSnmp';

// Global application state
const isResizing = ref(false);
const mibTreeKey = ref(0);

// Initialize composables
useTheme();
const { notifications, removeNotification } = useNotifications();
const { host, savedHosts, loadSavedHosts, handleDeleteHost } = useHostManager();
const { selectedOid, selectedOperation, handleOidSelect, handleLogEntrySelect } = useOidSelection();
const tabsManager = useTabsManager();

// The modal manager depends on SNMP and tabs logic, so we pass them in.
// The SNMP composable needs a way to open the SET modal and the instance modal, creating circular dependencies.
// To solve this, we define function placeholders that will be replaced once the modal manager is initialized.
let openSetModalFn = () => Promise.resolve();
let openOperationInstanceModalFn = () => {};

const snmpManager = useSnmp({
  openSetModalForOid: (...args) => openSetModalFn(...args),
  openOperationInstanceModal: (...args) => openOperationInstanceModalFn(...args),
});

const modalManager = useModalManager(snmpManager, tabsManager);

// Now that the modal manager is initialized, we can update the function references in the SNMP composable.
openSetModalFn = modalManager.openSetModalForOid;
openOperationInstanceModalFn = modalManager.openOperationInstanceModal;

// Event handlers
const handleMibLoaded = () => {
  mibTreeKey.value++; // Force refresh of the MIB tree
};

const handleResizeStart = () => {
  isResizing.value = true;
};

const handleResizeEnd = () => {
  isResizing.value = false;
};

onMounted(async () => {
  await loadSavedHosts();
});

</script>

<template>
  <div class="app-container" :class="{ 'app--is-resizing': isResizing }">
    <!-- Top Bar -->
    <TopBar
      v-model:host="host"
      v-model:selectedOid="selectedOid"
      v-model:operation="selectedOperation"
      :host-suggestions="savedHosts"
      @execute="snmpManager.handleExecuteSnmpOperation"
      @load-mib="modalManager.handleLoadMib"
      @delete-host="handleDeleteHost"
    />

    <!-- Main Content Area -->
    <div class="main-content">
      <Splitpanes
        class="main-split"
        @resize="handleResizeStart"
        @resized="handleResizeEnd"
      >
        <Pane size="25" min-size="20" max-size="50">
          <MibTreeSidebar
            :reload-key="mibTreeKey"
            :is-resizing="isResizing"
            :selected-oid="selectedOid"
            @oid-select="handleOidSelect"
            @open-table="(node) => tabsManager.handleOpenTableTab(node, host)"
            @request-operation="snmpManager.handleContextOperation"
            @open-graph="(node) => modalManager.handleOpenGraphRequest(node, host)"
          />
        </Pane>
        <Pane>
          <TabsPanel
            v-model:tabs="tabsManager.tabs.value"
            v-model:active-tab-id="tabsManager.activeTabId.value"
            @add-tab="tabsManager.handleAddTab"
            @update:tabs="tabsManager.handleTabsUpdate"
            @close-tab="tabsManager.handleCloseTab"
            @log-entry-select="handleLogEntrySelect"
            @table-data-updated="tabsManager.handleTableDataUpdate"
            @chart-state-updated="tabsManager.handleChartStateUpdate"
            @rename-tab="({ id, title }) => tabsManager.handleRenameTab(id, title)"
          />
        </Pane>
      </Splitpanes>

      <!-- Overlay to prevent text selection during resize -->
      <div v-if="isResizing" class="resize-overlay"></div>
    </div>

    <!-- MIB Manager Dialog -->
    <div v-if="modalManager.showMibManager.value" class="dialog-overlay" @click.self="modalManager.showMibManager.value = false">
      <MibManagerDialog
        @mib-loaded="handleMibLoaded"
        @close="modalManager.showMibManager.value = false"
      />
    </div>

    <GraphInstanceModal
      :show="modalManager.graphInstanceModal.visible"
      :node="modalManager.graphInstanceModal.node"
      @confirm="modalManager.handleGraphInstanceConfirm"
      @cancel="modalManager.handleGraphInstanceCancel"
    />

    <GraphInstanceModal
      :show="modalManager.operationInstanceModal.visible"
      :node="modalManager.operationInstanceModal.node"
      @confirm="modalManager.handleOperationInstanceConfirm"
      @cancel="modalManager.handleOperationInstanceCancel"
    />

    <SetValueModal
      :show="modalManager.setOperationModal.visible"
      :node="modalManager.setOperationModal.node"
      :current-value="modalManager.setOperationModal.currentValue"
      :loading="modalManager.setOperationModal.loadingCurrent"
      :load-error="modalManager.setOperationModal.loadError"
      @cancel="modalManager.handleSetModalCancel"
      @confirm="modalManager.handleSetModalConfirm"
    />

    <!-- Notifications -->
    <div class="notification-container">
      <ToastNotification
        v-for="notification in notifications"
        :key="notification.id"
        :notification="notification"
        @close="removeNotification(notification.id)"
      />
    </div>
  </div>
</template>

<style scoped>
.app-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  background-color: var(--md-sys-color-background);
}

.app--is-resizing * {
  user-select: none !important;
  cursor: col-resize !important;
}

.main-content {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.resize-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 9999;
}

.dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  animation: fadeIn 0.2s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.notification-container {
  position: fixed;
  bottom: 16px;
  right: 16px;
  z-index: 10001;
}
</style>
