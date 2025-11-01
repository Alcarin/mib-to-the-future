import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ref } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'

vi.mock('../../wailsjs/go/app/App')

const echartsMocks = vi.hoisted(() => ({
  initMock: vi.fn(() => ({
    setOption: vi.fn(),
    resize: vi.fn(),
    dispose: vi.fn()
  }))
}))

vi.mock('echarts/core', () => ({
  use: vi.fn(),
  init: echartsMocks.initMock
}))

vi.mock('@material/web/textfield/outlined-text-field.js', () => {
  if (!customElements.get('md-outlined-text-field')) {
    customElements.define('md-outlined-text-field', class extends HTMLElement {})
  }
  return {}
})

vi.mock('@material/web/button/filled-button.js', () => {
  if (!customElements.get('md-filled-button')) {
    customElements.define('md-filled-button', class extends HTMLElement {})
  }
  return {}
})

vi.mock('@material/web/switch/switch.js', () => {
  if (!customElements.get('md-switch')) {
    customElements.define('md-switch', class extends HTMLElement {
      constructor() {
        super()
        this.selected = false
      }
    })
  }
  return {}
})

import ChartTab from '../components/ChartTab.vue'
import { useChartPolling } from '../composables/useChartPolling'
vi.mock('../composables/useChartPolling')

const { initMock } = echartsMocks

const createComponent = (overrides = {}) => {
  return mount(ChartTab, {
    props: {
      tabInfo: {
        oid: '1.3.6.1.2.1.1.5.0',
        chartState: {
          pollingInterval: 5,
          isPolling: false,
          useLogScale: false,
          useDifference: false,
          useDerivative: false,
          enforceNonNegative: false,
          samples: [],
          lastRawValue: null,
          lastSampleTimestamp: null
        },
        ...overrides.tabInfo
      },
      hostConfig: {
        address: '192.0.2.10',
        port: 161,
        community: 'public',
        version: 'v2c',
        ...overrides.hostConfig
      }
    },
    attachTo: document.body
  })
}

