<script setup>
/**
 * @vue-component
 * @description Visualizza un grafico temporale basato su ECharts per un OID numerico,
 * permettendo il polling SNMP configurabile e la trasformazione dei dati.
 *
 * @vue-prop {Object} tabInfo - Metadati della tab, inclusi OID e stato del grafico.
 * @vue-prop {Object} hostConfig - Configurazione SNMP corrente per eseguire i GET.
 *
 * @vue-event {object} state-update - Emesso ogni volta che lo stato del grafico cambia.
 */
import { ref, computed, watch, onMounted, onBeforeUnmount, onActivated, nextTick } from 'vue'
import '@material/web/textfield/outlined-text-field.js'
import '@material/web/button/filled-button.js'
import '@material/web/switch/switch.js'
import '@material/web/iconbutton/icon-button.js'
import { SaveCSVFile } from '../../wailsjs/go/app/App'
import * as echarts from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { TooltipComponent, GridComponent, DataZoomComponent, LegendComponent, TitleComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

// Import dei moduli refactorizzati
import { readThemeColors, withAlpha } from '../utils/chartColors'
import { formatTimeLabel, cloneSamples } from '../utils/chartHelpers'
import { useChartPolling } from '../composables/useChartPolling'
import { useErrorHandler } from '../composables/useErrorHandler'

echarts.use([LineChart, TooltipComponent, GridComponent, DataZoomComponent, LegendComponent, TitleComponent, CanvasRenderer])

const props = defineProps({
  tabInfo: {
    type: Object,
    required: true
  },
  hostConfig: {
    type: Object,
    default: () => null
  }
})

const emit = defineEmits(['state-update'])

const { handleError } = useErrorHandler()

const hostSnapshot = computed(() => props.tabInfo?.hostSnapshot || props.hostConfig || null)

const hostLabel = computed(() => {
  const snapshot = hostSnapshot.value
  if (!snapshot || !snapshot.address) {
    return ''
  }
  const port = snapshot.port ? `:${snapshot.port}` : ''
  const version = snapshot.version ? ` · ${snapshot.version}` : ''
  return `${snapshot.address}${port}${version}`
})

const chartTitle = computed(() => {
  if (props.tabInfo?.displayName) {
    return props.tabInfo.displayName
  }
  const title = props.tabInfo?.title || ''
  if (title.toLowerCase().startsWith('graph:')) {
    const [, ...rest] = title.split(':')
    const trimmed = rest.join(':').trim()
    if (trimmed) {
      return trimmed
    }
  }
  return title || props.tabInfo?.oid || 'SNMP Chart'
})

const numericOid = computed(() => props.tabInfo?.oid || '')
const baseOid = computed(() => props.tabInfo?.baseOid || '')

// Initialize polling composable
const {
  pollingInterval,
  intervalInput,
  isPolling,
  isFetching,
  samples,
  lastRawValue,
  lastSampleTimestamp,
  startPolling,
  stopPolling,
  pausePolling,
  commitIntervalInput,
  lastError
} = useChartPolling(props, emit)

const themeColors = ref(readThemeColors())

const useLogScale = ref(Boolean(props.tabInfo?.chartState?.useLogScale))
const useDifference = ref(Boolean(props.tabInfo?.chartState?.useDifference))
const useDerivative = ref(Boolean(props.tabInfo?.chartState?.useDerivative))
const enforceNonNegative = ref(Boolean(props.tabInfo?.chartState?.enforceNonNegative))

const chartContainer = ref(null)
let chartInstance = null
let resizeObserver = null
let themeObserver = null

let isApplyingExternal = false

const hasSamples = computed(() => samples.value.length > 0)
const lastSample = computed(() =>
  samples.value.length > 0 ? samples.value[samples.value.length - 1] : null
)

const chartSeriesData = computed(() => {
  const clampValues = enforceNonNegative.value && (useDifference.value || useDerivative.value)
  return samples.value.map(sample => {
    let value = null
    if (useDerivative.value) {
      value = sample.derivative ?? null
    } else if (useDifference.value) {
      value = sample.difference ?? null
    } else {
      value = sample.rawValue ?? null
    }

    if (value !== null && value !== undefined && clampValues) {
      value = Math.max(0, value)
    }

    return [sample.timestamp, value]
  })
})

/**
 * @function escapeCsvValue
 * @description Gestisce le regole di escaping per una cella CSV evitando rotture di formato.
 * @param {*} value - Il valore da convertire.
 * @returns {string} La stringa sicura per il CSV.
 */
const escapeCsvValue = (value) => {
  if (value === null || value === undefined) {
    return ''
  }
  let stringValue = String(value)
  if (stringValue.includes('"')) {
    stringValue = stringValue.replace(/"/g, '""')
  }
  if (/[",\n]/.test(stringValue)) {
    return `"${stringValue}"`
  }
  return stringValue
}

/**
 * @function sanitizeFileName
 * @description Produce un nome file adatto al filesystem sostituendo caratteri non validi.
 * @param {string} input - La stringa di partenza.
 * @returns {string} Il nome file ripulito. 
 */
const sanitizeFileName = (input) => {
  if (!input) {
    return 'chart'
  }
  return input.replace(/[^a-z0-9-_]+/gi, '-').replace(/^-+|-+$/g, '').toLowerCase() || 'chart'
}

/**
 * @function exportSamples
 * @description Esporta i campioni raccolti dal grafico in un file CSV scaricabile.
 */
const exportSamples = async () => {
  if (!hasSamples.value) {
    return
  }

  const headers = [
    'Timestamp',
    'Label',
    'Raw Value',
    'Difference',
    'Derivative',
    'Display Value',
    'Response Time',
    'Resolved Name'
  ]

  const rows = samples.value.map(sample => {
    const timestampIso = Number.isFinite(sample.timestamp)
      ? new Date(sample.timestamp).toISOString()
      : ''

    return [
      escapeCsvValue(timestampIso),
      escapeCsvValue(sample.label),
      escapeCsvValue(sample.rawValue),
      escapeCsvValue(sample.difference),
      escapeCsvValue(sample.derivative),
      escapeCsvValue(sample.displayValue),
      escapeCsvValue(sample.responseTime),
      escapeCsvValue(sample.resolvedName)
    ].join(',')
  })

  const csv = [headers.join(','), ...rows].join('\n')
  const baseName = sanitizeFileName(props.tabInfo?.title || props.tabInfo?.oid || 'chart')

  try {
    await SaveCSVFile(`${baseName}-${Date.now()}.csv`, csv)
  } catch (err) {
    handleError(err, 'Failed to save CSV file')
  }
}

const seriesLabel = computed(() => {
  if (useDerivative.value) {
    return 'Derivative (per second)'
  }
  if (useDifference.value) {
    return 'Difference'
  }
  return 'Value'
})

watch(
  () => props.tabInfo?.chartState,
  (next) => {
    if (!next) return
    applyExternalState(next)
  },
  { deep: true }
)

watch(
  () => props.tabInfo?.id,
  (next, prev) => {
    if (next && next !== prev && props.tabInfo?.chartState) {
      applyExternalState(props.tabInfo.chartState)
    }
  }
)

watch(pollingInterval, (newVal, oldVal) => {
  if (newVal === oldVal) return
  intervalInput.value = String(newVal)
  if (isApplyingExternal) {
    return
  }
  // Note: startPolling already handles scheduling
  syncState()
})

watch(useLogScale, () => {
  if (isApplyingExternal) {
    updateChart()
    return
  }
  updateChart()
  syncState()
})

watch(useDifference, (next) => {
  if (next && !isApplyingExternal) {
    useDerivative.value = false
  }
  updateChart()
  if (isApplyingExternal) {
    return
  }
  syncState()
})

watch(useDerivative, () => {
  if (isApplyingExternal) {
    updateChart()
    return
  }
  if (useDerivative.value && !isApplyingExternal) {
    useDifference.value = false
  }
  updateChart()
  syncState()
})

watch(enforceNonNegative, () => {
  updateChart()
  if (isApplyingExternal) {
    return
  }
  syncState()
})

watch(samples, () => {
  updateChart()
  if (isApplyingExternal) {
    return
  }
  syncState()
}, { deep: true })

watch(themeColors, () => {
  updateChart()
}, { deep: true })

const applyExternalState = (state) => {
  if (state == null || typeof state !== 'object') {
    return
  }
  if (state.tabId && props.tabInfo?.id && state.tabId !== props.tabInfo.id) {
    return
  }
  if (state.oid && props.tabInfo?.oid && state.oid !== props.tabInfo.oid) {
    return
  }
  isApplyingExternal = true
  if (Number(state.pollingInterval) > 0) {
    pollingInterval.value = Number(state.pollingInterval)
  }
  if (Array.isArray(state.samples)) {
    samples.value = cloneSamples(state.samples)
  }
  if (state.lastRawValue !== undefined) {
    lastRawValue.value = state.lastRawValue
  }
  if (state.lastSampleTimestamp !== undefined) {
    lastSampleTimestamp.value = state.lastSampleTimestamp
  }
  if (typeof state.isPolling === 'boolean') {
    isPolling.value = state.isPolling
  }
  if (typeof state.useLogScale === 'boolean') {
    useLogScale.value = state.useLogScale
  }
  if (typeof state.useDifference === 'boolean') {
    useDifference.value = state.useDifference
  }
  if (typeof state.useDerivative === 'boolean') {
    useDerivative.value = state.useDerivative
  }
  if (typeof state.enforceNonNegative === 'boolean') {
    enforceNonNegative.value = state.enforceNonNegative
  }
  if (useDerivative.value && useDifference.value) {
    if (state.useDerivative) {
      useDifference.value = false
    } else if (state.useDifference) {
      useDerivative.value = false
    } else {
      useDifference.value = false
    }
  }
  nextTick(() => {
    try {
      updateChart()
    } finally {
      isApplyingExternal = false
    }
  })
}

const syncState = () => {
  if (isApplyingExternal) {
    return
  }
  emit('state-update', {
    tabId: props.tabInfo?.id || null,
    pollingInterval: pollingInterval.value,
    isPolling: isPolling.value,
    useLogScale: useLogScale.value,
    useDifference: useDifference.value,
    useDerivative: useDerivative.value,
    enforceNonNegative: enforceNonNegative.value,
    samples: cloneSamples(samples.value),
    lastRawValue: lastRawValue.value,
    lastSampleTimestamp: lastSampleTimestamp.value,
    oid: props.tabInfo?.oid || null
  })
}

const togglePolling = async () => {
  if (isPolling.value) {
    handleStopPolling()
  } else {
    await handleStartPolling()
  }
}

// Wrapper functions that integrate composable methods with syncState
const handleStartPolling = async () => {
  await startPolling()
  syncState()
}

const handleStopPolling = () => {
  stopPolling()
  syncState()
}

const formatAxisLabel = (value) => {
  if (value === null || value === undefined) {
    return ''
  }
  if (Math.abs(value) >= 1) {
    return value.toLocaleString(undefined, { maximumFractionDigits: 2 })
  }
  return value.toExponential(2)
}

const formatTooltip = (params) => {
  if (!Array.isArray(params) || params.length === 0) {
    return ''
  }
  const point = params[0]
  const sample = samples.value.find(item => item.timestamp === point.data?.[0])
  if (!sample) {
    return ''
  }
  const clampValues = enforceNonNegative.value && (useDifference.value || useDerivative.value)
  const valueDisplay = sample.displayValue ?? '—'
  const tooltipTime = formatTimeLabel(sample.timestamp ?? point.axisValue ?? point.data?.[0])
  const rows = [
    `<div class="chart-tooltip__title">${tooltipTime}</div>`,
    `<div class="chart-tooltip__row"><span class="chart-tooltip__label">Value:</span><span class="chart-tooltip__value">${valueDisplay}</span></div>`
  ]
  if (useDifference.value && sample.difference !== null && sample.difference !== undefined) {
    const diffValue = clampValues ? Math.max(0, sample.difference) : sample.difference
    rows.push(`<div class="chart-tooltip__row"><span class="chart-tooltip__label">Difference:</span><span class="chart-tooltip__value">${formatNumericValue(diffValue)}</span></div>`)
  }
  if (useDerivative.value) {
    const derivative = sample.derivative ?? null
    const derivedValue = derivative === null || derivative === undefined
      ? derivative
      : (clampValues ? Math.max(0, derivative) : derivative)
    rows.push(`<div class="chart-tooltip__row"><span class="chart-tooltip__label">Derivative:</span><span class="chart-tooltip__value">${formatNumericValue(derivedValue)}</span></div>`)
  }
  if (sample.responseTime !== null) {
    rows.push(`<div class="chart-tooltip__row"><span class="chart-tooltip__label">Latency:</span><span class="chart-tooltip__value">${sample.responseTime} ms</span></div>`)
  }
  return `<div class="chart-tooltip">${rows.join('')}</div>`
}

const formatNumericValue = (value) => {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return '—'
  }
  return value.toLocaleString(undefined, { maximumFractionDigits: 3 })
}

const handleIntervalInput = (event) => {
  intervalInput.value = event?.target?.value ?? ''
}

const handleLogScaleToggle = (event) => {
  useLogScale.value = Boolean(event?.target?.selected)
}

const handleDifferenceToggle = (event) => {
  const next = Boolean(event?.target?.selected)
  useDifference.value = next
  if (next) {
    useDerivative.value = false
  }
}

const handleDerivativeToggle = (event) => {
  const next = Boolean(event?.target?.selected)
  useDerivative.value = next
  if (next) {
    useDifference.value = false
  }
}

const handleNonNegativeToggle = (event) => {
  enforceNonNegative.value = Boolean(event?.target?.selected)
}

const updateChart = () => {
  if (!chartInstance) {
    return
  }

  const colors = themeColors.value

  chartInstance.setOption({
    backgroundColor: colors.surfaceContainer,
    textStyle: {
      color: colors.onSurface
    },
    xAxis: {
      type: 'time',
      boundaryGap: false,
      axisLabel: {
        formatter: (value) => formatTimeLabel(value),
        color: colors.onSurfaceVariant
      },
      axisLine: {
        lineStyle: { color: colors.outline }
      },
      splitLine: {
        show: false
      }
    },
    yAxis: {
      type: useLogScale.value ? 'log' : 'value',
      axisLabel: {
        formatter: formatAxisLabel,
        color: colors.onSurfaceVariant
      },
      axisLine: {
        lineStyle: { color: colors.outline }
      },
      splitLine: {
        show: true,
        lineStyle: {
          color: withAlpha(colors.outlineVariant, 0.35)
        }
      }
    },
    tooltip: {
      backgroundColor: withAlpha(colors.surface, 0.94),
      borderColor: withAlpha(colors.outline, 0.4),
      textStyle: {
        color: colors.onSurface
      }
    },
    dataZoom: [
      {
        type: 'slider',
        height: 24,
        bottom: 8,
        borderColor: colors.outline,
        backgroundColor: withAlpha(colors.onSurfaceVariant, 0.08),
        fillerColor: withAlpha(colors.primary, 0.22),
        handleStyle: {
          color: colors.primary
        },
        textStyle: {
          color: colors.onSurfaceVariant
        }
      },
      { type: 'inside' }
    ],
    grid: { left: 48, right: 24, top: 32, bottom: 48 },
    color: [colors.primary],
    series: [{
      name: seriesLabel.value,
      type: 'line',
      smooth: true,
      showSymbol: false,
      areaStyle: { color: withAlpha(colors.primary, 0.16) },
      lineStyle: { color: colors.primary, width: 2 },
      itemStyle: { color: colors.primary },
      emphasis: { focus: 'series' },
      data: chartSeriesData.value
    }]
  }, { notMerge: false, lazyUpdate: true })
}

const initChart = () => {
  if (!chartContainer.value) {
    return
  }
  themeColors.value = readThemeColors()
  chartInstance = echarts.init(chartContainer.value)
  chartInstance.setOption({
    animationDurationUpdate: 300,
    animationEasingUpdate: 'linear',
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'cross' },
      className: 'chart-tooltip-wrapper',
      formatter: formatTooltip
    }
  })
}

