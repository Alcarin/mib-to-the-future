import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import TableTab from '../components/TableTab.vue'
import ChartTab from '../components/ChartTab.vue'

vi.mock('../../wailsjs/go/app/App')

vi.mock('@material/web/chips/filter-chip.js', () => ({}))
vi.mock('@material/web/iconbutton/icon-button.js', () => ({}))
vi.mock('@material/web/button/filled-button.js', () => ({}))
vi.mock('@material/web/textfield/outlined-text-field.js', () => ({}))
vi.mock('@material/web/switch/switch.js', () => ({}))

vi.mock('echarts/core', () => {
  const resize = vi.fn()
  const setOption = vi.fn()
  const dispose = vi.fn()
  return {
    __esModule: true,
    init: vi.fn(() => ({ setOption, dispose, resize })),
    use: vi.fn()
  }
})

vi.mock('echarts/charts', () => ({ __esModule: true, LineChart: {} }))
vi.mock('echarts/components', () => ({
  __esModule: true,
  TooltipComponent: {},
  GridComponent: {},
  DataZoomComponent: {},
  LegendComponent: {},
  TitleComponent: {}
}))
vi.mock('echarts/renderers', () => ({ __esModule: true, CanvasRenderer: {} }))

import { FetchTableData, SaveCSVFile, SNMPGet } from '../../wailsjs/go/app/App'
import { useChartPolling } from '../composables/useChartPolling'
vi.mock('../composables/useChartPolling')

const ensureCustomElement = (name) => {
  if (typeof customElements !== 'undefined' && !customElements.get(name)) {
    customElements.define(name, class extends HTMLElement {})
  }
}

;['md-icon-button', 'md-filter-chip', 'md-filled-button', 'md-outlined-text-field', 'md-switch'].forEach(ensureCustomElement)

const globalStubs = {
  'md-icon-button': {
    name: 'MdIconButton',
    emits: ['click'],
    template: '<button class="md-icon-button" v-bind="$attrs" @click="$emit(\'click\', $event)"><slot /></button>'
  },
  'md-filter-chip': {
    name: 'MdFilterChip',
    emits: ['click'],
    template: '<button class="md-filter-chip" v-bind="$attrs" @click="$emit(\'click\', $event)"><slot /></button>'
  },
  'md-filled-button': {
    name: 'MdFilledButton',
    emits: ['click'],
    template: '<button class="md-filled-button" v-bind="$attrs" @click="$emit(\'click\', $event)"><slot /></button>'
  },
  'md-outlined-text-field': {
    name: 'MdOutlinedTextField',
    template: '<div class="md-outlined-text-field" v-bind="$attrs"><slot name="leading-icon" /><slot /></div>'
  },
  'md-switch': {
    name: 'MdSwitch',
    emits: ['change'],
    template: '<button class="md-switch" type="button" role="switch" v-bind="$attrs" @click="$emit(\'change\', $event)"><slot /></button>'
  }
}

describe('CSV export helpers', () => {
  beforeEach(() => {
    FetchTableData.mockClear()
    SaveCSVFile.mockClear()
    SaveCSVFile.mockResolvedValue(true)
    SNMPGet.mockClear()
  })

  it('esporta i dati della tabella usando SaveCSVFile con nome sanificato', async () => {
    const nowSpy = vi.spyOn(Date, 'now').mockReturnValue(1700000000000)
    const wrapper = mount(TableTab, {
      props: {
        tabInfo: {
          id: 'tab-1',
          title: 'Interfaccia/Temperatura #1',
          oid: '1.3.6.1',
          columns: [
            { key: 'name', label: 'Nome' },
            { key: 'value', label: 'Valore' }
          ],
          data: [
            { id: 'row-1', name: 'Alpha', value: 42 }
          ]
        },
        hostConfig: {}
      },
      global: { stubs: globalStubs }
    })

    await wrapper.vm.exportTable()

    expect(SaveCSVFile).toHaveBeenCalledTimes(1)
    const [filename, csvContent] = SaveCSVFile.mock.calls[0]
    expect(filename).toBe('interfaccia-temperatura-1-1700000000000.csv')
    expect(csvContent).toBe('Nome,Valore\nAlpha,42')

    nowSpy.mockRestore()
  })

  it('esporta i campioni del grafico usando SaveCSVFile e include i valori principali', async () => {
    useChartPolling.mockReturnValueOnce({
      pollingInterval: ref(5),
      intervalInput: ref('5'),
      isPolling: ref(false),
      isFetching: ref(false),
      errorMessage: ref(null),
      samples: ref([
        {
          id: 'sample-1',
          timestamp: 1700000000000,
          rawValue: 128,
          difference: 5,
          derivative: 2,
          displayValue: '128',
          responseTime: 40,
          resolvedName: 'ifInOctets'
        }
      ]),
      lastRawValue: ref(null),
      lastSampleTimestamp: ref(null),
      startPolling: vi.fn(),
      stopPolling: vi.fn(),
      pausePolling: vi.fn(),
      clearData: vi.fn(),
      commitIntervalInput: vi.fn(),
      setSamples: vi.fn(),
    })

    const nowSpy = vi.spyOn(Date, 'now').mockReturnValue(1700000000000)
    const wrapper = mount(ChartTab, {
      props: {
        tabInfo: {
          id: 'chart-1',
          title: 'Grafico Totale%',
          oid: '1.3.6.1.4.1',
          chartState: { // chartState is now ignored because we mock the composable
            samples: []
          }
        }
      },
      global: { stubs: globalStubs }
    })

    await wrapper.vm.exportSamples()

    expect(SaveCSVFile).toHaveBeenCalledTimes(1)
    const [filename, csvContent] = SaveCSVFile.mock.calls[0]
    expect(filename).toBe('grafico-totale-1700000000000.csv')
    expect(csvContent.startsWith('Timestamp,Label,Raw Value,Difference,Derivative,Display Value,Response Time,Resolved Name')).toBe(true)
    expect(csvContent).toContain('128')
    expect(csvContent).toContain('ifInOctets')

    nowSpy.mockRestore()
  })
})
