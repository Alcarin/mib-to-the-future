<script setup>
/**
 * @vue-component
 * @description A recursive sidebar component for browsing a MIB tree.
 * It handles both the root view (with search and split-pane details) and individual tree nodes.
 *
 * @vue-prop {Object} [node=null] - The MIB node object to render. Required for recursive/child instances.
 * @vue-prop {Number} [level=0] - The nesting level of the node, used for indentation.
 * @vue-prop {Array} [initialTree=null] - The complete MIB tree structure, provided to the root component.
 * @vue-prop {String} [selectedOid=''] - The OID of the currently selected node, used for highlighting.
 * @vue-prop {Boolean} [isResizing=false] - A boolean indicating if a resize operation is in progress.
 * @vue-prop {Number} [reloadKey=0] - A key that can be changed to force a reload of the MIB tree data.
 *
 * @vue-event {string} oid-select - Fired when a user selects a MIB node. Payload is the OID string.
 * @vue-event {object} open-table - Fired when a user requests to open the table view for a MIB table node. Payload is the node object.
 * @vue-event {void} resize-start - Fired when the split-pane resizing begins.
 * @vue-event {void} resize-end - Fired when the split-pane resizing ends.
 */
import { ref, computed, onMounted, watch, reactive, onBeforeUnmount, nextTick, provide, inject } from 'vue'
import { Splitpanes, Pane } from 'splitpanes'
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
import { useErrorHandler } from '../composables/useErrorHandler'
import '@material/web/textfield/outlined-text-field.js'
import '@material/web/iconbutton/icon-button.js'
import '@material/web/button/filled-button.js'
import DeLoreanHeader from './DeLoreanHeader.vue'
import OidDetailsPanel from './OidDetailsPanel.vue'
import ContextMenuOverlay from './ContextMenuOverlay.vue'
import BookmarkFolderModal from './BookmarkFolderModal.vue'
import AddBookmarkModal from './AddBookmarkModal.vue'

const NUMERIC_SYNTAX_HINTS = [
  'INTEGER',
  'COUNTER',
  'GAUGE',
  'TIMETICKS',
  'UNSIGNED',
  'FLOAT',
  'DOUBLE',
  'ENUMERATED'
]

const OPERATION_CONFIG = {
  get: { label: 'Execute GET', icon: 'play_arrow' },
  getnext: { label: 'Execute GET NEXT', icon: 'skip_next' },
  getbulk: { label: 'Execute GET BULK', icon: 'dataset' },
  walk: { label: 'Execute WALK', icon: 'hiking' },
  set: { label: 'Execute SET', icon: 'edit' }
}

const STORAGE_KEY_EXPANDED = 'mib-tree-expanded-nodes'
const BOOKMARK_ROOT_KEY = 'bookmarks'
const BOOKMARK_FOLDER_PREFIX = 'bookmark-folder:'
const DEFAULT_EXPANDED_NODES = ['1.3.6.1', '1.3.6.1.2', '1.3.6.1.2.1', BOOKMARK_ROOT_KEY]

const canUseLocalStorage = () => typeof window !== 'undefined' && typeof window.localStorage !== 'undefined'

const createDefaultExpandedNodes = () => new Set(DEFAULT_EXPANDED_NODES)

const mergeWithDefaults = (values) => {
  const baseline = createDefaultExpandedNodes()
  values.forEach((oid) => {
    if (typeof oid === 'string' && oid.length > 0) {
      baseline.add(oid)
    }
  })
  return baseline
}

const loadExpandedNodesFromStorage = () => {
  if (!canUseLocalStorage()) return null
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY_EXPANDED)
    if (!raw) return null
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return null
    return parsed.filter((item) => typeof item === 'string')
  } catch (error) {
    handleWarning('Impossibile caricare lo stato dei nodi espansi', { logToConsole: true, showNotification: false })
    return null
  }
}

const persistExpandedNodesToStorage = (set) => {
  if (!canUseLocalStorage()) return
  try {
    window.localStorage.setItem(STORAGE_KEY_EXPANDED, JSON.stringify(Array.from(set)))
  } catch (error) {
    handleWarning('Impossibile salvare lo stato dei nodi espansi', { logToConsole: true, showNotification: false })
  }
}

const isNodeWritable = (node) => {
  const access = (node?.access || '').toLowerCase()
  return access.includes('write')
}

// --- Refactored Context Menu Logic ---
// This logic now only exists in the root instance of the component.
const contextMenuState = reactive({
  visible: false,
  x: 0,
  y: 0,
  items: [],
  node: null
})

const addBookmarkModalState = reactive({
  visible: false,
  targetNode: null,
  selectedFolderKey: BOOKMARK_ROOT_KEY
})

const folderModalState = reactive({
  visible: false,
  mode: 'create',
  parentKey: BOOKMARK_ROOT_KEY,
  folderKey: null,
  name: ''
})

// Drag & Drop state - shared across all recursive instances via provide/inject
const injectedDraggedNodeKey = inject('draggedNodeKey', null)
const injectedDropTargetKey = inject('dropTargetKey', null)
const localDraggedNodeKey = ref(null)
const localDropTargetKey = ref(null)
const draggedNodeKey = injectedDraggedNodeKey || localDraggedNodeKey
const dropTargetKey = injectedDropTargetKey || localDropTargetKey

const getOriginalType = (type) => {
  if (!type) return 'scalar'
  if (type.startsWith('bookmark-')) {
    return type.substring('bookmark-'.length)
  }
  return type === 'bookmark' ? 'scalar' : type
}

const isBookmarkFolderType = (type) => type === 'bookmark-folder'

const isBookmarkEntryType = (type) => {
  if (!type) return false
  if (type === 'bookmark-root' || type === 'bookmark-folder') return false
  return type === 'bookmark' || type.startsWith('bookmark-')
}

const isBookmarkNode = (type) => isBookmarkEntryType(type)

const isBookmarkFolderNode = (node) => isBookmarkFolderType(node?.type)

