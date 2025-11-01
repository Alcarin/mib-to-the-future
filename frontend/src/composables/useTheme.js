// frontend/src/composables/useTheme.js
import { ref, watchEffect } from 'vue'

const theme = ref('dark')
const colorTheme = ref('purple')

/**
 * A Vue composable for managing application theme and color scheme.
 * It initializes the theme from localStorage or system preference, and provides
 * functions to toggle the theme and set the color scheme.
 *
 * @returns {{
 *   theme: import('vue').Ref<string>,
 *   toggleTheme: () => void,
 *   colorTheme: import('vue').Ref<string>,
 *   setColorTheme: (newColor: string) => void
 * }}
 */
export function useTheme() {
  // Init light/dark mode
  const savedMode = localStorage.getItem('mib-theme-mode')
  const systemDark = window.matchMedia('(prefers-color-scheme: dark)').matches
  theme.value = savedMode ?? (systemDark ? 'dark' : 'light')

  // Init color theme
  const savedColor = localStorage.getItem('mib-theme-color')
  colorTheme.value = savedColor ?? 'purple'

  // Watch and apply data attributes to <html>
  watchEffect(() => {
    // light/dark mode
    document.documentElement.dataset.theme = theme.value
    localStorage.setItem('mib-theme-mode', theme.value)

    // color theme
    document.documentElement.dataset.colorTheme = colorTheme.value
    localStorage.setItem('mib-theme-color', colorTheme.value)
  })

  function toggleTheme() {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
  }

  function setColorTheme(newColor) {
    colorTheme.value = newColor
  }

  return { theme, toggleTheme, colorTheme, setColorTheme }
}
