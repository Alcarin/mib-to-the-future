import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { nextTick } from 'vue'

vi.mock('@material/web/textfield/outlined-text-field.js', () => ({}))
vi.mock('@material/web/iconbutton/icon-button.js', () => ({}))
vi.mock('@material/web/button/filled-button.js', () => ({}))
vi.mock('@material/web/button/outlined-button.js', () => ({}))

import MibTreeSidebar from '../components/MibTreeSidebar.vue'
import {
  GetMIBTree,
  MoveBookmark,
  MoveBookmarkFolder
} from '../../wailsjs/go/app/App'

vi.mock('../../wailsjs/go/app/App')

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
          <input data-test="search-input" type="text" :value="search" @input="$emit('update:search', $event.target.value)" />
        </div>
      `
    },
    OidDetailsPanel: {
      props: ['node'],
      template: '<div class="oid-details" :data-node="node ? node.oid : null"></div>'
    },
    'md-icon-button': {
      template: `<button class="md-icon-button" type="button" @click="$emit('click', $event)" v-bind="$attrs"><slot /></button>`
    }
  }
}

const treeWithBookmarks = [
  {
    oid: 'bookmarks',
    name: 'Bookmarks',
    type: 'bookmark-root',
    children: [
      {
        oid: 'bookmark-folder:1',
        name: 'Folder 1',
        type: 'bookmark-folder',
        parentOid: 'bookmarks',
        children: [
          {
            oid: '1.3.6.1.2.1.1.1',
            name: 'sysDescr',
            type: 'bookmark',
            parentOid: 'bookmark-folder:1'
          }
        ]
      },
      {
        oid: 'bookmark-folder:2',
        name: 'Folder 2',
        type: 'bookmark-folder',
        parentOid: 'bookmarks',
        children: []
      },
      {
        oid: '1.3.6.1.2.1.1.5',
        name: 'sysName',
        type: 'bookmark',
        parentOid: 'bookmarks'
      }
    ]
  }
]

beforeEach(() => {
  vi.resetAllMocks()
  if (typeof window !== 'undefined' && window.localStorage) {
    window.localStorage.clear()
  }
})

describe('Bookmark Drag & Drop', () => {
  it('should share draggedNodeKey across recursive instances via provide/inject', async () => {
    GetMIBTree.mockResolvedValue(treeWithBookmarks)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const root = wrapper.vm.bookmarkRootNode
    const bookmark = root.children.find((child) => child.oid === '1.3.6.1.2.1.1.5')

    // Simulate drag start
    wrapper.vm.handleDragStart({ dataTransfer: { setData: vi.fn(), effectAllowed: '' } }, bookmark)
    await nextTick()

    // draggedNodeKey should be set
    expect(wrapper.vm.draggedNodeKey).toBe('1.3.6.1.2.1.1.5')

    wrapper.unmount()
  })

  it('should only mark target node with drop-target class, not all eligible nodes', async () => {
    GetMIBTree.mockResolvedValue(treeWithBookmarks)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const root = wrapper.vm.bookmarkRootNode
    const bookmark = root.children.find((child) => child.oid === '1.3.6.1.2.1.1.5')
    const folder1 = root.children.find((child) => child.oid === 'bookmark-folder:1')
    const folder2 = root.children.find((child) => child.oid === 'bookmark-folder:2')

    // Start dragging bookmark
    wrapper.vm.handleDragStart({ dataTransfer: { setData: vi.fn(), effectAllowed: '' } }, bookmark)
    await nextTick()

    // Before hovering, no target
    expect(wrapper.vm.dropTargetKey).toBeNull()
    expect(wrapper.vm.getNodeClass(folder1)).not.toContain('drop-target')
    expect(wrapper.vm.getNodeClass(folder2)).not.toContain('drop-target')

    // Hover over folder1
    wrapper.vm.handleDragOver({ preventDefault: () => {}, dataTransfer: { dropEffect: '' } }, folder1)
    await nextTick()

    // Only folder1 should have drop-target class
    expect(wrapper.vm.dropTargetKey).toBe('bookmark-folder:1')
    expect(wrapper.vm.getNodeClass(folder1)).toContain('drop-target')
    expect(wrapper.vm.getNodeClass(folder2)).not.toContain('drop-target')

    // No drop-eligible class should be applied to any node
    expect(wrapper.vm.getNodeClass(folder1)).not.toContain('drop-eligible')
    expect(wrapper.vm.getNodeClass(folder2)).not.toContain('drop-eligible')

    wrapper.unmount()
  })

  it('should move bookmark to different folder and reload tree', async () => {
    GetMIBTree.mockResolvedValue(treeWithBookmarks)
    MoveBookmark.mockResolvedValue(undefined)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const root = wrapper.vm.bookmarkRootNode
    const bookmark = root.children.find((child) => child.oid === '1.3.6.1.2.1.1.5')
    const folder2 = root.children.find((child) => child.oid === 'bookmark-folder:2')

    expect(bookmark.parentOid).toBe('bookmarks')

    // Simulate drag and drop
    wrapper.vm.handleDragStart({ dataTransfer: { setData: vi.fn(), effectAllowed: '' } }, bookmark)
    await nextTick()

    wrapper.vm.handleDragOver({ preventDefault: () => {}, dataTransfer: { dropEffect: '' } }, folder2)
    await nextTick()

    await wrapper.vm.handleDrop({ preventDefault: () => {} }, folder2)
    await flushPromises()

    // MoveBookmark should be called with correct params
    expect(MoveBookmark).toHaveBeenCalledWith('1.3.6.1.2.1.1.5', 'bookmark-folder:2')

    // GetMIBTree should be called again to reload
    expect(GetMIBTree).toHaveBeenCalledTimes(2) // Initial + reload

    wrapper.unmount()
  })

  it('should move folder to different parent folder', async () => {
    const treeWithNestedFolders = [
      {
        oid: 'bookmarks',
        name: 'Bookmarks',
        type: 'bookmark-root',
        children: [
          {
            oid: 'bookmark-folder:1',
            name: 'Parent Folder',
            type: 'bookmark-folder',
            parentOid: 'bookmarks',
            children: [
              {
                oid: 'bookmark-folder:2',
                name: 'Child Folder',
                type: 'bookmark-folder',
                parentOid: 'bookmark-folder:1',
                children: []
              }
            ]
          },
          {
            oid: 'bookmark-folder:3',
            name: 'Target Folder',
            type: 'bookmark-folder',
            parentOid: 'bookmarks',
            children: []
          }
        ]
      }
    ]

    GetMIBTree.mockResolvedValue(treeWithNestedFolders)
    MoveBookmarkFolder.mockResolvedValue(undefined)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const root = wrapper.vm.bookmarkRootNode
    const childFolder = root.children[0].children[0]
    const targetFolder = root.children.find((child) => child.oid === 'bookmark-folder:3')

    wrapper.vm.handleDragStart({ dataTransfer: { setData: vi.fn(), effectAllowed: '' } }, childFolder)
    await nextTick()

    await wrapper.vm.handleDrop({ preventDefault: () => {} }, targetFolder)
    await flushPromises()

    expect(MoveBookmarkFolder).toHaveBeenCalledWith('bookmark-folder:2', 'bookmark-folder:3')

    wrapper.unmount()
  })

  it('should not allow dropping node onto itself', async () => {
    GetMIBTree.mockResolvedValue(treeWithBookmarks)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const root = wrapper.vm.bookmarkRootNode
    const folder = root.children.find((child) => child.oid === 'bookmark-folder:1')

    wrapper.vm.handleDragStart({ dataTransfer: { setData: vi.fn(), effectAllowed: '' } }, folder)
    await nextTick()

    const canDrop = wrapper.vm.canDropOnNode(folder, folder)
    expect(canDrop).toBe(false)

    wrapper.unmount()
  })

  it('should not allow dropping parent folder into its own child', async () => {
    const treeWithNestedFolders = [
      {
        oid: 'bookmarks',
        name: 'Bookmarks',
        type: 'bookmark-root',
        children: [
          {
            oid: 'bookmark-folder:1',
            name: 'Parent',
            type: 'bookmark-folder',
            parentOid: 'bookmarks',
            children: [
              {
                oid: 'bookmark-folder:2',
                name: 'Child',
                type: 'bookmark-folder',
                parentOid: 'bookmark-folder:1',
                children: []
              }
            ]
          }
        ]
      }
    ]

    GetMIBTree.mockResolvedValue(treeWithNestedFolders)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const root = wrapper.vm.bookmarkRootNode
    const parentFolder = root.children[0]
    const childFolder = parentFolder.children[0]

    wrapper.vm.handleDragStart({ dataTransfer: { setData: vi.fn(), effectAllowed: '' } }, parentFolder)
    await nextTick()

    const canDrop = wrapper.vm.canDropOnNode(childFolder, parentFolder)
    expect(canDrop).toBe(false)

    wrapper.unmount()
  })

  it('should not move item if dropped on same parent', async () => {
    GetMIBTree.mockResolvedValue(treeWithBookmarks)
    MoveBookmark.mockResolvedValue(undefined)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const root = wrapper.vm.bookmarkRootNode
    const bookmark = root.children.find((child) => child.oid === '1.3.6.1.2.1.1.5')

    expect(bookmark.parentOid).toBe('bookmarks')

    wrapper.vm.handleDragStart({ dataTransfer: { setData: vi.fn(), effectAllowed: '' } }, bookmark)
    await nextTick()

    await wrapper.vm.handleDrop({ preventDefault: () => {} }, root)
    await flushPromises()

    // MoveBookmark should NOT be called since it's the same parent
    expect(MoveBookmark).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('should clear drag state after drop', async () => {
    GetMIBTree.mockResolvedValue(treeWithBookmarks)
    MoveBookmark.mockResolvedValue(undefined)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const root = wrapper.vm.bookmarkRootNode
    const bookmark = root.children.find((child) => child.oid === '1.3.6.1.2.1.1.5')
    const folder = root.children.find((child) => child.oid === 'bookmark-folder:1')

    wrapper.vm.handleDragStart({ dataTransfer: { setData: vi.fn(), effectAllowed: '' } }, bookmark)
    await nextTick()

    expect(wrapper.vm.draggedNodeKey).toBe('1.3.6.1.2.1.1.5')

    await wrapper.vm.handleDrop({ preventDefault: () => {} }, folder)
    await flushPromises()

    // State should be cleared
    expect(wrapper.vm.draggedNodeKey).toBeNull()
    expect(wrapper.vm.dropTargetKey).toBeNull()

    wrapper.unmount()
  })

  it('should find bookmark node in nested folder structure', async () => {
    const treeWithDeepNesting = [
      {
        oid: 'bookmarks',
        name: 'Bookmarks',
        type: 'bookmark-root',
        children: [
          {
            oid: 'bookmark-folder:1',
            name: 'Level 1',
            type: 'bookmark-folder',
            parentOid: 'bookmarks',
            children: [
              {
                oid: 'bookmark-folder:2',
                name: 'Level 2',
                type: 'bookmark-folder',
                parentOid: 'bookmark-folder:1',
                children: [
                  {
                    oid: '1.3.6.1.2.1.1.1',
                    name: 'Deep Bookmark',
                    type: 'bookmark',
                    parentOid: 'bookmark-folder:2'
                  }
                ]
              }
            ]
          }
        ]
      }
    ]

    GetMIBTree.mockResolvedValue(treeWithDeepNesting)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const found = wrapper.vm.findBookmarkNode('1.3.6.1.2.1.1.1')
    expect(found).toBeTruthy()
    expect(found.oid).toBe('1.3.6.1.2.1.1.1')
    expect(found.name).toBe('Deep Bookmark')

    wrapper.unmount()
  })

  it('should correctly identify draggable nodes', async () => {
    GetMIBTree.mockResolvedValue(treeWithBookmarks)

    const wrapper = mount(MibTreeSidebar, {
      props: { selectedOid: '' },
      global: baseGlobalConfig
    })

    await flushPromises()

    const root = wrapper.vm.bookmarkRootNode
    const folder = root.children.find((child) => child.type === 'bookmark-folder')
    const bookmark = root.children.find((child) => child.oid === '1.3.6.1.2.1.1.5')

    // Bookmark root should NOT be draggable
    expect(wrapper.vm.isDraggableNode(root)).toBe(false)

    // Bookmark folder should be draggable
    expect(wrapper.vm.isDraggableNode(folder)).toBe(true)

    // Bookmark entry should be draggable
    expect(wrapper.vm.isDraggableNode(bookmark)).toBe(true)

    wrapper.unmount()
  })
})
