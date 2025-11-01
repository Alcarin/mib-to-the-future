import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import TabsPanel from '../components/TabsPanel.vue'

vi.mock('@material/web/tabs/tabs.js', () => ({}))
vi.mock('@material/web/tabs/primary-tab.js', () => ({}))
vi.mock('@material/web/iconbutton/icon-button.js', () => ({}))

const baseTabs = [
  { id: 'log-1', title: 'Log 1', type: 'log', data: [{ message: 'entry' }] },
  { id: 'table-1', title: 'Table 1', type: 'table', data: [{ column: 'value' }] }
]

const mountPanel = (overrideProps = {}) => {
  return mount(TabsPanel, {
    props: {
      tabs: baseTabs,
      activeTabId: 'log-1',
      ...overrideProps
    },
    global: {
      stubs: {
        LogTab: { template: '<div class="log-tab-stub" />' },
        TableTab: { template: '<div class="table-tab-stub" />' },
        'md-tabs': { template: '<div class="md-tabs" v-bind="$attrs"><slot /></div>' },
        'md-primary-tab': { template: '<div class="md-primary-tab" v-bind="$attrs"><slot /></div>' },
        'md-icon-button': { template: '<button class="md-icon-button" type="button" v-bind="$attrs"><slot /></button>' }
      }
    }
  })
}

describe('TabsPanel', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('emits add-tab when the add button is clicked', async () => {
    const wrapper = mountPanel()

    await wrapper.find('.add-tab-btn').trigger('click')

    expect(wrapper.emitted()['add-tab']).toBeTruthy()
    wrapper.unmount()
  })

  it('emits close-tab with correct id when close icon is clicked', async () => {
    const wrapper = mountPanel()

    await wrapper.find('.close-tab-btn').trigger('click')

    expect(wrapper.emitted()['close-tab'][0]).toEqual(['log-1'])
    wrapper.unmount()
  })

  it('renders table component when active tab is of type table', () => {
    const wrapper = mountPanel({ activeTabId: 'table-1' })

    expect(wrapper.find('.table-tab-stub').exists()).toBe(true)
    expect(wrapper.find('.log-tab-stub').exists()).toBe(false)

    wrapper.unmount()
  })

  it('emits update:activeTabId when handleTabChange runs', () => {
    const wrapper = mountPanel()

    wrapper.vm.handleTabChange({ target: { activeTabIndex: 1 } })

    expect(wrapper.emitted()['update:activeTabId'][0]).toEqual(['table-1'])
    wrapper.unmount()
  })

  it('emits reordered tabs when a drag-and-drop completes', () => {
    const wrapper = mountPanel()

    wrapper.vm.handleDragStart('log-1', { dataTransfer: { effectAllowed: '' } })
    wrapper.vm.handleDrop(1)

    const emittedTabs = wrapper.emitted()['update:tabs'][0][0]
    expect(emittedTabs.map((tab) => tab.id)).toEqual(['table-1', 'log-1'])

    wrapper.unmount()
  })
})