const isBookmarkEntryNode = (node) => isBookmarkEntryType(node?.type)

const isBookmarkParentKey = (value) => {
  if (!value) return false
  if (value === BOOKMARK_ROOT_KEY) return true
  return typeof value === 'string' && value.startsWith(BOOKMARK_FOLDER_PREFIX)
}

const buildMenuItems = (node) => {
  if (!node) return []
  const inferredType = (node.type || (Array.isArray(node.children) && node.children.length ? 'node' : 'scalar')).toLowerCase()

  if (inferredType === 'bookmark-root') {
    return [{ key: 'new-folder', type: 'new-folder', label: 'New folder', icon: 'create_new_folder' }]
  }

  if (isBookmarkFolderType(inferredType)) {
    return [
      { key: 'new-folder', type: 'new-folder', label: 'New subfolder', icon: 'create_new_folder' },
      { key: 'rename-folder', type: 'rename-folder', label: 'Rename folder', icon: 'drive_file_rename_outline' },
      { key: 'delete-folder', type: 'delete-folder', label: 'Delete folder', icon: 'delete' }
    ]
  }

  const originalType = getOriginalType(inferredType)
  const syntax = String(node.syntax || '').toUpperCase()
  const hasChildren = Array.isArray(node.children) && node.children.length > 0

  const operations = []
  switch (originalType) {
    case 'scalar':
      operations.push('get', 'getnext')
      break
    case 'column':
      operations.push('get', 'getnext', 'walk')
      break
    case 'table':
      operations.push('walk', 'getbulk')
      break
    case 'node':
      operations.push('walk')
      break
    default:
      if (hasChildren) {
        operations.push('walk')
      } else {
        operations.push('get')
      }
      break
  }

  if (isNodeWritable(node)) {
    operations.push('set')
  }

  let items = operations
    .filter(op => Boolean(OPERATION_CONFIG[op]))
    .map(op => ({ key: `op-${op}`, type: 'operation', value: op, label: OPERATION_CONFIG[op].label, icon: OPERATION_CONFIG[op].icon }))

  const openItems = []
  if (originalType === 'table') {
    openItems.push({ key: 'open-table', type: 'open-table', label: 'Open as table', icon: 'table_chart' })
  }

  const hasNumericSyntax = NUMERIC_SYNTAX_HINTS.some(hint => syntax.includes(hint))
  if ((originalType === 'scalar' || originalType === 'column') && hasNumericSyntax) {
    openItems.push({ key: 'open-graph', type: 'open-graph', label: 'Open as chart', icon: 'show_chart' })
  }

  if (items.length > 0 && openItems.length > 0) {
    items.push({ key: 'divider-open', type: 'divider' })
  }

  items = [...items, ...openItems]

  if (isBookmarkEntryType(inferredType)) {
    items.push({ key: 'remove-bookmark', type: 'remove-bookmark', label: 'Remove bookmark', icon: 'bookmark_remove' })
  } else {
    items.push({ key: 'add-bookmark', type: 'add-bookmark', label: 'Add to bookmarks', icon: 'bookmark_add' })
  }

  return items
}

const showContextMenu = (event, node) => {
  const entries = buildMenuItems(node)
  if (entries.length === 0) {
    hideContextMenu()
    return
  }

  const estimatedWidth = 240
  const estimatedHeight = entries.filter(item => item.type !== 'divider').length * 40 + 16
  const viewportWidth = window.innerWidth || document.documentElement.clientWidth
  const viewportHeight = window.innerHeight || document.documentElement.clientHeight

  let posX = event.clientX ?? 0
  let posY = event.clientY ?? 0

  if (Number.isFinite(posX) && posX + estimatedWidth > viewportWidth) { posX = Math.max(8, viewportWidth - estimatedWidth) }
  if (Number.isFinite(posY) && posY + estimatedHeight > viewportHeight) { posY = Math.max(8, viewportHeight - estimatedHeight) }

  contextMenuState.items = entries
  contextMenuState.node = node
  contextMenuState.x = Number.isFinite(posX) ? posX : 0
  contextMenuState.y = Number.isFinite(posY) ? posY : 0
  contextMenuState.visible = true
}

const hideContextMenu = () => {
  contextMenuState.visible = false
  contextMenuState.node = null
  contextMenuState.items = []
}
// --- End of Refactored Context Menu Logic ---

const props = defineProps({
  node: Object,
  level: { type: Number, default: 0 },
  initialTree: { type: Array, default: null },
  selectedOid: { type: String, default: '' },
  isResizing: Boolean,
  reloadKey: { type: Number, default: 0 }
})

const emit = defineEmits(['oid-select', 'open-table', 'resize-start', 'resize-end', 'request-operation', 'open-graph', 'show-context-menu', 'bookmark-click'])

const mibTree = ref([])
const isRoot = computed(() => props.level === 0)

const searchQuery = ref('')
const loading = ref(true)
const loadError = ref(null)
const showErrorBanner = computed(() => isRoot.value && Boolean(loadError.value))
const { handleError, handleWarning } = useErrorHandler()

const loadMIBTree = async () => {
  loading.value = true
  loadError.value = null
  try {
    const tree = await GetMIBTree()
    mibTree.value = tree || []
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    loadError.value = message
    if (isRoot.value) {
      handleError(error, 'Errore durante il caricamento dell\'albero MIB')
    }
  } finally {
    loading.value = false
  }
}

const resolvedTree = computed(() => props.initialTree || mibTree.value)

const localBookmarkRootNode = computed(() => {
  if (!isRoot.value) return null
  const tree = resolvedTree.value || []
  return tree.find((node) => node.oid === BOOKMARK_ROOT_KEY) || null
})

const injectedBookmarkRootNode = inject('bookmarkRootNode', null)
const bookmarkRootNode = computed(() => injectedBookmarkRootNode?.value || localBookmarkRootNode.value)

// Shared reload function
const injectedReloadTree = inject('reloadMIBTree', null)
const reloadMIBTree = injectedReloadTree || loadMIBTree

