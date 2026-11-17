import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { describe, expect, it, vi } from 'vitest'
import VersionBadge from '../VersionBadge.vue'
import { useAppStore, useAuthStore } from '@/stores'
import { getRollbackVersions } from '@/api/admin/system'
import type * as SystemAPI from '@/api/admin/system'

vi.mock('@/api/admin/system', async (importOriginal) => ({
  ...await importOriginal<typeof SystemAPI>(),
  getRollbackVersions: vi.fn(),
}))

describe('VersionBadge manual rollback commands', () => {
  it('renders fork-specific script and GHCR commands for the selected release', async () => {
    vi.mocked(getRollbackVersions).mockResolvedValue({
      versions: [{ version: '2026.11.16', published_at: '2026-11-16T00:00:00Z', html_url: 'https://github.com/c4lnn/sub2api/releases/tag/v2026.11.16' }],
    })
    const pinia = createPinia()
    setActivePinia(pinia)
    const auth = useAuthStore(pinia)
    auth.user = { role: 'admin' } as typeof auth.user
    const app = useAppStore(pinia)
    app.versionLoaded = true
    app.currentVersion = '2026.11.17'
    app.latestVersion = '2026.11.17'
    app.buildType = 'release'
    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      messages: { en: { version: { rollback: 'Rollback', deployScript: 'Install script', deployDocker: 'Docker' } } },
    })
    const wrapper = mount(VersionBadge, { global: { plugins: [pinia, i18n], stubs: { Icon: true } } })

    await wrapper.get('button').trigger('click')
    await flushPromises()
    const rollbackButton = wrapper.findAll('button').find((button) => button.text().includes('Rollback'))
    expect(rollbackButton).toBeDefined()
    await rollbackButton!.trigger('click')
    await flushPromises()

    const releaseButton = wrapper.findAll('button').find((button) => button.text().includes('v2026.11.16'))
    expect(releaseButton).toBeDefined()
    await releaseButton!.trigger('click')
    await flushPromises()
    expect(wrapper.get('code').text()).toContain('https://raw.githubusercontent.com/c4lnn/sub2api/v2026.11.16/deploy/install.sh')

    const dockerTab = wrapper.findAll('button').find((button) => button.text() === 'Docker')
    expect(dockerTab).toBeDefined()
    await dockerTab!.trigger('click')
    expect(wrapper.get('code').text()).toContain('image: ghcr.io/c4lnn/sub2api:2026.11.16')
    expect(wrapper.get('code').text()).not.toContain(':v2026.11.16')
    wrapper.unmount()
  })
})
