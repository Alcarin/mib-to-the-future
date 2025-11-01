import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ToastNotification from '../components/ToastNotification.vue'

describe('ToastNotification.vue', () => {
  const notification = {
    id: 1,
    message: 'Test message',
    type: 'success',
  }

  it('renders notification message and icon', () => {
    const wrapper = mount(ToastNotification, {
      props: { notification },
    })

    expect(wrapper.text()).toContain('Test message')
    expect(wrapper.text()).toContain('check_circle')
  })

  it('emits close event after a timeout', async () => {
    vi.useFakeTimers()
    const wrapper = mount(ToastNotification, { props: { notification } })

    await vi.advanceTimersByTimeAsync(5000)

    expect(wrapper.emitted('close')).toBeTruthy()
    vi.useRealTimers()
  })

  it('emits close event when close button is clicked', async () => {
    const wrapper = mount(ToastNotification, { props: { notification } })

    await wrapper.find('.close-btn').trigger('click')

    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
