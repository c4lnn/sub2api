import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  createDisplayGroup, deleteDisplayGroup, duplicate, list, listDisplayGroups,
  updateDisplay, updateDisplayGroup,
} from '@/api/admin/channelMonitor'

const client = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: client }))

beforeEach(() => {
  vi.resetAllMocks()
  localStorage.clear()
  sessionStorage.clear()
})

describe('monitor display API consumer contracts', () => {
  it('consumes the groups items envelope and uses the dedicated group endpoints', async () => {
    const group = { id: 7, name: 'Group', sort_order: 10, created_at: '', updated_at: '' }
    client.get.mockResolvedValueOnce({ data: { items: [group] } })
    expect(await listDisplayGroups()).toEqual({ items: [group] })
    expect(client.get).toHaveBeenCalledWith('/admin/channel-monitor-display-groups')
    client.post.mockResolvedValueOnce({ data: group })
    expect(await createDisplayGroup({ name: 'Group', sort_order: 10 })).toEqual(group)
    expect(client.post).toHaveBeenCalledWith('/admin/channel-monitor-display-groups', { name: 'Group', sort_order: 10 })
    client.put.mockResolvedValueOnce({ data: { ...group, name: 'Renamed' } })
    await updateDisplayGroup(7, { name: 'Renamed' })
    expect(client.put).toHaveBeenCalledWith('/admin/channel-monitor-display-groups/7', { name: 'Renamed' })
    await deleteDisplayGroup(7)
    expect(client.delete).toHaveBeenCalledWith('/admin/channel-monitor-display-groups/7')
  })

  it('does not confuse omitted filters with the storage query sentinel', async () => {
    client.get.mockResolvedValue({ data: { items: [], total: 0 } })
    await list()
    expect(client.get.mock.lastCall?.[1].params).not.toHaveProperty('display_group_id')
    await list({ display_group_id: 0, search: 'query', enabled: false, provider: 'grok', page: 2 })
    expect(client.get.mock.lastCall?.[1].params).toEqual({ display_group_id: 0, search: 'query', enabled: false, provider: 'grok', page: 2 })
    await list({ display_group_id: 7 })
    expect(client.get.mock.lastCall?.[1].params).toEqual({ display_group_id: 7 })
  })

  it('preserves explicit null and omitted membership in narrow display updates', async () => {
    client.put.mockResolvedValue({ data: { id: 42, display_group_id: null, sort_order: 30 } })
    await updateDisplay(42, { display_group_id: null })
    expect(client.put).toHaveBeenLastCalledWith('/admin/channel-monitors/42/display', { display_group_id: null })
    await updateDisplay(42, { sort_order: 0 })
    expect(client.put).toHaveBeenLastCalledWith('/admin/channel-monitors/42/display', { sort_order: 0 })
    expect(client.put.mock.lastCall?.[1]).not.toHaveProperty('display_group_id')
  })

  it('reuses the duplicate operation after an ambiguous failure and consumes inherited display settings', async () => {
    localStorage.setItem('auth_user', JSON.stringify({ id: 7 }))
    const copy = { id: 43, name: 'Disabled copy', enabled: false, group_name: 'Historical label', display_group_id: 7, sort_order: 30 }
    client.post.mockRejectedValueOnce(new Error('network timeout')).mockResolvedValueOnce({ data: copy })
    await expect(duplicate(42)).rejects.toThrow('network timeout')
    expect(await duplicate(42)).toEqual(copy)
    expect(client.post).toHaveBeenCalledTimes(2)
    expect(client.post.mock.calls[0][0]).toBe('/admin/channel-monitors/42/duplicate')
    expect(client.post.mock.calls[0][1]).toBeUndefined()
    expect(client.post.mock.calls[1][2].headers['Idempotency-Key']).toBe(client.post.mock.calls[0][2].headers['Idempotency-Key'])
    expect(sessionStorage.length).toBe(0)
  })
})
