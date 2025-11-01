import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { nextTick } from 'vue'

vi.mock('@material/web/textfield/outlined-text-field.js', () => ({}))
vi.mock('@material/web/iconbutton/icon-button.js', () => ({}))
vi.mock('@material/web/button/filled-button.js', () => ({}))
vi.mock('@material/web/button/outlined-button.js', () => ({}))

const addNotificationMock = vi.fn()
vi.mock('../composables/useNotifications', () => ({
  useNotifications: () => ({
    notifications: { value: [] },
    addNotification: addNotificationMock,
    removeNotification: vi.fn()
  })
}))

const handleErrorMock = vi.fn()
const handleWarningMock = vi.fn()
vi.mock('../composables/useErrorHandler', () => ({
  useErrorHandler: () => ({
    handleError: handleErrorMock,
    handleWarning: handleWarningMock,
    clearLastError: vi.fn(),
    withErrorHandling: vi.fn(),
    lastError: { value: null },
    errorHistory: { value: [] },
    inferErrorType: vi.fn()
  })
}))

import MibTreeSidebar from '../components/MibTreeSidebar.vue'
import {
  GetMIBTree,
  AddBookmark,
  RemoveBookmark,
  CreateBookmarkFolder,
  DeleteBookmarkFolder,
  MoveBookmark,
  MoveBookmarkFolder,
  RenameBookmarkFolder
} from '../../wailsjs/go/app/App'

vi.mock('../../wailsjs/go/app/App')

const sampleTree = [
  {
    oid: '1.3.6.1',
    name: 'iso',
    type: 'node',
    children: [
      {
        oid: '1.3.6.1.2',
        name: 'mgmt',
        type: 'node'
      },
      {
        oid: '1.3.6.1.2.1.1.3',
        name: 'sysUpTime',
        type: 'scalar',
        syntax: 'TimeTicks',
        access: 'read-only'
      },
      {
        oid: '1.3.6.1.2.1.1.5',
        name: 'sysName',
        type: 'scalar',
        syntax: 'DisplayString',
        access: 'read-write'
      },
      {
        oid: '1.3.6.1.4',
        name: 'private',
        type: 'table'
      }
    ]
  }
]

vi.mock('splitpanes', () => ({
  Splitpanes: {
    name: 'Splitpanes',
    template: '<div class="splitpanes"><slot /></div>'
  },
  Pane: {
    name: 'Pane',
    template: '<div class="pane"><slot /></div>'
  }
}))

const baseGlobalConfig = {
  stubs: {
  DeLoreanHeader: {
    props: ['search'],
    emits: ['update:search'],
    template: `
      <div class="de-lorean-header">
        <input
          data-test="search-input"
          type="text"
          :value="search"
          @input="$emit('update:search', $event.target.value)"
        />
      </div>
    `
  },
  OidDetailsPanel: {
    props: ['node'],
    template: '<div class="oid-details" :data-node="node ? node.oid : null"></div>'
  },
  'md-filled-button': {
    template: `
      <button
        class="md-filled-button"
        type="button"
        :disabled="$attrs.disabled"
        @click="!$attrs.disabled && $emit('click', $event)"
      >
        <slot name="icon"></slot>
        <slot></slot>
      </button>
    `
  },
  'md-icon-button': {
    template: `<button class="md-icon-button" type="button" @click="$emit('click', $event)" v-bind="$attrs"><slot /></button>`
  }
  }
}

beforeEach(() => {
  vi.resetAllMocks()
  addNotificationMock.mockReset()
  handleErrorMock.mockReset()
  handleWarningMock.mockReset()
  if (typeof window !== 'undefined' && window.localStorage) {
    window.localStorage.clear()
  }
  GetMIBTree.mockResolvedValue(sampleTree)
})

