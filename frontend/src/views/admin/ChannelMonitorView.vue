<template>
  <AppLayout>
    <div class="w-full min-w-0 space-y-6 pb-8">
      <header
        class="page-header mb-0 rounded-3xl bg-white p-5 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700 sm:p-6"
      >
        <h1 class="page-title flex items-center gap-2 text-xl font-black text-gray-900 dark:text-white">
          <span class="inline-flex h-8 w-8 items-center justify-center rounded-xl bg-blue-50 text-blue-500 dark:bg-blue-900/30 dark:text-blue-400">
            <Icon name="chart" size="sm" />
          </span>
          {{ t('admin.channelMonitor.title') }}
        </h1>
        <p class="page-description mt-1.5 text-xs text-gray-500 dark:text-gray-400">
          {{ t(`channelMonitorV3.admin.description.${siteMode}`) }}
        </p>
        <div class="mt-4 flex flex-wrap items-center gap-x-3 gap-y-2 rounded-2xl bg-gray-50 px-4 py-3 dark:bg-dark-900/50" data-testid="monitor-mode-switch">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('channelMonitorV3.admin.siteMode') }}</span>
          <div class="inline-flex rounded-lg border border-gray-200 bg-white p-0.5 dark:border-dark-600 dark:bg-dark-800" role="radiogroup" :aria-label="t('channelMonitorV3.admin.siteMode')">
            <button
              v-for="mode in modes"
              :key="mode"
              type="button"
              role="radio"
              :aria-checked="siteMode === mode"
              class="rounded-md px-3 py-1 text-xs font-medium transition disabled:cursor-not-allowed"
              :class="siteMode === mode ? 'bg-primary-600 text-white shadow-sm' : 'text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-white'"
              :disabled="switchingMode || siteMode === 'off'"
              :data-testid="`monitor-mode-${mode}`"
              @click="pendingMode = mode"
            >
              {{ t(`channelMonitorV3.modes.${mode}`) }}
            </button>
          </div>
          <span class="text-xs text-gray-500 dark:text-gray-400">
            {{ siteMode === 'off' ? t('channelMonitorV3.admin.siteModeOff') : t('channelMonitorV3.admin.siteModeHint') }}
            <router-link v-if="siteMode === 'off'" to="/admin/settings" class="ml-1 text-primary-600 hover:underline dark:text-primary-400">{{ t('admin.settings.tabs.features') }}</router-link>
          </span>
        </div>
        <div class="mt-4 border-t border-gray-100 pt-4 dark:border-dark-700">
          <div
            class="tabs inline-flex w-full max-w-2xl flex-wrap sm:w-auto"
            role="tablist"
            :aria-label="t('channelMonitorV2.admin.tabAria')"
          >
            <button
              type="button"
              role="tab"
              class="tab flex-1 sm:flex-none"
              :class="adminMonitorTab === 'v3' ? 'tab-active' : ''"
              :aria-selected="adminMonitorTab === 'v3'"
              data-testid="monitor-tab-v3"
              @click="adminMonitorTab = 'v3'"
            >
              {{ t('channelMonitorV3.admin.tab') }}
            </button>
            <button
              type="button"
              role="tab"
              class="tab flex-1 sm:flex-none"
              :class="adminMonitorTab === 'v2' ? 'tab-active' : ''"
              :aria-selected="adminMonitorTab === 'v2'"
              @click="adminMonitorTab = 'v2'"
            >
              {{ t('channelMonitorV2.admin.tabV2') }}
            </button>
            <button
              type="button"
              role="tab"
              class="tab flex-1 sm:flex-none"
              :class="adminMonitorTab === 'legacy' ? 'tab-active' : ''"
              :aria-selected="adminMonitorTab === 'legacy'"
              @click="adminMonitorTab = 'legacy'"
            >
              {{ isV1Mode ? t('channelMonitorV2.admin.tabV1Active') : t('channelMonitorV2.admin.tabV1History') }}
            </button>
          </div>
        </div>
      </header>

      <V3SettingsPanel v-if="adminMonitorTab === 'v3'" />
      <MonitorSettingsPanel v-else-if="adminMonitorTab === 'v2'" />

      <div v-else class="grid min-w-0 items-start gap-4 lg:grid-cols-[16rem_minmax(0,1fr)]">
        <MonitorDisplayGroups
          :groups="displayGroups"
          :selected-id="selectedGroupId"
          :loading="groupsLoading"
          @select="selectGroup"
          @create="openGroupDialog()"
          @edit="openGroupDialog"
          @delete="deletingGroup = $event"
        />
        <div class="min-w-0 space-y-4">
          <div class="min-w-0 rounded-2xl bg-white p-4 dark:bg-dark-800">
            <h2 class="break-all text-lg font-semibold" data-testid="current-display-group">{{ currentGroupName }}</h2>
            <p v-if="selectedGroupId === null" class="mt-2 text-sm text-gray-500">{{ t('admin.channelMonitor.displayGroups.storageHint') }}</p>
          </div>
          <TablePageLayout>
      <template #filters>
        <MonitorFiltersBar
          v-model:search="searchQuery"
          v-model:provider="providerFilter"
          v-model:enabled="enabledFilter"
          :loading="loading"
          @reload="reload"
          @create="openCreateDialog"
          @manage-templates="showTemplateManager = true"
          @search-input="handleSearch"
        />
      </template>

      <template #table>
        <DataTable :columns="columns" :data="monitors" :loading="loading">
          <template #cell-name="{ row, value }">
            <div class="flex items-center gap-1.5">
              <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
              <HelpTooltip v-if="row.api_key_decrypt_failed" :content="t('admin.channelMonitor.apiKeyDecryptFailed')">
                <Icon name="exclamationTriangle" size="sm" class="text-red-500" />
              </HelpTooltip>
            </div>
          </template>

          <template #cell-provider="{ row }">
            <span class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium" :class="providerBadgeClass(row.provider)">
              {{ providerLabel(row.provider) }}
            </span>
            <!-- 三种检测模式并列展示，quota 系配额数据源与纯探活一眼可分 -->
            <span class="ml-1 inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium" :class="checkModeBadgeClass(row.check_mode)">
              {{ checkModeLabel(row.check_mode) }}
            </span>
          </template>

          <template #cell-primary_model="{ row }">
            <MonitorPrimaryModelCell :row="row" />
          </template>

          <template #cell-availability_7d="{ row }">
            <span class="text-sm text-gray-900 dark:text-gray-100">{{ formatAvailability(row) }}</span>
          </template>

          <template #cell-latency="{ row }">
            <span class="text-sm text-gray-900 dark:text-gray-100">{{ formatLatency(row.primary_latency_ms) }}</span>
          </template>

          <template #cell-enabled="{ row }">
            <Toggle :modelValue="row.enabled" @update:modelValue="toggleEnabled(row)" />
          </template>

          <template #cell-sort_order="{ row }">
            <MonitorDisplayOrderCell :row="row" :saving="savingDisplayIds.has(row.id)" @save="saveMonitorOrder" />
          </template>

          <template #cell-actions="{ row }">
            <MonitorActionsCell
              :row="row"
              :running="runningId === row.id"
              :duplicating="duplicatingIds.has(row.id)"
              @run="handleRunNow"
              @duplicate="handleDuplicate"
              @edit="openEditDialog"
              @delete="handleDelete"
              @move="openMoveDialog"
            />
          </template>

          <template #empty>
            <EmptyState
              :title="t(hasFilters ? 'admin.channelMonitor.displayGroups.noMatches' : 'admin.channelMonitor.displayGroups.emptyGroup')"
              :description="t(hasFilters ? 'admin.channelMonitor.displayGroups.noMatchesHint' : 'admin.channelMonitor.createFirstMonitor')"
              :action-text="hasFilters ? undefined : t('admin.channelMonitor.createButton')"
              @action="openCreateDialog"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="onPageChange"
          @update:pageSize="onPageSizeChange"
        />
      </template>
          </TablePageLayout>
        </div>
      </div>
    </div>

    <MonitorFormDialog
      :show="showDialog"
      :monitor="editing"
      :display-group-id="createGroupContext"
      @close="closeDialog"
      @saved="reload"
    />

    <MonitorTemplateManagerDialog
      :show="showTemplateManager"
      @close="showTemplateManager = false"
      @updated="reload"
    />

    <MonitorRunResultDialog
      :show="showRunResult"
      :results="runResults"
      @close="showRunResult = false"
    />

    <BaseDialog :show="showGroupDialog" :title="t(editingGroup ? 'admin.channelMonitor.displayGroups.edit' : 'admin.channelMonitor.displayGroups.create')" @close="closeGroupDialog">
      <form id="display-group-form" class="space-y-4" @submit.prevent="saveGroup">
        <label class="block input-label" for="display-group-name">{{ t('admin.channelMonitor.displayGroups.name') }}</label>
        <input id="display-group-name" v-model="groupForm.name" class="input" required :disabled="groupSaving" />
        <label class="block input-label" for="display-group-order">{{ t('admin.channelMonitor.displayGroups.order') }}</label>
        <input id="display-group-order" v-model="groupForm.sort_order" class="input" type="number" min="0" max="2147483647" step="1" required :disabled="groupSaving" />
        <p class="text-sm text-gray-500">{{ t('admin.channelMonitor.displayGroups.orderHint') }}</p>
      </form>
      <template #footer>
        <button class="btn btn-secondary" type="button" :disabled="groupSaving" @click="closeGroupDialog">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" form="display-group-form" type="submit" :disabled="groupSaving">{{ t('common.save') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="movingMonitor !== null" :title="t('admin.channelMonitor.displayGroups.move')" @close="closeMoveDialog">
      <form id="move-monitor-form" class="space-y-4" @submit.prevent="moveMonitor">
        <p class="break-all">{{ movingMonitor?.name }}</p>
        <label class="block input-label" for="monitor-move-target">{{ t('admin.channelMonitor.displayGroups.target') }}</label>
        <select id="monitor-move-target" v-model="moveTarget" class="input w-full" :disabled="moveSaving">
          <option v-for="group in displayGroups" :key="group.id" :value="group.id">{{ group.name }}</option>
          <option :value="null">{{ t('admin.channelMonitor.displayGroups.storage') }}</option>
        </select>
      </form>
      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="moveSaving" @click="closeMoveDialog">{{ t('common.cancel') }}</button>
        <button type="submit" class="btn btn-primary" form="move-monitor-form" :disabled="moveSaving || moveTarget === movingMonitor?.display_group_id">{{ t('common.save') }}</button>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="deletingGroup !== null"
      :title="t('admin.channelMonitor.displayGroups.delete')"
      :message="t('admin.channelMonitor.displayGroups.deleteConfirm', { name: deletingGroup?.name ?? '' })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="deleteGroup"
      @cancel="deletingGroup = null"
    />

    <ConfirmDialog
      :show="pendingMode !== null"
      :title="t('channelMonitorV3.admin.switchTitle')"
      :message="pendingMode ? t('channelMonitorV3.admin.switchConfirm', { mode: t(`channelMonitorV3.modes.${pendingMode}`), effect: t(`channelMonitorV3.admin.switchEffect.${pendingMode}`) }) : ''"
      :confirm-text="t('channelMonitorV3.admin.switchAction')"
      :cancel-text="t('common.cancel')"
      @confirm="applyMode"
      @cancel="pendingMode = null"
    />

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('common.delete')"
      :message="deleteConfirmMessage"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { adminAPI } from '@/api/admin'