// Use provide/inject to share expandedNodes across recursive instances
const injectedExpandedNodes = inject('expandedNodes', null)
const localExpandedNodes = ref(createDefaultExpandedNodes())
const expandedNodes = injectedExpandedNodes || localExpandedNodes

// If this is root, provide the shared refs to all children
if (isRoot.value) {
  provide('expandedNodes', expandedNodes)
  provide('draggedNodeKey', draggedNodeKey)
  provide('dropTargetKey', dropTargetKey)
  provide('bookmarkRootNode', localBookmarkRootNode)
  provide('reloadMIBTree', loadMIBTree)

  watch(
    expandedNodes,
    (current) => {
      persistExpandedNodesToStorage(current)
    },
    { deep: false }
  )
}

const folderOptions = computed(() => {
  const options = [{ key: BOOKMARK_ROOT_KEY, label: 'Bookmarks' }]
  const root = bookmarkRootNode.value
  if (!root) return options

  const visit = (node, ancestors = []) => {
    if (!node?.children) return
    node.children
      .filter((child) => isBookmarkFolderNode(child))
      .forEach((child) => {
        const path = [...ancestors, child.name || 'Untitled'].join(' / ')
        options.push({ key: child.oid, label: path })
        visit(child, [...ancestors, child.name || 'Untitled'])
      })
  }

  visit(root, [])
  return options
})

const findBookmarkNode = (oid, current = bookmarkRootNode.value) => {
  if (!current) return null
  if (current.oid === oid) return current
  const children = current.children || []
  for (const child of children) {
    if (child.oid === oid) {
      return child
    }
    if (isBookmarkFolderNode(child)) {
      const found = findBookmarkNode(oid, child)
      if (found) return found
    }
  }
  return null
}

onMounted(() => {
  if (!isRoot.value) return

  const storedExpanded = loadExpandedNodesFromStorage()
  if (storedExpanded && storedExpanded.length > 0) {
    expandedNodes.value = mergeWithDefaults(storedExpanded)
  }

  window.addEventListener('keydown', handleWindowKeydown)
  window.addEventListener('scroll', handleTreeScroll, true)
  loadMIBTree()
})

onBeforeUnmount(() => {
  if (!isRoot.value) return
  window.removeEventListener('keydown', handleWindowKeydown)
  window.removeEventListener('scroll', handleTreeScroll, true)
})

watch(() => props.reloadKey, (newVal, oldVal) => {
  if (isRoot.value && newVal !== oldVal) {
    hideContextMenu()
    loadMIBTree()
  }
})

const filteredTree = computed(() => {
  const tree = props.initialTree || mibTree.value
  if (!searchQuery.value) return collapseTree(tree)
  const query = searchQuery.value.toLowerCase()
  const filterNode = (node) => {
    if (node.oid === BOOKMARK_ROOT_KEY) {
        return node;
    }
    const matches = node.name.toLowerCase().includes(query) || node.oid.includes(query)
    if (node.children) {
      const filteredChildren = node.children.map(filterNode).filter(child => child !== null)
      if (filteredChildren.length > 0 || matches) {
        return { ...node, children: filteredChildren }
      }
    }
    return matches ? node : null
  }
  const filtered = tree.map(filterNode).filter(node => node !== null)
  return collapseTree(filtered)
})

const collapseTree = (nodes) => {
  if (!nodes) return []
  return nodes.map(collapseNode).filter(node => node !== null)
}

const collapseNode = (node) => {
  if (!node) return null
  const collapsedChildren = (node.children || []).map(collapseNode).filter(child => child !== null)
  const clone = { ...node, children: collapsedChildren, displayName: node.displayName || node.name }
  if (collapsedChildren.length === 1 && !shouldPreventCollapse(clone) && !shouldPreventCollapse(collapsedChildren[0])) {
    const child = collapsedChildren[0]
    const displayName = `${clone.displayName}.${child.displayName || child.name}`
    return { ...child, displayName }
  }
  return clone
}

const shouldPreventCollapse = (node) => {
  if (!node?.type) return false
  if (node.type === 'bookmark-root' || node.type === 'bookmark-folder') return true
  const originalType = getOriginalType(node.type)
  return originalType === 'table'
}

const updateExpandedNodes = (mutator) => {
  const next = new Set(expandedNodes.value)
  mutator(next)
  expandedNodes.value = next
}

const ensureNodeExpanded = (key) => {
  if (!key) return
  updateExpandedNodes((set) => set.add(key))
}

const removeExpandedNode = (key) => {
  if (!key) return
  updateExpandedNodes((set) => set.delete(key))
}

const toggleNode = (oid) => {
  updateExpandedNodes((set) => {
    if (set.has(oid)) {
      set.delete(oid)
    } else {
      set.add(oid)
    }
  })
}

const selectNode = (node) => {
  emit('oid-select', node.oid)
}

const openTable = (node) => {
  emit('open-table', node)
}

const getNodeIcon = (node) => {
    const type = node.type || ''
    const originalType = getOriginalType(type)

    if (type === 'bookmark-root') return 'folder_special'
    if (type === 'bookmark-folder') return expandedNodes.value?.has(node.oid) ? 'folder_open' : 'folder'
    if (isBookmarkNode(type)) return 'bookmark'
    if (originalType === 'table') return 'table_chart'
    if (isNodeWritable(node) && (originalType === 'scalar' || originalType === 'column')) return 'edit'
    if (originalType === 'scalar') return 'data_object'
    if (originalType === 'column') return 'view_column'
    if (node.children && node.children.length > 0) return 'folder'
    return 'description'
}

