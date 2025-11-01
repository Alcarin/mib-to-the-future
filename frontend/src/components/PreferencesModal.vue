<script setup>
/**
 * @vue-component
 * @description A modal dialog for managing user preferences, such as theme and color scheme.
 *
 * @vue-prop {Boolean} show - Controls the visibility of the modal.
 *
 * @vue-event {boolean} update:show - Emitted to update the visibility of the modal. The payload is a boolean.
 */
import { useTheme } from '../composables/useTheme.js'
import '@material/web/switch/switch.js'
import '@material/web/button/text-button.js'
import '@material/web/iconbutton/icon-button.js'

defineProps({
  show: Boolean
})

const emit = defineEmits(['update:show'])

const { theme, toggleTheme, colorTheme, setColorTheme } = useTheme()

const colorOptions = [
  { value: 'purple', label: 'Purple', color: '#6750A4' },
  { value: 'blue', label: 'Blue', color: '#0061a4' },
  { value: 'green', label: 'Green', color: '#006e2c' },
  { value: 'orange', label: 'Orange', color: '#B35900' },
  { value: 'red', label: 'Red', color: '#B3261E' }
]

/**
 * @function closeModal
 * @description Emette un evento per chiudere la modale.
 */
const closeModal = () => {
  emit('update:show', false)
}
</script>

<template>
  <div v-if="show" class="modal-overlay" @click.self="closeModal">
    <div class="modal-content">
      <div class="modal-header">
        <h2>Preferences</h2>
        <md-icon-button @click="closeModal" title="Close">
          <span class="material-symbols-outlined">close</span>
        </md-icon-button>
      </div>
      <div class="modal-body">
        <div class="preference-item">
          <label for="theme-select">Theme</label>
          <div class="theme-switcher">
            <span>Light</span>
            <md-switch
              :selected="theme === 'dark'"
              @change="toggleTheme"
            ></md-switch>
            <span>Dark</span>
          </div>
        </div>
        <div class="divider"></div>
        <div class="preference-item column">
          <label>Color Theme</label>
          <div class="color-selector">
            <label v-for="option in colorOptions" :key="option.value" class="color-option">
              <input
                type="radio"
                name="color-theme"
                :value="option.value"
                :checked="colorTheme === option.value"
                @change="setColorTheme(option.value)"
              />
              <span class="color-swatch" :style="{ backgroundColor: option.color }"></span>
              <span class="color-label">{{ option.label }}</span>
            </label>
          </div>
        </div>
      </div>
      <div class="modal-footer">
        <md-text-button @click="closeModal">Close</md-text-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
@import '../assets/styles/modal.css';

.preference-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-md) 0;
}
.preference-item.column {
  flex-direction: column;
  align-items: flex-start;
  gap: var(--spacing-md);
}
.theme-switcher {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.divider {
  height: 1px;
  background-color: var(--md-sys-color-outline-variant);
  margin: var(--spacing-xs) 0;
}

.color-selector {
  display: flex;
  gap: var(--spacing-md);
}

.color-option {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  cursor: pointer;
}

.color-option input[type="radio"] {
  display: none; /* Hide the default radio button */
}

.color-swatch {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: 2px solid var(--md-sys-color-outline);
  display: inline-block;
  transition: transform 0.2s ease;
}

.color-option input[type="radio"]:checked + .color-swatch {
  border-color: var(--md-sys-color-primary);
  transform: scale(1.2);
  box-shadow: 0 0 0 2px var(--md-sys-color-surface-container-high), 0 0 0 4px var(--md-sys-color-primary);
}

.color-label {
  font-size: 14px;
  color: var(--md-sys-color-on-surface);
}
</style>