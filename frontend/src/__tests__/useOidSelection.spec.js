import { describe, it, expect, beforeEach, vi } from 'vitest'

beforeEach(() => {
  vi.resetModules()
})

describe('useOidSelection', () => {
  it('aggiorna l\'OID selezionato quando viene invocato handleOidSelect', async () => {
    const { useOidSelection } = await import('../composables/useOidSelection')
    const { selectedOid, handleOidSelect } = useOidSelection()

    handleOidSelect('1.3.6.1.4')
    expect(selectedOid.value).toBe('1.3.6.1.4')
  })

  it('normalizza l\'OID proveniente da un log entry', async () => {
    const { useOidSelection } = await import('../composables/useOidSelection')
    const { selectedOid, handleLogEntrySelect } = useOidSelection()

    handleLogEntrySelect({ oid: '.1.3.6.1.4.0' })
    expect(selectedOid.value).toBe('1.3.6.1.4')
  })
})