const getNodeClass = (node) => {
  const classes = ['tree-node']
  if (node.type) classes.push(`node-${node.type}`)
  if (isDraggableNode(node)) classes.push('draggable')
  // Evidenzia solo se l'OID corrisponde E non è una selezione cross-type
  // (bookmark selezionato ma nodo è originale, o viceversa)
  if (props.selectedOid === node.oid) {
    const nodeIsBookmark = isBookmarkParentKey(node.parentOid)
    // Non evidenziare mai i bookmark quando si seleziona qualcosa
    // perché il click su bookmark seleziona sempre l'originale
    if (!nodeIsBookmark) {
      classes.push('selected')
    }
  }
  if (draggedNodeKey.value === node.oid) {
    classes.push('dragging')
  }
  // Show drop-target indicator only on the node currently under the mouse
  if (dropTargetKey.value === node.oid) {
    classes.push('drop-target')
  }
  return classes.join(' ')
}

const expandPathToNode = (targetOid, tree) => {
  // Trova il nodo nell'albero MIB (non nei bookmark) e ritorna il percorso
  const findPath = (nodes, currentPath = [], depth = 0) => {
    for (const node of nodes) {
      if (node.oid === BOOKMARK_ROOT_KEY) continue // Salta il ramo bookmarks

      const newPath = [...currentPath, node.oid]

      if (node.oid === targetOid) {
        // Trovato! Ritorna il percorso completo
        return newPath
      }

      if (node.children && node.children.length > 0) {
        const foundPath = findPath(node.children, newPath, depth + 1)
        if (foundPath) {
          return foundPath
        }
      }
    }
    return null
  }

  const pathToTarget = findPath(tree)

  if (pathToTarget) {
    // Espandi tutti i nodi antenati (escluso il target stesso)
    const ancestorsToExpand = pathToTarget.slice(0, -1)

    // IMPORTANTE: per forzare la reattività di Vue con i Set,
    // dobbiamo creare un nuovo Set invece di modificare quello esistente
    const newExpandedNodes = new Set(expandedNodes.value)

    ancestorsToExpand.forEach(oid => {
      newExpandedNodes.add(oid)
    })

    // Sostituisci il Set per triggerare la reattività
    expandedNodes.value = newExpandedNodes

    return true
  }

  return false
}

const scrollToNodeInTree = async () => {
  // Aspetta che Vue aggiorni il DOM dopo l'espansione
  await nextTick()

  // Aspetta più tempo per assicurarci che il DOM sia completamente renderizzato
  await new Promise(resolve => setTimeout(resolve, 150))

  // Aspettiamo un altro ciclo per la classe selected
  await nextTick()
  await new Promise(resolve => setTimeout(resolve, 100))

  // Cerchiamo il nodo selected nell'albero MIB (non nei bookmarks)
  const selectedNodes = document.querySelectorAll('.tree-node.selected')
  let targetNode = null

  for (const nodeElement of selectedNodes) {
    const parentOidAttr = nodeElement.getAttribute('data-parent-oid')
    const isBookmarkNode = isBookmarkParentKey(parentOidAttr)

    // Scrolla solo al nodo originale, non al bookmark
    if (!isBookmarkNode) {
      targetNode = nodeElement
      break
    }
  }

  if (targetNode) {
    const treeContainer = targetNode.closest('.tree-container')
    if (treeContainer) {
      const nodeTop = targetNode.offsetTop
      const containerHeight = treeContainer.clientHeight
      const scrollPosition = nodeTop - (containerHeight / 2) + (targetNode.offsetHeight / 2)
      treeContainer.scrollTo({ top: scrollPosition, behavior: 'smooth' })
    }
  }
}

const handleBookmarkClick = async (node) => {
  // Questa funzione viene chiamata quando un bookmark viene cliccato
  // Può essere chiamata dal componente corrente o propagata da un figlio
  if (isRoot.value) {
    // Se sono root, gestisco l'espansione
    const tree = props.initialTree || mibTree.value

    const expanded = expandPathToNode(node.oid, tree)

    selectNode(node)
    if (expanded) {
      await scrollToNodeInTree()
    }
  } else {
    // Se non sono root, propago l'evento al parent
    emit('bookmark-click', node)
  }
}

const handleClick = async (node) => {
  hideContextMenu()
  if (isBookmarkFolderNode(node)) {
    toggleNode(node.oid)
    return
  }
  if (isBookmarkEntryNode(node)) {
    await handleBookmarkClick(node)
    return
  }
  selectNode(node)
}

const handleDoubleClick = (node) => {
  hideContextMenu()
  const originalType = getOriginalType(node.type)
  if (originalType === 'table') {
    openTable(node)
  } else if (node.children && node.children.length > 0) {
    toggleNode(node.oid)
  }
}

const handleContextMenuItemSelect = async (item) => {
  if (!contextMenuState.node) {
    hideContextMenu()
    return
  }
  const node = contextMenuState.node
  switch (item.type) {
    case 'operation': 
        emit('request-operation', { node, operation: item.value }); 
        break
    case 'open-table': 
        openTable(node); 
        break
    case 'open-graph': 
        emit('open-graph', node); 
        break
    case 'add-bookmark':
        openAddBookmarkModal(node)
        break
    case 'remove-bookmark':
        await RemoveBookmark(node.oid)
        await reloadMIBTree()
        break
    case 'new-folder':
        openCreateFolderModal(node)
        break
    case 'rename-folder':
        openRenameFolderModal(node)
        break
    case 'delete-folder':
        await handleDeleteFolder(node)
        break
  }
  hideContextMenu()
}

const handleContextMenu = (event, node) => {
  // This is now the final handler, only in the root component.
  // For child components, this function will emit an event upwards.
  if (isRoot.value) {
    event.preventDefault()
    selectNode(node)
    showContextMenu(event, node)
  } else {
    emit('show-context-menu', event, node)
  }
}

const openAddBookmarkModal = (node) => {
  addBookmarkModalState.targetNode = node
  addBookmarkModalState.selectedFolderKey = BOOKMARK_ROOT_KEY
  addBookmarkModalState.visible = true
}

const openCreateFolderModal = (node) => {
  const parentKey = node.type === 'bookmark-root' ? BOOKMARK_ROOT_KEY : node.oid
  folderModalState.mode = 'create'
  folderModalState.parentKey = parentKey
  folderModalState.folderKey = null
  folderModalState.name = ''
  folderModalState.visible = true
}