import type {
  ChannelMonitor,
  CheckResult,
  ListParams,
  Provider,
  MonitorDisplayGroup,
} from '@/api/admin/channelMonitor'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'
import MonitorFiltersBar from '@/components/admin/monitor/MonitorFiltersBar.vue'
import MonitorFormDialog from '@/components/admin/monitor/MonitorFormDialog.vue'
import MonitorTemplateManagerDialog from '@/components/admin/monitor/MonitorTemplateManagerDialog.vue'
import MonitorRunResultDialog from '@/components/admin/monitor/MonitorRunResultDialog.vue'
import MonitorPrimaryModelCell from '@/components/admin/monitor/MonitorPrimaryModelCell.vue'
import MonitorActionsCell from '@/components/admin/monitor/MonitorActionsCell.vue'
import MonitorDisplayGroups from '@/components/admin/monitor/MonitorDisplayGroups.vue'
import MonitorDisplayOrderCell from '@/components/admin/monitor/MonitorDisplayOrderCell.vue'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'
import MonitorSettingsPanel from '@/features/channel-monitor-v2/MonitorSettingsPanel.vue'
import V3SettingsPanel from '@/features/channel-monitor-v3/V3SettingsPanel.vue'
import { setMonitorMode } from '@/api/channelMonitorV3'
import { getChannelMonitorMode, isChannelMonitorV1Mode, type ChannelMonitorMode } from '@/utils/featureFlags'

