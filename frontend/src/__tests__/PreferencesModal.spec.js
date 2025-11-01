import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import PreferencesModal from '../components/PreferencesModal.vue'

// Mock the composable
const mockToggleTheme = vi.fn()
const mockSetColorTheme = vi.fn()
vi.mock('../composables/useTheme.js', () => ({
  useTheme: () => ({
    theme: ref('dark'),
    toggleTheme: mockToggleTheme,
    colorTheme: ref('default'),
    setColorTheme: mockSetColorTheme,
  }),
}))

// Mock Material Web Components
vi.mock('@material/web/switch/switch.js', () => ({}))
vi.mock('@material/web/button/text-button.js', () => ({}))
vi.mock('@material/web/iconbutton/icon-button.js', () => ({}))

describe('PreferencesModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('emits update:show when close button is clicked', async () => {
    const wrapper = mount(PreferencesModal, { props: { show: true } })
    await wrapper.find('md-icon-button').trigger('click')
    expect(wrapper.emitted('update:show')[0]).toEqual([false])
  })

  it('calls toggleTheme when the theme switch is changed', async () => {
    const wrapper = mount(PreferencesModal, { props: { show: true } })
    const switchControl = wrapper.find('md-switch')
    await switchControl.trigger('change')
    expect(mockToggleTheme).toHaveBeenCalled()
  })

  it('calls setColorTheme when a color option is selected', async () => {
    const wrapper = mount(PreferencesModal, { props: { show: true } })
    const blueRadio = wrapper.find('input[value="blue"]')
    await blueRadio.trigger('change')
    expect(mockSetColorTheme).toHaveBeenCalledWith('blue')
  })
})
