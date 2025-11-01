<script setup>
/**
 * @vue-component
 * @description Modal dialog to perform SNMP SET operations on a writable MIB node.
 * It provides a dynamic form based on the node syntax (numeric, enum, string, OID, IP, bits).
 *
 * @vue-prop {boolean} show - Controls the visibility of the modal.
 * @vue-prop {object|null} node - The currently selected MIB node (must include name, oid, syntax, access).
 * @vue-prop {object|null} currentValue - Result of the pre-fetch SNMP GET (may be null when unavailable).
 * @vue-prop {boolean} loading - Indicates if the pre-fetch GET is in progress.
 * @vue-prop {string} loadError - Optional error message from the pre-fetch GET.
 *
 * @vue-event {void} cancel - Emitted when the user cancels the dialog.
 * @vue-event {object} confirm - Emitted with the payload to send to SNMP SET (valueType, value, displayValue, encoding?).
 */
import { computed, ref, watch } from 'vue'
import '@material/web/button/filled-button.js'
import '@material/web/button/text-button.js'
import '@material/web/iconbutton/icon-button.js'
import '@material/web/textfield/outlined-text-field.js'
import '@material/web/select/outlined-select.js'
import '@material/web/select/select-option.js'
import '@material/web/progress/circular-progress.js'
import '@material/web/checkbox/checkbox.js'

const props = defineProps({
  show: {
    type: Boolean,
    default: false
  },
  node: {
    type: Object,
    default: () => null
  },
  currentValue: {
    type: Object,
    default: () => null
  },
  loading: {
    type: Boolean,
    default: false
  },
  loadError: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['cancel', 'confirm'])

const createEmptyMetadata = () => ({
  raw: '',
  baseType: '',
  baseTypeUpper: '',
  inputKind: 'text',
  valueType: 'string',
  ranges: [],
  size: null,
  enums: [],
  bits: [],
  isEnum: false,
  isBits: false
})

function parseSyntaxMetadata(syntax) {
  if (!syntax || typeof syntax !== 'string') {
    return createEmptyMetadata()
  }

  const raw = syntax.trim()
  if (!raw) {
    const metadata = createEmptyMetadata()
    metadata.raw = raw
    return metadata
  }

  const metadata = createEmptyMetadata()
  metadata.raw = raw
  const normalized = raw.replace(/\s+/g, ' ').trim()

  const baseMatch = normalized.match(/^[A-Za-z0-9-]+(?:\s+[A-Za-z0-9-]+)?/)
  const baseType = baseMatch ? baseMatch[0] : ''
  const baseTypeUpper = baseType.toUpperCase()

  metadata.baseType = baseType
  metadata.baseTypeUpper = baseTypeUpper

  const enumSectionMatch = normalized.match(/\{([^}]*)\}/)
  if (enumSectionMatch) {
    const entries = enumSectionMatch[1].split(',').map(item => item.trim()).filter(Boolean)
    for (const entry of entries) {
      const enumMatch = entry.match(/^([A-Za-z0-9_-]+)\s*\(\s*(-?\d+)\s*\)$/)
      if (enumMatch) {
        metadata.enums.push({
          name: enumMatch[1],
          value: Number.parseInt(enumMatch[2], 10)
        })
      }
    }
  }

  const rangeMatches = [...normalized.matchAll(/(-?\d+)\s*\.\.\s*(-?\d+)/g)]
  if (rangeMatches.length > 0) {
    metadata.ranges = rangeMatches.map(match => ({
      min: Number.parseInt(match[1], 10),
      max: Number.parseInt(match[2], 10)
    }))
  }

  const sizeMatch = normalized.match(/SIZE\s*\(\s*(\d+)\s*\.\.\s*(\d+)\s*\)/i)
  if (sizeMatch) {
    metadata.size = {
      min: Number.parseInt(sizeMatch[1], 10),
      max: Number.parseInt(sizeMatch[2], 10)
    }
  }

  metadata.isBits = baseTypeUpper.includes('BITS') || baseTypeUpper.includes('BIT STRING')
  metadata.isEnum = baseTypeUpper.includes('ENUMERATED') || (!metadata.isBits && metadata.enums.length > 0)

  if (metadata.isBits && metadata.enums.length) {
    metadata.bits = metadata.enums.map(({ name, value }) => ({ name, value }))
    metadata.enums = []
  }

  const inputKind = detectInputKind(metadata)
  metadata.inputKind = inputKind
  metadata.valueType = mapValueType(metadata, inputKind)

  if (!metadata.size && inputKind === 'text') {
    metadata.size = { min: 0, max: 65535 }
  }

  return metadata
}

