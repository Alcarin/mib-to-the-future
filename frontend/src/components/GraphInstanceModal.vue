<script setup>
/**
 * @vue-component
 * @description Modal dialog requesting the instance identifier needed to poll a tabular OID column.
 *
 * @vue-prop {boolean} show - Controls visibility of the modal.
 * @vue-prop {object|null} node - The node metadata for which we are requesting an instance.
 *
 * @vue-event {void} cancel - Emitted when the user cancels the dialog.
 * @vue-event {string} confirm - Emitted with the selected instance identifier when the user confirms.
 */
import { ref, watch, computed } from 'vue'
import '@material/web/button/filled-button.js'
import '@material/web/button/text-button.js'
import '@material/web/iconbutton/icon-button.js'
import '@material/web/textfield/outlined-text-field.js'

const props = defineProps({
  show: {
    type: Boolean,
    default: false
  },
  node: {
    type: Object,
    default: () => null
  }
})

const emit = defineEmits(['cancel', 'confirm'])

const instanceId = ref('')
const errorMessage = ref('')

const nodeName = computed(() => props.node?.name || props.node?.oid || '')
const baseOid = computed(() => props.node?.oid || '')

watch(
  () => props.show,
  (visible) => {
    if (visible) {
      instanceId.value = ''
      errorMessage.value = ''
    }
  }
)

const close = () => {
  emit('cancel')
}

const handleConfirm = () => {
  const trimmed = instanceId.value.trim()
  if (!trimmed) {
    errorMessage.value = 'Instance ID is required.'
    return
  }
  errorMessage.value = ''
  emit('confirm', trimmed)
}

const handleKeyDown = (event) => {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    handleConfirm()
  } else if (event.key === 'Escape') {
    event.preventDefault()
    close()
  }
}
</script>

<template>
  <div v-if="show" class="modal-overlay" @click.self="close">
    <div class="modal-content" @keydown="handleKeyDown">
      <div class="modal-header">
        <h2>Select table instance</h2>
        <md-icon-button @click="close" title="Close dialog">
          <span class="material-symbols-outlined">close</span>
        </md-icon-button>
      </div>
      <div class="modal-body">
        <p class="modal-description">
          Choose the table instance to poll for <strong>{{ nodeName }}</strong>.
          Provide the numeric index (for example <code>2</code> or <code>2.10</code>) that identifies the row.
        </p>
        <md-outlined-text-field
          label="Instance ID"
          :value="instanceId"
          :error="Boolean(errorMessage)"
          :error-text="errorMessage"
          @input="instanceId = $event.target.value"
          @keyup.enter.stop.prevent="handleConfirm"
        >
          <span class="material-symbols-outlined" slot="leading-icon">data_object</span>
        </md-outlined-text-field>
        <p class="modal-hint">
          Base OID: <code>{{ baseOid }}</code>
        </p>
      </div>
      <div class="modal-footer">
        <md-text-button @click="close">Cancel</md-text-button>
        <md-filled-button @click="handleConfirm">Continue</md-filled-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
@import '../assets/styles/modal.css';

.modal-content {
  max-width: 520px;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.modal-header h2 {
  margin: 0;
  font-size: var(--md-sys-typescale-title-large-size);
}

.modal-body {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.modal-description {
  margin: 0;
  color: var(--md-sys-color-on-surface-variant);
  line-height: 1.4;
}

.modal-hint {
  margin: 0;
  font-size: 13px;
  color: var(--md-sys-color-on-surface-variant);
}

code {
  font-family: var(--font-family-mono, 'JetBrains Mono', monospace);
  font-size: 0.95em;
  padding: 2px 6px;
  border-radius: 6px;
  background-color: var(--md-sys-color-surface-container-highest);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--spacing-sm);
}
</style>
