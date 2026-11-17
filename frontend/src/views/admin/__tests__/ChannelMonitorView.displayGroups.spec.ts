import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import type { VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type * as VueI18n from 'vue-i18n'
import type { ChannelMonitor, ListParams, ListResponse, MonitorDisplayGroup } from '@/api/admin/channelMonitor'
import ChannelMonitorView from '@/views/admin/ChannelMonitorView.vue'

const api = vi.hoisted(() => ({
  list: vi.fn(), listDisplayGroups: vi.fn(), createDisplayGroup: vi.fn(), updateDisplayGroup: vi.fn(),
  deleteDisplayGroup: vi.fn(), updateDisplay: vi.fn(), create: vi.fn(), update: vi.fn(),
  duplicate: vi.fn(), del: vi.fn(), runNow: vi.fn(), templates: vi.fn(),
  showError: vi.fn(), showSuccess: vi.fn(),
}))
vi.mock('@/api/admin', () => ({ adminAPI: {
  channelMonitor: api,
  channelMonitorTemplate: { list: api.templates },
  accounts: { list: vi.fn().mockResolvedValue({ items: [] }), getById: vi.fn() },
} }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: api.showError, showSuccess: api.showSuccess }) }))
vi.mock('@/utils/featureFlags', () => ({ getChannelMonitorMode: () => 'v1', isChannelMonitorV1Mode: () => true }))
vi.mock('@/features/channel-monitor-v2/MonitorSettingsPanel.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/features/channel-monitor-v3/V3SettingsPanel.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/api/keys', () => ({ keysAPI: { list: vi.fn() } }))
vi.mock('@/api/groups', () => ({ userGroupsAPI: { getUserGroupRates: vi.fn() } }))
vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof VueI18n>('vue-i18n'),
  useI18n: () => ({ t: (key: string, values?: Record<string, unknown>) => values ? `${key}:${JSON.stringify(values)}` : key }),
}))

const Dialog = defineComponent({
  props: ['show', 'title'],
  template: '<section v-if="show" role="dialog"><h2>{{ title }}</h2><slot /><slot name="footer" /></section>',
})
const Confirm = defineComponent({
  props: ['show', 'message'], emits: ['confirm', 'cancel'],
  template: '<section v-if="show" role="alertdialog"><p>{{ message }}</p><button data-testid="confirm-delete" @click="$emit(\'confirm\')">Delete</button><button @click="$emit(\'cancel\')">Cancel</button></section>',
})

function group(id: number, name: string, sort_order = 0): MonitorDisplayGroup {
  return { id, name, sort_order, created_at: '', updated_at: '' }
}
function monitor(overrides: Partial<ChannelMonitor> = {}): ChannelMonitor {
  return {
    id: 42, name: 'Original monitor', provider: 'openai', api_mode: 'chat_completions',
    endpoint: 'https://example.test', api_key_masked: 'sk-***', primary_model: 'model', extra_models: [],
    group_name: 'Historical label', display_group_id: null, sort_order: 30, enabled: true,
    interval_seconds: 60, jitter_seconds: 0, last_checked_at: null, created_by: 1, created_at: '', updated_at: '',
    primary_status: '', primary_latency_ms: null, availability_7d: 0, extra_models_status: [],
    template_id: null, extra_headers: {}, body_override_mode: 'off', body_override: null,
    check_mode: 'probe', account_id: null, ...overrides,
  }
}
function response(items: ChannelMonitor[], params: ListParams = {}, total = items.length): ListResponse {
  return { items, total, page: params.page ?? 1, page_size: params.page_size ?? 20, pages: Math.ceil(total / (params.page_size ?? 20)) }
}
let wrapper: VueWrapper
let rows: ChannelMonitor[]
let groups: MonitorDisplayGroup[]
function render() {
  wrapper = mount(ChannelMonitorView, { global: { stubs: {
    AppLayout: { template: '<main><slot /></main>' },
    TablePageLayout: { template: '<section><slot name="filters" /><slot name="table" /><slot name="pagination" /></section>' },
    BaseDialog: Dialog, ConfirmDialog: Confirm, MonitorTemplateManagerDialog: true,
    MonitorRunResultDialog: true, MonitorPrimaryModelCell: true, Toggle: true,
    ModelTagInput: true, MonitorKeyPickerDialog: true, MonitorAdvancedRequestConfig: true,
    Icon: true, RouterLink: true,
  } } })
  return wrapper
}
function button(text: string) {
  const found = wrapper.findAll('button').find(item => item.text() === text)
  if (!found) throw new Error(`Missing button: ${text}`)
  return found
}
async function select(id: number | null) {
  await wrapper.get(`[data-testid="display-group-${id ?? 'storage'}"]`).trigger('click')
  await flushPromises()
}
async function openCreate() {
  await button('admin.channelMonitor.createButton').trigger('click')
  await flushPromises()
  await wrapper.get('input[placeholder="admin.channelMonitor.form.namePlaceholder"]').setValue('New monitor')
  await wrapper.get('[data-testid="monitor-endpoint"]').setValue('https://new.example.test')
  await wrapper.get('[data-testid="monitor-primary-model"]').setValue('new-model')
}

