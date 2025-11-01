import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'

vi.mock('../../wailsjs/go/app/App', () => ({
  SNMPGet: vi.fn(),
}))

const handleErrorMock = vi.fn()
vi.mock('../composables/useErrorHandler', () => ({
  useErrorHandler: () => ({
    handleError: handleErrorMock,
    lastError: { value: null },
  }),
}))

const createPollingWrapper = async (overrides = {}) => {
  vi.resetModules()
  const bridge = await import('../../wailsjs/go/app/App')
  bridge.SNMPGet.mockReset()
  const module = await import('../composables/useChartPolling')

  const baseProps = {
    tabInfo: {
      oid: '1.3.6.1.2.1.1.3.0',
      chartState: {},
    },
    hostConfig: {
      address: '127.0.0.1',
      port: 161,
      community: 'public',
      version: 'v2c',
    },
  }

  const props = {
    tabInfo: { ...baseProps.tabInfo, ...(overrides.tabInfo || {}) },
    hostConfig: { ...baseProps.hostConfig, ...(overrides.hostConfig || {}) },
  }

  const PollingHost = defineComponent({
    props: {
      tabInfo: {
        type: Object,
        required: true,
      },
      hostConfig: {
        type: Object,
        required: true,
      },
    },
    setup(componentProps, { emit, expose }) {
      const api = module.useChartPolling(componentProps, emit)
      expose({ api })
      return () => null
    },
  })

  const wrapper = mount(PollingHost, { props })
  await nextTick()
  return { wrapper, api: wrapper.vm.api, bridge }
}

describe('useChartPolling', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    handleErrorMock.mockClear()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('avvia il polling e accumula i campioni calcolando differenza e derivata', async () => {
    vi.useFakeTimers()
    const { wrapper, api, bridge } = await createPollingWrapper()
    const firstTs = new Date('2024-05-01T10:00:00Z')
    const secondTs = new Date(firstTs.getTime() + 1000)

    bridge.SNMPGet
      .mockResolvedValueOnce({
        timestamp: firstTs.toISOString(),
        value: '10',
      })
      .mockResolvedValueOnce({
        timestamp: secondTs.toISOString(),
        value: '15',
      })

    api.intervalInput.value = '1'
    await api.startPolling()
    expect(api.isPolling.value).toBe(true)
    expect(bridge.SNMPGet).toHaveBeenCalledTimes(1)
    expect(api.samples.value).toHaveLength(1)
    expect(api.samples.value[0]).toMatchObject({
      value: 10,
      difference: null,
      derivative: null,
    })

    await vi.advanceTimersByTimeAsync(1000)
    await Promise.resolve()
    await Promise.resolve()

    expect(bridge.SNMPGet).toHaveBeenCalledTimes(2)
    expect(api.samples.value).toHaveLength(2)
    expect(api.samples.value[0].value).toBe(10)
    expect(api.samples.value[1].value).toBe(15)
    expect(api.samples.value[1].difference).toBe(5)
    expect(api.samples.value[1].derivative).toBe(5)

    api.stopPolling()
    wrapper.unmount()
  })

  it('riporta un errore quando la configurazione SNMP è incompleta', async () => {
    const { wrapper, api } = await createPollingWrapper({
      hostConfig: { address: '' },
    })

    await api.startPolling()
    expect(handleErrorMock).toHaveBeenCalledWith(
      'Incomplete SNMP configuration. Check host and credentials.',
      'Configuration Error'
    )
    expect(api.isPolling.value).toBe(false)

    wrapper.unmount()
  })

  it('ripristina i dati quando viene invocato clearData', async () => {
    const { wrapper, api } = await createPollingWrapper()
    api.samples.value = [{ id: 'sample-1' }]
    api.lastRawValue.value = 10

    api.clearData()

    expect(api.samples.value).toEqual([])
    expect(api.lastRawValue.value).toBeNull()

    wrapper.unmount()
  })

  it('convalida l\'intervallo inserito mantenendo il valore precedente', async () => {
    const { wrapper, api } = await createPollingWrapper()
    api.intervalInput.value = '0'
    api.commitIntervalInput()
    expect(api.intervalInput.value).toBe('5')
    expect(api.pollingInterval.value).toBe(5)

    wrapper.unmount()
  })

  it('normalizza timeout e retries numerici anche se passati come stringhe', async () => {
    const { wrapper, api } = await createPollingWrapper({
      hostConfig: {
        address: '192.0.2.1',
        timeout: '8',
        retries: '3',
        port: '162'
      }
    })

    const bridge = await import('../../wailsjs/go/app/App')
    bridge.SNMPGet.mockResolvedValue({
      timestamp: new Date().toISOString(),
      value: '1'
    })

    await api.startPolling()
    expect(api.isPolling.value).toBe(true)
    expect(bridge.SNMPGet).toHaveBeenCalledWith(
      expect.objectContaining({
        host: '192.0.2.1',
        timeout: 8,
        retries: 3,
        port: 162
      }),
      expect.any(String)
    )

    api.stopPolling()
    wrapper.unmount()
  })

  it('mantiene i campi di sicurezza per SNMPv3', async () => {
    const { wrapper, api } = await createPollingWrapper({
      hostConfig: {
        address: '198.51.100.10',
        version: 'v3',
        community: 'public-ignored',
        securityLevel: 'authPriv',
        securityUsername: 'snmpuser',
        authProtocol: 'SHA',
        authPassword: 'secret',
        privProtocol: 'AES',
        privPassword: 'another-secret'
      }
    })

    const bridge = await import('../../wailsjs/go/app/App')
    bridge.SNMPGet.mockResolvedValue({
      timestamp: new Date().toISOString(),
      value: '42'
    })

    await api.startPolling()
    expect(api.isPolling.value).toBe(true)
    expect(bridge.SNMPGet).toHaveBeenCalledWith(
      expect.objectContaining({
        host: '198.51.100.10',
        version: 'v3',
        securityLevel: 'authPriv',
        securityUsername: 'snmpuser',
        authProtocol: 'SHA',
        authPassword: 'secret',
        privProtocol: 'AES',
        privPassword: 'another-secret'
      }),
      expect.any(String)
    )

    api.stopPolling()
    wrapper.unmount()
  })
})
