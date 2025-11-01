import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref, nextTick } from 'vue'
import Toolbar from '../components/Toolbar.vue'

const theme = ref('dark')
const colorTheme = ref('default')
const toggleTheme = vi.fn()
const setColorTheme = vi.fn()

vi.mock('../composables/useTheme.js', () => ({
  useTheme: () => ({
    theme,
    toggleTheme,
    colorTheme,
    setColorTheme
  })
}))

vi.mock('@iconify/vue', () => ({
  Icon: {
    name: 'IconStub',
    template: '<span class="icon-stub"></span>'
  }
}))

describe('Toolbar', () => {
  beforeEach(() => {
    theme.value = 'dark'
    colorTheme.value = 'default'
    toggleTheme.mockClear()
    setColorTheme.mockClear()

    if (typeof window.matchMedia !== 'function') {
      window.matchMedia = vi.fn().mockReturnValue({
        matches: false,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn()
      })
    }

    localStorage.clear()
  })

  it('marks the active navigation button with the tonal variant', () => {
    const wrapper = mount(Toolbar, {
      props: { tabAttiva: 'browser' }
    })

    const tonalButtons = wrapper.findAll('md-filled-tonal-button')
    expect(tonalButtons.length).toBeGreaterThan(0)
    expect(tonalButtons[0].text()).toContain('Browser MIB')

    const textButtons = wrapper.findAll('md-text-button')
    expect(textButtons[0].text()).toContain('Query SNMP')
    expect(textButtons[1].text()).toContain('Log')
    wrapper.unmount()
  })

  it('emits cambia-tab when a navigation button is clicked', async () => {
    const wrapper = mount(Toolbar, {
      props: { tabAttiva: 'browser' }
    })

    const textButtons = wrapper.findAll('md-text-button')
    const queryButton = textButtons.find(btn => btn.text().includes('Query SNMP'))
    expect(queryButton).toBeTruthy()
    await queryButton.trigger('click')

    expect(wrapper.emitted()['cambia-tab'][0]).toEqual(['query'])
    wrapper.unmount()
  })

  it('invokes toggleTheme and updates label when the theme button is clicked', async () => {
    const wrapper = mount(Toolbar, {
      props: { tabAttiva: 'browser' }
    })

    const textButtons = wrapper.findAll('md-text-button')
    const themeButton = textButtons.find(btn => btn.text().includes('Dark') || btn.text().includes('Light'))
    expect(themeButton).toBeTruthy()

    expect(themeButton.text()).toContain('Dark')

    await themeButton.trigger('click')
    expect(toggleTheme).toHaveBeenCalledTimes(1)

    theme.value = 'light'
    await nextTick()

    expect(themeButton.text()).toContain('Light')
    wrapper.unmount()
  })
})