const teardownChart = () => {
  if (resizeObserver && chartContainer.value) {
    resizeObserver.unobserve(chartContainer.value)
    resizeObserver = null
  }
  if (chartInstance) {
    chartInstance.dispose()
    chartInstance = null
  }
}

onMounted(() => {
  initChart()
  updateChart()

  if (chartContainer.value && 'ResizeObserver' in window) {
    resizeObserver = new ResizeObserver(() => {
      chartInstance?.resize()
    })
    resizeObserver.observe(chartContainer.value)
  } else {
    window.addEventListener('resize', resizeChart)
  }

  // Note: startPolling handles scheduling internally
  if (isPolling.value) {
    startPolling()
  }

  if (typeof MutationObserver !== 'undefined') {
    themeObserver = new MutationObserver((mutations) => {
      if (mutations.some(mutation => mutation.type === 'attributes')) {
        themeColors.value = readThemeColors()
      }
    })
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['data-color-theme', 'class']
    })
  }
})

onActivated(() => {
  nextTick(() => {
    chartInstance?.resize()
    updateChart()
  })
})

onBeforeUnmount(() => {
  pausePolling()
  teardownChart()
  window.removeEventListener('resize', resizeChart)
  if (themeObserver) {
    themeObserver.disconnect()
    themeObserver = null
  }
})

