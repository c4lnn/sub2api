import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import MonitorDetailDialog from '../MonitorDetailDialog.vue'
import type { UserMonitorDetail } from '@/api/channelMonitor'
const mocks = vi.hoisted(() => ({ status: vi.fn(), showError: vi.fn() }))
vi.mock('@/api/channelMonitor', () => ({ status: mocks.status }))
vi.mock('@/stores/app', () => ({ useAppStore: () => mocks }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...await importOriginal<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }) }))
enableAutoUnmount(afterEach)
beforeEach(() => vi.clearAllMocks())
function deferred() {
  return Promise.withResolvers<UserMonitorDetail>()
}
const detail = (model: string): UserMonitorDetail => ({
  id: 1, name: 'Monitor', provider: 'openai', group_name: 'Legacy',
  display_group_id: 10, display_group_name: 'Published', display_group_sort_order: 0, sort_order: 0,
  models: [{ model, latest_status: 'operational', latest_latency_ms: 100, availability_7d: 99,
    availability_15d: 98, availability_30d: 97, avg_latency_7d_ms: 100 }],
})
function open() {
  return mount(MonitorDetailDialog, { props: { show: true, monitorId: 1, title: 'Monitor' },
    global: { stubs: { BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /></div>' } } } })
}
describe('monitor detail request ownership', () => {
  it('keeps the new monitor response when the old response arrives last', async () => {
    const old = deferred()
    mocks.status.mockReturnValueOnce(old.promise).mockResolvedValueOnce(detail('new-model'))
    const w = open(); await w.setProps({ monitorId: 2 }); await flushPromises()
    old.resolve(detail('old-model')); await flushPromises()
    expect(w.text()).toContain('new-model'); expect(w.text()).not.toContain('old-model')
  })
  it('ignores a previous failure while the current monitor is loading', async () => {
    const old = deferred(); const current = deferred()
    mocks.status.mockReturnValueOnce(old.promise).mockReturnValueOnce(current.promise)
    const w = open(); await w.setProps({ show: false }); await w.setProps({ show: true })
    old.reject(new Error('old failure')); await flushPromises()
    expect(mocks.showError).not.toHaveBeenCalled(); expect(w.text()).toContain('common.loading')
    current.resolve(detail('current')); await flushPromises(); expect(w.text()).toContain('current')
  })
  it('reports a failure for the current monitor', async () => {
    mocks.status.mockRejectedValueOnce(new Error('current failure'))
    const w = open(); await flushPromises()
    expect(mocks.showError).toHaveBeenCalledWith('current failure')
    expect(w.text()).toContain('channelStatus.detailLoadError')
  })
  it('does not restore a hidden dialog when its detail response arrives late', async () => {
    const old = deferred()
    mocks.status.mockReturnValueOnce(old.promise)
    const w = open()
    await w.setProps({ show: false, monitorId: null })
    old.resolve(detail('hidden-model'))
    await flushPromises()
    expect(w.text()).not.toContain('hidden-model')
    expect(mocks.showError).not.toHaveBeenCalled()
    mocks.status.mockResolvedValueOnce(detail('republished-model'))
    await w.setProps({ show: true, monitorId: 1 })
    await flushPromises()
    expect(w.text()).toContain('republished-model')
    expect(w.text()).not.toContain('hidden-model')
    expect(mocks.status).toHaveBeenCalledTimes(2)
  })

  it('clears loaded details when the target is invalidated without reopening stale content', async () => {
    mocks.status.mockResolvedValueOnce(detail('previous-model'))
    const w = open()
    await flushPromises()
    expect(w.text()).toContain('previous-model')
    await w.setProps({ monitorId: null })
    expect(w.text()).not.toContain('previous-model')
    expect(w.text()).not.toContain('common.loading')
    expect(mocks.status).toHaveBeenCalledTimes(1)
  })
})
