import { describe, it, expect, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import App from '../App.vue'

vi.mock('../components/MibTreeSidebar.vue', () => ({
  default: {
    name: 'MibTreeSidebar',
    props: ['reloadKey', 'isResizing', 'selectedOid'],
    template: '<div class="sidebar-stub" />'
  }
}))

vi.mock('../components/TopBar.vue', () => ({
  default: {
    name: 'TopBar',
    props: ['host', 'selectedOid', 'operation', 'hostSuggestions'],
    emits: ['execute', 'load-mib', 'download-mib', 'delete-host', 'update:host', 'update:selectedOid', 'update:operation'],
    template: '<div class="topbar-stub" />'
  }
}))

vi.mock('../components/TabsPanel.vue', () => {
  const { defineComponent, h, computed } = require('vue')
  return {
    default: defineComponent({
      name: 'TabsPanel',
      props: {
        tabs: { type: Array, default: () => [] },
        activeTabId: { type: [String, Number], default: null },
        hostConfig: { type: Object, default: () => ({}) }
      },
      emits: ['add-tab', 'update:tabs', 'close-tab', 'log-entry-select', 'table-data-updated', 'chart-state-updated'],
      setup(props, { expose }) {
        const host = computed(() => props.hostConfig)
        expose({ host })
        return () => h('div', { class: 'tabs-panel-stub', 'data-host-address': host.value?.address ?? '' })
      }
    })
  }
})

vi.mock('../components/MibManagerDialog.vue', () => ({
  default: {
    name: 'MibManagerDialog',
    emits: ['mib-loaded', 'close'],
    template: '<div class="mib-manager-stub" />'
  }
}))

vi.mock('../components/GraphInstanceModal.vue', () => ({
  default: {
    name: 'GraphInstanceModal',
    props: ['show', 'node'],
    emits: ['confirm', 'cancel'],
    template: '<div class="graph-modal-stub" />'
  }
}))

vi.mock('../components/SetValueModal.vue', () => ({
  default: {
    name: 'SetValueModal',
    props: ['show', 'node', 'currentValue', 'loading', 'loadError'],
    emits: ['confirm', 'cancel'],
    template: '<div class="set-value-modal-stub" />'
  }
}))

vi.mock('../components/ToastNotification.vue', () => ({
  default: {
    name: 'ToastNotification',
    props: ['notification'],
    emits: ['close'],
    template: '<div class="toast-stub" />'
  }
}))

vi.mock('../composables/useHostManager', () => {
  const { ref } = require('vue')
  return {
    useHostManager: () => ({
      host: ref({
        address: '198.51.100.7',
        port: 161,
        community: 'public',
        version: 'v2c'
      }),
      savedHosts: ref([]),
      loadSavedHosts: vi.fn(),
      handleDeleteHost: vi.fn()
    })
  }
})

vi.mock('../composables/useOidSelection', () => ({
  useOidSelection: () => ({
    selectedOid: { value: '1.3.6.1.2.1.1.3.0' },
    selectedOperation: { value: 'get' },
    handleOidSelect: vi.fn(),
    handleLogEntrySelect: vi.fn()
  })
}))

const logTab = {
  id: 'log-default',
  title: 'Request Log',
  type: 'log',
  data: []
}

vi.mock('../composables/useTabsManager', () => ({
  useTabsManager: () => ({
    tabs: { value: [logTab] },
    activeTabId: { value: 'log-default' },
    handleAddTab: vi.fn(),
    handleTabsUpdate: vi.fn(),
    handleCloseTab: vi.fn(),
    handleTableDataUpdate: vi.fn(),
    handleChartStateUpdate: vi.fn(),
    handleOpenTableTab: vi.fn(),
    openGraphTab: vi.fn(),
    findTargetLogTab: () => logTab,
    prependLogEntry: vi.fn(),
    replaceLogEntry: vi.fn()
  })
}))

vi.mock('../composables/useModalManager', () => ({
  useModalManager: () => ({
    showMibManager: { value: false },
    graphInstanceModal: { visible: false, node: null },
    operationInstanceModal: { visible: false, node: null, operation: null, setPayload: null },
    setOperationModal: {
      visible: false,
      node: null,
      loadingCurrent: false,
      loadError: '',
      currentValue: null
    },
    handleLoadMib: vi.fn(),
    handleGraphInstanceConfirm: vi.fn(),
    handleGraphInstanceCancel: vi.fn(),
    handleOperationInstanceConfirm: vi.fn(),
    handleOperationInstanceCancel: vi.fn(),
    handleSetModalConfirm: vi.fn(),
    handleSetModalCancel: vi.fn(),
    openSetModalForOid: vi.fn(),
    openOperationInstanceModal: vi.fn(),
    handleOpenGraphRequest: vi.fn()
  })
}))

vi.mock('../composables/useSnmp', () => ({
  useSnmp: () => ({
    handleExecuteSnmpOperation: vi.fn(),
    handleContextOperation: vi.fn()
  })
}))

vi.mock('../composables/useTheme', () => ({
  useTheme: vi.fn()
}))

vi.mock('../composables/useNotifications', () => ({
  useNotifications: () => ({
    notifications: [],
    removeNotification: vi.fn(),
    addNotification: vi.fn()
  })
}))

vi.mock('@material/web/button/filled-button.js', () => ({}))
vi.mock('@material/web/iconbutton/icon-button.js', () => ({}))
vi.mock('splitpanes', () => ({
  Splitpanes: {
    name: 'Splitpanes',
    template: '<div class="splitpanes-stub"><slot /></div>'
  },
  Pane: {
    name: 'Pane',
    template: '<div class="pane-stub"><slot /></div>'
  }
}))

const createAppWrapper = () => mount(App, { attachTo: document.body })

describe('App host propagation', () => {
  it("passa l'host configurato al TabsPanel per il polling dei grafici", async () => {
    const wrapper = createAppWrapper()
    await flushPromises()

    const tabsPanel = wrapper.findComponent({ name: 'TabsPanel' })
    expect(tabsPanel.exists()).toBe(true)

    expect(tabsPanel.vm.$props.hostConfig.address).toBe('198.51.100.7')

    wrapper.unmount()
  })
})
