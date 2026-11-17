import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, shallowMount } from '@vue/test-utils'
import ChannelStatusV1View from '../ChannelStatusV1View.vue'
import type { UserMonitorDetail, UserMonitorView } from '@/api/channelMonitor'
import MonitorCardGrid from '@/components/user/monitor/MonitorCardGrid.vue'

const mocks = vi.hoisted(() => ({ list: vi.fn(), status: vi.fn(), showError: vi.fn() }))
vi.mock('@/api/channelMonitor', () => mocks)
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ cachedPublicSettings: { channel_monitor_enabled: true }, showError: mocks.showError }) }))
vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string) => key })
}))
enableAutoUnmount(afterEach)
afterEach(() => { vi.useRealTimers(); localStorage.clear() })

function monitor(overrides: Partial<UserMonitorView> = {}): UserMonitorView {
  return {
    id: 1, name: 'Monitor', provider: 'openai', group_name: 'Legacy',
    display_group_id: 10, display_group_name: 'Published', display_group_sort_order: 0, sort_order: 0,
    primary_model: 'model', primary_status: 'operational', primary_latency_ms: 100,
    primary_ping_latency_ms: 20, availability_7d: 99, extra_models: [], timeline: [], ...overrides,
  }
}

function detail(availability: number, id = 1, model = 'model'): UserMonitorDetail {
  return {
    id, name: `Monitor ${id}`, provider: 'openai', group_name: 'Legacy',
    display_group_id: 10, display_group_name: 'Published', display_group_sort_order: 0, sort_order: 0,
    models: [{ model, latest_status: 'operational', latest_latency_ms: 100, availability_7d: 99,
      availability_15d: availability, availability_30d: availability, avg_latency_7d_ms: 100 }],
  }
}

function deferred<T>() {
  return Promise.withResolvers<T>()
}
beforeEach(() => {
  vi.useFakeTimers()
  vi.resetAllMocks()
  localStorage.clear()
  localStorage.setItem('channel-status-auto-refresh', JSON.stringify({ enabled: true, interval_seconds: 30 }))
  mocks.list.mockImplementation(async () => ({ items: [monitor()] }))
  mocks.status.mockResolvedValue(detail(90))
})

function mountView() {
  return shallowMount(ChannelStatusV1View, {
    global: { stubs: {
      AppLayout: { template: '<div><slot /></div>' },
      MonitorHero: { props: ['loading', 'overallStatus'], emits: ['update:window', 'refresh'], template: `<div :data-loading="loading" :data-overall="overallStatus">
        <button class="seven" @click="$emit('update:window', '7d')">7d</button>
        <button class="fifteen" @click="$emit('update:window', '15d')">15d</button>
        <button class="thirty" @click="$emit('update:window', '30d')">30d</button>
        <button class="refresh" @click="$emit('refresh')">Refresh</button>
      </div>` },
      MonitorCardGrid: false,
      MonitorCard: { props: ['item', 'availabilityValue'], emits: ['click'], template: '<button @click="$emit(\'click\')">{{ item.name }}<output>{{ availabilityValue }}</output></button>' },
      MonitorDetailDialog: false,
      BaseDialog: { props: ['show', 'title'], emits: ['close'], template: '<div v-if="show" role="dialog"><h3>{{ title }}</h3><slot /><button class="close-detail" @click="$emit(\'close\')">Close</button></div>' }
    } }
  })
}