const openRenameFolderModal = (node) => {
  if (!isBookmarkFolderNode(node)) return
  folderModalState.mode = 'rename'
  folderModalState.folderKey = node.oid
  folderModalState.parentKey = node.parentOid || BOOKMARK_ROOT_KEY
  folderModalState.name = node.name || ''
  folderModalState.visible = true
}

const handleDeleteFolder = async (node) => {
  if (!isBookmarkFolderNode(node)) return
  const confirmMessage = node.children && node.children.length > 0
    ? `Delete folder "${node.name}" and all its contents?`
    : `Delete folder "${node.name}"?`
  const confirmed = window.confirm(confirmMessage)
  if (!confirmed) return
  try {
    await DeleteBookmarkFolder(node.oid)
    removeExpandedNode(node.oid)
    await reloadMIBTree()
  } catch (error) {
    handleError(error, 'Impossibile eliminare la cartella bookmark')
  }
}

const handleFolderModalConfirm = async (folderName) => {
  const name = String(folderName || '').trim()
  if (!name) return

  if (folderModalState.mode === 'create') {
    try {
      const dto = await CreateBookmarkFolder(name, folderModalState.parentKey)
      folderModalState.visible = false
      folderModalState.name = ''
      folderModalState.folderKey = null
      if (dto) {
        await reloadMIBTree()
        ensureNodeExpanded(dto.parentKey || BOOKMARK_ROOT_KEY)
        ensureNodeExpanded(dto.key)
        if (addBookmarkModalState.visible) {
          addBookmarkModalState.selectedFolderKey = dto.key
        }
      }
    } catch (error) {
      handleError(error, 'Impossibile creare la cartella bookmark')
    }
  } else if (folderModalState.mode === 'rename') {
    const folderKey = folderModalState.folderKey
    if (!folderKey) return
    try {
      await RenameBookmarkFolder(folderKey, name)
      folderModalState.visible = false
      folderModalState.name = ''
      folderModalState.folderKey = null
      await reloadMIBTree()
      ensureNodeExpanded(folderKey)
    } catch (error) {
      handleError(error, 'Impossibile rinominare la cartella bookmark')
    }
  }
}

const handleAddBookmarkConfirm = async (folderKey) => {
  if (!addBookmarkModalState.targetNode) return
  const targetKey = folderKey || BOOKMARK_ROOT_KEY
  try {
    await AddBookmark(addBookmarkModalState.targetNode.oid, targetKey)
    addBookmarkModalState.visible = false
    await reloadMIBTree()
    ensureNodeExpanded(targetKey)
  } catch (error) {
    handleError(error, 'Impossibile aggiungere il bookmark')
  } finally {
    addBookmarkModalState.targetNode = null
  }
}

const handleAddBookmarkCreateFolder = (parentKey) => {
  folderModalState.mode = 'create'
  folderModalState.parentKey = parentKey || BOOKMARK_ROOT_KEY
  folderModalState.folderKey = null
  folderModalState.name = ''
  folderModalState.visible = true
}

const isDraggableNode = (node) => {
  if (!node) return false
  if (node.oid === BOOKMARK_ROOT_KEY) return false
  return isBookmarkEntryNode(node) || isBookmarkFolderNode(node)
}

const isDescendantFolder = (candidateKey, folderKey) => {
  if (!candidateKey || !folderKey) return false
  if (candidateKey === folderKey) return true
  const start = findBookmarkNode(folderKey)
  if (!start) return false
  const stack = [...(start.children || [])]
  while (stack.length) {
    const current = stack.shift()
    if (!current) continue
    if (current.oid === candidateKey) {
      return true
    }
    if (isBookmarkFolderNode(current)) {
      stack.push(...(current.children || []))
    }
  }
  return false
}

const canDropOnNode = (target, dragged) => {
  if (!target || !dragged) return false
  if (dragged.oid === BOOKMARK_ROOT_KEY) return false
  if (isBookmarkEntryNode(dragged)) {
    return target.oid === BOOKMARK_ROOT_KEY || isBookmarkFolderNode(target)
  }
  if (isBookmarkFolderNode(dragged)) {
    if (!(target.oid === BOOKMARK_ROOT_KEY || isBookmarkFolderNode(target))) return false
    if (target.oid === dragged.oid) return false
    if (isBookmarkFolderNode(target) && isDescendantFolder(target.oid, dragged.oid)) return false
    return true
  }
  return false
}

const handleDragStart = (event, node) => {
  if (!isDraggableNode(node)) {
    event.preventDefault()
    return
  }
  draggedNodeKey.value = node.oid
  dropTargetKey.value = null
  if (event.dataTransfer) {
    event.dataTransfer.setData('text/plain', node.oid)
    event.dataTransfer.effectAllowed = 'move'
  }
}

const handleDragOver = (event, node) => {
  if (!draggedNodeKey.value) return
  const dragged = findBookmarkNode(draggedNodeKey.value)
  if (!dragged) return
  if (canDropOnNode(node, dragged)) {
    event.preventDefault()
    dropTargetKey.value = node.oid
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = 'move'
    }
  } else if (dropTargetKey.value === node.oid) {
    dropTargetKey.value = null
  }
}

const handleDragEnter = (event, node) => {
  if (!draggedNodeKey.value) return
  const dragged = findBookmarkNode(draggedNodeKey.value)
  if (dragged && canDropOnNode(node, dragged)) {
    event.preventDefault()
    dropTargetKey.value = node.oid
    if (isBookmarkFolderNode(node)) {
      ensureNodeExpanded(node.oid)
    }
  }
}

const handleDragLeave = (event, node) => {
  const current = event?.currentTarget
  const related = event?.relatedTarget
  if (current && related && current.contains(related)) {
    return
  }
  if (dropTargetKey.value === node.oid) {
    dropTargetKey.value = null
  }
}