const { t } = useI18n()
const appStore = useAppStore()
const isV1Mode = computed(() => isChannelMonitorV1Mode())
const modes: ChannelMonitorMode[] = ['v1', 'v2', 'v3']
const siteMode = computed(() => (appStore.cachedPublicSettings?.channel_monitor_enabled === false ? 'off' : getChannelMonitorMode()))
const tabForMode = (mode: ChannelMonitorMode) => (mode === 'v1' ? 'legacy' : mode)
const adminMonitorTab = ref<'v3' | 'v2' | 'legacy'>(tabForMode(getChannelMonitorMode()))
const pendingMode = ref<ChannelMonitorMode | null>(null)
const switchingMode = ref(false)

async function applyMode() {
  const mode = pendingMode.value
  pendingMode.value = null
  if (!mode || mode === siteMode.value) return
  switchingMode.value = true
  try {
    await setMonitorMode(mode)
    // Sidebar, the user page and the runners all follow the public settings.
    await appStore.fetchPublicSettings(true)
    adminMonitorTab.value = tabForMode(mode)
    appStore.showSuccess(t('channelMonitorV3.admin.switched', { mode: t(`channelMonitorV3.modes.${mode}`) }))
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('channelMonitorV3.admin.switchFailed')))
  } finally {
    switchingMode.value = false
  }
}
const {
  providerLabel,
  providerBadgeClass,
  checkModeLabel,
  checkModeBadgeClass,
  formatLatency,
  formatAvailability,
} = useChannelMonitorFormat()