describe('MibTreeSidebar (root)', () => {
  it('loads MIB tree data on mount and renders nodes', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: {
        selectedOid: ''
      },
      global: baseGlobalConfig
    })

    await flushPromises()

    expect(GetMIBTree).toHaveBeenCalledTimes(1)
    const nodeLabels = wrapper.findAll('.node-name').map((node) => node.text())
    expect(nodeLabels).toContain('iso')
    expect(nodeLabels).toContain('private')
    wrapper.unmount()
  })

  it('marks read-write scalars with edit icon and exposes SET action', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: {
        selectedOid: ''
      },
      global: baseGlobalConfig
    })

    await flushPromises()

    const writableNode = wrapper.findAll('.tree-node').find((el) => el.text().includes('sysName'))
    expect(writableNode, 'Writable node should be rendered').toBeTruthy()

    const icon = writableNode.find('.node-icon')
    expect(icon.exists()).toBe(true)
    expect(icon.text().trim()).toBe('edit')

    await writableNode.trigger('contextmenu', { clientX: 120, clientY: 130 })
    await flushPromises()
    await nextTick()

    const menuState = wrapper.vm.contextMenuState
    const setEntry = menuState.items.find((item) => item.type === 'operation' && item.value === 'set')
    expect(setEntry, 'SET entry should be available for writable nodes').toBeTruthy()
    expect(setEntry.label).toBe('Execute SET')

    wrapper.unmount()
  })

  it('shows an error banner and allows retry when GetMIBTree fails', async () => {
    const error = new Error('backend offline')
    GetMIBTree.mockRejectedValueOnce(error)

    const wrapper = mount(MibTreeSidebar, {
      props: {
        selectedOid: ''
      },
      global: baseGlobalConfig
    })

    await flushPromises()

    // Verifica che handleError sia stato chiamato con l'errore
    expect(handleErrorMock).toHaveBeenCalledWith(
      error,
      "Errore durante il caricamento dell'albero MIB"
    )

    const banner = wrapper.find('[data-test="mib-tree-error"]')
    expect(banner.exists()).toBe(true)
    expect(banner.text()).toContain('backend offline')

    await wrapper.find('[data-test="mib-tree-retry"]').trigger('click')
    await flushPromises()

    expect(GetMIBTree).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[data-test="mib-tree-error"]').exists()).toBe(false)

    wrapper.unmount()
  })

  it('filters nodes based on search query', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: {
        selectedOid: ''
      },
      global: baseGlobalConfig
    })

    await flushPromises()

    await wrapper.find('[data-test="search-input"]').setValue('private')
    await flushPromises()

    const nodeLabels = wrapper.findAll('.node-name').map((node) => node.text())
    expect(nodeLabels).toContain('private')
    expect(nodeLabels).not.toContain('mgmt')
    wrapper.unmount()
  })

  it('restores expanded nodes from localStorage when available', async () => {
    const storedNodes = ['1.3.6.1.4', 'custom-node']
    window.localStorage.setItem('mib-tree-expanded-nodes', JSON.stringify(storedNodes))

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const expandedNodes = wrapper.vm.expandedNodes
    expect(expandedNodes.has('1.3.6.1.4')).toBe(true)
    expect(expandedNodes.has('custom-node')).toBe(true)

    wrapper.unmount()
  })

  it('persists expanded node updates to localStorage', async () => {
    const storageProto = Object.getPrototypeOf(window.localStorage)
    const setItemSpy = vi.spyOn(storageProto, 'setItem')

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()
    setItemSpy.mockClear()

    await wrapper.vm.toggleNode('1.3.6.1.4')
    await nextTick()

    expect(setItemSpy).toHaveBeenCalled()
    const lastCall = setItemSpy.mock.calls.at(-1)
    expect(lastCall[0]).toBe('mib-tree-expanded-nodes')
    expect(JSON.parse(lastCall[1])).toEqual(expect.arrayContaining(['1.3.6.1.4']))

    setItemSpy.mockRestore()
    wrapper.unmount()
  })

  it('shows context menu options for scalar nodes', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: {
        selectedOid: ''
      },
      global: baseGlobalConfig
    })

    await flushPromises()

    const scalarNode = wrapper.findAll('.tree-node').find((el) => el.text().includes('sysUpTime'))
    expect(scalarNode, 'Scalar node should be rendered').toBeTruthy()

    await scalarNode.trigger('contextmenu', { clientX: 50, clientY: 60 })
    await flushPromises()
    await nextTick()
    const menuState = wrapper.vm.contextMenuState
    expect(menuState.visible).toBe(true)
    expect(wrapper.emitted()['oid-select']?.[0]).toEqual(['1.3.6.1.2.1.1.3'])

    const labels = menuState.items
      .filter((item) => item.type !== 'divider')
      .map((item) => item.label)
      .filter(Boolean)
    expect(labels).toContain('Execute GET')
    expect(labels).toContain('Execute GET NEXT')
    expect(labels).toContain('Open as chart')

    await wrapper.vm.handleContextMenuItemSelect(menuState.items[0])
    await flushPromises()

    const operationEvents = wrapper.emitted()['request-operation'] || []
    expect(operationEvents[0][0]).toMatchObject({
      operation: 'get',
      node: expect.objectContaining({ oid: '1.3.6.1.2.1.1.3' })
    })

    wrapper.unmount()
  })

  it('closes context menu on Escape key press', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: {
        selectedOid: ''
      },
      global: baseGlobalConfig
    })

    await flushPromises()

    const scalarNode = wrapper.findAll('.tree-node').find((el) => el.text().includes('sysUpTime'))
    await scalarNode.trigger('contextmenu')
    await flushPromises()
    await nextTick()

    expect(wrapper.vm.contextMenuState.visible).toBe(true)

    // Simulate Escape key press on the window
    const event = new KeyboardEvent('keydown', { key: 'Escape' })
    window.dispatchEvent(event)
    await flushPromises()
    await nextTick()

    expect(wrapper.vm.contextMenuState.visible).toBe(false)
    wrapper.unmount()
  })

  it('supports table actions from context menu', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: {
        selectedOid: ''
      },
      global: baseGlobalConfig
    })

    await flushPromises()

    const tableNode = wrapper.findAll('.tree-node').find((el) => el.text().includes('private'))
    expect(tableNode, 'Table node should be rendered').toBeTruthy()

    await tableNode.trigger('contextmenu', { clientX: 80, clientY: 90 })
    await flushPromises()
    await nextTick()

    let menuState = wrapper.vm.contextMenuState
    expect(menuState.visible).toBe(true)
    expect(menuState.items[0].label).toBe('Execute WALK')
    await wrapper.vm.handleContextMenuItemSelect(menuState.items[0])

    const operationEvents = wrapper.emitted()['request-operation'] || []
    expect(operationEvents[0][0]).toMatchObject({
      operation: 'walk',
      node: expect.objectContaining({ oid: '1.3.6.1.4' })
    })

    await tableNode.trigger('contextmenu', { clientX: 80, clientY: 90 })
    await flushPromises()
    await nextTick()

    menuState = wrapper.vm.contextMenuState
    const openTableItem = menuState.items.find((item) => item.type === 'open-table')
    expect(openTableItem, 'Open table action should be present').toBeTruthy()
    await wrapper.vm.handleContextMenuItemSelect(openTableItem)
    await flushPromises()

    const tableEvents = wrapper.emitted()['open-table'] || []
    expect(tableEvents[tableEvents.length - 1][0]).toMatchObject({ oid: '1.3.6.1.4' })

    wrapper.unmount()
  })
})

