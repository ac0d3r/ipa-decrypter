<template>
  <section class="panel">
    <PanelHeader :title="t('devices.listTitle')">
      <IconButton icon="plus" variant="primary" :title="t('common.add')" @click="openAdd" />
      <IconButton icon="refresh" :title="t('common.refresh')" :busy="busy" @click="load" />
    </PanelHeader>

    <table class="table">
      <thead>
        <tr>
          <th>{{ t('devices.colName') }}</th>
          <th>{{ t('devices.colAddress') }}</th>
          <th>{{ t('devices.colStatus') }}</th>
          <th>{{ t('devices.colActions') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="!items.length">
          <td colspan="4" class="muted">{{ t('common.noData') }}</td>
        </tr>
        <tr v-for="d in items" :key="d.id">
          <td>{{ d.name }}</td>
          <td>{{ d.host }}:{{ d.port }}</td>
          <td>
            <span class="badge" :class="statusClass(d.status)">{{ deviceStatus(d.status) }}</span>
            <div v-if="d.last_error" class="error">{{ d.last_error }}</div>
          </td>
          <td>
            <div class="table-actions">
              <IconButton icon="pulse" :title="t('common.health')" :disabled="busy" @click="health(d)" />
              <IconButton icon="trash" variant="danger" :title="t('common.delete')" :disabled="busy" @click="remove(d)" />
            </div>
          </td>
        </tr>
      </tbody>
    </table>
    <p v-if="error" class="error">{{ error }}</p>
  </section>

  <div v-if="addOpen" class="modal-backdrop" @click.self="closeAdd">
    <div class="panel modal">
      <h2>{{ t('devices.addTitle') }}</h2>
      <div class="grid two">
        <input v-model="form.name" :placeholder="t('devices.name')" />
        <input v-model="form.host" :placeholder="t('devices.host')" />
        <input v-model.number="form.port" type="number" :placeholder="t('devices.port')" />
      </div>
      <div class="modal-actions">
        <button @click="closeAdd">{{ t('common.cancel') }}</button>
        <button class="primary" @click="create" :disabled="busy">{{ t('devices.addProbe') }}</button>
      </div>
      <p v-if="addError" class="error">{{ addError }}</p>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { adminApi, statusClass } from '../api'
import IconButton from '../components/IconButton.vue'
import PanelHeader from '../components/PanelHeader.vue'
import { useI18n } from '../i18n'

const { t } = useI18n()
const items = ref([])
const busy = ref(false)
const error = ref('')
const addOpen = ref(false)
const addError = ref('')
const form = reactive({ name: '', host: '', port: 6347 })

function deviceStatus(status) {
  return t(`devices.status.${status}`) || status
}

function openAdd() {
  addOpen.value = true
  addError.value = ''
}

function closeAdd() {
  addOpen.value = false
  addError.value = ''
}

async function load() {
  busy.value = true
  error.value = ''
  try {
    items.value = await adminApi.listDevices()
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

async function create() {
  busy.value = true
  addError.value = ''
  try {
    await adminApi.createDevice({ ...form })
    form.name = ''
    form.host = ''
    closeAdd()
    await load()
  } catch (e) {
    addError.value = e.message
  } finally {
    busy.value = false
  }
}

async function health(d) {
  busy.value = true
  try {
    await adminApi.healthDevice(d.id)
    await load()
  } finally {
    busy.value = false
  }
}

async function remove(d) {
  if (!confirm(t('devices.deleteConfirm', { name: d.name }))) return
  busy.value = true
  try {
    await adminApi.deleteDevice(d.id)
    await load()
  } finally {
    busy.value = false
  }
}

onMounted(load)
</script>
