import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { ref, nextTick } from 'vue'

const selectedOid = ref('')
const selectedOperation = ref('get')

vi.mock('../composables/useOidSelection', () => ({
  useOidSelection: () => ({ selectedOid, selectedOperation }),
}))

const setup = async () => {
  vi.resetModules()
  const module = await import('../composables/useTabsManager')
  return module.useTabsManager()
}

const hostA = {
  address: '192.0.2.10',
  port: 161,
  community: 'public',
  version: 'v2c'
}

const hostB = {
  address: '198.51.100.5',
  port: 162,
  community: 'public',
  version: 'v2c'
}

describe('useTabsManager', () => {
  beforeEach(() => {
    selectedOid.value = ''
    selectedOperation.value = 'get'
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('aggiunge un nuovo tab di log e lo attiva', async () => {
    const manager = await setup()
    const nowSpy = vi.spyOn(Date, 'now').mockReturnValue(1700000000000)

    expect(manager.tabs.value).toHaveLength(1)
    manager.handleAddTab()
    await nextTick()

    expect(manager.tabs.value).toHaveLength(2)
    expect(manager.activeTabId.value).toBe('tab-1700000000000')

    nowSpy.mockRestore()
  })

  it('non chiude l\'ultimo tab disponibile', async () => {
    const manager = await setup()
    manager.handleCloseTab('log-default')
    expect(manager.tabs.value).toHaveLength(1)
  })

  it('chiude il tab richiesto e riattiva il primo', async () => {
    const manager = await setup()
    vi.spyOn(Date, 'now').mockReturnValue(1700000000000)
    manager.handleAddTab()
    await nextTick()

    const newlyAddedId = manager.tabs.value[1].id
    manager.handleCloseTab(newlyAddedId)
    expect(manager.tabs.value).toHaveLength(1)
    expect(manager.activeTabId.value).toBe('log-default')
  })

  it('crea un tab tabellare solo se non esiste già sullo stesso host', async () => {
    const manager = await setup()
    vi.spyOn(Date, 'now').mockReturnValue(1700000000000)

    manager.handleOpenTableTab({ oid: '1.2.3', name: 'ifTable' }, hostA)
    await nextTick()
    expect(manager.tabs.value).toHaveLength(2)
    expect(manager.tabs.value[1].hostSnapshot).toMatchObject({ address: '192.0.2.10' })

    manager.handleOpenTableTab({ oid: '1.2.3', name: 'ifTable' }, hostA)
    await nextTick()

    expect(manager.tabs.value).toHaveLength(2)
    expect(manager.activeTabId.value).toMatch(/^table-/)
  })

  it('permette tab distinti per lo stesso OID su host diversi', async () => {
    const manager = await setup()
    vi.spyOn(Date, 'now').mockReturnValue(1700000000000)

    manager.handleOpenTableTab({ oid: '1.2.3', name: 'ifTable' }, hostA)
    await nextTick()
    manager.handleOpenTableTab({ oid: '1.2.3', name: 'ifTable' }, hostB)
    await nextTick()

    expect(manager.tabs.value.filter(tab => tab.type === 'table')).toHaveLength(2)
    const hostLabels = manager.tabs.value
      .filter(tab => tab.type === 'table')
      .map(tab => tab.hostLabel)
    expect(hostLabels).toEqual(expect.arrayContaining(['192.0.2.10:161 · v2c', '198.51.100.5:162 · v2c']))
  })

  it('apre un tab grafico e imposta selectedOid correttamente', async () => {
    const manager = await setup()
    const nowSpy = vi.spyOn(Date, 'now').mockReturnValue(1700000000000)

    manager.openGraphTab({ oid: '1.3.6.1.2', name: 'sysUpTime', syntax: 'TimeTicks' }, '  .0 ', hostA)
    await nextTick()

    expect(selectedOid.value).toBe('1.3.6.1.2.0')
    expect(manager.tabs.value[1]).toMatchObject({
      type: 'chart',
      oid: '1.3.6.1.2.0',
      baseOid: '1.3.6.1.2',
      instanceId: '0',
    })
    expect(manager.activeTabId.value).toBe('chart-1700000000000')

    // Seconda chiamata deve riutilizzare il tab esistente
    manager.openGraphTab({ oid: '1.3.6.1.2', name: 'sysUpTime' }, '0', hostA)
    await nextTick()
    expect(manager.tabs.value).toHaveLength(2)

    nowSpy.mockRestore()
  })

  it('consente grafici distinti per lo stesso OID su host differenti', async () => {
    const manager = await setup()
    const nowSpy = vi.spyOn(Date, 'now').mockReturnValue(1700000000000)

    manager.openGraphTab({ oid: '1.3.6.1.4', name: 'sysInOctets' }, '', hostA)
    await nextTick()
    manager.openGraphTab({ oid: '1.3.6.1.4', name: 'sysInOctets' }, '', hostB)
    await nextTick()

    const chartTabs = manager.tabs.value.filter(tab => tab.type === 'chart')
    expect(chartTabs).toHaveLength(2)
    expect(chartTabs.map(tab => tab.hostLabel)).toEqual(expect.arrayContaining(['192.0.2.10:161 · v2c', '198.51.100.5:162 · v2c']))

    nowSpy.mockRestore()
  })

  it('aggiorna i dati tabellari e aggiunge metadati di refresh', async () => {
    const manager = await setup()
    vi.spyOn(Date, 'now').mockReturnValue(1700000000000)
    manager.handleOpenTableTab({ oid: '1.2.3', name: 'ifTable' }, hostA)
    await nextTick()

    const tableTabId = manager.tabs.value[1].id
    manager.handleTableDataUpdate(tableTabId, {
      columns: ['ifIndex', 'ifDescr'],
      rows: [{ ifIndex: 1, ifDescr: 'lo' }],
    })

    const tableTab = manager.tabs.value[1]
    expect(tableTab.data).toHaveLength(1)
    expect(tableTab.columns).toEqual(['ifIndex', 'ifDescr'])
    expect(tableTab.lastUpdated).toBeTruthy()
  })

  it('aggiorna lo stato del grafico normalizzando i campioni', async () => {
    const manager = await setup()
    const nowSpy = vi.spyOn(Date, 'now').mockReturnValue(1700000000000)
    manager.openGraphTab({ oid: '1.2.3', name: 'sysInOctets' }, '', hostA)
    await nextTick()

    const chartTab = manager.tabs.value[1]
    chartTab.type = 'chart'

    manager.handleChartStateUpdate(chartTab.id, {
      pollingInterval: '10',
      isPolling: 1,
      useLogScale: 0,
      useDifference: 1,
      samples: [
        { timestamp: 1, value: '10', perSecond: 5 },
        { timestamp: 2, value: '15', delta: 5 },
      ],
    })

    expect(chartTab.chartState.pollingInterval).toBe(10)
    expect(chartTab.chartState.isPolling).toBe(true)
    expect(chartTab.chartState.useDifference).toBe(true)
    expect(chartTab.chartState.samples[0]).toMatchObject({
      timestamp: 1,
      value: '10',
      derivative: 5,
    })
    expect(chartTab.chartState.samples[1]).toMatchObject({
      timestamp: 2,
      value: '15',
      difference: 5,
    })

    nowSpy.mockRestore()
  })

  it('gestisce la coda del log aggiungendo e sostituendo elementi', async () => {
    const manager = await setup()
    const logTab = manager.findTargetLogTab()
    const entry = manager.prependLogEntry(logTab, { id: 'a', status: 'pending' })
    expect(logTab.data[0]).toEqual(entry)

    const updated = manager.replaceLogEntry(logTab, entry, { status: 'success' })
    expect(updated.status).toBe('success')
    expect(logTab.data[0]).toEqual(updated)
  })

  it('trova il tab di log attivo o il primo disponibile', async () => {
    const manager = await setup()
    expect(manager.findTargetLogTab()).toMatchObject({ id: 'log-default' })

    vi.spyOn(Date, 'now').mockReturnValue(1700000000000)
    manager.handleAddTab()
    await nextTick()
    const newTabIndex = manager.tabs.value.findIndex(tab => tab.id.startsWith('tab-'))
    manager.tabs.value[newTabIndex].type = 'log'
    manager.activeTabId.value = manager.tabs.value[newTabIndex].id

    expect(manager.findTargetLogTab().id).toBe(manager.tabs.value[newTabIndex].id)
  })
})