const resizeChart = () => {
  chartInstance?.resize()
}

defineExpose({ exportSamples })
</script>

<template>
  <div class="chart-tab">
    <header class="chart-header">
      <div class="chart-header__info">
        <h3 class="chart-header__title">{{ chartTitle }}</h3>
        <div class="chart-header__oids">
          <span v-if="numericOid" class="chart-header__oid">{{ numericOid }}</span>
          <span
            v-if="baseOid && baseOid !== numericOid"
            class="chart-header__oid chart-header__oid--base"
          >
            Base: {{ baseOid }}
          </span>
        </div>
      </div>
      <div class="chart-header__meta">
        <span v-if="hostLabel" class="chart-header__host">Host: {{ hostLabel }}</span>
      </div>
    </header>

    <section class="chart-controls">
      <div class="chart-controls__left">
        <md-outlined-text-field
          label="Interval (s)"
          type="number"
          min="1"
          step="1"
          :value="intervalInput"
          @input="handleIntervalInput"
          @change="commitIntervalInput"
          @blur="commitIntervalInput"
          @keydown.enter.prevent="commitIntervalInput"
        >
          <span class="material-symbols-outlined" slot="leading-icon">schedule</span>
        </md-outlined-text-field>

        <md-filled-button
          class="chart-controls__action"
          @click="togglePolling"
        >
          <span class="material-symbols-outlined" slot="icon">
            {{ isPolling ? 'stop' : 'play_arrow' }}
          </span>
          {{ isPolling ? 'Stop' : 'Play' }}
        </md-filled-button>
      </div>

      <div class="chart-controls__right">
        <div class="chart-controls__toggles">
          <label class="chart-toggle">
            <md-switch
              :selected="useLogScale"
              @change="handleLogScaleToggle"
            ></md-switch>
            <span>Log scale</span>
          </label>
          <label class="chart-toggle">
            <md-switch
              :selected="useDifference"
              :disabled="useDerivative"
              @change="handleDifferenceToggle"
            ></md-switch>
            <span>Difference</span>
          </label>
          <label class="chart-toggle">
            <md-switch
              :selected="useDerivative"
              :disabled="useDifference"
              @change="handleDerivativeToggle"
            ></md-switch>
            <span>Derivative</span>
          </label>
          <label class="chart-toggle">
            <md-switch
              :selected="enforceNonNegative"
              :disabled="!(useDifference || useDerivative)"
              @change="handleNonNegativeToggle"
            ></md-switch>
            <span>Clamp non-negative</span>
          </label>
        </div>
        <md-icon-button
          class="chart-controls__export"
          @click="exportSamples"
          :disabled="!hasSamples"
          title="Export to CSV"
        >
          <span class="material-symbols-outlined">download</span>
        </md-icon-button>
      </div>
    </section>

    <section class="chart-display">
      <div ref="chartContainer" class="chart-canvas"></div>
      <div v-if="!hasSamples && !isPolling && !lastError" class="chart-placeholder">
        Press play to start collecting data and populate the chart.
      </div>
    </section>

    <footer class="chart-footer">
      <div class="chart-status">
        <span class="status-indicator" :class="{ 'status-indicator--active': isPolling }"></span>
        <span>
          {{ isPolling ? 'Polling running' : 'Polling stopped' }}
          <span v-if="isFetching" class="chart-status__fetching">(request in progress...)</span>
        </span>
        <span v-if="lastSample" class="chart-status__last-sample">
          Latest sample: {{ lastSample.label }} — {{ lastSample.displayValue ?? '—' }}
        </span>
      </div>
      <div v-if="lastError && lastError.message" class="chart-error">
        <span class="material-symbols-outlined">error</span>
        <span>{{ lastError.message }}</span>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.chart-tab {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: var(--spacing-md, 16px);
  padding: var(--spacing-md, 16px);
  background-color: var(--md-sys-color-surface-container);
  color: var(--md-sys-color-on-surface);
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--spacing-lg, 24px);
  flex-wrap: wrap;
}

