<script setup>
/**
 * @vue-component
 * @description Un componente per le notifiche toast.
 *
 * @vue-prop {Object} notification - L'oggetto notifica.
 * @vue-prop {String} notification.message - Il messaggio da visualizzare.
 * @vue-prop {String} [notification.type=info] - Il tipo di notifica (info, success, error).
 *
 * @vue-event {void} close - Emesso quando la notifica deve essere chiusa.
 */
import { onMounted } from 'vue'

defineProps({
  notification: {
    type: Object,
    required: true,
  },
})

const emit = defineEmits(['close'])

onMounted(() => {
  setTimeout(() => emit('close'), 5000)
})

const icon = {
  success: 'check_circle',
  error: 'error',
  info: 'info',
}
</script>

<template>
  <div :class="['toast', `toast--${notification.type || 'info'}`]">
    <span class="material-symbols-outlined">{{ icon[notification.type] || 'info' }}</span>
    <p>{{ notification.message }}</p>
    <button @click="emit('close')" class="close-btn">
      <span class="material-symbols-outlined">close</span>
    </button>
  </div>
</template>

<style scoped>
.toast {
  display: flex;
  align-items: center;
  padding: 16px;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  color: white;
  margin-top: 16px;
}

.toast--info {
  background-color: #2196f3;
}

.toast--success {
  background-color: #4caf50;
}

.toast--error {
  background-color: #f44336;
}

.toast p {
  margin: 0;
  margin-left: 16px;
}

.close-btn {
  background: none;
  border: none;
  color: white;
  cursor: pointer;
  margin-left: auto;
  padding: 0;
}
</style>