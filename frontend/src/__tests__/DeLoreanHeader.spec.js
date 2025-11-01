import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import DeLoreanHeader from '../components/DeLoreanHeader.vue'

// Mock Material Web Components
vi.mock('@material/web/iconbutton/icon-button.js', () => ({}))
vi.mock('@material/web/textfield/outlined-text-field.js', () => ({}));

describe('DeLoreanHeader.vue', () => {
  it('renders the search input and emits update:search event', async () => {
    const wrapper = mount(DeLoreanHeader, {
      props: {
        search: 'initial search',
      },
    })

    // Check if the search input is rendered with the correct initial value
    const searchInput = wrapper.find('md-outlined-text-field')
    expect(searchInput.exists()).toBe(true)
    expect(searchInput.attributes('value')).toBe('initial search')

    // Simulate user input
    const inputEl = searchInput.element;
    inputEl.value = 'new search';
    await searchInput.trigger('input');

    // Check if the component emits the 'update:search' event with the new value
    expect(wrapper.emitted('update:search')).toBeTruthy()
    expect(wrapper.emitted('update:search')[0][0]).toBe('new search')
  })
})
