/**
 * Composable for SNMP polling functionality in charts
 */
import { ref, onBeforeUnmount } from 'vue'
import { SNMPGet } from '../../wailsjs/go/app/App'
import { formatTimeLabel, limitSamples } from '../utils/chartHelpers'
import { useErrorHandler } from './useErrorHandler'

export function useChartPolling(props) {
  const { handleError, lastError } = useErrorHandler()

  // State
  const pollingInterval = ref(
    Number.isFinite(props.tabInfo?.chartState?.pollingInterval) && props.tabInfo.chartState.pollingInterval > 0
      ? props.tabInfo.chartState.pollingInterval
      : 5
  )
  const intervalInput = ref(String(pollingInterval.value))
  const isPolling = ref(Boolean(props.tabInfo?.chartState?.isPolling))
  const isFetching = ref(false)

  const samples = ref([])
  const lastRawValue = ref(null)
  const lastSampleTimestamp = ref(null)

  let pollingTimer = null

  // Helpers
  const toTimestamp = (ts) => {
    if (typeof ts === 'number') {
      return ts
    }
    if (typeof ts === 'string') {
      const parsed = new Date(ts)
      if (!isNaN(parsed.getTime())) {
        return parsed.getTime()
      }
    }
    return Date.now()
  }

  const toNumeric = (val) => {
    if (val === null || val === undefined || val === '') {
      return null
    }
    if (typeof val === 'number') {
      return isNaN(val) ? null : val
    }
    const str = String(val).trim()
    if (str === '') {
      return null
    }
    const num = Number(str)
    return isNaN(num) ? null : num
  }

  const buildSnmpConfig = () => {
    const hc = props.hostConfig || {}
    const rawAddress = hc.address ?? hc.host ?? ''
    const address = typeof rawAddress === 'string' ? rawAddress.trim() : ''
    if (!address) {
      return null
    }

    const version = typeof hc.version === 'string' && hc.version ? hc.version : 'v2c'
    const normalizedVersion = version.toLowerCase()

    const coerceNumber = (value, fallback) => {
      const numberValue = Number(value)
      return Number.isFinite(numberValue) && numberValue > 0 ? numberValue : fallback
    }

    const config = {
      host: address,
      port: coerceNumber(hc.port, 161),
      community: hc.community || 'public',
      version,
      timeout: coerceNumber(hc.timeout, 5),
      retries: coerceNumber(hc.retries, 1),
      contextName: hc.contextName ?? '',
      securityLevel: hc.securityLevel ?? '',
      securityUsername: hc.securityUsername ?? '',
      authProtocol: hc.authProtocol ?? '',
      authPassword: hc.authPassword ?? '',
      privProtocol: hc.privProtocol ?? '',
      privPassword: hc.privPassword ?? ''
    }

    if (normalizedVersion !== 'v3') {
      config.contextName = ''
      config.securityLevel = ''
      config.securityUsername = ''
      config.authProtocol = ''
      config.authPassword = ''
      config.privProtocol = ''
      config.privPassword = ''
    }

    return config
  }

  // Core polling functions
  const fetchSample = async (config) => {
    if (isFetching.value) {
      return
    }
    isFetching.value = true
    try {
      const result = await SNMPGet(config, props.tabInfo?.oid)
      if (!result) {
        throw new Error('No SNMP result received.')
      }
      appendSample(result)
    } catch (err) {
      handleError(err, 'SNMP request failed')
    } finally {
      isFetching.value = false
    }
  }

  const appendSample = (result) => {
    const timestampMs = toTimestamp(result?.timestamp)
    const numericRaw = toNumeric(result?.rawValue ?? result?.value)

    let difference = null
    let derivative = null
    if (numericRaw !== null && lastRawValue.value !== null && lastSampleTimestamp.value !== null) {
      difference = numericRaw - lastRawValue.value
      const deltaTime = (timestampMs - lastSampleTimestamp.value) / 1000
      if (deltaTime > 0 && difference !== null) {
        derivative = difference / deltaTime
      }
    }

    const newSample = {
      id: `sample-${timestampMs}-${samples.value.length}`,
      timestamp: timestampMs,
      label: formatTimeLabel(timestampMs),
      rawValue: numericRaw,
      difference,
      derivative,
      displayValue: result?.displayValue ?? result?.value ?? '',
      value: numericRaw
    }

    samples.value = limitSamples([...samples.value, newSample])
    lastRawValue.value = numericRaw
    lastSampleTimestamp.value = timestampMs
  }

  const pausePolling = () => {
    if (pollingTimer) {
      clearInterval(pollingTimer)
      pollingTimer = null
    }
  }

  const schedulePolling = () => {
    pausePolling()
    if (!isPolling.value) {
      return
    }
    const intervalMs = Math.max(1000, Math.round(Number(pollingInterval.value) * 1000))
    pollingTimer = window.setInterval(async () => {
      const config = buildSnmpConfig()
      if (!config) {
        handleError('Invalid SNMP configuration. Polling stopped.', 'Configuration Error')
        stopPolling()
        return
      }
      await fetchSample(config)
    }, intervalMs)
  }

  const commitIntervalInput = () => {
    const parsed = Number(intervalInput.value)
    if (Number.isFinite(parsed) && parsed >= 1) {
      pollingInterval.value = parsed
    } else {
      intervalInput.value = String(pollingInterval.value)
    }
  }

  const startPolling = async () => {
    const oid = props.tabInfo?.oid
    if (!oid) {
      handleError('Invalid OID provided for polling.', 'Configuration Error')
      return
    }

    commitIntervalInput()

    const intervalSeconds = Number(pollingInterval.value)
    if (!Number.isFinite(intervalSeconds) || intervalSeconds <= 0) {
      handleError('Set a valid polling interval (>= 1s).', 'Configuration Error')
      return
    }

    const config = buildSnmpConfig()
    if (!config) {
      handleError('Incomplete SNMP configuration. Check host and credentials.', 'Configuration Error')
      return
    }

    isPolling.value = true

    await fetchSample(config)
    schedulePolling()
  }

  const stopPolling = () => {
    pausePolling()
    isPolling.value = false
  }

  const clearData = () => {
    samples.value = []
    lastRawValue.value = null
    lastSampleTimestamp.value = null
  }

  // Cleanup
  onBeforeUnmount(() => {
    pausePolling()
  })

  return {
    // State
    pollingInterval,
    intervalInput,
    isPolling,
    isFetching,
    samples,
    lastRawValue,
    lastSampleTimestamp,
    lastError,

    // Methods
    startPolling,
    stopPolling,
    pausePolling,
    clearData,
    commitIntervalInput,
    setSamples: (newSamples) => { samples.value = newSamples }
  }
}
