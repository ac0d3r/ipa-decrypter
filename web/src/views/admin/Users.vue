<template>
  <section class="panel">
    <PanelHeader :title="t('users.listTitle')">
      <IconButton icon="plus" variant="primary" :title="t('common.add')" @click="openAdd" />
      <IconButton icon="refresh" :title="t('common.refresh')" :busy="busy" @click="load" />
    </PanelHeader>

    <table class="table">
      <thead>
        <tr>
          <th>{{ t('users.colUsername') }}</th>
          <th>{{ t('users.colRole') }}</th>
          <th>{{ t('users.colCreated') }}</th>
          <th>{{ t('users.colActions') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="!items.length">
          <td colspan="4" class="muted">{{ t('common.noData') }}</td>
        </tr>
        <tr v-for="u in items" :key="u.id">
          <td>{{ u.username }}</td>
          <td>
            <span class="badge" :class="u.is_admin ? 'ok' : ''">
              {{ u.is_admin ? t('users.roleAdmin') : t('users.roleUser') }}
            </span>
          </td>
          <td class="muted">{{ fmt(u.created_at) }}</td>
          <td>
            <div class="row-actions">
              <button type="button" class="action-btn" :disabled="busy" @click="openEdit(u)">
                {{ t('users.edit') }}
              </button>
              <button type="button" class="action-btn danger" :disabled="busy" @click="remove(u)">
                {{ t('common.delete') }}
              </button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
    <p v-if="error" class="error">{{ error }}</p>
  </section>

  <div v-if="addOpen" class="modal-backdrop" @click.self="closeAdd">
    <div class="panel modal">
      <h2>{{ t('users.addTitle') }}</h2>
      <div class="grid two">
        <input v-model="form.username" :placeholder="t('users.username')" autocomplete="off" />
        <input v-model="form.password" type="password" :placeholder="t('users.password')" autocomplete="new-password" />
        <label class="checkbox-row">
          <input type="checkbox" v-model="form.is_admin" />
          <span>{{ t('users.isAdmin') }}</span>
        </label>
      </div>
      <div class="modal-actions">
        <button @click="closeAdd" :disabled="busy">{{ t('common.cancel') }}</button>
        <button class="primary" @click="create" :disabled="busy">{{ t('common.add') }}</button>
      </div>
      <p v-if="formError" class="error">{{ formError }}</p>
    </div>
  </div>

  <div v-if="editOpen" class="modal-backdrop" @click.self="closeEdit">
    <div class="panel modal">
      <h2>{{ t('users.editTitle') }}</h2>
      <p class="muted">{{ editForm.username }}</p>
      <div class="grid two">
        <input v-model="editForm.password" type="password" :placeholder="t('users.newPassword')" autocomplete="new-password" />
        <label class="checkbox-row">
          <input type="checkbox" v-model="editForm.is_admin" />
          <span>{{ t('users.isAdmin') }}</span>
        </label>
      </div>
      <div class="modal-actions">
        <button @click="closeEdit" :disabled="busy">{{ t('common.cancel') }}</button>
        <button class="primary" @click="saveEdit" :disabled="busy">{{ t('common.confirm') }}</button>
      </div>
      <p v-if="formError" class="error">{{ formError }}</p>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { adminApi } from '../../api'
import IconButton from '../../components/IconButton.vue'
import PanelHeader from '../../components/PanelHeader.vue'
import { useI18n } from '../../i18n'

const { t, locale } = useI18n()
const items = ref([])
const busy = ref(false)
const error = ref('')
const formError = ref('')
const addOpen = ref(false)
const editOpen = ref(false)
const form = reactive({ username: '', password: '', is_admin: false })
const editForm = reactive({ id: '', username: '', password: '', is_admin: false })

function fmt(s) {
  try {
    return new Date(s).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US')
  } catch {
    return s
  }
}

async function load() {
  busy.value = true
  error.value = ''
  try {
    items.value = (await adminApi.listUsers()) || []
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

function openAdd() {
  form.username = ''
  form.password = ''
  form.is_admin = false
  formError.value = ''
  addOpen.value = true
}

function closeAdd() {
  addOpen.value = false
}

async function create() {
  busy.value = true
  formError.value = ''
  try {
    await adminApi.createUser({
      username: form.username.trim(),
      password: form.password,
      is_admin: form.is_admin,
    })
    closeAdd()
    await load()
  } catch (e) {
    formError.value = e.message
  } finally {
    busy.value = false
  }
}

function openEdit(u) {
  editForm.id = u.id
  editForm.username = u.username
  editForm.password = ''
  editForm.is_admin = !!u.is_admin
  formError.value = ''
  editOpen.value = true
}

function closeEdit() {
  editOpen.value = false
}

async function saveEdit() {
  busy.value = true
  formError.value = ''
  try {
    const body = { is_admin: editForm.is_admin }
    if (editForm.password.trim()) body.password = editForm.password.trim()
    await adminApi.updateUser(editForm.id, body)
    closeEdit()
    await load()
  } catch (e) {
    formError.value = e.message
  } finally {
    busy.value = false
  }
}

async function remove(u) {
  if (!confirm(t('users.deleteConfirm', { username: u.username }))) return
  busy.value = true
  error.value = ''
  try {
    await adminApi.deleteUser(u.id)
    await load()
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.row-actions { display: flex; gap: 0.35rem; flex-wrap: wrap; }
.action-btn {
  font-size: 0.75rem;
  padding: 0.28rem 0.5rem;
}
.action-btn.danger { color: var(--red); border-color: #4a2020; }
.grid.two {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.65rem;
  margin: 0.75rem 0;
}
.checkbox-row {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  font-size: 0.82rem;
  color: var(--text-2);
  grid-column: 1 / -1;
}
.checkbox-row input { width: auto; }
.modal { width: min(420px, 100%); margin: 0; }
.modal h2 { margin: 0 0 0.35rem; font-size: 1rem; }
.modal-actions { display: flex; justify-content: flex-end; gap: 0.5rem; }
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  z-index: 40;
}
</style>
