import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { nextTick } from 'vue'
import TableTab from '../components/TableTab.vue'

vi.mock('../../wailsjs/go/app/App', () => ({
  FetchTableData: vi.fn(),
  SaveCSVFile: vi.fn()
}))

const handleErrorMock = vi.fn()
vi.mock('../composables/useErrorHandler', () => ({
  useErrorHandler: () => ({
    handleError: handleErrorMock,
    lastError: { value: null },
  }),
}))

import { FetchTableData } from '../../wailsjs/go/app/App'

describe('TableTab.vue', () => {
  const baseTabInfo = {
    id: 'table-123',
    oid: '1.3.6.1.2.1.2.2',
    title: 'ifTable',
    type: 'table'
  }

  const hostConfig = {
    address: '127.0.0.1',
    port: 161,
    community: 'public',
    version: 'v2c'
  }

  const createResponse = () => ({
    columns: [
      { key: 'ifIndex', label: 'If Index', oid: '1.3.6.1.2.1.2.2.1.1', type: 'number' },
      { key: 'ifDescr', label: 'If Descr', oid: '1.3.6.1.2.1.2.2.1.2', type: 'string' },
      { key: 'ifSpeed', label: 'If Speed', oid: '1.3.6.1.2.1.2.2.1.5', type: 'number' },
      { key: 'ifPhysAddress', label: 'If Phys Address', oid: '1.3.6.1.2.1.2.2.1.6', type: 'string' },
      { key: 'ifOperStatus', label: 'If Oper Status', oid: '1.3.6.1.2.1.2.2.1.8', type: 'number' }
    ],
    rows: [
      {
        __instance: '1',
        ifIndex: '1',
        ifDescr: 'eth0',
        ifSpeed: '1000000000',
        ifPhysAddress: '00:11:22:33:44:55',
        ifOperStatus: '1'
      },
      {
        __instance: '2',
        ifIndex: '2',
        ifDescr: 'eth1',
        ifSpeed: '1000000000',
        ifPhysAddress: '',
        ifOperStatus: '2'
      },
      {
        __instance: '3',
        ifIndex: '3',
        ifDescr: 'lo',
        ifSpeed: '0',
        ifPhysAddress: null,
        ifOperStatus: '1'
      }
    ]
  })

  beforeEach(() => {
    FetchTableData.mockReset()
    handleErrorMock.mockClear()
  })

  it('loads data on mount if not already present and emits event', async () => {
    const response = createResponse()
    FetchTableData.mockResolvedValue(response)

    const wrapper = mount(TableTab, {
      props: {
        tabInfo: { ...baseTabInfo, data: [], columns: [] },
        hostConfig
      }
    })

    await flushPromises()

    expect(FetchTableData).toHaveBeenCalledTimes(1)
    expect(wrapper.emitted('data-updated')[0][0]).toEqual({
      columns: response.columns,
      rows: response.rows
    })
  })

  it('does not load data on mount if already present', async () => {
    const response = createResponse()
    FetchTableData.mockResolvedValue(response)

    mount(TableTab, {
      props: {
        tabInfo: { ...baseTabInfo, data: response.rows, columns: response.columns },
        hostConfig
      }
    })

    await flushPromises()

    expect(FetchTableData).not.toHaveBeenCalled()
  })

  it('renders table data correctly when passed in props', async () => {
    const response = createResponse()
    const wrapper = mount(TableTab, {
      props: {
        tabInfo: { 
          ...baseTabInfo, 
          data: response.rows, 
          columns: response.columns, 
          lastUpdated: new Date().toISOString()
        },
        hostConfig
      }
    })

    await flushPromises()

    const rows = wrapper.findAll('.data-row')
    expect(rows.length).toBe(3)
    expect(wrapper.text()).toContain('eth0')
    expect(wrapper.text()).toContain('1000000000')
    expect(wrapper.find('.last-updated').exists()).toBe(true)
    expect(wrapper.text()).toContain('Last updated:')
    expect(wrapper.text()).toContain('Host: 127.0.0.1:161 · v2c')
  })

  it('sorts the table when a header is clicked', async () => {
    const response = createResponse()
    const wrapper = mount(TableTab, {
      props: {
        tabInfo: { ...baseTabInfo, data: response.rows, columns: response.columns },
        hostConfig
      }
    })

    await flushPromises()

    const descHeader = wrapper.findAll('th').find(w => w.text().includes('If Descr'))
    await descHeader.trigger('click')
    await nextTick()

    let rows = wrapper.findAll('.data-row')
    expect(rows[0].text()).toContain('eth0')
    expect(rows[1].text()).toContain('eth1')
    expect(rows[2].text()).toContain('lo')

    await descHeader.trigger('click')
    await nextTick()

    rows = wrapper.findAll('.data-row')
    expect(rows[0].text()).toContain('lo')
    expect(rows[1].text()).toContain('eth1')
    expect(rows[2].text()).toContain('eth0')
  })

  it('sorts numeric string values in natural order', async () => {
    const columns = [
      { key: 'idx', label: 'Index', type: 'string' },
      { key: 'name', label: 'Name', type: 'string' }
    ]
    const rows = [
      { __instance: '1', idx: '1', name: 'alpha' },
      { __instance: '10', idx: '10', name: 'gamma' },
      { __instance: '2', idx: '2', name: 'beta' }
    ]

    const wrapper = mount(TableTab, {
      props: {
        tabInfo: { ...baseTabInfo, data: rows, columns },
        hostConfig
      }
    })

    await flushPromises()

    const indexHeader = wrapper.findAll('th').find(w => w.text().includes('Index'))
    await indexHeader.trigger('click')
    await nextTick()

    let firstColumn = wrapper.findAll('.data-row').map(row => row.findAll('td')[0].text().trim())
    expect(firstColumn).toEqual(['1', '2', '10'])

    await indexHeader.trigger('click')
    await nextTick()

    firstColumn = wrapper.findAll('.data-row').map(row => row.findAll('td')[0].text().trim())
    expect(firstColumn).toEqual(['10', '2', '1'])
  })

  it('filters the table based on search query', async () => {
    const response = createResponse()
    const wrapper = mount(TableTab, {
      props: {
        tabInfo: { ...baseTabInfo, data: response.rows, columns: response.columns },
        hostConfig
      }
    })

    await flushPromises()

    wrapper.vm.searchQuery = 'eth1'
    await wrapper.vm.$nextTick()

    const rows = wrapper.findAll('.data-row')
    expect(rows.length).toBe(1)
    expect(rows[0].text()).toContain('eth1')
  })

  it('renders placeholder for empty values in the table', async () => {
    const response = createResponse()
    const wrapper = mount(TableTab, {
      props: {
        tabInfo: { ...baseTabInfo, data: response.rows, columns: response.columns },
        hostConfig
      }
    })

    await flushPromises()

    const rows = wrapper.findAll('.data-row')
    expect(rows[2].text()).toContain('—')
  })

  it('displays an error message when data fetching fails', async () => {
    const error = new Error('SNMP request failed')
    FetchTableData.mockRejectedValue(error)

    mount(TableTab, {
      props: {
        tabInfo: { ...baseTabInfo, data: [], columns: [] },
        hostConfig
      }
    })

    await flushPromises()

    expect(handleErrorMock).toHaveBeenCalledWith(error, 'Failed to fetch table data')
  })
});
