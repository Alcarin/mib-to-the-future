import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

vi.mock('../../wailsjs/go/app/App', () => ({
  ListHosts: vi.fn(),
  DeleteHost: vi.fn(),
}))

const setup = async () => {
  vi.resetModules()
  const appBridge = await import('../../wailsjs/go/app/App')
  appBridge.ListHosts.mockReset()
  appBridge.DeleteHost.mockReset()
  const { useHostManager } = await import('../composables/useHostManager')
  return { appBridge, manager: useHostManager() }
}

describe('useHostManager', () => {
  let consoleError

  beforeEach(() => {
    consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
  })

  afterEach(() => {
    consoleError.mockRestore()
  })

  it('carica e ordina gli host in base all\'ultimo utilizzo', async () => {
    const { appBridge, manager } = await setup()
    appBridge.ListHosts.mockResolvedValue([
      {
        address: '10.0.0.1',
        port: '162',
        community: 'public',
        lastUsedAt: '2024-02-01T10:00:00Z',
      },
      {
        address: '10.0.0.2',
        port: '161',
        community: 'private',
        writeCommunity: '',
        lastUsedAt: '2024-03-01T09:30:00Z',
      },
    ])

    await manager.loadSavedHosts()

    expect(appBridge.ListHosts).toHaveBeenCalledTimes(1)
    expect(manager.savedHosts.value).toHaveLength(2)
    expect(manager.savedHosts.value[0].address).toBe('10.0.0.2')
    expect(manager.host.value.address).toBe('10.0.0.2')
    expect(manager.host.value.port).toBe(161) // coercePort converte stringa in numero
    expect(manager.host.value.writeCommunity).toBe('private') // fallback al valore comunità
  })

  it('ripristina lo stato di default quando non ci sono host salvati', async () => {
    const { appBridge, manager } = await setup()
    appBridge.ListHosts.mockResolvedValue([])

    await manager.loadSavedHosts()

    expect(manager.savedHosts.value).toEqual([])
    expect(manager.host.value).toMatchObject({
      address: '127.0.0.1',
      port: 161,
      community: 'public',
      version: 'v2c',
    })
  })

  it('gestisce errori di caricamento e azzera la cache locale', async () => {
    const { appBridge, manager } = await setup()
    appBridge.ListHosts.mockRejectedValue(new Error('boom'))

    await manager.loadSavedHosts()

    expect(consoleError).toHaveBeenCalledWith('Failed to load saved hosts:', expect.any(Error))
    expect(manager.savedHosts.value).toEqual([])
    expect(manager.host.value.address).toBe('127.0.0.1')
  })

  it('non sostituisce la configurazione corrente dopo l\'inizializzazione', async () => {
    const { appBridge, manager } = await setup()
    appBridge.ListHosts.mockResolvedValue([
      { address: '10.0.0.2', lastUsedAt: '2024-03-01T09:30:00Z' },
      { address: '10.0.0.1', lastUsedAt: '2024-02-01T10:00:00Z' },
    ])

    await manager.loadSavedHosts()

    manager.host.value.address = '192.168.0.50'
    appBridge.ListHosts.mockResolvedValue([
      { address: '10.0.0.2', lastUsedAt: '2024-03-01T09:30:00Z' },
    ])

    await manager.loadSavedHosts()

    expect(manager.host.value.address).toBe('192.168.0.50')
    expect(manager.savedHosts.value[0].address).toBe('10.0.0.2')
  })

  it('chiama DeleteHost e ricarica la lista dopo la cancellazione', async () => {
    const { appBridge, manager } = await setup()
    appBridge.ListHosts.mockResolvedValueOnce([{ address: 'a', lastUsedAt: '2024-01-01' }])
    await manager.loadSavedHosts()

    appBridge.ListHosts.mockResolvedValueOnce([])

    await manager.handleDeleteHost(' a ')

    expect(appBridge.DeleteHost).toHaveBeenCalledWith('a')
    expect(appBridge.ListHosts).toHaveBeenCalledTimes(2)
    expect(manager.savedHosts.value).toEqual([])
  })

  it('ignora la cancellazione quando l\'indirizzo è vuoto', async () => {
    const { appBridge, manager } = await setup()
    await manager.handleDeleteHost('   ')
    expect(appBridge.DeleteHost).not.toHaveBeenCalled()
  })
})
