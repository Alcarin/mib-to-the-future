import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import MibManagerDialog from '../components/MibManagerDialog.vue'
import { ListMIBModules, GetMIBStats, LoadMIBFile, DeleteMIBModule, GetMIBModuleDetails } from '../../wailsjs/go/app/App'

vi.mock('../../wailsjs/go/app/App')

describe('MibManagerDialog.vue', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    // Mock confirm dialog
    global.confirm = vi.fn(() => true)
    ListMIBModules.mockResolvedValue([
      { name: 'BASE', nodeCount: 3, scalarCount: 1, tableCount: 0, columnCount: 0, typeCount: 2, skippedNodes: 0, missingImports: [] },
      { name: 'IF-MIB', nodeCount: 12, scalarCount: 6, tableCount: 2, columnCount: 4, typeCount: 1, skippedNodes: 0, missingImports: [] }
    ])
    GetMIBStats.mockResolvedValue({ modules: 2, total_nodes: 123, scalar: 50, table: 10 })
    GetMIBModuleDetails.mockResolvedValue({
      module: 'BASE',
      tree: [],
      stats: { nodeCount: 3, scalarCount: 1, tableCount: 0, columnCount: 0, typeCount: 2, skippedNodes: 0, missingCount: 0 },
      missingImports: []
    })
  })

  it('loads and displays MIB modules on mount', async () => {
    const wrapper = mount(MibManagerDialog)

    await flushPromises()

    expect(ListMIBModules).toHaveBeenCalled()
    expect(GetMIBStats).toHaveBeenCalled()
    expect(GetMIBModuleDetails).toHaveBeenCalledWith('BASE')

    const moduleItems = wrapper.findAll('.module-item-card')
    expect(moduleItems).toHaveLength(2)
    expect(moduleItems[0].text()).toContain('BASE')
    expect(moduleItems[1].text()).toContain('IF-MIB')
    expect(wrapper.text()).toContain('123')
  })

  it('emits close event when close button is clicked', async () => {
    const wrapper = mount(MibManagerDialog)
    await wrapper.find('md-icon-button[title="Close"]').trigger('click')
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('calls LoadMIBFile and emits mib-loaded for each loaded module', async () => {
    LoadMIBFile.mockResolvedValue(['NEW-MIB', 'SECOND-MIB'])
    ListMIBModules.mockResolvedValueOnce([
      { name: 'BASE', nodeCount: 3, scalarCount: 1, tableCount: 0, columnCount: 0, typeCount: 2, skippedNodes: 0, missingImports: [] },
      { name: 'IF-MIB', nodeCount: 12, scalarCount: 6, tableCount: 2, columnCount: 4, typeCount: 1, skippedNodes: 0, missingImports: [] }
    ])
    const wrapper = mount(MibManagerDialog)
    await flushPromises() // for initial load

    await wrapper.find('md-filled-button').trigger('click')
    await flushPromises()

    expect(LoadMIBFile).toHaveBeenCalled()
    expect(wrapper.emitted('mib-loaded')).toBeTruthy()
    expect(wrapper.emitted('mib-loaded')).toHaveLength(2)
    expect(wrapper.emitted('mib-loaded')[0]).toEqual(['NEW-MIB'])
    expect(wrapper.emitted('mib-loaded')[1]).toEqual(['SECOND-MIB'])
  })

  it('calls DeleteMIBModule and emits mib-loaded on success', async () => {
    const wrapper = mount(MibManagerDialog)
    await flushPromises()

    const cards = wrapper.findAll('.module-item-card')
    await cards[1].trigger('click')
    await flushPromises()

    await wrapper.find('.module-delete-btn').trigger('click')
    await flushPromises()

    expect(global.confirm).toHaveBeenCalledWith('Are you sure you want to delete module "IF-MIB"?')
    expect(DeleteMIBModule).toHaveBeenCalledWith('IF-MIB')
    expect(wrapper.emitted('mib-loaded')).toBeTruthy()
  })
})
