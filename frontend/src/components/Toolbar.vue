<script setup>
/**
 * @vue-component
 * @description A toolbar component with navigation buttons and a theme switcher.
 *
 * @vue-prop {String} tabAttiva - The key of the currently active tab.
 *
 * @vue-event {string} cambia-tab - Emitted when a tab is clicked. The payload is the key of the tab to switch to.
 */
import { useTheme } from '../composables/useTheme.js'
import '@material/web/button/filled-tonal-button.js'
import '@material/web/button/text-button.js'

defineProps({ tabAttiva: String })
defineEmits(['cambia-tab'])
const { theme, toggleTheme } = useTheme()
</script>

<template>
  <header class="toolbar">
    <div class="brand">
      <span class="material-symbols-outlined logo">rocket_launch</span>
      <span class="title">MIB to the Future</span>
    </div>
    <nav class="toolbar-actions">
      <md-filled-tonal-button v-if="tabAttiva === 'browser'" @click="$emit('cambia-tab', 'browser')">
        <span slot="icon" class="material-symbols-outlined">folder_tree</span>
        Browser MIB
      </md-filled-tonal-button>
      <md-text-button v-else @click="$emit('cambia-tab', 'browser')">
        <span slot="icon" class="material-symbols-outlined">folder_tree</span>
        Browser MIB
      </md-text-button>

      <md-filled-tonal-button v-if="tabAttiva === 'query'" @click="$emit('cambia-tab', 'query')">
        <span slot="icon" class="material-symbols-outlined">lan</span>
        Query SNMP
      </md-filled-tonal-button>
      <md-text-button v-else @click="$emit('cambia-tab', 'query')">
        <span slot="icon" class="material-symbols-outlined">lan</span>
        Query SNMP
      </md-text-button>

      <md-filled-tonal-button v-if="tabAttiva === 'log'" @click="$emit('cambia-tab', 'log')">
        <span slot="icon" class="material-symbols-outlined">list_alt</span>
        Log
      </md-filled-tonal-button>
      <md-text-button v-else @click="$emit('cambia-tab', 'log')">
        <span slot="icon" class="material-symbols-outlined">list_alt</span>
        Log
      </md-text-button>

      <span style="flex-grow: 1;"></span>

      <md-text-button @click="toggleTheme">
        <span slot="icon" class="material-symbols-outlined">brightness_6</span>
        {{ theme === 'dark' ? 'Dark' : 'Light' }}
      </md-text-button>
    </nav>
  </header>
</template>