function detectInputKind(metadata) {
  const upper = metadata.baseTypeUpper
  if (metadata.isBits) {
    return 'bits'
  }
  if (metadata.isEnum) {
    return 'enum'
  }
  if (upper.includes('IPADDRESS')) {
    return 'ip'
  }
  if (upper.includes('OBJECT IDENTIFIER')) {
    return 'oid'
  }
  if (
    upper.includes('OCTET STRING') ||
    upper.includes('DISPLAYSTRING') ||
    (upper.includes('STRING') && !upper.includes('BIT STRING'))
  ) {
    return 'text'
  }
  if (
    upper.includes('INTEGER') ||
    upper.includes('UNSIGNED') ||
    upper.includes('UINTEGER') ||
    upper.includes('COUNTER') ||
    upper.includes('GAUGE') ||
    upper.includes('TIMETICKS')
  ) {
    return 'number'
  }
  if (upper.includes('OPAQUE')) {
    return 'text'
  }
  return 'text'
}

function mapValueType(metadata, inputKind) {
  const upper = metadata.baseTypeUpper
  if (metadata.isBits) return 'bits'
  if (inputKind === 'enum') return 'integer'

  if (upper.includes('COUNTER64')) return 'counter64'
  if (upper.includes('COUNTER32')) return 'counter32'
  if (upper.includes('GAUGE32')) return 'gauge32'
  if (upper.includes('UNSIGNED32') || upper.includes('UINTEGER32')) return 'unsigned32'
  if (upper.includes('TIMETICKS')) return 'timeticks'
  if (upper.includes('IPADDRESS')) return 'ipaddress'
  if (upper.includes('OBJECT IDENTIFIER')) return 'objectIdentifier'
  if (upper.includes('OPAQUE')) return 'opaque'
  if (upper.includes('INTEGER')) return 'integer'

  if (inputKind === 'number') {
    return 'integer'
  }

  return 'string'
}

function deriveInitialFormValue(metadata, currentValue) {
  if (!currentValue) {
    if (metadata.inputKind === 'enum') {
      return metadata.enums.length > 0 ? String(metadata.enums[0].value) : ''
    }
    return ''
  }

  switch (metadata.inputKind) {
    case 'enum': {
      const raw = currentValue.rawValue ?? currentValue.value ?? ''
      const parsed = Number.parseInt(raw, 10)
      if (!Number.isNaN(parsed) && metadata.enums.some(option => option.value === parsed)) {
        return String(parsed)
      }
      return metadata.enums.length > 0 ? String(metadata.enums[0].value) : ''
    }
    case 'number': {
      const raw = currentValue.rawValue ?? currentValue.value ?? currentValue.displayValue
      const parsed = Number(raw)
      return Number.isFinite(parsed) ? String(parsed) : ''
    }
    case 'ip':
    case 'oid':
    case 'text':
    default: {
      const raw = currentValue.displayValue ?? currentValue.value ?? ''
      return raw == null ? '' : String(raw)
    }
  }
}

function deriveInitialBits(metadata, currentValue) {
  if (!currentValue || !metadata.bits.length) {
    return []
  }

  const raw = currentValue.rawValue ?? currentValue.value ?? ''
  if (typeof raw !== 'string') {
    return []
  }

  const bits = []
  for (const bit of metadata.bits) {
    if (raw.includes(bit.name) || raw.includes(`(${bit.value})`)) {
      bits.push(bit.value)
    }
  }
  return bits
}