.chart-header__info {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.chart-header__title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.chart-header__oids {
  display: flex;
  gap: var(--spacing-sm, 8px);
  flex-wrap: wrap;
  font-size: 12px;
  color: var(--md-sys-color-on-surface-variant);
}

.chart-header__oid {
  padding: 2px 8px;
  border-radius: var(--border-radius-sm);
  background-color: var(--md-sys-color-surface-container-highest);
  font-family: 'Courier New', monospace;
}

.chart-header__oid--base {
  background-color: var(--md-sys-color-surface-container);
}

.chart-header__meta {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm, 8px);
  color: var(--md-sys-color-on-surface-variant);
  font-size: 12px;
}

.chart-header__host {
  padding: 2px 10px;
  border-radius: var(--border-radius-sm);
  background-color: var(--md-sys-color-surface-container-high);
}

.chart-controls {
  display: flex;
  justify-content: space-between;
  gap: var(--spacing-lg, 24px);
  flex-wrap: wrap;
}

.chart-controls__left {
  display: flex;
  align-items: center;
  gap: var(--spacing-md, 16px);
  flex-wrap: wrap;
}

.chart-controls__right {
  display: flex;
  align-items: center;
  gap: var(--spacing-md, 16px);
  flex-wrap: wrap;
  justify-content: flex-end;
}