const monitors = ref<ChannelMonitor[]>([])
const loading = ref(false)
const runningId = ref<number | null>(null)
const searchQuery = ref('')
const providerFilter = ref<Provider | ''>('')
const enabledFilter = ref<'' | 'true' | 'false'>('')
const pagination = reactive({ page: 1, page_size: getPersistedPageSize(), total: 0 })
const displayGroups = ref<MonitorDisplayGroup[]>([])
const groupsLoading = ref(false)
const selectedGroupId = ref<number | null>(null)
const currentGroupName = computed(() => selectedGroupId.value === null
  ? t('admin.channelMonitor.displayGroups.storage')
  : displayGroups.value.find(group => group.id === selectedGroupId.value)?.name ?? '')
const hasFilters = computed(() => Boolean(searchQuery.value.trim() || providerFilter.value || enabledFilter.value))
const createGroupContext = ref<number | null>(null)
const showGroupDialog = ref(false)
const editingGroup = ref<MonitorDisplayGroup | null>(null)
const groupForm = reactive<{ name: string; sort_order: number | string }>({ name: '', sort_order: 0 })
const groupSaving = ref(false)
const deletingGroup = ref<MonitorDisplayGroup | null>(null)
const groupDeleting = ref(false)
const movingMonitor = ref<ChannelMonitor | null>(null)
const moveTarget = ref<number | null>(null)
const moveSaving = ref(false)
const savingDisplayIds = reactive(new Set<number>())

const showDialog = ref(false)
const showTemplateManager = ref(false)
const editing = ref<ChannelMonitor | null>(null)
const showDeleteDialog = ref(false)
const deleting = ref<ChannelMonitor | null>(null)
const showRunResult = ref(false)
const runResults = ref<CheckResult[]>([])
const duplicatingIds = reactive(new Set<number>())

let abortController: AbortController | null = null
let searchTimeout: ReturnType<typeof setTimeout> | null = null
let groupsRequest = 0

const columns = computed<Column[]>(() => [
  { key: 'name', label: t('admin.channelMonitor.columns.name'), sortable: false },
  { key: 'provider', label: t('admin.channelMonitor.columns.provider'), sortable: false },
  { key: 'primary_model', label: t('admin.channelMonitor.columns.primaryModel'), sortable: false },
  { key: 'availability_7d', label: t('admin.channelMonitor.columns.availability7d'), sortable: false },
  { key: 'latency', label: t('admin.channelMonitor.columns.latency'), sortable: false },
  { key: 'enabled', label: t('admin.channelMonitor.columns.enabled'), sortable: false },
  { key: 'sort_order', label: t('admin.channelMonitor.displayGroups.order'), sortable: false },
  { key: 'actions', label: t('admin.channelMonitor.columns.actions'), sortable: false },
])

