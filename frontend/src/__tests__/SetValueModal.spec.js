import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import SetValueModal from '../components/SetValueModal.vue'

describe('SetValueModal', () => {
  const textFieldStub = defineComponent({
    name: 'MdOutlinedTextField',
    props: {
      value: {
        type: [String, Number],
        default: ''
      },
      type: {
        type: String,
        default: 'text'
      }
    },
    emits: ['input'],
    setup(props, { emit, attrs }) {
      return () => h('input', {
        ...attrs,
        value: props.value,
        type: props.type || attrs.type || 'text',
        onInput: (event) => emit('input', event)
      })
    }
  })

  const buttonComponent = defineComponent({
    name: 'MdButtonStub',
    emits: ['click'],
    setup(_, { emit, slots, attrs }) {
      return () => h(
        'button',
        {
          type: 'button',
          ...attrs,
          onClick: () => emit('click')
        },
        slots.default ? slots.default() : []
      )
    }
  })

  const selectStub = {
    props: ['value'],
    emits: ['change'],
    template: '<select :value="value" @change="$emit(\'change\', $event)"><slot /></select>'
  }

  const optionStub = {
    props: ['value'],
    template: '<option :value="value"><slot /></option>'
  }

  const checkboxStub = {
    props: ['checked'],
    emits: ['change'],
    template: '<input type="checkbox" :checked="checked" @change="$emit(\'change\', $event)" />'
  }

  const global = {
    components: {
      'md-outlined-text-field': textFieldStub,
      'md-filled-button': buttonComponent,
      'md-text-button': buttonComponent,
      'md-icon-button': buttonComponent
    },
    stubs: {
      'md-outlined-select': selectStub,
      'md-select-option': optionStub,
      'md-circular-progress': { template: '<div class="progress"></div>' },
      'md-checkbox': checkboxStub
    }
  }

  const baseNode = {
    name: 'sysLocation',
    oid: '1.3.6.1.2.1.1.6',
    type: 'scalar',
    syntax: 'INTEGER (0..100)'
  }

  it('renders summary information', () => {
    const wrapper = mount(SetValueModal, {
      props: {
        show: true,
        node: baseNode
      },
      global
    })

    expect(wrapper.text()).toContain('sysLocation')
    expect(wrapper.text()).toContain('1.3.6.1.2.1.1.6')
    expect(wrapper.text()).toContain('INTEGER (0..100)')
  })

  it('emits confirm payload for numeric syntax', async () => {
    const wrapper = mount(SetValueModal, {
      props: {
        show: true,
        node: baseNode
      },
      global
    })

    const setupState = wrapper.vm.$.setupState
    setupState.formValue = '42'
    await setupState.submit()

    const emitted = wrapper.emitted('confirm')
    expect(emitted).toBeTruthy()
    const payload = emitted[0][0]
    expect(payload.valueType).toBe('integer')
    expect(payload.value).toBe(42)
    expect(payload.displayValue).toBe('42')
  })
})