function validateFormValue(metadata, inputValue) {
  const trimmed = typeof inputValue === 'string' ? inputValue.trim() : inputValue

  if (metadata.inputKind === 'number') {
    if (trimmed === '' || trimmed === null || trimmed === undefined) {
      return { valid: false, message: 'Value is required.' }
    }
    const parsed = Number(trimmed)
    if (!Number.isFinite(parsed)) {
      return { valid: false, message: 'Enter a valid number.' }
    }
    if (metadata.ranges.length) {
      const withinRange = metadata.ranges.some(({ min, max }) => parsed >= min && parsed <= max)
      if (!withinRange) {
        return { valid: false, message: 'Value is outside the allowed range.' }
      }
    }
    return { valid: true, message: '' }
  }

  if (metadata.inputKind === 'enum') {
    if (trimmed === '' || trimmed === null || trimmed === undefined) {
      return { valid: false, message: 'Select an option.' }
    }
    return { valid: true, message: '' }
  }

  if (metadata.inputKind === 'ip') {
    if (trimmed === '') {
      return { valid: false, message: 'IPv4 address is required.' }
    }
    const ipv4Pattern = /^(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}$/
    if (!ipv4Pattern.test(trimmed)) {
      return { valid: false, message: 'Provide a valid IPv4 address.' }
    }
    return { valid: true, message: '' }
  }

  if (metadata.inputKind === 'oid') {
    if (trimmed === '') {
      return { valid: false, message: 'OID is required.' }
    }
    const oidPattern = /^(\d+)(\.\d+)*$/
    if (!oidPattern.test(trimmed)) {
      return { valid: false, message: 'Use dotted numeric notation (e.g., 1.3.6.1.2.1).' }
    }
    return { valid: true, message: '' }
  }

  if (metadata.inputKind === 'text' && metadata.size) {
    const length = String(inputValue ?? '').length
    if (length < metadata.size.min || length > metadata.size.max) {
      return { valid: false, message: `Length must be between ${metadata.size.min} and ${metadata.size.max}.` }
    }
  }

  return { valid: true, message: '' }
}

function buildPayload(metadata, inputValue) {
  const result = {
    valueType: metadata.valueType,
    value: null,
    displayValue: ''
  }

  switch (metadata.inputKind) {
    case 'enum': {
      const numericValue = Number.parseInt(inputValue, 10)
      const option = metadata.enums.find(item => item.value === numericValue)
      result.value = numericValue
      result.displayValue = option ? `${option.name} (${option.value})` : String(numericValue)
      break
    }
    case 'number': {
      const numericValue = Number(inputValue)
      result.value = numericValue
      result.displayValue = String(inputValue)
      break
    }
    case 'ip':
    case 'oid':
    case 'text': {
      result.value = String(inputValue ?? '')
      result.displayValue = result.value
      break
    }
    default: {
      result.value = String(inputValue ?? '')
      result.displayValue = result.value
      break
    }
  }

  return result
}

function buildBitsPayload(metadata, bitSet) {
  const sortedBits = Array.from(bitSet).sort((a, b) => a - b)
  const buffer = bitsToBytes(sortedBits)
  const labels = metadata.bits
    .filter(bit => sortedBits.includes(bit.value))
    .map(bit => bit.name)
  return {
    valueType: 'bits',
    value: Array.from(buffer),
    displayValue: labels.length ? labels.join(', ') : '0'
  }
}

function bitsToBytes(selectedBits) {
  if (!selectedBits.length) {
    return new Uint8Array(0)
  }
  const maxBit = Math.max(...selectedBits)
  const length = Math.floor(maxBit / 8) + 1
  const bytes = new Uint8Array(length)
  for (const bit of selectedBits) {
    const byteIndex = Math.floor(bit / 8)
    const bitIndex = bit % 8
    bytes[byteIndex] |= 1 << (7 - bitIndex)
  }
  return bytes
}

const syntaxMetadata = computed(() => parseSyntaxMetadata(props.node?.syntax))

const formValue = ref('')
const bitSelection = ref(new Set())
const touched = ref(false)

const nodeName = computed(() => props.node?.name || props.node?.oid || '')
const nodeOid = computed(() => props.node?.oid || '')
const nodeSyntax = computed(() => props.node?.syntax || 'N/A')
const nodeTypeLabel = computed(() => props.node?.type || 'N/A')

const currentDisplayValue = computed(() => props.currentValue?.displayValue ?? props.currentValue?.value ?? '')

watch(
  () => props.show,
  (visible) => {
    if (visible) {
      initializeForm()
      touched.value = false
    }
  }
)

watch(
  () => props.currentValue,
  () => {
    if (!props.show) {
      return
    }
    initializeForm()
  }
)

watch(
  () => syntaxMetadata.value,
  () => {
    if (!props.show) {
      return
    }
    initializeForm()
  }
)

