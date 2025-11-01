import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import OidDetailsPanel from '../components/OidDetailsPanel.vue'

describe('OidDetailsPanel.vue', () => {
  const sampleNode = {
    name: 'sysDescr',
    oid: '1.3.6.1.2.1.1.1.0',
    type: 'OCTET STRING',
    syntax: 'OctetString',
    access: 'read-only',
    status: 'current',
    description: 'A textual description of the entity.',
  }

  it('renders node details when a node is provided', () => {
    const wrapper = mount(OidDetailsPanel, {
      props: {
        node: sampleNode,
      },
    })

    expect(wrapper.text()).toContain('OID Details')
    expect(wrapper.text()).toContain('sysDescr')
    expect(wrapper.text()).toContain('1.3.6.1.2.1.1.1.0')
    expect(wrapper.text()).toContain('A textual description of the entity.')
  })

  it('renders a placeholder when no node is provided', () => {
    const wrapper = mount(OidDetailsPanel, {
      props: {
        node: null,
      },
    })

    expect(wrapper.find('.details-panel-placeholder').exists()).toBe(true)
    expect(wrapper.text()).toContain('Select a node to see its details')
  })

  it('renders N/A for missing fields', () => {
    const partialNode = { name: 'test', oid: '1.2.3' }
    const wrapper = mount(OidDetailsPanel, {
      props: { node: partialNode },
    })

    expect(wrapper.text()).toContain('Type')
    expect(wrapper.text()).toContain('N/A')
    expect(wrapper.text()).toContain('No description available')
  })
})
