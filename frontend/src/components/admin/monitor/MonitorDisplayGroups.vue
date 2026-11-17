<template>
  <aside class="min-w-0 space-y-3 rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h2 class="font-semibold">{{ t('admin.channelMonitor.displayGroups.title') }}</h2>
      <button type="button" class="btn btn-secondary" data-testid="create-display-group" :disabled="loading" @click="$emit('create')">{{ t('admin.channelMonitor.displayGroups.create') }}</button>
    </div>
    <p v-if="loading" role="status" class="text-sm text-gray-500">{{ t('common.loading') }}</p>
    <nav class="space-y-2" :aria-label="t('admin.channelMonitor.displayGroups.title')">
      <div v-for="group in groups" :key="group.id" class="min-w-0 rounded-lg border border-gray-200 p-2 dark:border-dark-600">
        <button type="button" class="min-h-11 w-full break-all rounded-lg px-2 py-2 text-left text-sm focus-visible:ring-2 focus-visible:ring-primary-500" :class="selectedId === group.id ? 'bg-primary-50 font-semibold text-primary-700 dark:bg-primary-900/30 dark:text-primary-300' : ''" :aria-current="selectedId === group.id ? 'true' : undefined" :data-testid="`display-group-${group.id}`" @click="$emit('select', group.id)">{{ group.name }}</button>
        <div class="mt-1 flex flex-wrap items-center gap-2 text-xs">
          <span class="text-gray-500">{{ t('admin.channelMonitor.displayGroups.orderValue', { value: group.sort_order }) }}</span>
          <button type="button" class="min-h-11 px-2 text-primary-600" :aria-label="t('admin.channelMonitor.displayGroups.editNamed', { name: group.name })" :data-testid="`edit-display-group-${group.id}`" @click="$emit('edit', group)">{{ t('common.edit') }}</button>
          <button type="button" class="min-h-11 px-2 text-red-600" :aria-label="t('admin.channelMonitor.displayGroups.deleteNamed', { name: group.name })" :data-testid="`delete-display-group-${group.id}`" @click="$emit('delete', group)">{{ t('common.delete') }}</button>
        </div>
      </div>
      <button type="button" class="min-h-11 w-full break-all rounded-lg border border-dashed border-gray-300 px-3 py-3 text-left text-sm dark:border-dark-600" :class="selectedId === null ? 'bg-primary-50 font-semibold text-primary-700 dark:bg-primary-900/30 dark:text-primary-300' : ''" :aria-current="selectedId === null ? 'true' : undefined" data-testid="display-group-storage" @click="$emit('select', null)">{{ t('admin.channelMonitor.displayGroups.storage') }}</button>
    </nav>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { MonitorDisplayGroup } from '@/api/admin/channelMonitor'

defineProps<{ groups: MonitorDisplayGroup[]; selectedId: number | null; loading: boolean }>()
defineEmits<{
  (e: 'select', id: number | null): void
  (e: 'create'): void
  (e: 'edit', group: MonitorDisplayGroup): void
  (e: 'delete', group: MonitorDisplayGroup): void
}>()
const { t } = useI18n()
</script>