function initializeForm() {
  const metadata = syntaxMetadata.value
  touched.value = false

  if (metadata.isBits) {
    const initialBits = deriveInitialBits(metadata, props.currentValue)
    bitSelection.value = new Set(initialBits)
    formValue.value = ''
    return
  }

  const initialValue = deriveInitialFormValue(metadata, props.currentValue)
  formValue.value = initialValue
}

const validationState = computed(() => {
  const metadata = syntaxMetadata.value
  if (metadata.isBits) {
    return { valid: true, message: '' }
  }
  return validateFormValue(metadata, formValue.value)
})

const canSubmit = computed(() => {
  if (!props.node) {
    return false
  }
  if (syntaxMetadata.value.isBits) {
    return true
  }
  return validationState.value.valid && formValue.value !== null
})

const dialogError = computed(() => {
  if (!touched.value) return ''
  return validationState.value.valid ? '' : validationState.value.message
})

const handleCancel = () => {
  emit('cancel')
}

const handleKeyDown = (event) => {
  if (event.key === 'Escape') {
    event.preventDefault()
    emit('cancel')
  } else if (event.key === 'Enter' && !event.shiftKey && syntaxMetadata.value.inputKind !== 'text') {
    event.preventDefault()
    submit()
  }
}

const toggleBit = (bit) => {
  const next = new Set(bitSelection.value)
  if (next.has(bit)) {
    next.delete(bit)
  } else {
    next.add(bit)
  }
  bitSelection.value = next
}

const submit = () => {
  touched.value = true
  if (!canSubmit.value) {
    return
  }

  const metadata = syntaxMetadata.value
  const payload = metadata.isBits
    ? buildBitsPayload(metadata, bitSelection.value)
    : buildPayload(metadata, formValue.value)

  emit('confirm', payload)
}
</script>

<template>
  <div v-if="show" class="modal-overlay" @click.self="handleCancel">
    <div class="modal-content" @keydown="handleKeyDown">
      <div class="modal-header">
        <h2>SNMP SET</h2>
        <md-icon-button @click="handleCancel" title="Close dialog">
          <span class="material-symbols-outlined">close</span>
        </md-icon-button>
      </div>

      <div class="modal-body">
        <section class="summary">
          <div class="summary-item">
            <span class="summary-label">Name</span>
            <span class="summary-value">{{ nodeName }}</span>
          </div>
          <div class="summary-item">
            <span class="summary-label">OID</span>
            <code class="summary-code">{{ nodeOid }}</code>
          </div>
          <div class="summary-item">
            <span class="summary-label">Type</span>
            <span class="summary-value">{{ nodeTypeLabel }}</span>
          </div>
          <div class="summary-item">
            <span class="summary-label">Syntax</span>
            <span class="summary-value">{{ nodeSyntax }}</span>
          </div>
        </section>

        <section class="current-value">
          <span class="summary-label">Current value</span>
          <div class="current-value__content">
            <md-circular-progress v-if="loading" indeterminate class="current-value__spinner"></md-circular-progress>
            <span v-else-if="props.loadError" class="current-value__error">{{ props.loadError }}</span>
            <span v-else-if="currentDisplayValue" class="current-value__value">{{ currentDisplayValue }}</span>
            <span v-else class="current-value__placeholder">No value available</span>
          </div>
        </section>

        <section class="form-section">
          <template v-if="syntaxMetadata.inputKind === 'enum'">
            <md-outlined-select
              label="New value"
              :value="String(formValue)"
              :error="Boolean(dialogError)"
              :error-text="dialogError"
              @change="formValue = $event.target.value"
            >
              <md-select-option
                v-for="option in syntaxMetadata.enums"
                :key="option.value"
                :value="String(option.value)"
              >
                {{ option.name }} ({{ option.value }})
              </md-select-option>
            </md-outlined-select>
          </template>

          <template v-else-if="syntaxMetadata.inputKind === 'number'">
            <md-outlined-text-field
              label="New value"
              type="number"
              :value="String(formValue)"
              :error="Boolean(dialogError)"
              :error-text="dialogError"
              @input="formValue = $event.target.value"
            ></md-outlined-text-field>
            <p v-if="syntaxMetadata.ranges.length" class="field-hint">
              Allowed range:
              <span v-for="(range, idx) in syntaxMetadata.ranges" :key="idx">
                {{ range.min }}..{{ range.max }}<span v-if="idx < syntaxMetadata.ranges.length - 1"> or </span>
              </span>
            </p>
          </template>

          <template v-else-if="syntaxMetadata.inputKind === 'ip'">
            <md-outlined-text-field
              label="IPv4 address"
              placeholder="192.168.0.1"
              :value="String(formValue)"
              :error="Boolean(dialogError)"
              :error-text="dialogError"
              @input="formValue = $event.target.value"
            ></md-outlined-text-field>
          </template>

          <template v-else-if="syntaxMetadata.inputKind === 'oid'">
            <md-outlined-text-field
              label="Object Identifier"
              placeholder="1.3.6.1.2.1.1.5.0"
              :value="String(formValue)"
              :error="Boolean(dialogError)"
              :error-text="dialogError"
              @input="formValue = $event.target.value"
            ></md-outlined-text-field>
            <p class="field-hint">Use dotted numeric notation, e.g., <code>1.3.6.1.2.1.1.5.0</code>.</p>
          </template>

          <template v-else-if="syntaxMetadata.inputKind === 'bits'">
            <div class="bits-list">
              <label
                v-for="bit in syntaxMetadata.bits"
                :key="bit.value"
                class="bits-option"
              >
                <md-checkbox
                  :checked="bitSelection.has(bit.value)"
                  @change="toggleBit(bit.value)"
                ></md-checkbox>
                <span class="bits-option__label">
                  {{ bit.name }} ({{ bit.value }})
                </span>
              </label>
            </div>
            <p class="field-hint">Selected flags will be set to 1; others will be cleared.</p>
          </template>

          <template v-else>
            <md-outlined-text-field
              label="New value"
              :value="String(formValue)"
              :error="Boolean(dialogError)"
              :error-text="dialogError"
              @input="formValue = $event.target.value"
            ></md-outlined-text-field>
            <p v-if="syntaxMetadata.size" class="field-hint">
              Length between {{ syntaxMetadata.size.min }} and {{ syntaxMetadata.size.max }} characters.
            </p>
          </template>
        </section>
      </div>

      <div class="modal-footer">
        <md-text-button @click="handleCancel">Cancel</md-text-button>
        <md-filled-button :disabled="!canSubmit" @click="submit">Apply</md-filled-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