const handleDragEnd = () => {
  draggedNodeKey.value = null
  dropTargetKey.value = null
}

const handleDrop = async (event, node) => {
  if (!draggedNodeKey.value) return
  const dragged = findBookmarkNode(draggedNodeKey.value)
  if (!dragged || !canDropOnNode(node, dragged)) {
    draggedNodeKey.value = null
    dropTargetKey.value = null
    return
  }
  event.preventDefault()
  const destinationKey = node.oid === BOOKMARK_ROOT_KEY ? BOOKMARK_ROOT_KEY : node.oid
  try {
    const currentParent = dragged.parentOid || BOOKMARK_ROOT_KEY
    if (currentParent === destinationKey && isBookmarkEntryNode(dragged)) {
      draggedNodeKey.value = null
      dropTargetKey.value = null
      return
    }
    if (currentParent === destinationKey && isBookmarkFolderNode(dragged)) {
      draggedNodeKey.value = null
      dropTargetKey.value = null
      return
    }
    if (isBookmarkEntryNode(dragged)) {
      await MoveBookmark(dragged.oid, destinationKey)
    } else if (isBookmarkFolderNode(dragged)) {
      await MoveBookmarkFolder(dragged.oid, destinationKey)
      ensureNodeExpanded(dragged.oid)
    }
    ensureNodeExpanded(destinationKey)
    await reloadMIBTree()
  } catch (error) {
    handleError(error, 'Impossibile spostare l\'elemento bookmark')
  } finally {
    draggedNodeKey.value = null
    dropTargetKey.value = null
  }
}

const handleWindowKeydown = (event) => {
  if (event.key === 'Escape' && contextMenuState.visible) {
    hideContextMenu()
  }
}

const handleTreeScroll = () => {
  if (contextMenuState.visible) {
    hideContextMenu()
  }
}

const findNodeByOidInMibTree = (oid, tree) => {
  // Cerca il nodo solo nell'albero MIB, escludendo i bookmark
  for (const node of tree) {
    if (node.oid === BOOKMARK_ROOT_KEY) continue // Salta il ramo bookmarks
    if (node.oid === oid) return node
    if (node.children) {
      const found = findNodeByOidInMibTree(oid, node.children)
      if (found) return found
    }
  }
  return null
}

const selectedNodeDetails = computed(() => {
  if (!props.selectedOid || !isRoot.value) return null
  // Cerca sempre il nodo originale nell'albero MIB, non nei bookmark
  return findNodeByOidInMibTree(props.selectedOid, props.initialTree || mibTree.value)
})

</script>

<template>
  <aside v-if="isRoot" class="mib-sidebar">
    <DeLoreanHeader v-model:search="searchQuery" />

    <div
      v-if="showErrorBanner"
      class="tree-error-banner"
      data-test="mib-tree-error"
    >
      <span class="material-symbols-outlined tree-error-banner__icon">error</span>
      <div class="tree-error-banner__content">
        <p class="tree-error-banner__title">Impossibile caricare l'albero MIB</p>
        <p class="tree-error-banner__details">{{ loadError }}</p>
      </div>
      <md-filled-button
        data-test="mib-tree-retry"
        :disabled="loading"
        @click="loadMIBTree"
      >
        <span slot="icon" class="material-symbols-outlined">refresh</span>
        Riprova
      </md-filled-button>
    </div>

    <Splitpanes
      horizontal
      class="sidebar-split"
      @resize="emit('resize-start')"
      @resized="emit('resize-end')"
    >
      <Pane>
        <div class="tree-container" @scroll.passive="handleTreeScroll">
          <MibTreeSidebar
            v-for="rootNode in filteredTree"
            :key="rootNode.oid"
            :node="rootNode"
            :level="1"
            :selected-oid="selectedOid"
            @oid-select="(oid) => emit('oid-select', oid)"
            @open-table="(node) => emit('open-table', node)"
            @show-context-menu="handleContextMenu"
            @bookmark-click="handleBookmarkClick"
          />
        </div>
      </Pane>
      <Pane v-if="selectedNodeDetails" size="30" min-size="20">
        <OidDetailsPanel :node="selectedNodeDetails" />
      </Pane>
    </Splitpanes>
    <div v-if="!selectedNodeDetails" class="details-panel-placeholder">
      <OidDetailsPanel :node="null" />
    </div>
    <ContextMenuOverlay
      :state="contextMenuState"
      @close="hideContextMenu"
      @select="handleContextMenuItemSelect"
    />
    <AddBookmarkModal
      :show="addBookmarkModalState.visible"
      :node-name="addBookmarkModalState.targetNode?.name || ''"
      :node-oid="addBookmarkModalState.targetNode?.oid || ''"
      :folders="folderOptions"
      :selected-key="addBookmarkModalState.selectedFolderKey"
      @update:show="(value) => (addBookmarkModalState.visible = value)"
      @confirm="handleAddBookmarkConfirm"
      @create-folder="handleAddBookmarkCreateFolder"
    />
    <BookmarkFolderModal
      :show="folderModalState.visible"
      :title="folderModalState.mode === 'create' ? 'New folder' : 'Rename folder'"
      :confirm-label="folderModalState.mode === 'create' ? 'Create' : 'Rename'"
      :default-name="folderModalState.name"
      @update:show="(value) => (folderModalState.visible = value)"
      @confirm="handleFolderModalConfirm"
    />
  </aside>
  <div v-else>
    <div
      :class="getNodeClass(node)"
      :data-parent-oid="node.parentOid"
      :style="{ paddingLeft: `${level * 16 + 8}px` }"
      @click.left="handleClick(node)"
      @dblclick="handleDoubleClick(node)"
      @contextmenu.stop.prevent="handleContextMenu($event, node)"
      :draggable="isDraggableNode(node) ? 'true' : 'false'"
      @dragstart.stop="handleDragStart($event, node)"
      @dragover.stop.prevent="handleDragOver($event, node)"
      @dragenter.stop.prevent="handleDragEnter($event, node)"
      @dragleave.stop="handleDragLeave($event, node)"
      @drop.stop.prevent="handleDrop($event, node)"
      @dragend="handleDragEnd"
    >
      <md-icon-button v-if="node.children && node.children.length > 0" @click.stop="toggleNode(node.oid)" class="expand-btn">
        <span class="material-symbols-outlined expand-icon">
          {{ expandedNodes.has(node.oid) ? 'expand_more' : 'chevron_right' }}
        </span>
      </md-icon-button>
      <span v-else class="expand-spacer"></span>

      <span class="material-symbols-outlined node-icon">
        {{ getNodeIcon(node) }}
      </span>

      <span class="node-name">{{ node.displayName || node.name }}</span>
    </div>
    <template v-if="node && node.children && expandedNodes.has(node.oid)">
      <MibTreeSidebar
        v-for="child in node.children"
        :key="child.oid"
        :node="child"
        :selected-oid="selectedOid"
        @oid-select="(oid) => emit('oid-select', oid)"
        @open-table="(node) => emit('open-table', node)"
        @show-context-menu="(event, node) => emit('show-context-menu', event, node)"
        @bookmark-click="handleBookmarkClick"
        :level="props.level + 1"
      />
    </template>
  </div>