const deleteConfirmMessage = computed(() => {
  const name = deleting.value?.name || ''
  return t('admin.channelMonitor.deleteConfirm', { name })
})

async function reload() {
  if (abortController) abortController.abort()
  const ctrl = new AbortController()
  abortController = ctrl
  loading.value = true
  try {
    const params: ListParams = {
      page: pagination.page,
      page_size: pagination.page_size,
      display_group_id: selectedGroupId.value ?? 0,
    }
    if (providerFilter.value) params.provider = providerFilter.value
    if (enabledFilter.value === 'true') params.enabled = true
    if (enabledFilter.value === 'false') params.enabled = false
    if (searchQuery.value.trim()) params.search = searchQuery.value.trim()

    const res = await adminAPI.channelMonitor.list(params, { signal: ctrl.signal })
    if (ctrl.signal.aborted || abortController !== ctrl) return
    const lastPage = Math.max(1, Math.ceil(res.total / pagination.page_size))
    if (pagination.page > lastPage) {
      pagination.page = lastPage
      await reload()
      return
    }
    monitors.value = res.items || []
    pagination.total = res.total
  } catch (err: unknown) {
    if (ctrl.signal.aborted || abortController !== ctrl) return
    const e = err as { name?: string; code?: string }
    if (e?.name === 'AbortError' || e?.code === 'ERR_CANCELED') return
    appStore.showError(extractApiErrorMessage(err, t('admin.channelMonitor.loadError')))
  } finally {
    if (abortController === ctrl) {
      loading.value = false
      abortController = null
    }
  }
}

function handleSearch() {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    pagination.page = 1
    reload()
  }, 300)
}

function onPageChange(page: number) {
  pagination.page = page
  reload()
}

function onPageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  reload()
}

function openCreateDialog() {
  createGroupContext.value = selectedGroupId.value
  editing.value = null
  showDialog.value = true
}

function openEditDialog(row: ChannelMonitor) {
  editing.value = row
  showDialog.value = true
}

function closeDialog() {
  showDialog.value = false
  editing.value = null
}