describe('channel monitor automatic detail refresh', () => {
  it.each(['fifteen', 'thirty'])('refreshes the cached %s-day availability on timer ticks', async (tab) => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('.' + tab).trigger('click')
    await flushPromises()
    expect(wrapper.get('output').text()).toBe('90')
    mocks.status.mockResolvedValue(detail(98))
    await vi.advanceTimersByTimeAsync(30000)
    await flushPromises()
    expect(wrapper.get('output').text()).toBe('98')
    expect(mocks.status).toHaveBeenCalledTimes(2)
    expect(wrapper.get('[data-loading]').attributes('data-loading')).toBe('false')
  })

  it('keeps the seven-day view on the list endpoint without extra detail requests', async () => {
    const wrapper = mountView()
    await flushPromises()
    await vi.advanceTimersByTimeAsync(30000)
    await flushPromises()
    expect(mocks.list).toHaveBeenCalledTimes(2)
    expect(mocks.status).not.toHaveBeenCalled()
    expect(wrapper.get('output').text()).toBe('99')
  })

  it('still refreshes cached availability on manual reload', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('.fifteen').trigger('click')
    await flushPromises()
    mocks.status.mockResolvedValue(detail(98))
    await wrapper.get('.refresh').trigger('click')
    await flushPromises()
    expect(wrapper.get('output').text()).toBe('98')
  })

  it('opens the correct details from different groups without extra list or grouping requests', async () => {
    mocks.list.mockResolvedValue({ items: [
      monitor({ name: 'First channel', display_group_name: 'GPT', primary_model: 'model-1' }),
      monitor({ id: 2, name: 'Second channel', display_group_id: 20, display_group_name: 'Claude', primary_model: 'model-2', primary_status: 'failed', availability_7d: 80 }),
    ] })
    mocks.status.mockImplementation(async (id: number) => detail(99, id, `detail-model-${id}`))
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.findAll('section h2').map(heading => heading.text())).toEqual(['GPT', 'Claude'])
    expect(mocks.list).toHaveBeenCalledTimes(1)
    expect(mocks.status).not.toHaveBeenCalled()

    const first = wrapper.findAll('section')[0]!
    await first.get('button').trigger('click')
    await flushPromises()
    expect(wrapper.get('[role="dialog"]').text()).toContain('First channel')
    expect(wrapper.get('[role="dialog"]').text()).toContain('detail-model-1')
    expect(mocks.status).toHaveBeenLastCalledWith(1)
    await wrapper.get('.close-detail').trigger('click')

    const second = wrapper.findAll('section')[1]!
    await second.get('button').trigger('click')
    await flushPromises()
    expect(wrapper.get('[role="dialog"]').text()).toContain('Second channel')
    expect(wrapper.get('[role="dialog"]').text()).toContain('detail-model-2')
    expect(wrapper.get('[role="dialog"]').text()).not.toContain('detail-model-1')
    expect(mocks.status.mock.calls.map(call => call[0])).toEqual([1, 2])
    expect(mocks.list).toHaveBeenCalledTimes(1)
  })

  it('keeps statistics and dialog identity when formal groups are renamed and cards move', async () => {
    mocks.list.mockResolvedValueOnce({ items: [monitor(), monitor({ id: 2, name: 'Second', primary_model: 'second', display_group_id: 20, display_group_name: 'B' })] })
    mocks.status.mockImplementation(async (id: number) => detail(id === 1 ? 91 : 82, id, id === 1 ? 'model' : 'second'))
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('.fifteen').trigger('click')
    await flushPromises()
    await wrapper.findAll('section button')[0]!.trigger('click')
    await flushPromises()
    mocks.list.mockResolvedValue({ items: [
      monitor({ display_group_id: 20, display_group_name: 'Renamed B', sort_order: 20, name: 'Renamed monitor' }),
      monitor({ id: 2, name: 'Second', primary_model: 'second', display_group_id: 20, display_group_name: 'Renamed B', sort_order: 10 }),
    ] })
    await wrapper.get('.refresh').trigger('click')
    await flushPromises()
    expect(wrapper.findAll('h2').map(heading => heading.text())).toEqual(['Renamed B'])
    expect(wrapper.findAll('output').map(output => output.text())).toEqual(['82', '91'])
    expect(wrapper.get('[role="dialog"]').text()).toContain('Renamed monitor')
    expect(wrapper.get('[role="dialog"]').text()).not.toContain('second')
  })

  it('closes an invisible monitor dialog, clears its target and rejects a stale click', async () => {
    const old = deferred<UserMonitorDetail>()
    mocks.status.mockReturnValueOnce(old.promise)
    const wrapper = mountView()
    await flushPromises()
    const staleRow = monitor()
    await wrapper.get('section button').trigger('click')
    expect(wrapper.find('[role="dialog"]').exists()).toBe(true)
    mocks.list.mockResolvedValue({ items: [] })
    await wrapper.get('.refresh').trigger('click')
    await flushPromises()
    expect(wrapper.findAll('section')).toHaveLength(0)
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    wrapper.getComponent(MonitorCardGrid).vm.$emit('cardClick', staleRow)
    await flushPromises()
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    old.resolve(detail(90, 1, 'hidden-model'))
    await flushPromises()
    expect(wrapper.text()).not.toContain('hidden-model')
    expect(mocks.status).toHaveBeenCalledTimes(1)
  })

  it('discards cached windows after removal even if the same monitor is published again', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('.fifteen').trigger('click')
    await flushPromises()
    expect(wrapper.get('output').text()).toBe('90')
    await wrapper.get('.seven').trigger('click')
    mocks.list.mockResolvedValue({ items: [] })
    await wrapper.get('.refresh').trigger('click')
    await flushPromises()
    mocks.list.mockResolvedValue({ items: [monitor()] })
    await wrapper.get('.refresh').trigger('click')
    await flushPromises()
    const current = deferred<UserMonitorDetail>()
    mocks.status.mockReturnValueOnce(current.promise)
    await wrapper.get('.fifteen').trigger('click')
    expect(wrapper.get('output').text()).toBe('')
    current.resolve(detail(98))
    await flushPromises()
    expect(wrapper.get('output').text()).toBe('98')
    expect(mocks.status).toHaveBeenCalledTimes(2)
  })

  it.each(['success', 'failure'])('invalidates in-flight ownership across removal and republication (%s)', async (outcome) => {
    const old = deferred<UserMonitorDetail>()
    const current = deferred<UserMonitorDetail>()
    mocks.status.mockReturnValueOnce(old.promise).mockReturnValueOnce(current.promise)
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('.fifteen').trigger('click')
    await wrapper.get('.seven').trigger('click')
    mocks.list.mockResolvedValue({ items: [] })
    await wrapper.get('.refresh').trigger('click')
    await flushPromises()
    mocks.list.mockResolvedValue({ items: [monitor()] })
    await wrapper.get('.refresh').trigger('click')
    await flushPromises()
    await wrapper.get('.fifteen').trigger('click')
    expect(mocks.status).toHaveBeenCalledTimes(2)
    if (outcome === 'success') old.resolve(detail(11))
    else old.reject(new Error('hidden detail failure'))
    await flushPromises()
    expect(wrapper.get('output').text()).toBe('')
    expect(mocks.showError).not.toHaveBeenCalled()
    await wrapper.get('.thirty').trigger('click')
    expect(mocks.status).toHaveBeenCalledTimes(2)
    current.resolve(detail(98))
    await flushPromises()
    expect(wrapper.get('output').text()).toBe('98')
  })

  it('does not let a slower pre-refresh request overwrite newer visible statistics', async () => {
    const old = deferred<UserMonitorDetail>()
    mocks.status.mockReturnValueOnce(old.promise).mockResolvedValueOnce(detail(98))
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('.fifteen').trigger('click')
    await wrapper.get('.refresh').trigger('click')
    await flushPromises()
    expect(wrapper.get('output').text()).toBe('98')
    old.resolve(detail(11))
    await flushPromises()
    expect(wrapper.get('output').text()).toBe('98')
  })

  it('keeps visible abnormal cards and overall status consistent when the published set changes', async () => {
    mocks.list.mockResolvedValueOnce({ items: [monitor(), monitor({ id: 2, name: 'Abnormal', primary_status: 'failed', display_group_id: 20, display_group_name: 'B' })] })
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.get('[data-overall]').attributes('data-overall')).toBe('degraded')
    expect(wrapper.text()).toContain('Abnormal')
    mocks.list.mockResolvedValue({ items: [monitor()] })
    await wrapper.get('.refresh').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-overall]').attributes('data-overall')).toBe('operational')
    expect(wrapper.text()).not.toContain('Abnormal')
  })

  it('does not discard visible cards or cached statistics when list refresh fails', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('.fifteen').trigger('click')
    await flushPromises()
    mocks.list.mockRejectedValueOnce(new Error('list failure'))
    await wrapper.get('.refresh').trigger('click')
    await flushPromises()
    expect(wrapper.get('output').text()).toBe('90')
    expect(mocks.status).toHaveBeenCalledTimes(1)
    expect(mocks.showError).toHaveBeenCalledWith('list failure')
  })

  it('loads the selected statistics window when it changes before the initial list resolves', async () => {
    const initial = deferred<{ items: UserMonitorView[] }>()
    mocks.list.mockReturnValueOnce(initial.promise)
    const wrapper = mountView()
    await wrapper.get('.fifteen').trigger('click')
    expect(mocks.status).not.toHaveBeenCalled()
    initial.resolve({ items: [monitor()] })
    await flushPromises()
    expect(wrapper.get('output').text()).toBe('90')
    expect(mocks.status).toHaveBeenCalledTimes(1)
  })
})
