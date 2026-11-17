import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount, type VueWrapper } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import type { UserMonitorView } from '@/api/channelMonitor'
import zh from '@/i18n/locales/zh'
import en from '@/i18n/locales/en'
import MonitorCardGrid from '../MonitorCardGrid.vue'
import MonitorCard from '../MonitorCard.vue'

vi.mock('@/utils/featureFlags', () => ({ isChannelMonitorQuotaVisible: () => false }))
enableAutoUnmount(afterEach)

function item(id: number, groupId: number, groupName: string, overrides: Partial<UserMonitorView> = {}): UserMonitorView {
  return {
    id,
    name: `Channel ${id}`,
    provider: 'openai',
    group_name: 'legacy-only',
    display_group_id: groupId,
    display_group_name: groupName,
    display_group_sort_order: 0,
    sort_order: 0,
    primary_model: `model-${id}`,
    primary_status: 'operational',
    primary_latency_ms: 100,
    primary_ping_latency_ms: 20,
    availability_7d: 99,
    extra_models: [],
    timeline: [],
    ...overrides,
  }
}

function render(items: UserMonitorView[], loading = false, locale: 'zh' | 'en' = 'zh') {
  return mount(MonitorCardGrid, {
    props: { items, loading, window: '7d', countdownSeconds: 30, detailCache: {} },
    global: {
      plugins: [createI18n({ legacy: false, locale, messages: { zh, en } })],
      stubs: { MonitorMetricPair: true, MonitorAvailabilityRow: true, MonitorTimeline: true },
    },
  })
}

function headings(wrapper: VueWrapper) {
  return wrapper.findAll('h2').map(heading => heading.text())
}

function visibleGroups(wrapper: VueWrapper) {
  return wrapper.findAll('section').map(section => ({
    name: section.get('header').text(),
    cards: section.findAllComponents(MonitorCard).map(card => card.text()),
  }))
}

describe('V1 monitor display groups', () => {
  it('sorts interleaved groups and cards by persistent order then ID across providers', () => {
    const wrapper = render([
      item(8, 3, 'A', { display_group_sort_order: 20 }),
      item(7, 2, 'B', { sort_order: 30 }),
      item(6, 2, 'B', { sort_order: 10, provider: 'anthropic' }),
      item(4, 1, 'C'), item(5, 2, 'B', { sort_order: 10 }),
    ])
    expect(visibleGroups(wrapper)).toEqual([
      { name: 'C', cards: [expect.stringContaining('Channel 4')] },
      { name: 'B', cards: [5, 6, 7].map(id => expect.stringContaining(`Channel ${id}`)) },
      { name: 'A', cards: [expect.stringContaining('Channel 8')] },
    ])
    expect(wrapper.findAllComponents(MonitorCard)).toHaveLength(5)
  })

  it('uses formal IDs despite colliding legacy labels and names resembling the old fallback', () => {
    const wrapper = render([
      item(1, 10, '未分组', { group_name: 'Shared legacy' }),
      item(2, 20, 'Shared legacy', { group_name: 'Shared legacy' }),
      item(3, 10, '未分组', { group_name: 'Other legacy', provider: 'anthropic' }),
    ])
    expect(visibleGroups(wrapper)).toEqual([
      { name: '未分组', cards: [expect.stringContaining('Channel 1'), expect.stringContaining('Channel 3')] },
      { name: 'Shared legacy', cards: [expect.stringContaining('Channel 2')] },
    ])
    for (const card of wrapper.findAllComponents(MonitorCard)) {
      expect(card.text()).not.toContain('Shared legacy')
      expect(card.text()).not.toContain('Other legacy')
      expect(card.text()).not.toContain('未分组')
    }
  })

  it('does not merge different group IDs even when their displayed names collide during refresh', () => {
    const wrapper = render([item(2, 20, 'Same'), item(1, 10, 'Same')])
    expect(visibleGroups(wrapper)).toEqual([
      { name: 'Same', cards: [expect.stringContaining('Channel 1')] },
      { name: 'Same', cards: [expect.stringContaining('Channel 2')] },
    ])
  })

  it('updates formal names, membership and ordering without retaining empty sections', async () => {
    const wrapper = render([item(1, 10, 'A'), item(2, 20, 'B')])
    await wrapper.setProps({ items: [
      item(1, 20, 'Renamed B', { sort_order: 20 }),
      item(2, 20, 'Renamed B', { sort_order: 10 }),
    ] })
    expect(visibleGroups(wrapper)).toEqual([
      { name: 'Renamed B', cards: [expect.stringContaining('Channel 2'), expect.stringContaining('Channel 1')] },
    ])
    await wrapper.setProps({ items: [] })
    expect(headings(wrapper)).toEqual([])
    expect(wrapper.findAllComponents(MonitorCard)).toHaveLength(0)
    expect(wrapper.text()).toContain('暂无可显示的渠道')
  })

  it('sorts reference arrays without mutating props or copying timelines on countdown ticks', async () => {
    const rows = [item(2, 10, 'A'), item(1, 10, 'A')]
    for (const row of rows) {
      Object.freeze(row.timeline)
      Object.freeze(row)
    }
    Object.freeze(rows)
    const wrapper = render(rows)
    const cards = wrapper.findAllComponents(MonitorCard)
    expect(cards.map(card => card.props('item'))).toEqual([rows[1], rows[0]])
    expect(cards[0]!.props('item')).toBe(rows[1])
    expect(cards[0]!.props('item').timeline).toBe(rows[1]!.timeline)
    await wrapper.setProps({ countdownSeconds: 29 })
    expect(wrapper.findAllComponents(MonitorCard)[0]!.props('item')).toBe(rows[1])
    expect(rows.map(row => row.id)).toEqual([2, 1])
  })

  it.each(['zh', 'en'] as const)('shows only the formal title in %s and preserves card click identity and abnormal status', async (locale) => {
    const longName = 'LongFormalGroup'.repeat(6)
    const row = item(1, 10, longName, { primary_status: 'failed' })
    const wrapper = render([row], false, locale)
    expect(headings(wrapper)).toEqual([longName])
    expect(wrapper.get('header').text()).toBe(longName)
    const card = wrapper.getComponent(MonitorCard)
    expect(card.text()).not.toContain(longName)
    expect(card.text()).not.toContain('legacy-only')
    for (const label of ['Channel 1', 'OpenAI', 'model-1']) expect(card.text()).toContain(label)
    expect(card.props('item').primary_status).toBe('failed')
    await card.trigger('click')
    expect(wrapper.emitted('cardClick')?.[0]?.[0]).toEqual(row)
  })

  it('shows the loading skeleton and then the original empty state, without empty sections', async () => {
    const wrapper = render([], true)
    expect(wrapper.find('.animate-pulse').exists()).toBe(true)
    expect(headings(wrapper)).toEqual([])
    await wrapper.setProps({ loading: false })
    expect(wrapper.find('.animate-pulse').exists()).toBe(false)
    expect(wrapper.text()).toContain('暂无可显示的渠道')
    expect(wrapper.findAll('section')).toHaveLength(0)
  })

  it('keeps existing sections visible during refresh', async () => {
    const wrapper = render([item(1, 10, 'A')])
    await wrapper.setProps({ loading: true })
    expect(headings(wrapper)).toEqual(['A'])
    expect(wrapper.getComponent(MonitorCard).text()).toContain('Channel 1')
    expect(wrapper.find('.animate-pulse').exists()).toBe(false)
  })
})