describe('MibTreeSidebar (node)', () => {
  it('emits oid-select when a node is clicked', async () => {
    const node = { oid: '1.3.6.1.2', name: 'mgmt', type: 'folder' }

    const wrapper = mount(MibTreeSidebar, {
      props: {
        node,
        level: 1,
        selectedOid: ''
      },
      global: baseGlobalConfig
    })

    await wrapper.find('.tree-node').trigger('click')
    expect(wrapper.emitted()['oid-select'][0]).toEqual([node.oid])
    wrapper.unmount()
  })

  it('emits open-table when a table node is double clicked', async () => {
    const node = { oid: '1.3.6.1.4', name: 'private', type: 'table' }

    const wrapper = mount(MibTreeSidebar, {
      props: {
        node,
        level: 1,
        selectedOid: ''
      },
      global: baseGlobalConfig
    })

    await wrapper.find('.tree-node').trigger('dblclick')
    expect(wrapper.emitted()['open-table'][0]).toEqual([node])
    wrapper.unmount()
  })
})

describe('MibTreeSidebar (bookmarks)', () => {
  const treeWithBookmarks = [
    {
      oid: 'bookmarks',
      name: 'Bookmarks',
      type: 'bookmark-root',
      children: [
        {
          oid: '1.3.6.1.2.1.1.3',
          name: 'sysUpTime',
          parentOid: 'bookmarks',
          type: 'bookmark',
          syntax: 'TimeTicks',
          access: 'read-only'
        },
        {
          oid: '1.3.6.1.4',
          name: 'private',
          parentOid: 'bookmarks',
          type: 'bookmark-table',
          children: []
        },
        {
          oid: '1.3.6.1.2.1.2.2.1.10',
          name: 'ifInOctets',
          parentOid: 'bookmarks',
          type: 'bookmark-column',
          syntax: 'Counter32'
        }
      ]
    },
    ...sampleTree
  ]

  const treeWithBookmarkFolder = [
    {
      oid: 'bookmarks',
      name: 'Bookmarks',
      type: 'bookmark-root',
      children: [
        {
          oid: 'bookmark-folder:1',
          name: 'Saved nodes',
          parentOid: 'bookmarks',
          type: 'bookmark-folder',
          children: [
            {
              oid: '1.3.6.1.2.1.1.5',
              name: 'sysName',
              parentOid: 'bookmark-folder:1',
              type: 'bookmark',
              syntax: 'OctetString',
              access: 'read-only'
            }
          ]
        }
      ]
    },
    ...sampleTree
  ]

  const treeWithNestedBookmarkFolders = [
    {
      oid: 'bookmarks',
      name: 'Bookmarks',
      type: 'bookmark-root',
      children: [
        {
          oid: 'bookmark-folder:1',
          name: 'Network',
          parentOid: 'bookmarks',
          type: 'bookmark-folder',
          children: [
            {
              oid: 'bookmark-folder:2',
              name: 'Interfaces',
              parentOid: 'bookmark-folder:1',
              type: 'bookmark-folder',
              children: []
            }
          ]
        }
      ]
    },
    ...sampleTree
  ]

  beforeEach(() => {
    GetMIBTree.mockResolvedValue(treeWithBookmarks)
    AddBookmark.mockResolvedValue(undefined)
    RemoveBookmark.mockResolvedValue(undefined)
  })

  it('renders bookmark root and bookmark nodes', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const nodeLabels = wrapper.findAll('.node-name').map((node) => node.text())
    expect(nodeLabels).toContain('Bookmarks')
    expect(nodeLabels).toContain('sysUpTime')
    expect(nodeLabels).toContain('private')

    wrapper.unmount()
  })

  it('keeps a single-bookmark folder expanded as folder', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '', initialTree: treeWithBookmarkFolder },
      global: baseGlobalConfig
    })

    await flushPromises()

    const tree = wrapper.vm.filteredTree
    const root = tree.find((node) => node.oid === 'bookmarks')
    expect(root).toBeTruthy()
    const folder = root.children.find((child) => child.type === 'bookmark-folder')
    expect(folder).toBeTruthy()
    expect(folder.displayName || folder.name).toBe('Saved nodes')
    expect(Array.isArray(folder.children)).toBe(true)
    expect(folder.children.length).toBe(1)
    expect(folder.children[0].name).toBe('sysName')

    wrapper.unmount()
  })

  it('shows correct icon for bookmark root and bookmark nodes', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const bookmarkRootNode = wrapper.findAll('.tree-node').find(el => el.text().includes('Bookmarks'))
    expect(bookmarkRootNode).toBeTruthy()
    const rootIcon = bookmarkRootNode.find('.node-icon')
    expect(rootIcon.text().trim()).toBe('folder_special')

    const bookmarkNode = wrapper.findAll('.tree-node.node-bookmark').at(0)
    expect(bookmarkNode).toBeTruthy()
    const bookmarkIcon = bookmarkNode.find('.node-icon')
    expect(bookmarkIcon.text().trim()).toBe('bookmark')

    wrapper.unmount()
  })

  it('shows correct context menu for bookmark with original type operations', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    // Test bookmark di tipo scalar
    const bookmarkScalar = wrapper.findAll('.tree-node.node-bookmark').at(0)
    await bookmarkScalar.trigger('contextmenu', { clientX: 100, clientY: 100 })
    await flushPromises()
    await nextTick()

    let menuState = wrapper.vm.contextMenuState
    expect(menuState.visible).toBe(true)

    const labels = menuState.items
      .filter(item => item.type !== 'divider')
      .map(item => item.label)

    expect(labels).toContain('Execute GET')
    expect(labels).toContain('Execute GET NEXT')
    expect(labels).toContain('Remove bookmark')
    expect(labels).not.toContain('Add to bookmarks')

    wrapper.unmount()
  })

  it('shows correct context menu for bookmark-table with table operations', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const bookmarkTable = wrapper.findAll('.tree-node.node-bookmark-table').at(0)
    await bookmarkTable.trigger('contextmenu', { clientX: 100, clientY: 100 })
    await flushPromises()
    await nextTick()

    const menuState = wrapper.vm.contextMenuState
    const labels = menuState.items
      .filter(item => item.type !== 'divider')
      .map(item => item.label)

    expect(labels).toContain('Execute WALK')
    expect(labels).toContain('Execute GET BULK')
    expect(labels).toContain('Open as table')
    expect(labels).toContain('Remove bookmark')

    wrapper.unmount()
  })

  it('shows correct context menu for bookmark-column with chart option', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const bookmarkColumn = wrapper.findAll('.tree-node.node-bookmark-column').at(0)
    await bookmarkColumn.trigger('contextmenu', { clientX: 100, clientY: 100 })
    await flushPromises()
    await nextTick()

    const menuState = wrapper.vm.contextMenuState
    const labels = menuState.items
      .filter(item => item.type !== 'divider')
      .map(item => item.label)

    expect(labels).toContain('Open as chart')
    expect(labels).toContain('Remove bookmark')

    wrapper.unmount()
  })

  it('shows correct context menu for bookmark-node with walk operation', async () => {
    // Mock tree con bookmark di tipo node (es. "system")
    const treeWithNodeBookmark = [
      {
        oid: 'bookmarks',
        name: 'Bookmarks',
        type: 'bookmark-root',
        children: [
          {
            oid: '1.3.6.1.2.1.1',
            name: 'system',
            parentOid: 'bookmarks',
            type: 'bookmark-node',
            children: [] // I bookmark non hanno children
          }
        ]
      },
      {
        oid: '1.3.6.1',
        name: 'iso',
        type: 'node',
        children: []
      }
    ]

    GetMIBTree.mockResolvedValue(treeWithNodeBookmark)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const bookmarkNode = wrapper.find('.tree-node.node-bookmark-node')
    expect(bookmarkNode.exists()).toBe(true)

    await bookmarkNode.trigger('contextmenu', { clientX: 100, clientY: 100 })
    await flushPromises()
    await nextTick()

    const menuState = wrapper.vm.contextMenuState
    const labels = menuState.items
      .filter(item => item.type !== 'divider')
      .map(item => item.label)

    // Un bookmark-node deve avere walk anche senza children
    expect(labels).toContain('Execute WALK')
    expect(labels).toContain('Remove bookmark')
    expect(labels).not.toContain('Add to bookmarks')

    wrapper.unmount()
  })

  it('does not highlight bookmark when original node is selected', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '1.3.6.1.2.1.1.3' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const bookmarkNode = wrapper.findAll('.tree-node.node-bookmark').at(0)
    expect(bookmarkNode.classes()).not.toContain('selected')

    // Nota: il nodo originale potrebbe non essere renderizzato se è collassato nell'albero
    // ma il bookmark non deve MAI essere evidenziato quando si seleziona per OID

    wrapper.unmount()
  })

  it('highlights only bookmark when bookmark is selected', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '1.3.6.1.2.1.1.3' },
      global: baseGlobalConfig
    })

    await flushPromises()

    // Clicca sul bookmark
    const bookmarkNode = wrapper.findAll('.tree-node.node-bookmark').at(0)
    await bookmarkNode.trigger('click')
    await flushPromises()

    // Verifica che il bookmark non sia evidenziato (perché il click seleziona l'originale)
    expect(bookmarkNode.classes()).not.toContain('selected')

    wrapper.unmount()
  })

  it('retrieves original node details when bookmark is selected', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '1.3.6.1.2.1.1.3' },
      global: baseGlobalConfig
    })

    await flushPromises()

    // selectedNodeDetails dovrebbe cercare nell'albero MIB, non nei bookmark
    const details = wrapper.vm.selectedNodeDetails
    expect(details).toBeTruthy()
    expect(details.type).toBe('scalar')
    expect(details.type).not.toBe('bookmark')

    wrapper.unmount()
  })

  it('emits bookmark-click when bookmark is clicked in child component', async () => {
    const bookmarkNode = {
      oid: '1.3.6.1.2.1.1.3',
      name: 'sysUpTime',
      parentOid: 'bookmarks',
      type: 'bookmark'
    }

    const wrapper = mount(MibTreeSidebar, {
      props: {
        node: bookmarkNode,
        level: 2, // Non-root component
        selectedOid: ''
      },
      global: baseGlobalConfig
    })

    await wrapper.find('.tree-node').trigger('click')
    // I componenti non-root emettono bookmark-click, non oid-select
    expect(wrapper.emitted()['bookmark-click']).toBeTruthy()
    expect(wrapper.emitted()['bookmark-click'][0]).toEqual([bookmarkNode])

    wrapper.unmount()
  })

  it('opens modal and confirms AddBookmark when add-bookmark menu item is selected', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const originalNode = wrapper.findAll('.tree-node').find(el =>
      el.text().includes('sysUpTime') && !el.classes().includes('node-bookmark')
    )

    if (originalNode) {
      await originalNode.trigger('contextmenu', { clientX: 100, clientY: 100 })
      await flushPromises()
      await nextTick()

      const menuState = wrapper.vm.contextMenuState
      const addBookmarkItem = menuState.items.find(item => item.type === 'add-bookmark')

      if (addBookmarkItem) {
        AddBookmark.mockClear()
        await wrapper.vm.handleContextMenuItemSelect(addBookmarkItem)
        expect(wrapper.vm.addBookmarkModalState.visible).toBe(true)
        const targetOid = wrapper.vm.addBookmarkModalState.targetNode?.oid
        await wrapper.vm.handleAddBookmarkConfirm('bookmarks')
        await flushPromises()
        expect(AddBookmark).toHaveBeenCalledWith(targetOid, 'bookmarks')
      }
    }

    wrapper.unmount()
  })

  it('creates a bookmark folder from the context menu', async () => {
    GetMIBTree.mockResolvedValue(treeWithBookmarkFolder)
    CreateBookmarkFolder.mockResolvedValue({
      id: 2,
      name: 'Child',
      key: 'bookmark-folder:2',
      parentKey: 'bookmark-folder:1',
      createdAt: new Date().toISOString()
    })

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const root = wrapper.vm.bookmarkRootNode
    expect(root).toBeTruthy()
    const folderNode = root.children.find((child) => child.type === 'bookmark-folder')
    expect(folderNode).toBeTruthy()

    wrapper.vm.showContextMenu({ clientX: 120, clientY: 120 }, folderNode)
    await nextTick()

    const newFolderItem = wrapper.vm.contextMenuState.items.find((item) => item.type === 'new-folder')
    expect(newFolderItem).toBeTruthy()

    await wrapper.vm.handleContextMenuItemSelect(newFolderItem)
    expect(wrapper.vm.folderModalState.visible).toBe(true)

    await wrapper.vm.handleFolderModalConfirm('Child')
    await flushPromises()

    expect(CreateBookmarkFolder).toHaveBeenCalledWith('Child', 'bookmark-folder:1')

    wrapper.unmount()
  })

  it('renames a bookmark folder from the context menu', async () => {
    GetMIBTree.mockResolvedValue(treeWithBookmarkFolder)
    RenameBookmarkFolder.mockResolvedValue(undefined)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const root = wrapper.vm.bookmarkRootNode
    expect(root).toBeTruthy()
    const folderNode = root.children.find((child) => child.type === 'bookmark-folder')
    expect(folderNode).toBeTruthy()

    wrapper.vm.showContextMenu({ clientX: 140, clientY: 140 }, folderNode)
    await nextTick()

    const renameItem = wrapper.vm.contextMenuState.items.find((item) => item.type === 'rename-folder')
    expect(renameItem).toBeTruthy()

    await wrapper.vm.handleContextMenuItemSelect(renameItem)
    expect(wrapper.vm.folderModalState.visible).toBe(true)
    expect(wrapper.vm.folderModalState.mode).toBe('rename')

    await wrapper.vm.handleFolderModalConfirm('Renamed folder')
    await flushPromises()

    expect(RenameBookmarkFolder).toHaveBeenCalledWith('bookmark-folder:1', 'Renamed folder')

    wrapper.unmount()
  })

  it('deletes a bookmark folder after confirmation', async () => {
    GetMIBTree.mockResolvedValue(treeWithBookmarkFolder)
    DeleteBookmarkFolder.mockResolvedValue(undefined)
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const root = wrapper.vm.bookmarkRootNode
    expect(root).toBeTruthy()
    const folderNode = root.children.find((child) => child.type === 'bookmark-folder')
    expect(folderNode).toBeTruthy()

    wrapper.vm.showContextMenu({ clientX: 160, clientY: 160 }, folderNode)
    await nextTick()

    const deleteItem = wrapper.vm.contextMenuState.items.find((item) => item.type === 'delete-folder')
    expect(deleteItem).toBeTruthy()

    await wrapper.vm.handleContextMenuItemSelect(deleteItem)
    await flushPromises()

    expect(DeleteBookmarkFolder).toHaveBeenCalledWith('bookmark-folder:1')

    confirmSpy.mockRestore()
    wrapper.unmount()
  })

  it('moves a bookmark via drag and drop', async () => {
    GetMIBTree.mockResolvedValue(treeWithBookmarkFolder)
    MoveBookmark.mockResolvedValue(undefined)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const root = wrapper.vm.bookmarkRootNode
    expect(root).toBeTruthy()
    const folderNode = root.children.find((child) => child.type === 'bookmark-folder')
    expect(folderNode).toBeTruthy()
    const bookmarkNode = folderNode.children.find((child) => child.type === 'bookmark')
    expect(bookmarkNode).toBeTruthy()

    MoveBookmark.mockClear()

    wrapper.vm.handleDragStart({ dataTransfer: { setData: vi.fn(), effectAllowed: '' } }, bookmarkNode)
    wrapper.vm.handleDragEnter({ preventDefault: () => {} }, root)
    wrapper.vm.handleDragOver({ preventDefault: () => {}, dataTransfer: { dropEffect: '' } }, root)
    await wrapper.vm.handleDrop({ preventDefault: () => {} }, root)
    await flushPromises()

    expect(MoveBookmark).toHaveBeenCalledWith('1.3.6.1.2.1.1.5', 'bookmarks')

    wrapper.unmount()
  })

  it('moves a bookmark folder via drag and drop', async () => {
    GetMIBTree.mockResolvedValue(treeWithNestedBookmarkFolders)
    MoveBookmarkFolder.mockResolvedValue(undefined)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const root = wrapper.vm.bookmarkRootNode
    expect(root).toBeTruthy()
    const parentFolder = root.children.find((child) => child.oid === 'bookmark-folder:1')
    const childFolder = parentFolder?.children.find((child) => child.oid === 'bookmark-folder:2')
    expect(parentFolder).toBeTruthy()
    expect(childFolder).toBeTruthy()

    MoveBookmarkFolder.mockClear()

    wrapper.vm.handleDragStart({ dataTransfer: { setData: vi.fn(), effectAllowed: '' } }, childFolder)
    wrapper.vm.handleDragEnter({ preventDefault: () => {} }, root)
    wrapper.vm.handleDragOver({ preventDefault: () => {}, dataTransfer: { dropEffect: '' } }, root)
    await wrapper.vm.handleDrop({ preventDefault: () => {} }, root)
    await flushPromises()

    expect(MoveBookmarkFolder).toHaveBeenCalledWith('bookmark-folder:2', 'bookmarks')

    wrapper.unmount()
  })

  it('imposta il target di drop quando un bookmark è sopra una cartella', async () => {
    GetMIBTree.mockResolvedValue(treeWithBookmarkFolder)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()
    await nextTick()

    const root = wrapper.vm.bookmarkRootNode
    expect(root).toBeTruthy()
    const folder = root.children.find((child) => child.type === 'bookmark-folder')
    const bookmark = folder?.children.find((child) => child.type !== 'bookmark-folder')
    expect(folder).toBeTruthy()
    expect(bookmark).toBeTruthy()

    const setData = vi.fn()
    wrapper.vm.handleDragStart({ dataTransfer: { setData, effectAllowed: '' } }, bookmark)
    await nextTick()

    const preventDefault = vi.fn()
    wrapper.vm.handleDragOver({ preventDefault, dataTransfer: { dropEffect: '' } }, folder)
    await nextTick()
    await nextTick()

    expect(preventDefault).toHaveBeenCalled()
    expect(wrapper.vm.dropTargetKey).toBe(folder.oid)

    wrapper.unmount()
  })

  it('calls RemoveBookmark when remove-bookmark menu item is selected', async () => {
    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const bookmarkNode = wrapper.findAll('.tree-node.node-bookmark').at(0)
    await bookmarkNode.trigger('contextmenu', { clientX: 100, clientY: 100 })
    await flushPromises()
    await nextTick()

    const menuState = wrapper.vm.contextMenuState
    const removeBookmarkItem = menuState.items.find(item => item.type === 'remove-bookmark')

    expect(removeBookmarkItem).toBeTruthy()
    await wrapper.vm.handleContextMenuItemSelect(removeBookmarkItem)
    await flushPromises()

    expect(RemoveBookmark).toHaveBeenCalled()

    wrapper.unmount()
  })

  it('expands ancestor nodes when clicking on nested bookmark', async () => {
    // Tree con nodi nested per testare l'espansione
    const nestedTree = [
      {
        oid: 'bookmarks',
        name: 'Bookmarks',
        type: 'bookmark-root',
        children: [
          {
            oid: '1.3.6.1.2.1.2.2',
            name: 'ifTable',
            parentOid: 'bookmarks',
            type: 'bookmark-table'
          }
        ]
      },
      {
        oid: '1.3.6.1',
        name: 'iso',
        type: 'node',
        children: [
          {
            oid: '1.3.6.1.2',
            name: 'mgmt',
            type: 'node',
            children: [
              {
                oid: '1.3.6.1.2.1',
                name: 'mib-2',
                type: 'node',
                children: [
                  {
                    oid: '1.3.6.1.2.1.2',
                    name: 'interfaces',
                    type: 'node',
                    children: [
                      {
                        oid: '1.3.6.1.2.1.2.2',
                        name: 'ifTable',
                        type: 'table'
                      }
                    ]
                  }
                ]
              }
            ]
          }
        ]
      }
    ]

    GetMIBTree.mockResolvedValue(nestedTree)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    // Verifica che inizialmente il nodo interfaces non sia espanso
    // (alcuni nodi sono espansi di default: '1.3.6.1', '1.3.6.1.2', '1.3.6.1.2.1')
    expect(wrapper.vm.expandedNodes.has('1.3.6.1.2.1.2')).toBe(false) // interfaces dovrebbe essere chiuso

    // Trova il bookmark ifTable node object per chiamare handleClick direttamente
    const ifTableBookmark = nestedTree[0].children[0]

    // Chiama handleClick direttamente invece di trigger('click')
    // perché il test non propaga correttamente gli eventi async
    await wrapper.vm.handleClick(ifTableBookmark)
    await flushPromises()
    await nextTick()

    // Verifica che tutti i nodi antenati siano stati espansi
    // IMPORTANTE: Leggi expandedNodes DOPO il click perché viene sostituito con un nuovo Set
    const expandedNodesAfterClick = wrapper.vm.expandedNodes
    expect(expandedNodesAfterClick.has('1.3.6.1')).toBe(true)
    expect(expandedNodesAfterClick.has('1.3.6.1.2')).toBe(true)
    expect(expandedNodesAfterClick.has('1.3.6.1.2.1')).toBe(true)
    expect(expandedNodesAfterClick.has('1.3.6.1.2.1.2')).toBe(true)

    // Verifica che l'OID sia stato selezionato
    expect(wrapper.emitted()['oid-select']).toBeTruthy()
    const selectEvents = wrapper.emitted()['oid-select']
    const lastSelectEvent = selectEvents[selectEvents.length - 1]
    expect(lastSelectEvent).toEqual(['1.3.6.1.2.1.2.2'])

    wrapper.unmount()
  })

  it('expandPathToNode returns correct path for nested nodes', async () => {
    const nestedTree = [
      {
        oid: '1.3.6.1',
        name: 'iso',
        type: 'node',
        children: [
          {
            oid: '1.3.6.1.2',
            name: 'mgmt',
            type: 'node',
            children: [
              {
                oid: '1.3.6.1.2.1',
                name: 'mib-2',
                type: 'node',
                children: [
                  {
                    oid: '1.3.6.1.2.1.2',
                    name: 'interfaces',
                    type: 'node',
                    children: [
                      {
                        oid: '1.3.6.1.2.1.2.2',
                        name: 'ifTable',
                        type: 'table'
                      }
                    ]
                  }
                ]
              }
            ]
          }
        ]
      }
    ]

    GetMIBTree.mockResolvedValue(nestedTree)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    // Chiama direttamente expandPathToNode per testare la logica
    const result = wrapper.vm.expandPathToNode('1.3.6.1.2.1.2.2', nestedTree)

    expect(result).toBe(true)

    const expandedNodes = wrapper.vm.expandedNodes
    // Tutti gli antenati dovrebbero essere espansi (ma non il target stesso necessariamente)
    expect(expandedNodes.has('1.3.6.1')).toBe(true)
    expect(expandedNodes.has('1.3.6.1.2')).toBe(true)
    expect(expandedNodes.has('1.3.6.1.2.1')).toBe(true)
    expect(expandedNodes.has('1.3.6.1.2.1.2')).toBe(true)

    wrapper.unmount()
  })
})
