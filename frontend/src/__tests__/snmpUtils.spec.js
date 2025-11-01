import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

const loadUtils = async () => import('../utils/snmp')

describe('snmp utils', () => {
  beforeEach(() => {
    vi.resetModules()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('normalizza e converte gli OID come previsto', async () => {
    const utils = await loadUtils()
    expect(utils.normalizeOid('.1.2.3')).toBe('1.2.3')
    expect(utils.normalizeOid('1.2.3')).toBe('1.2.3')
    expect(utils.canonicalTreeOid('1.2.3.0')).toBe('1.2.3')
  })

  it('genera identificatori di log univoci e incrementali', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2024-01-01T00:00:00Z'))
    const utils = await loadUtils()
    const first = utils.createLogEntryId()
    const second = utils.createLogEntryId()
    expect(first).toBe('log-1704067200000-1')
    expect(second).toBe('log-1704067200000-2')
    expect(second).not.toBe(first)
  })

  it('arricchisce le entry di log con OID normalizzato e nome risolto', async () => {
    const utils = await loadUtils()
    const entry = utils.buildLogEntry({
      oid: '.1.2.3.0',
      resolvedName: 'sysUpTime',
      value: '42',
    })
    expect(entry.oid).toBe('1.2.3.0')
    expect(entry.oidName).toBe('sysUpTime')
  })

  it('estrae i valori e la sintassi dai risultati SNMP', async () => {
    const utils = await loadUtils()
    const sample = { value: 10, displayValue: '10', rawValue: 10, syntax: 'Counter32' }
    expect(utils.getResultDisplayValue(sample)).toBe('10')
    expect(utils.getResultRawValue(sample)).toBe(10)
    expect(utils.getResultSyntax(sample)).toBe('Counter32')
  })

  it('valuta il tipo dei nodi e normalizza gli identificativi istanza', async () => {
    const utils = await loadUtils()
    expect(utils.isWritableNode({ access: 'read-write' })).toBe(true)
    expect(utils.getOriginalNodeType('bookmark-column')).toBe('column')
    expect(utils.sanitizeInstanceId(' .1.0 ')).toBe('1.0')
    expect(utils.buildInstanceOid('1.2.3', ' 1.0 ')).toBe('1.2.3.1.0')
    expect(utils.buildInstanceOid('1.2.3', '1.2.3.4')).toBe('1.2.3.4')
  })
})
