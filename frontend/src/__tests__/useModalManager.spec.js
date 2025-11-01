import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

const addNotification = vi.fn()

vi.mock('../composables/useNotifications', () => ({
  useNotifications: () => ({
    addNotification,
  }),
}))

vi.mock('../../wailsjs/go/app/App', () => ({
  GetMIBNode: vi.fn(),
}))

const getOriginalNodeType = (type) => {
  if (!type) return 'scalar'
  if (type.startsWith('bookmark-')) {
    return type.slice('bookmark-'.length)
  }
  return type === 'bookmark' ? 'scalar' : type
}

const createManager = async () => {
  vi.resetModules()
  const appBridge = await import('../../wailsjs/go/app/App')
  appBridge.GetMIBNode.mockReset()
  const module = await import('../composables/useModalManager')

  const tabsContext = {
    openGraphTab: vi.fn(),
  }

  const snmpContext = {
    handleExecuteSnmpOperation: vi.fn(),
    getOriginalNodeType,
  }

  const manager = module.useModalManager(snmpContext, tabsContext)

  return { manager, appBridge, snmpContext, tabsContext }
}

describe('useModalManager', () => {
  let consoleError

  beforeEach(() => {
    addNotification.mockReset()
    consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
  })

  afterEach(() => {
    consoleError.mockRestore()
    vi.clearAllMocks()
  })

  it('apre il gestore MIB quando richiesto', async () => {
    const { manager } = await createManager()
    expect(manager.showMibManager.value).toBe(false)
    manager.handleLoadMib()
    expect(manager.showMibManager.value).toBe(true)
  })

  it('richiede conferma per i grafici con istanza e delega a openGraphTab', async () => {
    const { manager, tabsContext } = await createManager()
    const node = { oid: '1.2.3', type: 'column' }
    manager.handleOpenGraphRequest(node)

    expect(manager.graphInstanceModal.visible).toBe(true)
    expect(manager.graphInstanceModal.node).toMatchObject(node)
    expect(tabsContext.openGraphTab).not.toHaveBeenCalled()

    manager.handleGraphInstanceConfirm(' 0 ')
    expect(tabsContext.openGraphTab).toHaveBeenCalledWith(node, ' 0 ')
    expect(manager.graphInstanceModal.visible).toBe(false)
  })

  it('apre direttamente il grafico per nodi scalari', async () => {
    const { manager, tabsContext } = await createManager()
    const node = { oid: '1.2.3', type: 'scalar' }
    manager.handleOpenGraphRequest(node)
    expect(tabsContext.openGraphTab).toHaveBeenCalledWith(node)
  })

  it('mostra il modal SET e carica il valore corrente', async () => {
    const { manager, snmpContext } = await createManager()
    snmpContext.handleExecuteSnmpOperation.mockResolvedValue({ result: { value: 'ok' } })

    await manager.openSetModalForOid('1.2.3.0', {
      oid: '1.2.3',
      access: 'read-write',
      name: 'sysName',
    })

    expect(snmpContext.handleExecuteSnmpOperation).toHaveBeenCalledWith({
      operation: 'get',
      oid: '1.2.3.0',
      skipSetModal: true,
      skipInstanceModal: true,
      node: expect.objectContaining({ oid: '1.2.3' }),
      silent: true,
    })
    expect(manager.setOperationModal.visible).toBe(true)
    expect(manager.setOperationModal.targetOid).toBe('1.2.3.0')
    expect(manager.setOperationModal.currentValue).toEqual({ value: 'ok' })
  })

  it('recupera il nodo dal backend quando non è fornito', async () => {
    const { manager, appBridge, snmpContext } = await createManager()
    appBridge.GetMIBNode.mockResolvedValue({
      oid: '1.2.3',
      access: 'read-write',
      name: 'sysContact',
    })
    snmpContext.handleExecuteSnmpOperation.mockResolvedValue({ result: null })

    await manager.openSetModalForOid('1.2.3')

    expect(appBridge.GetMIBNode).toHaveBeenCalledWith('1.2.3')
    expect(manager.setOperationModal.visible).toBe(true)
    expect(manager.setOperationModal.node).toMatchObject({ name: 'sysContact' })
  })

  it('notifica quando il nodo non è scrivibile', async () => {
    const { manager } = await createManager()
    await manager.openSetModalForOid('1.2.3', {
      oid: '1.2.3',
      access: 'read-only',
      name: 'sysDescr',
    })

    expect(addNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        severity: 'warning',
        message: expect.stringContaining('not writable'),
      })
    )
    expect(manager.setOperationModal.visible).toBe(false)
  })

  it('notifica gli errori del backend durante il caricamento del nodo', async () => {
    const { manager, appBridge } = await createManager()
    appBridge.GetMIBNode.mockRejectedValue(new Error('boom'))

    await manager.openSetModalForOid('1.2.3')

    expect(addNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        severity: 'error',
        message: expect.stringContaining('Unable to load metadata'),
      })
    )
  })

  it('invia il payload di conferma SET e azzera lo stato del modal', async () => {
    const { manager, snmpContext } = await createManager()
    manager.setOperationModal.visible = true
    manager.setOperationModal.targetOid = '1.2.3'
    manager.setOperationModal.node = { oid: '1.2.3', name: 'sysContact' }

    await manager.handleSetModalConfirm({ valueType: 'i', value: '42' })

    expect(snmpContext.handleExecuteSnmpOperation).toHaveBeenCalledWith({
      operation: 'set',
      oid: '1.2.3',
      skipSetModal: true,
      setPayload: { valueType: 'i', value: '42' },
      node: { oid: '1.2.3', name: 'sysContact' },
    })
    expect(manager.setOperationModal.visible).toBe(false)
  })

  it('annulla il modal SET ripristinando lo stato', async () => {
    const { manager } = await createManager()
    manager.setOperationModal.visible = true
    manager.setOperationModal.targetOid = '1.2.3'

    manager.handleSetModalCancel()

    expect(manager.setOperationModal.visible).toBe(false)
    expect(manager.setOperationModal.targetOid).toBe('')
  })
})
