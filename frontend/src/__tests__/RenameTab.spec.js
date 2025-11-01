import { describe, it, expect, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import TabsPanel from '../components/TabsPanel.vue';

describe('TabsPanel Tab Renaming', () => {
  let wrapper;
  const tabs = [
    { id: 'log-default', title: 'Request Log', type: 'log', data: [] },
    { id: 'tab-1', title: 'My Tab', type: 'log', data: [] },
  ];

  beforeEach(() => {
    wrapper = mount(TabsPanel, {
      props: {
        tabs,
        activeTabId: 'tab-1',
      },
      global: {
        stubs: {
          'md-tabs': true,
          'md-primary-tab': true,
          'md-icon-button': true,
        },
      },
    });
  });

  it('should display the tab title', () => {
    const tabTitles = wrapper.findAll('.tab-title');
    expect(tabTitles[1].text()).toBe('My Tab');
  });

  it('should switch to an input field on double-click', async () => {
    await wrapper.find('[data-testid="tab-tab-1"]').trigger('dblclick');
    const input = wrapper.find('.tab-title-input');
    expect(input.exists()).toBe(true);
    expect(input.element.value).toBe('My Tab');
  });

  it('should emit a rename-tab event on blur', async () => {
    await wrapper.find('[data-testid="tab-tab-1"]').trigger('dblclick');
    const input = wrapper.find('.tab-title-input');
    await input.setValue('New Tab Name');
    await input.trigger('blur');

    expect(wrapper.emitted('rename-tab')).toBeTruthy();
    expect(wrapper.emitted('rename-tab')[0][0]).toEqual({ id: 'tab-1', title: 'New Tab Name' });
  });

  it('should emit a rename-tab event on Enter key press', async () => {
    await wrapper.find('[data-testid="tab-tab-1"]').trigger('dblclick');
    const input = wrapper.find('.tab-title-input');
    await input.setValue('Another New Name');
    await input.trigger('keydown.enter');

    expect(wrapper.emitted('rename-tab')).toBeTruthy();
    expect(wrapper.emitted('rename-tab')[0][0]).toEqual({ id: 'tab-1', title: 'Another New Name' });
  });

  it('should allow renaming the default log tab', async () => {
    await wrapper.find('[data-testid="tab-log-default"]').trigger('dblclick');
    const input = wrapper.find('.tab-title-input');
    expect(input.exists()).toBe(true);
  });
});