describe('ChartTab', () => {
  let originalResizeObserver
  const startPollingMock = vi.fn()
  const stopPollingMock = vi.fn()

  beforeEach(() => {
    originalResizeObserver = global.ResizeObserver
    global.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
    initMock.mockClear()
    startPollingMock.mockClear()
    stopPollingMock.mockClear()

    useChartPolling.mockReturnValue({
      pollingInterval: ref(5),
      intervalInput: ref('5'),
      isPolling: ref(false),
      isFetching: ref(false),
      samples: ref([]),
      lastRawValue: ref(null),
      lastSampleTimestamp: ref(null),
      startPolling: startPollingMock,
      stopPolling: stopPollingMock,
      pausePolling: vi.fn(),
      clearData: vi.fn(),
      commitIntervalInput: vi.fn(),
      setSamples: vi.fn(),
      lastError: ref(null)
    })
  })

  afterEach(() => {
    if (originalResizeObserver) {
      global.ResizeObserver = originalResizeObserver
    } else {
      delete global.ResizeObserver
    }
  })

  it('initializes ECharts and renders controls', async () => {
    const wrapper = createComponent()
    await flushPromises()

    expect(initMock).toHaveBeenCalledTimes(1)

    const intervalField = wrapper.find('md-outlined-text-field')
    expect(intervalField.exists()).toBe(true)

    const toggles = wrapper.findAll('md-switch')
    expect(toggles).toHaveLength(4)

    wrapper.unmount()
  })

  it('emits state updates when polling starts', async () => {
    const isPolling = ref(false)
    startPollingMock.mockImplementation(async () => {
      isPolling.value = true
    })
    useChartPolling.mockReturnValueOnce({
      pollingInterval: ref(5),
      intervalInput: ref('5'),
      isPolling,
      isFetching: ref(false),
      samples: ref([{ rawValue: 42 }]),
      lastRawValue: ref(null),
      lastSampleTimestamp: ref(null),
      startPolling: startPollingMock,
      stopPolling: stopPollingMock,
      pausePolling: vi.fn(),
      clearData: vi.fn(),
      commitIntervalInput: vi.fn(),
      setSamples: vi.fn(),
      lastError: ref(null)
    })

    const wrapper = createComponent()
    await flushPromises()

    const actionButton = wrapper.find('md-filled-button')
    await actionButton.trigger('click')
    await flushPromises()

    expect(startPollingMock).toHaveBeenCalled()

    const events = wrapper.emitted('state-update') || []
    const sampleEvent = events.map(event => event[0]).find(payload => payload.isPolling)
    expect(sampleEvent).toBeDefined()
    expect(sampleEvent?.isPolling).toBe(true)

    wrapper.unmount()
  })

  it('emits transformations when toggles change', async () => {
    const wrapper = createComponent()
    await flushPromises()

    const toggles = wrapper.findAll('md-switch')
    expect(toggles).toHaveLength(4)

    // Log scale toggle
    toggles[0].element.selected = true
    toggles[0].element.dispatchEvent(new Event('change'))
    await flushPromises()

    // Enable difference and clamp
    toggles[1].element.selected = true
    toggles[1].element.dispatchEvent(new Event('change'))
    await flushPromises()

    toggles[3].element.selected = true
    toggles[3].element.dispatchEvent(new Event('change'))
    await flushPromises()

    // Switch to derivative (should disable difference)
    toggles[2].element.selected = true
    toggles[2].element.dispatchEvent(new Event('change'))
    await flushPromises()

    const events = wrapper.emitted('state-update') || []
    const lastPayload = events[events.length - 1]?.[0]
    expect(lastPayload?.useLogScale).toBe(true)
    expect(lastPayload?.useDifference).toBe(false)
    expect(lastPayload?.useDerivative).toBe(true)
    expect(lastPayload?.enforceNonNegative).toBe(true)

    wrapper.unmount()
  })

  it('mostra l\'errore del polling quando presente', async () => {
    const lastError = ref({ message: 'Incomplete SNMP configuration. Check host and credentials.' })
    useChartPolling.mockReturnValueOnce({
      pollingInterval: ref(5),
      intervalInput: ref('5'),
      isPolling: ref(false),
      isFetching: ref(false),
      samples: ref([]),
      lastRawValue: ref(null),
      lastSampleTimestamp: ref(null),
      startPolling: startPollingMock,
      stopPolling: stopPollingMock,
      pausePolling: vi.fn(),
      clearData: vi.fn(),
      commitIntervalInput: vi.fn(),
      setSamples: vi.fn(),
      lastError
    })

    const wrapper = createComponent({
      hostConfig: { address: '192.0.2.10' }
    })
    await flushPromises()

    const errorBanner = wrapper.find('.chart-error')
    expect(errorBanner.exists()).toBe(true)
    expect(errorBanner.text()).toContain('Incomplete SNMP configuration')

    wrapper.unmount()
  })

  it('displays an error message when polling fails', async () => {
    const lastError = ref(null)
    const startPollingMock = vi.fn().mockImplementation(() => {
      lastError.value = { message: 'SNMP request failed: boom' };
    });

    useChartPolling.mockReturnValueOnce({
      pollingInterval: ref(5),
      intervalInput: ref('5'),
      isPolling: ref(false),
      isFetching: ref(false),
      samples: ref([]),
      lastRawValue: ref(null),
      lastSampleTimestamp: ref(null),
      startPolling: startPollingMock,
      stopPolling: vi.fn(),
      pausePolling: vi.fn(),
      clearData: vi.fn(),
      commitIntervalInput: vi.fn(),
      setSamples: vi.fn(),
      lastError
    })

    const wrapper = createComponent()
    await flushPromises()

    const actionButton = wrapper.find('md-filled-button')
    await actionButton.trigger('click')
    await flushPromises()

    expect(startPollingMock).toHaveBeenCalled()
    const errorBanner = wrapper.find('.chart-error')
    expect(errorBanner.exists()).toBe(true)
    expect(errorBanner.text()).toContain('SNMP request failed: boom')

    wrapper.unmount()
  })
});

    

    
