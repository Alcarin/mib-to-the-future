import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { ref } from 'vue'

const hostState = ref({
  address: '127.0.0.1',
  port: 161,
  community: 'public',
  writeCommunity: 'private',
  version: 'v2c',
})
const loadSavedHosts = vi.fn()
const selectedOid = ref('')
const selectedOperation = ref('get')
const addNotification = vi.fn()
const logTab = { id: 'log-default', type: 'log', data: [] }

const findTargetLogTab = vi.fn(() => logTab)
const prependLogEntry = vi.fn((tab, entry) => {
  const next = Array.isArray(tab.data) ? tab.data : []
  tab.data = [entry, ...next]
  return entry
})
const replaceLogEntry = vi.fn((tab, target, patch) => {
  const current = Array.isArray(tab.data) ? tab.data : []
  const index = current.findIndex((item) => item === target || item?.id === target?.id)
  const updated = { ...target, ...patch }
  if (index !== -1) {
    const copy = [...current]
    copy.splice(index, 1, updated)
    tab.data = copy
  }
  return updated
})

vi.mock('../composables/useHostManager', () => ({
  useHostManager: () => ({
    host: hostState,
    loadSavedHosts,
  }),
}))

vi.mock('../composables/useOidSelection', () => ({
  useOidSelection: () => ({
    selectedOid,
    selectedOperation,
  }),
}))

vi.mock('../composables/useTabsManager', () => ({
  useTabsManager: () => ({
    findTargetLogTab,
    prependLogEntry,
    replaceLogEntry,
  }),
}))

vi.mock('../composables/useNotifications', () => ({
  useNotifications: () => ({
    addNotification,
  }),
}))

vi.mock('../../wailsjs/go/app/App', () => ({
  SNMPGet: vi.fn(),
  SNMPGetNext: vi.fn(),
  SNMPWalk: vi.fn(),
  SNMPGetBulk: vi.fn(),
  SNMPSet: vi.fn(),
}))

const setup = async () => {
  vi.resetModules()
  const bridge = await import('../../wailsjs/go/app/App')
  bridge.SNMPGet.mockReset()
  bridge.SNMPGetNext.mockReset()
  bridge.SNMPWalk.mockReset()
  bridge.SNMPGetBulk.mockReset()
  bridge.SNMPSet.mockReset()
  const module = await import('../composables/useSnmp')
  return { bridge, useSnmp: module.useSnmp }
}

