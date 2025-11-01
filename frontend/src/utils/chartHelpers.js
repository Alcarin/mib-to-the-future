/**
 * Helper functions for chart data manipulation
 */

/**
 * Maximum number of data points to keep in the chart
 */
export const MAX_POINTS = 500

/**
 * Formats a timestamp for chart labels
 * @param {number} timestampMs - Timestamp in milliseconds
 * @returns {string} Formatted time string (HH:MM:SS)
 */
export function formatTimeLabel(timestampMs) {
  const date = new Date(timestampMs)
  const options = {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  }
  return date.toLocaleTimeString(undefined, options)
}

/**
 * Deep clones an array of sample objects
 * @param {Array} items - Array of sample objects
 * @returns {Array} Cloned array
 */
export function cloneSamples(items) {
  if (!Array.isArray(items)) {
    return []
  }
  return items.map(sample => {
    if (!sample || typeof sample !== 'object') {
      return sample
    }
    return { ...sample }
  })
}

/**
 * Parses a value to number, handling various formats
 * @param {string|number} val - Value to parse
 * @returns {number|null} Parsed number or null if invalid
 */
export function parseValue(val) {
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

/**
 * Calculates the difference between consecutive values
 * @param {Array} samples - Array of sample objects with 'value' property
 * @returns {Array} New array with difference values
 */
export function applyDifference(samples) {
  if (!Array.isArray(samples) || samples.length < 2) {
    return samples
  }

  const result = []
  for (let i = 1; i < samples.length; i++) {
    const prev = samples[i - 1]
    const curr = samples[i]

    const prevVal = parseValue(prev?.value)
    const currVal = parseValue(curr?.value)

    if (prevVal !== null && currVal !== null) {
      result.push({
        timestamp: curr.timestamp,
        value: currVal - prevVal,
        rawValue: curr.rawValue
      })
    }
  }

  return result
}

/**
 * Calculates the derivative (rate of change) between consecutive values
 * @param {Array} samples - Array of sample objects with 'timestamp' and 'value'
 * @returns {Array} New array with derivative values
 */
export function applyDerivative(samples) {
  if (!Array.isArray(samples) || samples.length < 2) {
    return samples
  }

  const result = []
  for (let i = 1; i < samples.length; i++) {
    const prev = samples[i - 1]
    const curr = samples[i]

    const prevVal = parseValue(prev?.value)
    const currVal = parseValue(curr?.value)

    if (prevVal !== null && currVal !== null) {
      const timeDelta = (curr.timestamp - prev.timestamp) / 1000 // seconds
      if (timeDelta > 0) {
        const derivative = (currVal - prevVal) / timeDelta
        result.push({
          timestamp: curr.timestamp,
          value: derivative,
          rawValue: curr.rawValue
        })
      }
    }
  }

  return result
}

/**
 * Enforces non-negative values by clamping to zero
 * @param {Array} samples - Array of sample objects
 * @returns {Array} New array with non-negative values
 */
export function enforceNonNegativeValues(samples) {
  if (!Array.isArray(samples)) {
    return samples
  }

  return samples.map(sample => {
    const val = parseValue(sample?.value)
    if (val === null) {
      return sample
    }
    return {
      ...sample,
      value: Math.max(0, val)
    }
  })
}

/**
 * Generates CSV content from chart samples
 * @param {Array} samples - Array of sample objects
 * @param {string} oid - OID being monitored
 * @returns {string} CSV formatted string
 */
export function generateCSV(samples, oid) {
  if (!Array.isArray(samples) || samples.length === 0) {
    return ''
  }

  const header = 'Timestamp,Value,Raw Value,OID\n'
  const rows = samples.map(sample => {
    const timestamp = new Date(sample.timestamp).toISOString()
    const value = sample.value ?? ''
    const rawValue = sample.rawValue ?? ''
    return `${timestamp},${value},${rawValue},${oid}`
  })

  return header + rows.join('\n')
}

/**
 * Limits the number of samples to MAX_POINTS
 * @param {Array} samples - Array of sample objects
 * @returns {Array} Trimmed array
 */
export function limitSamples(samples) {
  if (!Array.isArray(samples)) {
    return []
  }
  if (samples.length <= MAX_POINTS) {
    return samples
  }
  return samples.slice(samples.length - MAX_POINTS)
}
