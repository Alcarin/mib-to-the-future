<script setup>
import { ref, watch } from 'vue'
import '@material/web/iconbutton/icon-button.js'
import '@material/web/button/filled-button.js'
import '@material/web/button/outlined-button.js'
import '@material/web/textfield/outlined-text-field.js'

const props = defineProps({
  show: { type: Boolean, default: false },
  title: { type: String, default: 'New folder' },
  confirmLabel: { type: String, default: 'Create' },
  defaultName: { type: String, default: '' }
})

const emit = defineEmits(['update:show', 'confirm'])

const name = ref(props.defaultName)
const errorMessage = ref('')

watch(() => props.defaultName, (value) => {
  name.value = value || ''
  errorMessage.value = ''
})

watch(() => props.show, (visible) => {
  if (visible) {
    name.value = props.defaultName || ''
    errorMessage.value = ''
  }
})

const closeModal = () => {
  emit('update:show', false)
}

const confirm = () => {
  const trimmed = (name.value || '').trim()
  if (!trimmed) {
    errorMessage.value = 'Name is required'
    return
  }
  emit('confirm', trimmed)
}
</script>

<template>
  <div v-if="show" class="modal-overlay" @click.self="closeModal">
    <div class="modal-content">
      <div class="modal-header">
        <h2>{{ title }}</h2>
        <md-icon-button @click="closeModal" title="Close">
          <span class="material-symbols-outlined">close</span>
        </md-icon-button>
      </div>
      <div class="modal-body">
        <md-outlined-text-field
          label="Folder name"
          v-model="name"
          :error="Boolean(errorMessage)"
          :error-text="errorMessage"
          @keyup.enter="confirm"
          autofocus
        ></md-outlined-text-field>
      </div>
      <div class="modal-footer">
        <md-outlined-button @click="closeModal">Cancel</md-outlined-button>
        <md-filled-button @click="confirm">{{ confirmLabel }}</md-filled-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
@import '../assets/styles/modal.css';

.modal-body {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}
</style>
