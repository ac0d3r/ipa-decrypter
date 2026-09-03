<template>
  <section class="panel">
    <PanelHeader :title="t('vouchers.listTitle')">
      <IconButton icon="plus" variant="primary" :title="t('common.add')" @click="openAdd" />
      <IconButton icon="refresh" :title="t('common.refresh')" :busy="busy" @click="load" />
    </PanelHeader>
    <p class="hint">{{ t('vouchers.hint') }}</p>

    <div class="filters">
      <div class="tabs">
        <button
          v-for="tab in filterTabs"
          :key="tab"
          type="button"
          :class="{ active: filter === tab }"
          @click="setFilter(tab)"
        >
          {{ t(tab === 'user' ? 'vouchers.filterUser' : `vouchers.${tab === 'unused' ? 'unused' : 'used'}`) }}
        </button>
      </div>
      <select
        v-if="filter === 'user'"
        v-model="userID"
        class="user-pick"
        :disabled="!redeemers.length"
        @change="onUserChange"
      >
        <option v-if="!redeemers.length" value="">{{ t('vouchers.pickUser') }}</option>
        <option v-for="u in redeemers" :key="u.id" :value="u.id">{{ u.username }}</option>
      </select>
    </div>

    <table class="table">
      <thead>
        <tr>
          <th>{{ t('vouchers.colCode') }}</th>
          <th>{{ t('vouchers.colQuota') }}</th>
          <th>{{ t('vouchers.colStatus') }}</th>
          <th>{{ t('vouchers.colRedeemedBy') }}</th>
          <th>{{ t('vouchers.colCreated') }}</th>
          <th>{{ t('users.colActions') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="!items.length">
          <td colspan="6" class="muted">{{ t('common.noData') }}</td>
        </tr>
        <tr v-for="v in items" :key="v.id">
          <td class="mono">{{ v.code }}</td>
          <td>{{ v.quota }}</td>
          <td>
            <span class="badge" :class="v.redeemed ? '' : 'ok'">
              {{ v.redeemed ? t('vouchers.used') : t('vouchers.unused') }}
            </span>
          </td>
          <td class="muted">{{ v.redeemed_username || '—' }}</td>
          <td class="muted">{{ fmt(v.created_at) }}</td>
          <td>
            <button
              v-if="!v.redeemed"
              type="button"
              class="action-btn danger"
              :disabled="busy"
              @click="remove(v)"
            >
              {{ t('common.delete') }}
            </button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-if="total > 0" class="pager">
      <button type="button" :disabled="page <= 1 || busy" @click="goPage(page - 1)">
        {{ t('vouchers.prev') }}
      </button>
      <span class="muted">{{ t('vouchers.pageInfo', { page, pages, total }) }}</span>
      <button type="button" :disabled="page >= pages || busy" @click="goPage(page + 1)">
        {{ t('vouchers.next') }}
      </button>
    </div>
    <p v-if="error" class="error">{{ error }}</p>
  </section>

  <div v-if="addOpen" class="modal-backdrop" @click.self="closeAdd">
    <div class="panel modal">
      <h2>{{ t('vouchers.addTitle') }}</h2>
      <div class="fields">
        <label>
          <span>{{ t('vouchers.count') }}</span>
          <input v-model.number="form.count" type="number" min="1" max="50" />
        </label>
        <label>
          <span>{{ t('vouchers.quota') }}</span>
          <input v-model.number="form.quota" type="number" min="1" />
        </label>
      </div>
      <div class="modal-actions">
        <button @click="closeAdd" :disabled="busy">{{ t('common.cancel') }}</button>
        <button class="primary" @click="create" :disabled="busy">{{ t('common.add') }}</button>
      </div>
      <p v-if="formError" class="error">{{ formError }}</p>
    </div>
  </div>

  <div v-if="codesOpen" class="modal-backdrop" @click.self="closeCodes">
    <div class="panel modal">
      <h2>{{ t('vouchers.codesTitle') }}</h2>
      <pre class="codes">{{ createdCodes.join('\n') }}</pre>
      <div class="modal-actions">
        <button class="primary" @click="copyCodes">{{ copied ? t('vouchers.copied') : t('vouchers.copy') }}</button>
        <button @click="closeCodes">{{ t('common.confirm') }}</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { adminApi } from '../../api'
import IconButton from '../../components/IconButton.vue'
import PanelHeader from '../../components/PanelHeader.vue'
import { useI18n } from '../../i18n'

const PAGE_SIZE = 20
const filterTabs = ['unused', 'used', 'user']

const { t, locale } = useI18n()
const items = ref([])
const busy = ref(false)
const error = ref('')
const formError = ref('')
const addOpen = ref(false)
const codesOpen = ref(false)
const createdCodes = ref([])
const copied = ref(false)
const form = reactive({ count: 5, quota: 5 })
const filter = ref('unused')
const userID = ref('')
const redeemers = ref([])
const page = ref(1)
const total = ref(0)
const pages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))