async function toggleEnabled(row: ChannelMonitor) {
  const next = !row.enabled
  try {
    await adminAPI.channelMonitor.update(row.id, { enabled: next })
    row.enabled = next
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}

async function handleRunNow(row: ChannelMonitor) {
  if (!isV1Mode.value) {
    appStore.showError(t('admin.channelMonitor.runFailed'))
    return
  }
  if (runningId.value != null) return
  runningId.value = row.id
  try {
    const res = await adminAPI.channelMonitor.runNow(row.id)
    runResults.value = res.results || []
    showRunResult.value = true
    appStore.showSuccess(t('admin.channelMonitor.runSuccess'))
    // Refresh row to get latest status from backend
    void reload()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.channelMonitor.runFailed')))
  } finally {
    runningId.value = null
  }
}

async function handleDuplicate(row: ChannelMonitor) {
  if (row.api_key_decrypt_failed) {
    appStore.showError(t('admin.channelMonitor.duplicateKeyUnavailable'))
    return
  }
  if (duplicatingIds.has(row.id)) return

  duplicatingIds.add(row.id)
  try {
    const duplicate = await adminAPI.channelMonitor.duplicate(row.id)
    appStore.showSuccess(t('admin.channelMonitor.duplicateSuccess', { name: duplicate.name }))
    await reload()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.channelMonitor.duplicateFailed')))
  } finally {
    duplicatingIds.delete(row.id)
  }
}

function handleDelete(row: ChannelMonitor) {
  deleting.value = row
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!deleting.value) return
  try {
    await adminAPI.channelMonitor.del(deleting.value.id)
    appStore.showSuccess(t('admin.channelMonitor.deleteSuccess'))
    showDeleteDialog.value = false
    deleting.value = null
    reload()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}

async function loadGroups() {
  const request = ++groupsRequest
  groupsLoading.value = true
  try {
    const response = await adminAPI.channelMonitor.listDisplayGroups()
    if (request !== groupsRequest) return
    displayGroups.value = response.items.slice().sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
  } catch (err: unknown) {
    if (request !== groupsRequest) return
    appStore.showError(extractApiErrorMessage(err, t('admin.channelMonitor.displayGroups.loadError')))
  } finally {
    if (request === groupsRequest) groupsLoading.value = false
  }
}

function selectGroup(id: number | null) {
  if (selectedGroupId.value === id) return
  if (searchTimeout) clearTimeout(searchTimeout)
  selectedGroupId.value = id
  pagination.page = 1
  monitors.value = []
  pagination.total = 0
  void reload()
}

function openGroupDialog(group?: MonitorDisplayGroup) {
  editingGroup.value = group ?? null
  groupForm.name = group?.name ?? ''
  groupForm.sort_order = group?.sort_order ?? 0
  showGroupDialog.value = true
}

function closeGroupDialog() {
  if (!groupSaving.value) showGroupDialog.value = false
}

function validOrder(value: number | string) {
  return value !== '' && Number.isInteger(Number(value)) && Number(value) >= 0 && Number(value) <= 2147483647
}

async function saveGroup() {
  if (groupSaving.value) return
  const name = groupForm.name.trim()
  if (!name || Array.from(name).length > 100 || !validOrder(groupForm.sort_order)) {
    appStore.showError(t('admin.channelMonitor.displayGroups.invalidInput'))
    return
  }
  const target = editingGroup.value
  groupSaving.value = true
  try {
    const payload = { name, sort_order: Number(groupForm.sort_order) }
    const saved = target
      ? await adminAPI.channelMonitor.updateDisplayGroup(target.id, payload)
      : await adminAPI.channelMonitor.createDisplayGroup(payload)
    groupsRequest++
    groupsLoading.value = false
    displayGroups.value = [...displayGroups.value.filter(group => group.id !== saved.id), saved]
      .sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
    showGroupDialog.value = false
    if (!target) selectGroup(saved.id)
    appStore.showSuccess(t('admin.channelMonitor.displayGroups.saved'))
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    groupSaving.value = false
  }
}

async function deleteGroup() {
  const target = deletingGroup.value
  if (!target || groupDeleting.value) return
  groupDeleting.value = true
  try {
    await adminAPI.channelMonitor.deleteDisplayGroup(target.id)
    groupsRequest++
    groupsLoading.value = false
    displayGroups.value = displayGroups.value.filter(group => group.id !== target.id)
    deletingGroup.value = null
    if (selectedGroupId.value === null) { pagination.page = 1; await reload() }
    else selectGroup(null)
    appStore.showSuccess(t('admin.channelMonitor.displayGroups.deleted'))
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    groupDeleting.value = false
  }
}

function openMoveDialog(row: ChannelMonitor) {
  movingMonitor.value = row
  moveTarget.value = row.display_group_id
}

function closeMoveDialog() {
  if (!moveSaving.value) movingMonitor.value = null
}

async function moveMonitor() {
  const target = movingMonitor.value
  if (!target || moveSaving.value) return
  const destination = moveTarget.value
  moveSaving.value = true
  try {
    await adminAPI.channelMonitor.updateDisplay(target.id, { display_group_id: destination })
    movingMonitor.value = null
    appStore.showSuccess(t('admin.channelMonitor.displayGroups.moved'))
    await reload()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    moveSaving.value = false
  }
}

async function saveMonitorOrder(row: ChannelMonitor, order: number) {
  if (savingDisplayIds.has(row.id) || !validOrder(order)) return
  savingDisplayIds.add(row.id)
  try {
    const saved = await adminAPI.channelMonitor.updateDisplay(row.id, { sort_order: order })
    row.sort_order = saved.sort_order
    appStore.showSuccess(t('admin.channelMonitor.displayGroups.saved'))
    await reload()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    savingDisplayIds.delete(row.id)
  }
}

watch([providerFilter, enabledFilter], () => {
  pagination.page = 1
  void reload()
})

watch(adminMonitorTab, (tab) => {
  if (tab === 'legacy') { void loadGroups(); void reload() }
})
onMounted(() => {
  if (adminMonitorTab.value === 'legacy') { void loadGroups(); void reload() }
})
onUnmounted(() => {
  if (searchTimeout) clearTimeout(searchTimeout)
  groupsRequest++
  abortController?.abort()
})
</script>
