import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import type { ChannelMonitor } from '@/api/admin/channelMonitor'
import MonitorDisplayOrderCell from '../MonitorDisplayOrderCell.vue'
import MonitorDisplayGroups from '../MonitorDisplayGroups.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({
  t: (key: string, values?: Record<string, unknown>) => values ? `${key}:${JSON.stringify(values)}` : key,
}) }))

function row(): ChannelMonitor {
  return {
    id: 1, name: 'Monitor', provider: 'openai', api_mode: 'chat_completions', endpoint: '',
    api_key_masked: '', primary_model: 'model', extra_models: [], group_name: 'Historical label',
    display_group_id: 8, sort_order: 20, enabled: true, interval_seconds: 60, jitter_seconds: 0,
    last_checked_at: null, created_by: 1, created_at: '', updated_at: '', primary_status: '',
    primary_latency_ms: null, availability_7d: 0, extra_models_status: [], template_id: null,
    extra_headers: {}, body_override_mode: 'off', body_override: null, check_mode: 'probe', account_id: null,
  }
}

describe('monitor display controls', () => {
  it('requires explicit save and keeps drafts distinct from the persisted order', async () => {
    const monitor = row()
    const wrapper = mount(MonitorDisplayOrderCell, { props: { row: monitor, saving: false } })
    await wrapper.get('input').setValue(10)
    expect(wrapper.emitted('save')).toBeUndefined()
    expect(wrapper.get('[data-testid="saved-monitor-order"]').text()).toContain('20')
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('save')).toEqual([[monitor, 10]])
    expect(monitor.sort_order).toBe(20)
    await wrapper.setProps({ row: { ...monitor, sort_order: 10 } })
    expect((wrapper.get('input').element as HTMLInputElement).value).toBe('10')
    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })

  it.each(['', -1, 1.5, 2147483648])('does not submit invalid order %s', async (value) => {
    const wrapper = mount(MonitorDisplayOrderCell, { props: { row: row(), saving: false } })
    await wrapper.get('input').setValue(value)
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('save')).toBeUndefined()
    wrapper.unmount()
  })

  it.each([0, 2147483647])('accepts the order boundary %s', async (value) => {
    const wrapper = mount(MonitorDisplayOrderCell, { props: { row: row(), saving: false } })
    await wrapper.get('input').setValue(value)
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('save')?.[0][1]).toBe(value)
    wrapper.unmount()
  })

  it('disables sorting while a save is pending', async () => {
    const wrapper = mount(MonitorDisplayOrderCell, { props: { row: row(), saving: true } })
    expect(wrapper.get('input').attributes('disabled')).toBeDefined()
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('save')).toBeUndefined()
    wrapper.unmount()
  })

  it('selects by identity, including a real group with the storage label, and renders full long names', async () => {
    const longName = '展示组'.repeat(33)
    const storageName = 'admin.channelMonitor.displayGroups.storage'
    const groups = [
      { id: 8, name: storageName, sort_order: 0, created_at: '', updated_at: '' },
      { id: 9, name: longName, sort_order: 10, created_at: '', updated_at: '' },
    ]
    const wrapper = mount(MonitorDisplayGroups, { props: { groups, selectedId: null, loading: false } })
    expect(wrapper.get('[data-testid="display-group-9"]').text()).toBe(longName)
    expect(wrapper.get('[data-testid="display-group-9"]').classes()).toContain('break-all')
    await wrapper.get('[data-testid="display-group-8"]').trigger('click')
    await wrapper.get('[data-testid="display-group-storage"]').trigger('click')
    expect(wrapper.emitted('select')).toEqual([[8], [null]])
    await wrapper.get('[data-testid="edit-display-group-9"]').trigger('click')
    expect(wrapper.emitted('edit')).toEqual([[groups[1]]])
    expect(wrapper.get('[data-testid="delete-display-group-8"]').attributes('aria-label')).toContain(storageName)
    expect(wrapper.find('[data-testid="delete-display-group-storage"]').exists()).toBe(false)
    wrapper.unmount()
  })
})