function fmt(s) {
  try {
    return new Date(s).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US')
  } catch {
    return s
  }
}

async function fetchPage() {
  const data = await adminApi.listVouchers({
    status: filter.value,
    user: filter.value === 'user' ? userID.value : '',
    offset: (page.value - 1) * PAGE_SIZE,
    limit: PAGE_SIZE,
  })
  items.value = data?.items || []
  total.value = data?.total || 0
  if (filter.value === 'user') {
    redeemers.value = data?.users || []
    if (userID.value && !redeemers.value.some((u) => u.id === userID.value)) {
      userID.value = ''
    }
    if (!userID.value && redeemers.value.length) {
      userID.value = redeemers.value[0].id
      await fetchPage()
      return
    }
  } else {
    redeemers.value = []
  }
  const maxPage = Math.max(1, Math.ceil(total.value / PAGE_SIZE))
  if (page.value > maxPage) {
    page.value = maxPage
    if (total.value > 0) await fetchPage()
  }
}

async function load() {
  busy.value = true
  error.value = ''
  try {
    await fetchPage()
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

function setFilter(next) {
  if (filter.value === next) return
  filter.value = next
  page.value = 1
  if (next !== 'user') userID.value = ''
  load()
}

function onUserChange() {
  page.value = 1
  load()
}

function goPage(next) {
  page.value = next
  load()
}

function openAdd() {
  form.count = 5
  form.quota = 5
  formError.value = ''
  addOpen.value = true
}

function closeAdd() {
  addOpen.value = false
}

function closeCodes() {
  codesOpen.value = false
  createdCodes.value = []
  copied.value = false
}

async function copyCodes() {
  try {
    await navigator.clipboard.writeText(createdCodes.value.join('\n'))
    copied.value = true
  } catch {
    copied.value = false
  }
}

async function create() {
  busy.value = true
  formError.value = ''
  try {
    const out = await adminApi.createVouchers({
      count: form.count,
      quota: form.quota,
    })
    createdCodes.value = out?.codes || []
    closeAdd()
    codesOpen.value = true
    copied.value = false
    filter.value = 'unused'
    userID.value = ''
    page.value = 1
    await load()
  } catch (e) {
    formError.value = e.message
  } finally {
    busy.value = false
  }
}

async function remove(v) {
  if (!confirm(t('vouchers.deleteUnused'))) return
  busy.value = true
  error.value = ''
  try {
    await adminApi.deleteVoucher(v.id)
    if (items.value.length === 1 && page.value > 1) page.value -= 1
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
.hint {
  margin: 0 0 0.75rem;
  font-size: 0.75rem;
  color: var(--text-3);
}
.filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.6rem;
  margin: 0 0 0.75rem;
}
.tabs {
  display: flex;
  gap: 0.35rem;
}
.tabs button.active {
  background: var(--accent);
  border-color: var(--accent);
  color: #fff;
}
.user-pick {
  width: auto;
  min-width: 10rem;
  max-width: 16rem;
}
.pager {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.6rem;
  margin-top: 0.75rem;
  font-size: 0.82rem;
}
.mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.82rem; }
.action-btn {
  font-size: 0.75rem;
  padding: 0.28rem 0.5rem;
}
.action-btn.danger { color: var(--red); border-color: #4a2020; }
.fields {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin: 0.75rem 0;
}
.fields label {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: 0.82rem;
  color: var(--text-2);
}
.codes {
  background: var(--surface-2);
  padding: 0.75rem;
  border-radius: 8px;
  font-size: 0.85rem;
  line-height: 1.6;
  max-height: 240px;
  overflow: auto;
}
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
