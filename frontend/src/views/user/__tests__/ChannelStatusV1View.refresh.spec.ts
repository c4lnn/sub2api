import { flushPromises, shallowMount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ChannelStatusV1View from '../ChannelStatusV1View.vue'
import type { UserMonitorListResponse, UserMonitorView } from '@/api/channelMonitor'

const { list } = vi.hoisted(() => ({ list: vi.fn() }))
vi.mock('@/api/channelMonitor', () => ({ list, status: vi.fn() }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ cachedPublicSettings: { channel_monitor_enabled: true }, showError: vi.fn() }) }))
vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string) => key }),
}))
const mountView = () => shallowMount(ChannelStatusV1View, {
  global: { stubs: {
    AppLayout: { template: '<div><slot /></div>' },
    MonitorHero: { props: ['autoRefresh', 'loading'], emits: ['refresh'], template: `<div :data-loading="loading">
      <button class="interval" @click="autoRefresh.setInterval(120)">120 seconds</button>
      <button class="refresh" @click="$emit('refresh')">Refresh</button>
      <button class="disable" @click="autoRefresh.setEnabled(false)">Disable</button>
    </div>` },
    MonitorCardGrid: false,
    EmptyState: false,
    MonitorCard: { props: ['item'], template: '<button>{{ item.name }}</button>' },
  } },
})

function monitor(): UserMonitorView {
  return {
    id: 1, name: 'Published channel', provider: 'openai', group_name: 'Old label',
    display_group_id: 10, display_group_name: 'Formal group', display_group_sort_order: 0, sort_order: 0,
    primary_model: 'model', primary_status: 'operational', primary_latency_ms: 100,
    primary_ping_latency_ms: 20, availability_7d: 99, extra_models: [], timeline: [],
  }
}
let wrapper: ReturnType<typeof mountView>
beforeEach(() => { vi.useFakeTimers(); localStorage.clear(); list.mockReset().mockResolvedValue({ items: [] }) })
afterEach(() => { wrapper?.unmount(); vi.useRealTimers(); localStorage.clear() })

describe('channel monitor refresh interval', () => {
  it('keeps automatic refresh disabled after reopening the page', async () => {
    wrapper = mountView(); await flushPromises()
    await wrapper.get('.disable').trigger('click')
    wrapper.unmount()
    wrapper = mountView(); await flushPromises()
    const calls = list.mock.calls.length
    await vi.advanceTimersByTimeAsync(240000)
    expect(list).toHaveBeenCalledTimes(calls)
    expect(JSON.parse(localStorage.getItem('channel-status-auto-refresh')!).enabled).toBe(false)
    await wrapper.get('.refresh').trigger('click'); await flushPromises()
    expect(list).toHaveBeenCalledTimes(calls + 1)
  })

  it('starts automatic refresh for a first visit', async () => {
    wrapper = mountView(); await flushPromises()
    await vi.advanceTimersByTimeAsync(120000)
    expect(list.mock.calls.length).toBeGreaterThan(1)
  })

  it('refreshes at the selected interval on successive automatic refreshes', async () => {
    wrapper = mountView(); await flushPromises()
    await wrapper.get('.interval').trigger('click')
    await vi.advanceTimersByTimeAsync(119000)
    expect(list).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(1000)
    expect(list).toHaveBeenCalledTimes(2)
    await vi.advanceTimersByTimeAsync(119000)
    expect(list).toHaveBeenCalledTimes(2)
    await vi.advanceTimersByTimeAsync(1000)
    expect(list).toHaveBeenCalledTimes(3)
  })

  it('preserves the selected interval after a manual refresh', async () => {
    wrapper = mountView(); await flushPromises()
    await wrapper.get('.interval').trigger('click')
    await wrapper.get('.refresh').trigger('click'); await flushPromises()
    expect(list).toHaveBeenCalledTimes(2)
    await vi.advanceTimersByTimeAsync(119000)
    expect(list).toHaveBeenCalledTimes(2)
    await vi.advanceTimersByTimeAsync(1000)
    expect(list).toHaveBeenCalledTimes(3)
  })

  it('honors a persisted interval after the initial load', async () => {
    localStorage.setItem('channel-status-auto-refresh', JSON.stringify({ enabled: true, interval_seconds: 120 }))
    wrapper = mountView(); await flushPromises()
    await vi.advanceTimersByTimeAsync(119000)
    expect(list).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(1000)
    expect(list).toHaveBeenCalledTimes(2)
    await wrapper.get('.disable').trigger('click')
    await vi.advanceTimersByTimeAsync(120000)
    expect(list).toHaveBeenCalledTimes(2)
  })

  it('retains published cards while refreshing and shows the original empty state when none remain', async () => {
    list.mockResolvedValueOnce({ items: [monitor()] })
    wrapper = mountView()
    await flushPromises()
    const pending = Promise.withResolvers<UserMonitorListResponse>()
    list.mockReturnValueOnce(pending.promise)
    await wrapper.get('.refresh').trigger('click')
    expect(wrapper.text()).toContain('Published channel')
    expect(wrapper.find('.animate-pulse').exists()).toBe(false)
    expect(wrapper.get('[data-loading]').attributes('data-loading')).toBe('true')
    pending.resolve({ items: [] })
    await flushPromises()
    expect(wrapper.text()).not.toContain('Published channel')
    expect(wrapper.findAll('section')).toHaveLength(0)
    expect(wrapper.text()).toContain('channelStatus.empty.title')
    expect(wrapper.get('[data-loading]').attributes('data-loading')).toBe('false')
  })

  it('ignores a superseded list response that would republish a removed monitor', async () => {
    list.mockResolvedValueOnce({ items: [monitor()] })
    wrapper = mountView()
    await flushPromises()
    const old = Promise.withResolvers<UserMonitorListResponse>()
    list.mockReturnValueOnce(old.promise).mockResolvedValueOnce({ items: [] })
    await wrapper.get('.refresh').trigger('click')
    await wrapper.get('.refresh').trigger('click')
    await flushPromises()
    expect(wrapper.text()).not.toContain('Published channel')
    old.resolve({ items: [monitor()] })
    await flushPromises()
    expect(wrapper.text()).not.toContain('Published channel')
    expect(wrapper.findAll('section')).toHaveLength(0)
    expect(wrapper.get('[data-loading]').attributes('data-loading')).toBe('false')
  })
})
