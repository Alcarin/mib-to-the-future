/**
 * Utility functions for chart color manipulation and theme integration
 */

/**
 * Reads a CSS custom property value from the document root
 * @param {string} name - CSS variable name (e.g., '--md-sys-color-primary')
 * @param {string} fallback - Fallback value if variable is not found
 * @returns {string} The CSS variable value or fallback
 */
export function readCssVar(name, fallback) {
  if (typeof window === 'undefined') {
    return fallback
  }
  const styles = getComputedStyle(document.documentElement)
  const value = styles.getPropertyValue(name)
  if (!value || !value.trim()) {
    return fallback
  }
  return value.trim()
}

/**
 * Converts hex color to RGB object
 * @param {string} hex - Hex color (e.g., '#6750A4' or '#abc')
 * @returns {{r: number, g: number, b: number}|null} RGB object or null if invalid
 */
export function hexToRgb(hex) {
  const normalized = hex.replace('#', '')
  if (normalized.length === 3) {
    const r = parseInt(normalized[0] + normalized[0], 16)
    const g = parseInt(normalized[1] + normalized[1], 16)
    const b = parseInt(normalized[2] + normalized[2], 16)
    return { r, g, b }
  }
  if (normalized.length === 6) {
    const r = parseInt(normalized.slice(0, 2), 16)
    const g = parseInt(normalized.slice(2, 4), 16)
    const b = parseInt(normalized.slice(4, 6), 16)
    return { r, g, b }
  }
  return null
}

/**
 * Parses RGB/RGBA string to RGB object
 * @param {string} input - RGB string (e.g., 'rgb(100, 50, 200)')
 * @returns {{r: number, g: number, b: number}|null} RGB object or null if invalid
 */
export function rgbStringToRgb(input) {
  const match = input.match(/rgba?\(([^)]+)\)/)
  if (!match) {
    return null
  }
  const parts = match[1].split(',').map(part => Number(part.trim()))
  if (parts.length < 3) {
    return null
  }
  const [r, g, b] = parts
  return { r, g, b }
}

/**
 * Adds alpha channel to any color format
 * @param {string} color - Color in hex, rgb, or rgba format
 * @param {number} alpha - Alpha value (0-1)
 * @returns {string} RGBA color string
 */
export function withAlpha(color, alpha) {
  if (!color) {
    return `rgba(0, 0, 0, ${alpha})`
  }
  if (color.startsWith('rgba')) {
    const match = color.match(/rgba?\(([^)]+)\)/)
    if (match) {
      const [r, g, b] = match[1].split(',').map(Number)
      return `rgba(${r}, ${g}, ${b}, ${alpha})`
    }
  }
  if (color.startsWith('rgb')) {
    const rgb = rgbStringToRgb(color)
    if (rgb) {
      return `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, ${alpha})`
    }
  }
  if (color.startsWith('#')) {
    const rgb = hexToRgb(color)
    if (rgb) {
      return `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, ${alpha})`
    }
  }
  return color
}

/**
 * Reads Material Design 3 theme colors from CSS variables
 * @returns {Object} Object with theme color properties
 */
export function readThemeColors() {
  return {
    primary: readCssVar('--md-sys-color-primary', '#6750A4'),
    surface: readCssVar('--md-sys-color-surface', '#FFFBFE'),
    surfaceContainer: readCssVar('--md-sys-color-surface-container', '#F3EDF7'),
    onSurface: readCssVar('--md-sys-color-on-surface', '#1C1B1F'),
    onSurfaceVariant: readCssVar('--md-sys-color-on-surface-variant', '#49454F'),
    outline: readCssVar('--md-sys-color-outline', '#79747E'),
    outlineVariant: readCssVar('--md-sys-color-outline-variant', '#CAC4D0')
  }
}