describe('useSnmp', () => {
  let consoleError
  let consoleWarn

  beforeEach(() => {
    hostState.value = {
      address: '127.0.0.1',
      port: 161,
      community: 'public',
      writeCommunity: 'private',
      version: 'v2c',
    }
    selectedOid.value = ''
    selectedOperation.value = 'get'
    loadSavedHosts.mockReset()
    addNotification.mockReset()
    findTargetLogTab.mockClear()
    prependLogEntry.mockClear()
    replaceLogEntry.mockClear()
    logTab.data = []
    consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    consoleWarn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    global.alert = vi.fn()
  })

  afterEach(() => {
    consoleError.mockRestore()
    consoleWarn.mockRestore()
    vi.clearAllMocks()
  })

  it('richiede l\'apertura del modal SET quando necessario', async () => {
    const { useSnmp } = await setup()
    selectedOid.value = '1.2.3.4'
    const openSetModalForOid = vi.fn().mockResolvedValue()
    const snmp = useSnmp({ openSetModalForOid })

    const result = await snmp.handleExecuteSnmpOperation({ operation: 'set' })

    expect(openSetModalForOid).toHaveBeenCalledWith('1.2.3.4', null)
    expect(result).toEqual({ modalOpened: true })
  })

  it('esegue un SNMP GET aggiornando il log', async () => {
    const { bridge, useSnmp } = await setup()
    selectedOid.value = '1.3.6.1.2'
    const snmp = useSnmp({ openSetModalForOid: vi.fn() })
    bridge.SNMPGet.mockResolvedValue({
      oid: '.1.3.6.1.2.0',
      value: '42',
      syntax: 'Counter32',
      status: 'success',
      responseTime: 12,
      resolvedName: 'sysUpTime',
    })

    const output = await snmp.handleExecuteSnmpOperation()

    expect(bridge.SNMPGet).toHaveBeenCalledWith(
      expect.objectContaining({
        host: '127.0.0.1',
        community: 'public',
        version: 'v2c',
        writeCommunity: 'private',
      }),
      '1.3.6.1.2'
    )
    expect(prependLogEntry).toHaveBeenCalledTimes(1)
    expect(replaceLogEntry).toHaveBeenCalled()
    expect(logTab.data[0]).toMatchObject({
      status: 'success',
      value: '42',
      oid: '1.3.6.1.2.0',
      syntax: 'Counter32',
    })
    expect(output.result).toMatchObject({ value: '42' })
    expect(loadSavedHosts).toHaveBeenCalledTimes(1)
  })

  it('gestisce un SNMP WALK aggiungendo ogni risposta al log', async () => {
    const { bridge, useSnmp } = await setup()
    selectedOid.value = '1.3.6.1.2'
    const snmp = useSnmp({ openSetModalForOid: vi.fn() })
    const baseTime = new Date('2024-04-01T10:00:00Z').toISOString()
    bridge.SNMPWalk.mockResolvedValue([
      { oid: '.1.3.6.1.2.0', value: '10', responseTime: 5, timestamp: baseTime, syntax: 'Integer' },
      { oid: '.1.3.6.1.2.1', value: '11', responseTime: 6, timestamp: baseTime, syntax: 'Integer' },
    ])

    const response = await snmp.handleExecuteSnmpOperation({ operation: 'walk', oid: '1.3.6.1.2' })

    expect(bridge.SNMPWalk).toHaveBeenCalled()
    expect(prependLogEntry).toHaveBeenCalledTimes(2)
    expect(replaceLogEntry).toHaveBeenCalledTimes(1)
    expect(logTab.data).toHaveLength(2)
    expect(response.result).toHaveLength(2)
  })

  it('esegue un SET con payload e notifica il successo', async () => {
    const { bridge, useSnmp } = await setup()
    selectedOid.value = '1.2.3.4'
    selectedOperation.value = 'set'
    const snmp = useSnmp({ openSetModalForOid: vi.fn() })
    bridge.SNMPSet.mockResolvedValue({ status: 'success', oid: '.1.2.3.4', value: '5' })

    const result = await snmp.handleExecuteSnmpOperation({
      operation: 'set',
      skipSetModal: true,
      oid: '1.2.3.4',
      setPayload: { valueType: 'i', value: '5', displayValue: '5' },
    })

    expect(bridge.SNMPSet).toHaveBeenCalledWith(
      expect.objectContaining({ community: 'private' }),
      '1.2.3.4',
      'i',
      '5'
    )
    expect(result.entry.status).toBe('success')
    expect(addNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        severity: 'success',
        message: expect.stringContaining('completed successfully'),
      })
    )
  })

  it('propaga l\'errore e notifica il fallimento di un SET', async () => {
    const { bridge, useSnmp } = await setup()
    selectedOid.value = '1.2.3.4'
    selectedOperation.value = 'set'
    const snmp = useSnmp({ openSetModalForOid: vi.fn() })
    bridge.SNMPSet.mockRejectedValue(new Error('boom'))

    const result = await snmp.handleExecuteSnmpOperation({
      operation: 'set',
      skipSetModal: true,
      oid: '1.2.3.4',
      setPayload: { valueType: 'i', value: '5', displayValue: '5' },
    })

    expect(result.error).toBeInstanceOf(Error)
    expect(addNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        severity: 'error',
        message: expect.stringContaining('failed'),
      })
    )
  })

  it('handleContextOperation seleziona l\'OID e delega a handleExecuteSnmpOperation', async () => {
    const { bridge, useSnmp } = await setup()
    bridge.SNMPGet.mockResolvedValue({ oid: '.1.2', value: 'ok' })
    const snmp = useSnmp({ openSetModalForOid: vi.fn() })
    await snmp.handleContextOperation({ node: { oid: '1.2', type: 'scalar' }, operation: 'get' })

    expect(selectedOid.value).toBe('1.2')
    expect(bridge.SNMPGet).toHaveBeenCalledWith(expect.any(Object), '1.2')
  })
})
