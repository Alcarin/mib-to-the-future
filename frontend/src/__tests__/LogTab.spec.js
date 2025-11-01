import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import LogTab from '../components/LogTab.vue'

vi.mock('@material/web/chips/filter-chip.js', () => ({}))
vi.mock('@material/web/iconbutton/icon-button.js', () => ({}))

vi.mock('../../wailsjs/go/app/App', () => ({
  __esModule: true,
  SaveCSVFile: vi.fn()
}))

import { SaveCSVFile } from '../../wailsjs/go/app/App'

const ensureCustomElement = (name) => {
  if (typeof customElements !== 'undefined' && !customElements.get(name)) {
    customElements.define(name, class extends HTMLElement {})
  }
}

;['md-filter-chip', 'md-icon-button'].forEach(ensureCustomElement)

beforeEach(() => {
  SaveCSVFile.mockReset()
  SaveCSVFile.mockResolvedValue(true)
})

const sampleLogs = [
  {
    id: 'log-1',
    timestamp: '2025-10-26T10:00:00.000Z',
    status: 'success',
    operation: 'GET',
    oid: '1.3.6.1.2.1.1.1.0',
    oidName: 'sysDescr',
    host: 'localhost',
    value: 'Test System',
    responseTime: 10,
  },
  {
    id: 'log-2',
    timestamp: '2025-10-26T10:01:00.000Z',
    status: 'error',
    operation: 'SET',
    oid: '1.3.6.1.2.1.1.5.0',
    oidName: 'sysName',
    host: 'localhost',
    value: 'New Name',
    responseTime: 20,
  },
  {
    id: 'log-3',
    timestamp: '2025-10-26T10:02:00.000Z',
    status: 'pending',
    operation: 'WALK',
    oid: '1.3.6.1.2.1.2',
    oidName: 'interfaces',
    host: 'remotehost',
    value: '',
    responseTime: null,
  },
]

describe('LogTab.vue', () => {
  it('renders log entries correctly', () => {
    const wrapper = mount(LogTab, {
      props: {
        data: sampleLogs,
      },
    })

    const rows = wrapper.findAll('.log-row')
    expect(rows.length).toBe(3)
    expect(wrapper.text()).toContain('sysDescr')
    expect(wrapper.text()).toContain('Test System')
  })

  it('filters log entries by status', async () => {
    const wrapper = mount(LogTab, {
      props: {
        data: sampleLogs,
      },
    })

    // Find and click the 'Success' filter button
    const successButton = wrapper.findAll('md-filter-chip')[1]
    await successButton.trigger('click')

    const rows = wrapper.findAll('.log-row')
    expect(rows.length).toBe(1)
    expect(wrapper.text()).toContain('sysDescr')
    expect(wrapper.text()).not.toContain('sysName')
  })

  it('filters log entries by search query', async () => {
    const wrapper = mount(LogTab, {
      props: {
        data: sampleLogs,
      },
    })

    wrapper.vm.searchQuery = 'remotehost'
    await wrapper.vm.$nextTick()

    const rows = wrapper.findAll('.log-row')
    expect(rows.length).toBe(1)
    expect(wrapper.text()).toContain('interfaces')
    expect(wrapper.text()).not.toContain('sysDescr')
  })

  it('shows an empty state when there are no logs', () => {
    const wrapper = mount(LogTab, {
      props: {
        data: [],
      },
    })

    expect(wrapper.find('.empty-state').exists()).toBe(true)
    expect(wrapper.text()).toContain('No log entries')
  })

  it('aggiorna i contatori di stato quando i dati cambiano', async () => {
    const wrapper = mount(LogTab, {
      props: {
        data: sampleLogs,
      },
    })

    const getChipLabel = (name) => {
      const chips = wrapper.findAll('md-filter-chip')
      const chip = chips.find((item) => item.attributes('label')?.startsWith(name))
      return chip?.attributes('label')
    }

    await nextTick()
    expect(getChipLabel('All')).toBe('All (3)')
    expect(getChipLabel('Pending')).toBe('Pending (1)')

    await wrapper.setProps({
      data: [
        { ...sampleLogs[0], status: 'success' },
        { ...sampleLogs[1], status: 'success' },
      ],
    })
    await nextTick()

    expect(getChipLabel('All')).toBe('All (2)')
    expect(getChipLabel('Success')).toBe('Success (2)')
    expect(getChipLabel('Pending')).toBeUndefined()
  })

  it('emette l\'evento entry-select quando si seleziona una riga', async () => {
    const wrapper = mount(LogTab, {
      props: {
        data: sampleLogs,
      },
    })

    await wrapper.findAll('.log-row')[0].trigger('click')

    expect(wrapper.emitted('entry-select')).toBeTruthy()
    expect(wrapper.emitted('entry-select')[0][0]).toMatchObject({
      oid: '1.3.6.1.2.1.1.1.0',
      name: 'sysDescr'
    })
  })

  it('usa SaveCSVFile per esportare il log con nome sanificato', async () => {
    vi.spyOn(Date, 'now').mockReturnValue(1700000000000)
    const wrapper = mount(LogTab, {
      props: {
        data: sampleLogs,
        tabInfo: { title: 'Log SNMP #1' }
      }
    })

    await wrapper.vm.exportLog()

    expect(SaveCSVFile).toHaveBeenCalledTimes(1)
    const [filename, csvContent] = SaveCSVFile.mock.calls[0]
    expect(filename).toBe('log-snmp-1-1700000000000.csv')
    expect(csvContent.split('\n')[0]).toBe('Timestamp,Operation,OID,Host,Status,Value,Response Time')

    Date.now.mockRestore()
  })
})
