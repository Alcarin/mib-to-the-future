// frontend/src/composables/useNotifications.js
import { ref } from 'vue'

const notifications = ref([])

/**
 * Un composable Vue per la gestione delle notifiche.
 *
 * @returns {{
 *   notifications: import('vue').Ref<Array<Object>>,
 *   addNotification: (notification: Object) => void,
 *   removeNotification: (id: number) => void,
 * }}
 */
export function useNotifications() {
  const addNotification = (notification) => {
    const id = Date.now()
    notifications.value.push({ ...notification, id })
  }

  const removeNotification = (id) => {
    const index = notifications.value.findIndex((n) => n.id === id)
    if (index !== -1) {
      notifications.value.splice(index, 1)
    }
  }

  return { notifications, addNotification, removeNotification }
}