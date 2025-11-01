import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import TopBar from '../components/TopBar.vue'

// Mock child components and Material Web Components
const HostSettingsModal = {
  props: ['show', 'host'],
  template: '<div v-if="show" class="host-settings-modal-stub"></div>',
}

const PreferencesModal = {
  props: ['show'],
  template: '<div v-if="show" class="preferences-modal-stub"></div>',
}

vi.mock('@material/web/button/outlined-button.js', () => ({}))
vi.mock('@material/web/button/filled-button.js', () => ({}))
vi.mock('@material/web/iconbutton/icon-button.js', () => ({}))
vi.mock('@material/web/textfield/outlined-text-field.js', () => ({}))
vi.mock('@material/web/select/outlined-select.js', () => ({}))
vi.mock('@material/web/select/select-option.js', () => ({}))
vi.mock('@material/web/menu/menu.js', () => ({ setup: () => ({ open: false }) }))
vi.mock('@material/web/menu/menu-item.js', () => ({}))
vi.mock('@material/web/divider/divider.js', () => ({}))

describe('TopBar.vue', () => {
  const host = { address: 'localhost', port: 161, community: 'public', version: 'v2c' }

  it('emits update events when inputs are changed', async () => {
    const wrapper = mount(TopBar, {
      props: { host, selectedOid: '.1.3.6', operation: 'get' },
      global: { stubs: { HostSettingsModal, PreferencesModal } },
    })

    // Host
    const hostInput = wrapper.find('md-outlined-text-field[label="Host"]');
    const hostInputEl = hostInput.element;
    hostInputEl.value = '127.0.0.1';
    hostInputEl.dispatchEvent(new Event('input', { bubbles: true }));
    await flushPromises();
    expect(wrapper.emitted('update:host')[0][0].address).toBe('127.0.0.1')

    // OID
    const oidInput = wrapper.find('md-outlined-text-field[label="OID"]');
    const oidInputEl = oidInput.element;
    oidInputEl.value = '.1.3.6.1.2';
    oidInputEl.dispatchEvent(new Event('input', { bubbles: true }));
    await flushPromises();
    expect(wrapper.emitted('update:selectedOid')[0]).toEqual(['.1.3.6.1.2'])

    // Operation
    const opSelect = wrapper.find('md-outlined-select');
    const opSelectEl = opSelect.element;
    opSelectEl.value = 'walk';
    opSelectEl.dispatchEvent(new Event('change', { bubbles: true }));
    await flushPromises();
    expect(wrapper.emitted('update:operation')[0]).toEqual(['walk'])
  })

  it('emits execute event when execute button is clicked', async () => {
    const wrapper = mount(TopBar, {
      props: { host, selectedOid: '.1.3.6', operation: 'get' },
      global: { stubs: { HostSettingsModal, PreferencesModal } },
    })

    await wrapper.find('md-filled-button').trigger('click')
    expect(wrapper.emitted('execute')).toBeTruthy()
  })

  it('opens the host settings modal', async () => {
    const wrapper = mount(TopBar, {
      props: { host, selectedOid: '.1.3.6', operation: 'get' },
      global: { stubs: { HostSettingsModal, PreferencesModal } },
    })

    expect(wrapper.find('.host-settings-modal-stub').exists()).toBe(false)
    await wrapper.find('md-icon-button[title="Host Settings"]').trigger('click')
    expect(wrapper.find('.host-settings-modal-stub').exists()).toBe(true)
  })

  it('opens the preferences modal', async () => {
    const wrapper = mount(TopBar, {
      props: { host, selectedOid: '.1.3.6', operation: 'get' },
      global: { stubs: { HostSettingsModal, PreferencesModal } },
    })

    // Mock menu open state
    wrapper.vm.$refs.fileMenu.open = true
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.preferences-modal-stub').exists()).toBe(false)
    // Find the menu item for preferences and click it
    const menuItems = wrapper.findAll('md-menu-item')
    const prefsItem = menuItems.find(item => item.text().includes('Preferences'))
    await prefsItem.trigger('click')

    expect(wrapper.find('.preferences-modal-stub').exists()).toBe(true)
  })

  it('shows host suggestions and emits selection', async () => {
    const wrapper = mount(TopBar, {
      props: {
        host: { address: '', port: 161, community: 'public', version: 'v2c' },
        selectedOid: '.1.3.6',
        operation: 'get',
        hostSuggestions: [
          { address: 'router.local', port: 162, community: 'private', version: 'v3' },
          { address: 'server.lab', port: 161, community: 'public', version: 'v2c' }
        ]
      },
      global: { stubs: { HostSettingsModal, PreferencesModal } },
    })

    const hostField = wrapper.find('md-outlined-text-field[label="Host"]')
    await hostField.trigger('focusin')
    await flushPromises()

    const suggestions = wrapper.findAll('button.host-suggestion__select')
    expect(suggestions.length).toBe(2)

    await suggestions[0].trigger('mousedown')
    await flushPromises()

    const emitted = wrapper.emitted('update:host')
    const lastEmission = emitted[emitted.length - 1][0]
    expect(lastEmission.address).toBe('router.local')
    expect(lastEmission.port).toBe(162)
    expect(lastEmission.community).toBe('private')
    expect(lastEmission.version).toBe('v3')

    expect(wrapper.findAll('button.host-suggestion__select').length).toBe(0)
  })

  it('emits delete-host when clicking the delete icon on a suggestion', async () => {
    const wrapper = mount(TopBar, {
      props: {
        host: { address: '', port: 161, community: 'public', version: 'v2c' },
        selectedOid: '.1.3.6',
        operation: 'get',
        hostSuggestions: [
          { address: 'router.local', port: 162, community: 'private', version: 'v3' }
        ]
      },
      global: { stubs: { HostSettingsModal, PreferencesModal } },
    })

    const hostField = wrapper.find('md-outlined-text-field[label="Host"]')
    await hostField.trigger('focusin')
    await flushPromises()

    const deleteButton = wrapper.find('md-icon-button.host-suggestion__delete')
    expect(deleteButton.exists()).toBe(true)

    await deleteButton.trigger('click')
    expect(wrapper.emitted('delete-host')[0]).toEqual(['router.local'])
  })
})
