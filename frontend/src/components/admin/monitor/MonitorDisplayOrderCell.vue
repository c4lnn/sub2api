<template>
  <form class="flex flex-wrap items-center gap-2" @submit.prevent="save">
    <input v-model="draft" type="number" min="0" max="2147483647" step="1" required class="input w-28" :aria-label="t('admin.channelMonitor.displayGroups.monitorOrder', { name: row.name })" :disabled="saving" data-testid="monitor-sort-order" />
    <button type="submit" class="btn btn-secondary" :disabled="saving || Number(draft) === row.sort_order" data-testid="save-monitor-order">{{ t('common.save') }}</button>
    <span class="text-xs text-gray-500" data-testid="saved-monitor-order">{{ t('admin.channelMonitor.displayGroups.savedOrder', { value: row.sort_order }) }}</span>
  </form>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ChannelMonitor } from '@/api/admin/channelMonitor'
const props = defineProps<{ row: ChannelMonitor; saving: boolean }>()
const emit = defineEmits<{ (e: 'save', row: ChannelMonitor, order: number): void }>()
const { t } = useI18n()
const draft = ref<number | string>(props.row.sort_order)
watch(() => [props.row.id, props.row.sort_order], () => { draft.value = props.row.sort_order })
function save() {
  const order = Number(draft.value)
  if (draft.value === '' || !Number.isInteger(order) || order < 0 || order > 2147483647 || props.saving) return
  emit('save', props.row, order)
}
</script>