beforeEach(() => {
  localStorage.clear()
  vi.resetAllMocks()
  groups = [group(1, 'Group A', 20), group(3, 'Group C', 10), group(2, 'Group B', 10)]
  rows = [monitor()]
  api.listDisplayGroups.mockImplementation(async () => ({ items: groups }))
  api.templates.mockResolvedValue({ items: [] })
  api.list.mockImplementation(async (params: ListParams) => {
    const filtered = rows.filter(row => (row.display_group_id ?? 0) === params.display_group_id
      && (!params.search || row.name.includes(params.search))
      && (params.provider === undefined || row.provider === params.provider)
      && (params.enabled === undefined || row.enabled === params.enabled))
      .sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
      .map(row => ({ ...row }))
    const size = params.page_size ?? 20
    return response(filtered.slice(((params.page ?? 1) - 1) * size, (params.page ?? 1) * size), params, filtered.length)
  })
  api.updateDisplay.mockImplementation(async (id: number, payload: Partial<ChannelMonitor>) => {
    const row = rows.find(item => item.id === id)!
    Object.assign(row, payload)
    return { ...row }
  })
  api.create.mockResolvedValue(monitor({ id: 99 }))
  api.update.mockResolvedValue(monitor())
})
afterEach(() => { wrapper?.unmount(); vi.useRealTimers() })

