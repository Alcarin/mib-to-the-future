<script setup>
/**
 * @vue-component
 * @description A modal dialog for configuring SNMP host settings.
 *
 * @vue-prop {Object} host - The host configuration object.
 * @vue-prop {Boolean} show - Controls the visibility of the modal.
 * @vue-prop {Array} snmpVersions - An array of available SNMP versions.
 *
 * @vue-event {Object} update:host - Emitted when the host configuration is updated. The payload is the updated host object.
 * @vue-event {boolean} update:show - Emitted to update the visibility of the modal. The payload is a boolean.
 */
import { computed } from 'vue'
import '@material/web/button/filled-button.js'
import '@material/web/button/text-button.js'
import '@material/web/iconbutton/icon-button.js'
import '@material/web/textfield/outlined-text-field.js'
import '@material/web/select/outlined-select.js'

const props = defineProps({
  host: Object,
  show: Boolean,
  snmpVersions: Array
})

const emit = defineEmits(['update:host', 'update:show'])

const hostModel = computed(() => props.host ?? {})

const isV3 = computed(() => hostModel.value?.version === 'v3')

const securityLevels = [
  { value: 'noAuthNoPriv', label: 'NoAuthNoPriv' },
  { value: 'authNoPriv', label: 'AuthNoPriv' },
  { value: 'authPriv', label: 'AuthPriv' }
]

const authProtocols = [
  { value: 'MD5', label: 'MD5' },
  { value: 'SHA', label: 'SHA' },
  { value: 'SHA224', label: 'SHA-224' },
  { value: 'SHA256', label: 'SHA-256' },
  { value: 'SHA384', label: 'SHA-384' },
  { value: 'SHA512', label: 'SHA-512' }
]

const privProtocols = [
  { value: 'DES', label: 'DES' },
  { value: 'AES', label: 'AES' },
  { value: 'AES192', label: 'AES-192' },
  { value: 'AES192C', label: 'AES-192-C' },
  { value: 'AES256', label: 'AES-256' },
  { value: 'AES256C', label: 'AES-256-C' }
]

/**
 * @function updateHostField
 * @description Emette un evento con la configurazione host aggiornata.
 * @param {string} field - Il campo della configurazione da aggiornare.
 * @param {unknown} value - Il nuovo valore per il campo.
 */
function updateHostField(field, value) {
  emit('update:host', { ...(props.host ?? {}), [field]: value })
}

/**
 * @function closeModal
 * @description Emette un evento per chiudere la modale.
 */
function closeModal() {
  emit('update:show', false)
}
</script>

<template>
  <div v-if="show" class="modal-overlay" @click.self="closeModal">
    <div class="modal-content">
      <div class="modal-header">
        <h2>Host Settings</h2>
        <md-icon-button @click="closeModal" title="Close">
          <span class="material-symbols-outlined">close</span>
        </md-icon-button>
      </div>
      <div class="modal-body">
        <!-- SNMP Configuration -->
        <div class="control-group">
          <md-outlined-text-field
            label="Port"
            type="number"
            class="narrow"
            :value="hostModel?.port ?? ''"
            @input="(e) => updateHostField('port', Number(e.target.value))"
          ></md-outlined-text-field>

          <md-outlined-select
            label="Version"
            :value="hostModel?.version ?? ''"
            @change="(e) => updateHostField('version', e.target.value)"
          >
            <md-select-option v-for="ver in snmpVersions" :key="ver.value" :value="ver.value">
              {{ ver.label }}
            </md-select-option>
          </md-outlined-select>

          <template v-if="!isV3">
            <md-outlined-text-field
              label="Community"
              class="flex-grow"
              :value="hostModel?.community ?? ''"
              @input="(e) => updateHostField('community', e.target.value)"
            ></md-outlined-text-field>

            <md-outlined-text-field
              label="Write Community"
              class="flex-grow"
              :value="hostModel?.writeCommunity ?? ''"
              @input="(e) => updateHostField('writeCommunity', e.target.value)"
            ></md-outlined-text-field>
          </template>
        </div>

        <!-- SNMPv3 Configuration -->
        <div v-if="isV3" class="v3-controls">
          <div class="control-group">
            <md-outlined-select
              label="Security Level"
              class="flex-grow"
              :value="hostModel?.securityLevel ?? ''"
              @change="(e) => updateHostField('securityLevel', e.target.value)"
            >
              <md-select-option v-for="level in securityLevels" :key="level.value" :value="level.value">
                {{ level.label }}
              </md-select-option>
            </md-outlined-select>

            <md-outlined-text-field
              label="Security Username"
              class="flex-grow"
              :value="hostModel?.securityUsername ?? ''"
              @input="(e) => updateHostField('securityUsername', e.target.value)"
            ></md-outlined-text-field>
          </div>

          <div class="control-group">
            <md-outlined-select
              label="Auth Protocol"
              class="flex-grow"
              :value="hostModel?.authProtocol ?? ''"
              @change="(e) => updateHostField('authProtocol', e.target.value)"
            >
              <md-select-option v-for="proto in authProtocols" :key="proto.value" :value="proto.value">
                {{ proto.label }}
              </md-select-option>
            </md-outlined-select>

            <md-outlined-text-field
              label="Auth Password"
              type="password"
              class="flex-grow"
              :value="hostModel?.authPassword ?? ''"
              @input="(e) => updateHostField('authPassword', e.target.value)"
            ></md-outlined-text-field>
          </div>

          <div class="control-group">
            <md-outlined-select
              label="Privacy Protocol"
              class="flex-grow"
              :value="hostModel?.privProtocol ?? ''"
              @change="(e) => updateHostField('privProtocol', e.target.value)"
            >
              <md-select-option v-for="proto in privProtocols" :key="proto.value" :value="proto.value">
                {{ proto.label }}
              </md-select-option>
            </md-outlined-select>

            <md-outlined-text-field
              label="Privacy Password"
              type="password"
              class="flex-grow"
              :value="hostModel?.privPassword ?? ''"
              @input="(e) => updateHostField('privPassword', e.target.value)"
            ></md-outlined-text-field>
          </div>

          <md-outlined-text-field
            label="Context Name"
            :value="hostModel?.contextName ?? ''"
            @input="(e) => updateHostField('contextName', e.target.value)"
          ></md-outlined-text-field>
        </div>
      </div>
      <div class="modal-footer">
        <md-text-button @click="closeModal">Cancel</md-text-button>
        <md-filled-button @click="closeModal">Done</md-filled-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
@import '../assets/styles/modal.css';

.modal-content {
  max-width: 600px;
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

.control-group {
  display: flex;
  gap: var(--spacing-md);
  align-items: center;
  margin-bottom: var(--spacing-md);
}

.v3-controls {
  margin-top: var(--spacing-lg);
  border-top: 1px solid var(--md-sys-color-outline-variant);
  padding-top: var(--spacing-lg);
}

.narrow {
  width: 100px;
}

.flex-grow {
  flex: 1;
}
</style>