</template>

<style scoped>
.mib-sidebar {
  display: flex;
  flex-direction: column;
  background-color: var(--md-sys-color-surface-container);
  border-right: 1px solid var(--md-sys-color-outline-variant);
  height: 100%;
  overflow: hidden;
}

.sidebar-header {
  padding: var(--spacing-md);
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
  background-color: var(--md-sys-color-surface-container);
}

.sidebar-title {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  font-size: var(--md-sys-typescale-title-medium-size);
  font-weight: 500;
  color: var(--md-sys-color-on-surface);
  margin: 0 0 var(--spacing-md) 0;
  cursor: pointer;
  position: relative;
  overflow: hidden;
}

.sidebar-title .material-symbols-outlined {
  color: var(--md-sys-color-primary);
}

.search-input {
  width: 100%;
  --md-outlined-text-field-container-shape: var(--border-radius-full);
  --md-outlined-text-field-container-height: 48px;
  --md-outlined-text-field-input-text-prefix-padding-inline-start: 0;
}

.tree-error-banner {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  margin: 8px 16px 0;
  padding: 16px;
  border-radius: 16px;
  background-color: var(--md-sys-color-error-container);
  color: var(--md-sys-color-on-error-container);
}

.tree-error-banner__icon {
  font-size: 24px;
  line-height: 1;
  margin-top: 2px;
}

.tree-error-banner__content {
  flex: 1;
}

.tree-error-banner__title {
  margin: 0;
  font-weight: 600;
}

.tree-error-banner__details {
  margin: 4px 0 0;
  font-size: 0.9rem;
  opacity: 0.85;
  word-break: break-word;
}

.tree-container {
  overflow-y: auto;
  padding: var(--spacing-xs) 0;
  height: 100%;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  cursor: pointer;
  user-select: none;
  transition: background-color 0.15s ease;
  color: var(--md-sys-color-on-surface);
}

.tree-node.draggable {
  cursor: grab;
}

.tree-node.draggable > * {
  pointer-events: none;
}

.tree-node.draggable .expand-btn {
  pointer-events: auto;
}

.tree-node.dragging {
  opacity: 0.55;
  cursor: grabbing;
}

.tree-node.drop-target {
  outline: 2px dashed var(--md-sys-color-primary);
  outline-offset: -2px;
  background-color: var(--md-sys-color-primary-container);
  color: var(--md-sys-color-on-primary-container);
}

.tree-node:hover {
  background-color: var(--md-sys-color-surface-container-highest);
}

.tree-node.selected {
  background-color: var(--md-sys-color-primary-container);
  color: var(--md-sys-color-on-primary-container);
}

.expand-btn {
  --md-icon-button-icon-size: 20px;
}

.expand-icon {
  font-size: 20px;
  color: var(--md-sys-color-on-surface-variant);
}

.expand-spacer {
  width: 40px;
  height: 40px;
}

.node-icon {
  font-size: 20px;
  color: var(--md-sys-color-on-surface-variant);
}

.tree-node.node-table .node-icon {
  color: var(--md-sys-color-tertiary);
}

.tree-node.node-scalar .node-icon {
  color: var(--md-sys-color-secondary);
}

.tree-node.node-bookmark-root .node-icon {
  color: var(--md-sys-color-primary);
}

.tree-node[class*="node-bookmark"] .node-icon {
  color: var(--md-sys-color-primary);
}

.node-name {
  flex: 1;
  font-size: 14px;
  font-weight: 500;
}


/* Scrollbar styling */
.tree-container::-webkit-scrollbar {
  width: 8px;
}

.tree-container::-webkit-scrollbar-track {
  background: var(--md-sys-color-surface-container);
}

.tree-container::-webkit-scrollbar-thumb {
  background: var(--md-sys-color-outline-variant);
  border-radius: 4px;
}

.tree-container::-webkit-scrollbar-thumb:hover {
  background: var(--md-sys-color-outline);
}

.sidebar-split {
  display: flex;
  flex: 1;
}

.sidebar-split :deep(.splitpanes__pane) {
  transition: width 0.2s ease-out, height 0.2s ease-out;
}

.details-panel-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: var(--spacing-md);
  min-height: 120px;
  color: var(--md-sys-color-on-surface-variant);
}

.details-panel-placeholder .material-symbols-outlined {
  font-size: 48px;
}

.details-panel-placeholder p {
  margin-top: var(--spacing-sm);
  font-size: var(--md-sys-typescale-body-medium-size);
}

.delorean-icon {
  width: 38px;
  height: 38px;
  object-fit: contain;
  position: absolute;
  left: 8px;
  top: 50%;
  transform: translateY(-50%);
  pointer-events: none;
  z-index: 5;
  will-change: transform;
}