describe('ChannelMonitorView display groups', () => {
  it('defaults to unpublished storage, sorts real groups by order/ID and keeps storage last', async () => {
    groups[0] = group(1, 'admin.channelMonitor.displayGroups.storage', 20)
    render()
    await flushPromises()
    expect(api.list.mock.calls[0][0]).toMatchObject({ display_group_id: 0, page: 1 })
    const choices = wrapper.findAll('nav [data-testid^="display-group-"]')
    expect(choices.map(item => item.attributes('data-testid'))).toEqual(['display-group-2', 'display-group-3', 'display-group-1', 'display-group-storage'])
    expect(wrapper.get('[data-testid="display-group-storage"]').attributes('aria-current')).toBe('true')
    expect(wrapper.text()).toContain('admin.channelMonitor.displayGroups.storageHint')
    await select(1)
    expect(wrapper.get('[data-testid="display-group-1"]').attributes('aria-current')).toBe('true')
    expect(api.list.mock.lastCall?.[0]).toMatchObject({ display_group_id: 1 })
  })

  it('creates a trimmed group with numeric order and selects the returned identity', async () => {
    api.createDisplayGroup.mockResolvedValue(group(8, 'New group', 5))
    render(); await flushPromises()
    await wrapper.get('[data-testid="create-display-group"]').trigger('click')
    await wrapper.get('#display-group-name').setValue(' New group ')
    await wrapper.get('#display-group-order').setValue(5)
    await wrapper.get('#display-group-form').trigger('submit')
    await flushPromises()
    expect(api.createDisplayGroup).toHaveBeenCalledWith({ name: 'New group', sort_order: 5 })
    expect(wrapper.get('[data-testid="current-display-group"]').text()).toBe('New group')
    expect(api.list.mock.lastCall?.[0]).toMatchObject({ display_group_id: 8, page: 1 })
    expect(wrapper.find('#display-group-form').exists()).toBe(false)
  })

  it('renames and orders a group only after a successful explicit save', async () => {
    api.updateDisplayGroup.mockRejectedValueOnce(new Error('duplicate name')).mockResolvedValueOnce(group(1, 'Renamed', 0))
    render(); await flushPromises(); await select(1)
    await wrapper.get('[data-testid="edit-display-group-1"]').trigger('click')
    await wrapper.get('#display-group-name').setValue(' Renamed ')
    await wrapper.get('#display-group-order').setValue(0)
    expect(wrapper.get('[data-testid="current-display-group"]').text()).toBe('Group A')
    await wrapper.get('#display-group-form').trigger('submit'); await flushPromises()
    expect(wrapper.get('[data-testid="current-display-group"]').text()).toBe('Group A')
    expect(api.showError).toHaveBeenCalledWith('duplicate name')
    expect(wrapper.find('#display-group-form').exists()).toBe(true)
    await wrapper.get('#display-group-form').trigger('submit'); await flushPromises()
    expect(api.updateDisplayGroup).toHaveBeenLastCalledWith(1, { name: 'Renamed', sort_order: 0 })
    expect(wrapper.get('[data-testid="current-display-group"]').text()).toBe('Renamed')
    expect(wrapper.findAll('nav [data-testid^="display-group-"]')[0].text()).toBe('Renamed')
  })

  it.each(['', 'x'.repeat(101), '   '])('rejects invalid group name without a write', async (name) => {
    render(); await flushPromises()
    await wrapper.get('[data-testid="create-display-group"]').trigger('click')
    await wrapper.get('#display-group-name').setValue(name)
    await wrapper.get('#display-group-form').trigger('submit'); await flushPromises()
    expect(api.createDisplayGroup).not.toHaveBeenCalled()
    expect(api.showError).toHaveBeenCalledWith('admin.channelMonitor.displayGroups.invalidInput')
  })

  it('retains filters while switching groups, resets pagination and distinguishes no matches from empty groups', async () => {
    vi.useFakeTimers()
    render(); await flushPromises()
    const search = wrapper.get('input[placeholder="admin.channelMonitor.searchPlaceholder"]')
    await search.setValue('nothing')
    await vi.advanceTimersByTimeAsync(300)
    await flushPromises()
    expect(wrapper.text()).toContain('admin.channelMonitor.displayGroups.noMatches')
    await select(1)
    expect(api.list.mock.lastCall?.[0]).toMatchObject({ display_group_id: 1, search: 'nothing', page: 1 })
    expect((search.element as HTMLInputElement).value).toBe('nothing')
    await search.setValue(''); await vi.advanceTimersByTimeAsync(300); await flushPromises()
    expect(wrapper.text()).toContain('admin.channelMonitor.displayGroups.emptyGroup')
    expect(wrapper.text()).not.toContain('admin.channelMonitor.displayGroups.noMatchesHint')
  })

  it.each(['', -1, 1.5, 2147483648])('rejects invalid group order %s without a write', async (order) => {
    render(); await flushPromises()
    await wrapper.get('[data-testid="create-display-group"]').trigger('click')
    await wrapper.get('#display-group-name').setValue('Valid name')
    await wrapper.get('#display-group-order').setValue(order)
    await wrapper.get('#display-group-form').trigger('submit'); await flushPromises()
    expect(api.createDisplayGroup).not.toHaveBeenCalled()
    expect(api.showError).toHaveBeenCalledWith('admin.channelMonitor.displayGroups.invalidInput')
  })

  it('counts Unicode characters rather than UTF-16 units for group names', async () => {
    const name = String.fromCodePoint(0x1f600).repeat(100)
    api.createDisplayGroup.mockResolvedValue(group(8, name))
    render(); await flushPromises()
    await wrapper.get('[data-testid="create-display-group"]').trigger('click')
    await wrapper.get('#display-group-name').setValue(name)
    await wrapper.get('#display-group-form').trigger('submit'); await flushPromises()
    expect(api.createDisplayGroup).toHaveBeenCalledWith({ name, sort_order: 0 })
    expect(wrapper.get('[data-testid="current-display-group"]').text()).toBe(name)
  })

  it('never lets a late success or error from another group replace the selected group', async () => {
    const pendingA = Promise.withResolvers<ListResponse>()
    const pendingB = Promise.withResolvers<ListResponse>()
    render(); await flushPromises()
    api.list.mockImplementationOnce(() => pendingA.promise)
    await wrapper.get('[data-testid="display-group-1"]').trigger('click')
    api.list.mockImplementationOnce(() => pendingB.promise)
    await wrapper.get('[data-testid="display-group-2"]').trigger('click')
    rows = [monitor({ id: 7, name: 'Current C monitor', display_group_id: 3 })]
    await select(3)
    pendingA.resolve(response([monitor({ name: 'Stale A monitor', display_group_id: 1 })]))
    pendingB.reject(new Error('Stale B error'))
    await flushPromises()
    expect(wrapper.text()).toContain('Current C monitor')
    expect(wrapper.text()).not.toContain('Stale A monitor')
    expect(api.showError).not.toHaveBeenCalledWith('Stale B error')
    expect(wrapper.get('[data-testid="current-display-group"]').text()).toBe('Group C')
  })

  it.each([null, 1])('creates in the group captured at opening (%s), not a later selected group', async (id) => {
    render(); await flushPromises()
    if (id !== null) await select(id)
    await openCreate()
    await select(2)
    await wrapper.get('#channel-monitor-form').trigger('submit'); await flushPromises()
    expect(api.create).toHaveBeenCalledWith(expect.objectContaining({ name: 'New monitor', display_group_id: id }))
    expect(api.create.mock.calls[0][0]).not.toHaveProperty('group_name')
    expect(wrapper.find('#channel-monitor-form').exists()).toBe(false)
    expect(wrapper.get('[data-testid="current-display-group"]').text()).toBe('Group B')
  })

  it('edits configuration without resubmitting hidden legacy labels or display fields', async () => {
    render(); await flushPromises()
    await wrapper.get('[data-testid="monitor-edit"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('input[placeholder="admin.channelMonitor.form.groupNamePlaceholder"]').exists()).toBe(false)
    await wrapper.get('[data-testid="monitor-primary-model"]').setValue('edited-model')
    await select(2)
    await wrapper.get('#channel-monitor-form').trigger('submit'); await flushPromises()
    expect(api.update).toHaveBeenCalledWith(42, expect.objectContaining({ primary_model: 'edited-model' }))
    for (const key of ['group_name', 'display_group_id', 'sort_order', 'api_key']) expect(api.update.mock.calls[0][1]).not.toHaveProperty(key)
  })

  it('moves to a real group and back to explicit null without changing config or order', async () => {
    render(); await flushPromises()
    await wrapper.get('[data-testid="monitor-move"]').trigger('click')
    await wrapper.get('#monitor-move-target').setValue('1')
    await wrapper.get('#move-monitor-form').trigger('submit'); await flushPromises()
    expect(api.updateDisplay).toHaveBeenLastCalledWith(42, { display_group_id: 1 })
    expect(wrapper.text()).not.toContain('Original monitor')
    expect(rows[0]).toMatchObject({ display_group_id: 1, sort_order: 30, group_name: 'Historical label', enabled: true })
    await select(1)
    await wrapper.get('[data-testid="monitor-move"]').trigger('click')
    const storage = wrapper.get('#monitor-move-target').findAll('option').at(-1)!
    await wrapper.get('#monitor-move-target').setValue(storage.element.value)
    await wrapper.get('#move-monitor-form').trigger('submit'); await flushPromises()
    expect(api.updateDisplay).toHaveBeenLastCalledWith(42, { display_group_id: null })
    expect(rows[0].sort_order).toBe(30)
  })

  it('keeps the original membership and open movement dialog when a deleted target fails', async () => {
    api.updateDisplay.mockRejectedValueOnce(new Error('Target no longer exists'))
    render(); await flushPromises()
    await wrapper.get('[data-testid="monitor-move"]').trigger('click')
    await wrapper.get('#monitor-move-target').setValue('1')
    await wrapper.get('#move-monitor-form').trigger('submit'); await flushPromises()
    expect(rows[0].display_group_id).toBeNull()
    expect(wrapper.find('#move-monitor-form').exists()).toBe(true)
    expect(api.showError).toHaveBeenCalledWith('Target no longer exists')
    expect(api.showSuccess).not.toHaveBeenCalled()
  })

  it('keeps failed sort edits distinct from persisted values, then saves and refreshes successfully', async () => {
    api.updateDisplay.mockRejectedValueOnce(new Error('order failed'))
    render(); await flushPromises()
    const input = wrapper.get('[data-testid="monitor-sort-order"]')
    await input.setValue(5)
    expect(api.updateDisplay).not.toHaveBeenCalled()
    await wrapper.get('[data-testid="save-monitor-order"]').trigger('submit'); await flushPromises()
    expect(wrapper.get('[data-testid="saved-monitor-order"]').text()).toContain('30')
    expect(rows[0].sort_order).toBe(30)
    expect((input.element as HTMLInputElement).value).toBe('5')
    expect(api.showError).toHaveBeenCalledWith('order failed')
    expect(api.showSuccess).not.toHaveBeenCalled()
    await wrapper.get('[data-testid="save-monitor-order"]').trigger('submit'); await flushPromises()
    expect(api.updateDisplay).toHaveBeenLastCalledWith(42, { sort_order: 5 })
    expect(wrapper.get('[data-testid="saved-monitor-order"]').text()).toContain('5')
    expect(rows[0].display_group_id).toBeNull()
  })

  it('backs up to a valid page after moving the only monitor on the last page', async () => {
    let moved = false
    api.list.mockImplementation(async (params: ListParams) => {
      const size = params.page_size ?? 20
      return response(params.page === 2 ? (moved ? [] : [monitor()]) : [monitor({ id: 1, name: 'First page' })], params, moved ? size : size + 1)
    })
    api.updateDisplay.mockImplementation(async () => { moved = true; return monitor({ display_group_id: 1 }) })
    render(); await flushPromises()
    await button('pagination.next').trigger('click'); await flushPromises()
    expect(api.list.mock.lastCall?.[0]).toMatchObject({ page: 2 })
    await wrapper.get('[data-testid="monitor-move"]').trigger('click')
    await wrapper.get('#monitor-move-target').setValue('1')
    await wrapper.get('#move-monitor-form').trigger('submit'); await flushPromises()
    expect(api.list.mock.lastCall?.[0]).toMatchObject({ page: 1 })
    expect(wrapper.text()).toContain('First page')
    expect(wrapper.text()).not.toContain('Original monitor')
  })

  it('resets a later page to page one when switching groups', async () => {
    api.list.mockImplementation(async (params: ListParams) => response([monitor()], params, (params.page_size ?? 20) + 1))
    render(); await flushPromises()
    await button('pagination.next').trigger('click'); await flushPromises()
    await select(1)
    expect(api.list.mock.lastCall?.[0]).toMatchObject({ page: 1, display_group_id: 1 })
  })

  it('shows the server refusal to delete a group containing only filtered-out disabled monitors', async () => {
    vi.useFakeTimers()
    rows = [monitor({ enabled: false, display_group_id: 1 })]
    api.deleteDisplayGroup.mockRejectedValueOnce(new Error('Group contains disabled monitors'))
    render(); await flushPromises(); await select(1)
    await wrapper.get('input[placeholder="admin.channelMonitor.searchPlaceholder"]').setValue('nothing')
    await vi.advanceTimersByTimeAsync(300); await flushPromises()
    expect(wrapper.find('[data-testid="monitor-edit"]').exists()).toBe(false)
    await wrapper.get('[data-testid="delete-display-group-1"]').trigger('click')
    await wrapper.get('[data-testid="confirm-delete"]').trigger('click'); await flushPromises()
    expect(api.deleteDisplayGroup).toHaveBeenCalledWith(1)
    expect(api.showError).toHaveBeenCalledWith('Group contains disabled monitors')
    expect(wrapper.find('[data-testid="display-group-1"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="current-display-group"]').text()).toBe('Group A')
    expect(rows).toHaveLength(1)
    expect(api.del).not.toHaveBeenCalled()
  })

  it('deletes only the empty group and returns to storage', async () => {
    api.deleteDisplayGroup.mockResolvedValue(undefined)
    render(); await flushPromises(); await select(1)
    await wrapper.get('[data-testid="delete-display-group-1"]').trigger('click')
    await wrapper.get('[data-testid="confirm-delete"]').trigger('click'); await flushPromises()
    expect(wrapper.find('[data-testid="display-group-1"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="display-group-storage"]').attributes('aria-current')).toBe('true')
    expect(api.list.mock.lastCall?.[0]).toMatchObject({ display_group_id: 0, page: 1 })
    expect(api.del).not.toHaveBeenCalled()
  })

  it('duplicates within the selected group, preserving the disabled-copy workflow and blocking repeated clicks', async () => {
    rows = [monitor({ display_group_id: 1, enabled: true })]
    const pending = Promise.withResolvers<ChannelMonitor>()
    api.duplicate.mockImplementationOnce(() => pending.promise)
    render(); await flushPromises(); await select(1)
    await wrapper.get('[data-testid="monitor-duplicate"]').trigger('click')
    await wrapper.get('[data-testid="monitor-duplicate"]').trigger('click')
    expect(api.duplicate).toHaveBeenCalledTimes(1)
    const copy = monitor({ id: 43, display_group_id: 1, enabled: false, name: 'Disabled copy' })
    rows.push(copy); pending.resolve(copy); await flushPromises()
    expect(api.duplicate).toHaveBeenCalledWith(42)
    expect(wrapper.text()).toContain('Disabled copy')
    expect(api.list.mock.lastCall?.[0]).toMatchObject({ display_group_id: 1 })
    expect(api.showSuccess).toHaveBeenCalledWith(expect.stringContaining('admin.channelMonitor.duplicateSuccess'))
    expect(api.create).not.toHaveBeenCalled()
  })
})
