<template>
  <section class="panel">
    <PanelHeader :title="t('accounts.listTitle')">
      <IconButton icon="plus" variant="primary" :title="t('common.add')" @click="openAdd" />
      <IconButton icon="refresh" :title="t('common.refresh')" :busy="busy" @click="load" />
    </PanelHeader>

    <table class="table accounts-table">
      <thead>
        <tr>
          <th>{{ t('accounts.colLabel') }}</th>
          <th>{{ t('accounts.colRegion') }}</th>
          <th>{{ t('accounts.colEmail') }}</th>
          <th>{{ t('accounts.colStatus') }}</th>
          <th>{{ t('accounts.colCredential') }}</th>
          <th>{{ t('accounts.colLog') }}</th>
          <th>{{ t('accounts.colActions') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="!items.length">
          <td colspan="7" class="muted">{{ t('common.noData') }}</td>
        </tr>
        <tr v-for="a in items" :key="a.id">
          <td class="col-label">{{ displayLabel(a) }}</td>
          <td class="col-region">
            <span v-if="a.region" class="region-chip">
              <CountryFlag :code="a.region" />
              {{ a.region.toUpperCase() }}
            </span>
            <span v-else class="region-chip unknown">{{ t('accounts.regionUnknown') }}</span>
          </td>
          <td class="col-email">{{ a.email }}</td>
          <td class="col-status">
            <span class="badge" :class="statusClass(a.status)">{{ accountStatus(a.status) }}</span>
          </td>
          <td class="col-credential">
            <div class="cred-cell">
              <span
                class="badge"
                :class="a.has_credential ? 'ok' : 'cred-missing'"
                :title="a.has_credential ? t('accounts.credentialHintSet') : t('accounts.credentialHintMissing')"
              >
                {{ a.has_credential ? t('accounts.credentialSet') : t('accounts.credentialMissing') }}
              </span>
              <div class="table-actions">
                <IconButton
                  icon="key"
                  :title="t('accounts.credentialUpload')"
                  :disabled="busy"
                  @click="pickCredential(a)"
                />
                <IconButton
                  icon="trash"
                  variant="danger"
                  :title="t('accounts.credentialClear')"
                  :disabled="busy || !a.has_credential"
                  @click="clearCred(a)"
                />
              </div>
            </div>
          </td>
          <td class="col-log">
            <span v-if="a.last_error" class="log-error" :title="a.last_error">{{ a.last_error }}</span>
            <span v-else class="muted">—</span>
          </td>
          <td class="col-actions">
            <div class="table-actions">
              <IconButton icon="login" :title="t('common.login')" :disabled="busy" @click="startLogin(a)" />
              <IconButton icon="check" :title="t('accounts.verify')" :disabled="busy" @click="refresh(a)" />
              <IconButton icon="logout" variant="danger" :title="t('common.revoke')" :disabled="busy" @click="revoke(a)" />
              <IconButton icon="trash" variant="danger" :title="t('common.delete')" :disabled="busy" @click="remove(a)" />
            </div>
          </td>
        </tr>
      </tbody>
    </table>
    <p v-if="error" class="error">{{ error }}</p>
  </section>

  <div v-if="addOpen" class="modal-backdrop" @click.self="closeAdd">
    <div class="panel modal">
      <h2>{{ t('accounts.addTitle') }}</h2>
      <div class="grid two">
        <input v-model="form.email" :placeholder="t('accounts.email')" />
        <input v-model="form.password" type="password" :placeholder="t('accounts.password')" />
        <input v-model="form.label" :placeholder="t('accounts.label')" />
        <input v-model="form.region" :placeholder="t('accounts.region')" />
        <label class="checkbox-row">
          <input type="checkbox" v-model="form.login_now" />
          <span>{{ t('accounts.loginNow') }}</span>
        </label>
      </div>
      <p class="muted">{{ t('accounts.hint') }}</p>
      <div class="modal-actions">
        <button @click="closeAdd" :disabled="busy">{{ t('common.cancel') }}</button>
        <button class="primary" @click="create" :disabled="busy">{{ t('common.add') }}</button>
      </div>
      <p v-if="addError" class="error">{{ addError }}</p>
    </div>
  </div>

  <div v-if="twoFA.open" class="modal-backdrop" @click.self="closeTwoFA">
    <div class="panel modal">
      <h2>{{ t('accounts.twoFATitle') }}</h2>
      <p class="muted">{{ t('accounts.twoFAHint', { email: twoFA.email }) }}</p>
      <input
        v-model="twoFA.code"
        :placeholder="t('accounts.twoFACode')"
        maxlength="8"
        autocomplete="one-time-code"
        @keyup.enter="submitTwoFA"
      />
      <div class="modal-actions">
        <button @click="closeTwoFA" :disabled="busy">{{ t('common.cancel') }}</button>
        <button class="primary" @click="submitTwoFA" :disabled="busy || !twoFA.code.trim()">
          {{ t('accounts.confirmLogin') }}
        </button>
      </div>
      <p v-if="twoFA.error" class="error">{{ twoFA.error }}</p>
    </div>
  </div>
  <input ref="credInput" type="file" accept=".bin,.dat,application/octet-stream" hidden @change="uploadCred" />
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ApiError, adminApi, statusClass } from '../api'
import IconButton from '../components/IconButton.vue'
import PanelHeader from '../components/PanelHeader.vue'
import CountryFlag from '../components/CountryFlag.vue'
import { useI18n } from '../i18n'

const { t } = useI18n()
const items = ref([])
const busy = ref(false)
const error = ref('')
const addOpen = ref(false)
const addError = ref('')
const form = reactive({
  email: '',
  password: '',
  region: '',
  label: '',
  login_now: true,
})
const twoFA = reactive({
  open: false,
  accountId: '',
  email: '',
  code: '',
  error: '',
})
const credInput = ref(null)
const credAccountId = ref('')

function pickCredential(a) {
  credAccountId.value = a.id
  credInput.value.value = ''
  credInput.value.click()
}

async function uploadCred(event) {
  const file = event.target.files?.[0]
  const accountId = credAccountId.value
  if (!file || !accountId) return
  busy.value = true
  error.value = ''
  try {
    await adminApi.uploadCredential(accountId, file)
    await load()
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

async function clearCred(a) {
  if (!confirm(t('accounts.credentialClearConfirm', { email: a.email }))) return
  busy.value = true
  try {
    await adminApi.clearCredential(a.id)
    await load()
  } finally {
    busy.value = false
  }
}

function accountStatus(status) {
  return t(`accounts.status.${status}`) || status
}

function displayLabel(a) {
  if (a.label && a.label !== a.email) return a.label
  return '—'
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
    items.value = await adminApi.listAccounts()
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

function openTwoFA(account) {
  twoFA.open = true
  twoFA.accountId = account.id
  twoFA.email = account.email
  twoFA.code = ''
  twoFA.error = ''
}

function closeTwoFA() {
  twoFA.open = false
  twoFA.accountId = ''
  twoFA.email = ''
  twoFA.code = ''
  twoFA.error = ''
}

async function handleLoginFlow(accountId, authCode = '') {
  try {
    await adminApi.loginAccount(accountId, authCode)
    closeTwoFA()
    error.value = ''
    await load()
    return true
  } catch (e) {
    if (e instanceof ApiError && e.needs2FA) {
      openTwoFA(e.account || { id: accountId, email: '' })
      await load()
      return false
    }
    throw e
  }
}

async function create() {
  busy.value = true
  addError.value = ''
  try {
    const created = await adminApi.createAccount({ ...form })
    form.password = ''
    closeAdd()
    if (form.login_now) {
      await load()
      if (created?.status === 'needs_2fa') openTwoFA(created)
    } else {
      await load()
    }
  } catch (e) {
    if (e instanceof ApiError && e.needs2FA && e.account) {
      form.password = ''
      closeAdd()
      openTwoFA(e.account)
      await load()
    } else {
      addError.value = e.message
      await load()
    }
  } finally {
    busy.value = false
  }
}

async function startLogin(a) {
  busy.value = true
  error.value = ''
  try {
    await handleLoginFlow(a.id, '')
  } catch (e) {
    error.value = e.message
    await load()
  } finally {
    busy.value = false
  }
}

async function submitTwoFA() {
  if (!twoFA.accountId || !twoFA.code.trim()) return
  busy.value = true
  twoFA.error = ''
  try {
    await handleLoginFlow(twoFA.accountId, twoFA.code.trim())
  } catch (e) {
    twoFA.error = e.message
    await load()
  } finally {
    busy.value = false
  }
}

async function refresh(a) {
  busy.value = true
  try {
    await adminApi.refreshAccount(a.id)
    await load()
  } finally {
    busy.value = false
  }
}

async function revoke(a) {
  busy.value = true
  try {
    await adminApi.revokeAccount(a.id)
    await load()
  } finally {
    busy.value = false
  }
}

async function remove(a) {
  if (!confirm(t('accounts.deleteConfirm', { email: a.email }))) return
  busy.value = true
  try {
    await adminApi.deleteAccount(a.id)
    await load()
  } finally {
    busy.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.accounts-table {
  table-layout: fixed;
  width: 100%;
}
.accounts-table th,
.accounts-table td {
  text-align: center;
  vertical-align: middle;
  padding: 0.5rem 0.3rem;
}
.accounts-table th:nth-child(1),
.accounts-table td:nth-child(1) {
  width: 9%;
}
.accounts-table th:nth-child(2),
.accounts-table td:nth-child(2) {
  width: 9%;
}
.accounts-table th:nth-child(3),
.accounts-table td:nth-child(3) {
  width: 22%;
  word-break: break-word;
}
.accounts-table th:nth-child(4),
.accounts-table td:nth-child(4) {
  width: 11%;
  white-space: nowrap;
}
.accounts-table th:nth-child(5),
.accounts-table td:nth-child(5) {
  width: 18%;
}
.accounts-table th:nth-child(6),
.accounts-table td:nth-child(6) {
  width: 14%;
}
.accounts-table th:nth-child(7),
.accounts-table td:nth-child(7) {
  width: 17%;
}
.col-label {
  font-weight: 500;
  line-height: 1.35;
  word-break: break-word;
}
.region-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.25rem;
  font-size: 0.7rem;
  line-height: 1;
  padding: 0.15rem 0.38rem;
  border-radius: 5px;
  background: var(--surface-3);
  color: var(--text-2);
  text-transform: uppercase;
}
.region-chip.unknown {
  padding: 0.15rem 0.32rem;
  color: var(--text-4);
  background: transparent;
  border: 1px dashed var(--line);
  text-transform: lowercase;
}
.cred-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.3rem;
}
.badge.cred-missing {
  opacity: 0.55;
  color: var(--text-4);
}
.col-actions .table-actions,
.cred-cell .table-actions {
  justify-content: center;
}
.log-error {
  color: var(--red);
  font-size: 0.76rem;
  line-height: 1.35;
  word-break: break-word;
  overflow-wrap: anywhere;
  display: -webkit-box;
  line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.checkbox-row {
  grid-column: 1 / -1;
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  cursor: pointer;
  font-size: 0.86rem;
  color: var(--text-2);
  user-select: none;
}
.checkbox-row input[type="checkbox"] {
  width: auto;
  margin: 0;
  padding: 0;
  flex-shrink: 0;
}
</style>
