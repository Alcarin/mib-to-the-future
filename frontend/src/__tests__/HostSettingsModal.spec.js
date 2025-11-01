import { mount } from '@vue/test-utils'
import { describe, it, expect, vi, beforeAll } from 'vitest'
import HostSettingsModal from '../components/HostSettingsModal.vue'

beforeAll(() => {
  // Mock attachInternals per jsdom
  if (!Element.prototype.attachInternals) {
    Element.prototype.attachInternals = vi.fn(() => ({
      setFormValue: vi.fn(),
    }));
  }
})

describe('HostSettingsModal', () => {
  const snmpVersions = [
    { value: 'v1', label: 'SNMPv1' },
    { value: 'v2c', label: 'SNMPv2c' },
    { value: 'v3', label: 'SNMPv3' }
  ]

  const global = {
    config: {
      compilerOptions: {
        isCustomElement: (tag) => tag.startsWith('md-'),
      },
    },
  };

  it('renders correctly when shown', () => {
    const wrapper = mount(HostSettingsModal, {
      props: {
        show: true,
        host: { port: 161, version: 'v2c', community: 'public' },
        snmpVersions
      },
      global,
    })
    expect(wrapper.find('.modal-overlay').exists()).toBe(true)
    expect(wrapper.find('h2').text()).toBe('Host Settings')
  })

  it('does not render when hidden', () => {
    const wrapper = mount(HostSettingsModal, {
      props: { show: false, host: {}, snmpVersions },
      global,
    })
    expect(wrapper.find('.modal-overlay').exists()).toBe(false)
  })

  it('emits update:show when close button is clicked', async () => {
    const wrapper = mount(HostSettingsModal, {
      props: { show: true, host: {}, snmpVersions },
      global,
    })
    await wrapper.find('md-icon-button').trigger('click')
    expect(wrapper.emitted('update:show')[0]).toEqual([false])
  })

  it('emits update:host when a field is changed', async () => {
    const wrapper = mount(HostSettingsModal, {
      props: {
        show: true,
        host: { port: 161, version: 'v2c', community: 'public' },
        snmpVersions
      },
      global,
    })

    const portInput = wrapper.findAll('md-outlined-text-field')[0]
    portInput.element.value = 162
    await portInput.trigger('input')
    expect(wrapper.emitted('update:host')[0][0]).toEqual({ port: 162, version: 'v2c', community: 'public' })
  })

  it('shows community string for v2c', () => {
    const wrapper = mount(HostSettingsModal, {
      props: {
        show: true,
        host: { version: 'v2c' },
        snmpVersions
      },
      global,
    })
    expect(wrapper.findAll('md-outlined-text-field').length).toBe(3)
  })

  it('hides community string and shows v3 fields for v3', async () => {
    const wrapper = mount(HostSettingsModal, {
      props: {
        show: true,
        host: { version: 'v2c' },
        snmpVersions
      },
      global,
    })

    expect(wrapper.findAll('md-outlined-text-field').length).toBe(3)
    expect(wrapper.find('.v3-controls').exists()).toBe(false)

    await wrapper.setProps({ host: { version: 'v3' } })

    expect(wrapper.findAll('md-outlined-text-field').length).toBe(5)
    expect(wrapper.find('.v3-controls').exists()).toBe(true)
  })
})