@import '../assets/styles/modal.css';

.modal-content {
  max-width: 580px;
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
  gap: var(--spacing-lg);
}

.summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: var(--spacing-md);
  background-color: var(--md-sys-color-surface-container-high);
  padding: var(--spacing-md);
  border-radius: 16px;
}

.summary-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.summary-label {
  font-size: 12px;
  text-transform: uppercase;
  color: var(--md-sys-color-on-surface-variant);
  letter-spacing: 0.08em;
}

.summary-value {
  font-size: 14px;
  color: var(--md-sys-color-on-surface);
  word-break: break-word;
}

.summary-code {
  padding: 2px 8px;
  border-radius: 8px;
  background-color: var(--md-sys-color-surface-container-highest);
  font-family: var(--font-family-mono, 'JetBrains Mono', monospace);
  color: var(--md-sys-color-on-surface);
}

.current-value {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.current-value__content {
  min-height: 36px;
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: 8px 12px;
  border-radius: 12px;
  background-color: var(--md-sys-color-surface-container);
}

.current-value__spinner {
  --md-circular-progress-size: 28px;
}

.current-value__value {
  font-family: var(--font-family-mono, 'JetBrains Mono', monospace);
  font-size: 14px;
  color: var(--md-sys-color-on-surface);
  word-break: break-word;
}

.current-value__placeholder {
  font-size: 14px;
  color: var(--md-sys-color-on-surface-variant);
}

.current-value__error {
  font-size: 14px;
  color: var(--md-sys-color-error);
}

.form-section {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.field-hint {
  margin: 0;
  font-size: 13px;
  color: var(--md-sys-color-on-surface-variant);
}

.form-section code {
  font-family: var(--font-family-mono, 'JetBrains Mono', monospace);
}

.bits-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 180px;
  overflow-y: auto;
}

.bits-option {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 6px 4px;
  border-radius: 8px;
  background-color: var(--md-sys-color-surface-container-low);
}

.bits-option__label {
  font-size: 14px;
  color: var(--md-sys-color-on-surface);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--spacing-sm);
}
</style>
