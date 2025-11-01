import { describe, it, expect, beforeEach } from 'vitest'
import { useNotifications } from '../composables/useNotifications'

describe('useNotifications', () => {
  let composable

  beforeEach(() => {
    // Since the state is global, we need to reset it before each test
    const { notifications, addNotification, removeNotification } = useNotifications()
    composable = { notifications, addNotification, removeNotification }
    composable.notifications.value = []
  })

  it('adds a notification', () => {
    composable.addNotification({ message: 'Test', type: 'info' })
    expect(composable.notifications.value.length).toBe(1)
    expect(composable.notifications.value[0].message).toBe('Test')
  })

  it('removes a notification', () => {
    composable.addNotification({ message: 'Test', type: 'info' })
    const id = composable.notifications.value[0].id
    composable.removeNotification(id)
    expect(composable.notifications.value.length).toBe(0)
  })
})
