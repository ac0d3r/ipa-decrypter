<template>
  <section class="panel">
    <PanelHeader :title="t('tasks.listTitle')">
      <IconButton icon="refresh" :title="t('common.refresh')" :busy="busy" @click="load" />
    </PanelHeader>

    <table class="table">
      <thead>
        <tr>
          <th>{{ t('tasks.colApp') }}</th>
          <th>{{ t('tasks.colRegion') }}</th>
          <th>{{ t('tasks.colStatus') }}</th>
          <th>{{ t('tasks.colTime') }}</th>
          <th>{{ t('tasks.colActions') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="!items.length">
          <td colspan="5" class="muted">{{ t('common.noData') }}</td>
        </tr>
        <tr v-for="item in items" :key="item.id">
          <td>
            <div>{{ item.app_name }}</div>
            <div class="muted">{{ item.bundle_id }}</div>
          </td>
          <td>
            {{ item.region }}
            <span v-if="item.display_version" class="muted"> · v{{ item.display_version }}</span>
            <span v-else-if="item.external_version_id" class="muted"> · {{ item.external_version_id }}</span>
          </td>
          <td>
            <span class="badge" :class="statusClass(item.status)">{{ taskStatus(item.status) }}</span>
          </td>
          <td class="muted">{{ fmt(item.updated_at) }}</td>
          <td>
            <div class="table-actions">
              <IconButton
                v-if="canDelete(item)"
                icon="trash"
                variant="danger"
                :title="t('common.delete')"
                :disabled="busy"
                @click="remove(item)"
              />
            </div>
          </td>
        </tr>
      </tbody>
    </table>
    <p v-if="error" class="error">{{ error }}</p>
  </section>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { adminApi, statusClass } from '../../api'
import IconButton from '../../components/IconButton.vue'
import PanelHeader from '../../components/PanelHeader.vue'
import { useI18n } from '../../i18n'

const { t, locale } = useI18n()
const items = ref([])
const busy = ref(false)
const error = ref('')

function taskStatus(status) {
  return t(`tasks.status.${status}`) || status
}

function canDelete(item) {
  return item.status === 'completed' || item.status === 'failed'
}

async function load() {
  busy.value = true
  try {
    items.value = await adminApi.listTasks()
  } finally {
    busy.value = false
  }
}

async function remove(item) {
  if (!canDelete(item)) return
  if (!confirm(t('tasks.deleteConfirm', { name: item.app_name || item.bundle_id }))) return
  busy.value = true
  error.value = ''
  try {
    await adminApi.deleteTask(item.id)
    await load()
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

function fmt(s) {
  try {
    return new Date(s).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US')
  } catch {
    return s
  }
}

onMounted(load)
</script>