.chart-controls__action {
  --md-filled-button-container-shape: 999px;
}

.chart-controls__toggles {
  display: flex;
  gap: var(--spacing-lg, 24px);
  align-items: center;
  flex-wrap: wrap;
}

.chart-controls__export {
  --md-icon-button-icon-size: 22px;
}

.chart-toggle {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm, 8px);
  color: var(--md-sys-color-on-surface-variant);
}

.chart-display {
  position: relative;
  flex: 1;
  min-height: 280px;
  background-color: var(--md-sys-color-surface-container-high);
  border-radius: 16px;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.12);
  overflow: hidden;
}

.chart-canvas {
  width: 100%;
  height: 100%;
}

.chart-placeholder {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  text-align: center;
  color: var(--md-sys-color-on-surface-variant);
  background: repeating-linear-gradient(
    -45deg,
    transparent,
    transparent 20px,
    rgba(255, 255, 255, 0.04) 20px,
    rgba(255, 255, 255, 0.04) 40px
  );
}

.chart-footer {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm, 8px);
}

.chart-status {
  display: flex;
  align-items: center;
  gap: var(--spacing-md, 16px);
  flex-wrap: wrap;
  font-size: 14px;
  color: var(--md-sys-color-on-surface-variant);
}

.status-indicator {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background-color: var(--md-sys-color-outline);
  transition: background-color 0.2s ease;
}

.status-indicator--active {
  background-color: var(--md-sys-color-primary);
  box-shadow: 0 0 0 4px rgba(124, 77, 255, 0.14);
}

.chart-status__fetching {
  font-style: italic;
}

.chart-status__last-sample {
  color: var(--md-sys-color-on-surface);
}

.chart-error {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm, 8px);
  padding: 12px 16px;
  border-radius: 12px;
  background-color: rgba(179, 38, 30, 0.12);
  color: var(--md-sys-color-error);
  font-size: 14px;
}

.chart-tooltip {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.chart-tooltip__title {
  font-weight: 600;
  color: inherit;
}

.chart-tooltip__row {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  font-size: 13px;
  color: inherit;
}

.chart-tooltip__label {
  font-weight: 500;
  margin-right: 12px;
  color: inherit;
  opacity: 0.82;
}

.chart-tooltip__value {
  font-variant-numeric: tabular-nums;
  color: inherit;
}

@media (max-width: 768px) {
  .chart-controls {
    flex-direction: column;
    align-items: stretch;
  }

  .chart-controls__left,
  .chart-controls__right {
    width: 100%;
    justify-content: space-between;
  }

  .chart-controls__toggles {
    width: 100%;
    justify-content: space-between;
  }
}
</style>