.title-text {
  display: inline-flex;
  align-items: center;
  margin-left: 56px;
  font-weight: 600;
  letter-spacing: 0.01em;
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.fire-trail {
  position: absolute;
  top: calc(50% + 6px);
  left: 56px;
  height: 30px;
  width: 0;
  opacity: 0;
  transform: translateY(-50%);
  pointer-events: none;
  z-index: 1;
  --segment-spacing: 54px;
  --segment-width: 72px;
  --active-width: 0px;
  filter: none;
  overflow: visible;
  will-change: width, opacity;
}

.fire-trail .trail-lane {
  position: absolute;
  top: 0;
  bottom: 0;
  left: 0;
  width: var(--active-width);
  pointer-events: none;
  mix-blend-mode: screen;
  overflow: hidden;
}

.fire-trail .flame-segment {
  position: absolute;
  bottom: 0;
  left: calc(var(--seg-index, 0) * var(--segment-spacing));
  width: var(--segment-width);
  height: 100%;
  border-radius: 999px;
  background:
    linear-gradient(
      180deg,
      hsl(calc(34 + var(--hue-shift, 0)) 87% 65%) 0%,
      hsl(calc(25 + var(--hue-shift, 0)) 92% 54%) 48%,
      hsl(calc(22 + var(--hue-shift, 0)) 85% 32%) 100%
    );
  clip-path: polygon(0% 100%, 0% 58%, 10% 34%, 22% 52%, 36% 24%, 48% 54%, 62% 20%, 76% 50%, 90% 32%, 100% 58%, 100% 100%);
  box-shadow:
    0 0 16px rgba(255, 132, 0, 0.32),
    0 0 32px rgba(255, 70, 0, 0.18);
  filter: brightness(0.78);
  opacity: 0;
  transform: translateY(calc(var(--seg-jitter, 0) * 1px)) scale(0.88);
  transform-origin: center bottom;
  mix-blend-mode: screen;
  will-change: transform, opacity, filter;
  pointer-events: none;
}

.fire-trail .flame-segment::before {
  content: '';
  position: absolute;
  inset: 18% 22% 26%;
  border-radius: inherit;
  background: radial-gradient(circle at 50% 0%, rgba(255, 255, 255, 0.92) 0%, rgba(255, 194, 90, 0.52) 58%, rgba(255, 64, 0, 0.16) 100%);
  opacity: 0.9;
  mix-blend-mode: screen;
}

.fire-trail .flame-segment::after {
  content: '';
  position: absolute;
  inset: 42% 12% -6%;
  border-radius: inherit;
  background: radial-gradient(ellipse at 50% 0%, rgba(255, 192, 70, 0.4) 0%, rgba(255, 100, 0, 0.2) 55%, rgba(0, 0, 0, 0) 100%);
  opacity: 0.5;
  mix-blend-mode: screen;
}

.fire-trail .flame-segment--lower {
  transform-origin: center bottom;
}

.fire-trail .flame-segment--lower::after {
  inset: 58% 18% -16%;
  background: radial-gradient(ellipse at 50% 0%, rgba(255, 180, 60, 0.32) 0%, rgba(255, 110, 0, 0.18) 60%, rgba(0, 0, 0, 0) 100%);
}

.trail-blocker {
  position: absolute;
  top: -12px;
  bottom: -10px;
  left: 0;
  background: var(--md-sys-color-surface-container);
  box-shadow: none;
  pointer-events: none;
  mix-blend-mode: normal;
  z-index: 4;
}

.fire-trail .ember-line {
  position: absolute;
  left: 0;
  right: 0;
  bottom: -8px;
  height: 6px;
  border-radius: 999px;
  background: linear-gradient(90deg, rgba(255, 128, 0, 0.9) 0%, rgba(255, 212, 150, 0.72) 45%, rgba(255, 110, 0, 0.85) 100%);
  box-shadow: 0 0 12px rgba(255, 118, 0, 0.55);
  opacity: 0;
  transform: scaleX(0.3);
  transform-origin: left center;
  filter: blur(0.5px);
  pointer-events: none;
}

.fire-trail .spark {
  position: absolute;
  bottom: 18px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(255, 249, 228, 0.95) 0%, rgba(255, 196, 90, 0.75) 55%, rgba(255, 90, 0, 0.2) 100%);
  box-shadow:
    0 0 12px rgba(255, 192, 80, 0.85),
    0 0 22px rgba(255, 110, 0, 0.5);
  opacity: 0;
  transform: translate(-50%, 0) scale(0.6);
  mix-blend-mode: screen;
  pointer-events: none;
  z-index: 2;
}

.fire-trail .spark::after {
  content: '';
  position: absolute;
  inset: -3px -2px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.85) 0%, rgba(255, 255, 255, 0) 70%);
  pointer-events: none;
}

.spark-a { left: 20%; }
.spark-b { left: 45%; }
.spark-c { left: 70%; }
.spark-d { left: 90%; }

@keyframes flameWave {
  0% {
    transform: scaleY(0.92) translateY(2px);
    filter: blur(0.3px) brightness(0.96);
  }
  50% {
    transform: scaleY(1.12) translateY(-3px);
    filter: blur(0.45px) brightness(1.18);
  }
  100% {
    transform: scaleY(0.98) translateY(0);
    filter: blur(0.32px) brightness(1.05);
  }
}

@keyframes corePulse {
  0% {
    transform: scaleY(0.94);
    opacity: 0.82;
  }
  50% {
    transform: scaleY(1.08) translateY(-2px);
    opacity: 1;
  }
  100% {
    transform: scaleY(0.96);
    opacity: 0.85;
  }
}

@keyframes heatHaze {
  0% {
    transform: scaleX(1) translateY(0);
    opacity: 0.8;
  }
  50% {
    transform: scaleX(1.05) translateY(-1px);
    opacity: 0.95;
  }
  100% {
    transform: scaleX(1.02) translateY(1px);
    opacity: 0.82;
  }
}
</style>
