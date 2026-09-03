<template>
  <div class="user-menu" ref="root">
    <button type="button" class="admin-link menu-trigger" @click="toggle">
      {{ authState.username || t('auth.username') }}
    </button>
    <div v-if="open" class="menu">
      <div class="quota">
        <div class="quota-row">
          <span>{{ t('auth.quotaDecrypt') }}</span>
          <span>{{ fmtQuota(quota.decrypt) }}</span>
        </div>
        <div class="quota-row">
          <span>{{ t('auth.quotaDownload') }}</span>
          <span>{{ fmtQuota(quota.download) }}</span>
        </div>
      </div>
      <form v-if="!authState.isAdmin" class="redeem" @submit.prevent="redeem">
        <input v-model="redeemCode" :placeholder="t('auth.redeemPlaceholder')" autocomplete="off" />
        <button type="submit" :disabled="redeemBusy || !redeemCode.trim()">{{ t('auth.redeem') }}</button>
        <p v-if="redeemError" class="redeem-msg error">{{ redeemError }}</p>
        <p v-else-if="redeemOk" class="redeem-msg ok">{{ redeemOk }}</p>
      </form>
      <button type="button" class="menu-logout" :disabled="logoutBusy" @click="logout">
        {{ t('auth.logout') }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { authApi } from '../api'
import { authState, clearAuthState } from '../auth'
import { useI18n } from '../i18n'

const { t } = useI18n()
const open = ref(false)
const logoutBusy = ref(false)
const root = ref(null)

const quota = computed(() => authState.quota || {})
const redeemCode = ref('')
const redeemBusy = ref(false)
const redeemError = ref('')
const redeemOk = ref('')

function fmtQuota(q) {
  if (!q) return '—'
  const used = q.used ?? 0
  const limit = q.limit
  if (limit == null) return String(used)
  if (limit < 0) return `${used} / ${t('users.limitUnlimited')}`
  return `${used} / ${limit}`
}

async function refreshQuota() {
  try {
    const me = await authApi.me()
    authState.quota = me?.quota || null
    if (me?.username) authState.username = me.username
  } catch {
    /* ignore */
  }
}

async function toggle() {
  open.value = !open.value
  if (open.value) await refreshQuota()
}

function onDocClick(e) {
  if (!open.value) return
  if (root.value && !root.value.contains(e.target)) open.value = false
}

async function redeem() {
  redeemBusy.value = true
  redeemError.value = ''
  redeemOk.value = ''
  try {
    const me = await authApi.redeem(redeemCode.value)
    authState.quota = me?.quota || null
    redeemOk.value = t('auth.redeemOk', { quota: me?.added ?? '' })
    redeemCode.value = ''
  } catch (e) {
    redeemError.value = e.message
  } finally {
    redeemBusy.value = false
  }
}

async function logout() {
  logoutBusy.value = true
  try {
    await authApi.logout()
  } catch {
    /* ignore */
  } finally {
    clearAuthState()
    logoutBusy.value = false
    open.value = false
  }
}

onMounted(() => document.addEventListener('click', onDocClick))
onUnmounted(() => document.removeEventListener('click', onDocClick))
</script>

<style scoped>
.user-menu { position: relative; }
.admin-link {
  font: inherit;
  font-size: .68rem;
  color: var(--text-2);
  padding: 0.28rem 0.55rem;
  border: 1px solid var(--line);
  border-radius: 8px;
  line-height: 1.2;
  background: transparent;
  cursor: pointer;
  max-width: 9rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.menu {
  position: absolute;
  right: 0;
  top: calc(100% + 6px);
  min-width: 200px;
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: 10px;
  padding: 0.45rem;
  z-index: 30;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.35);
}
.quota {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  padding: 0.25rem 0.35rem 0.5rem;
  border-bottom: 1px solid var(--line);
  margin-bottom: 0.4rem;
}
.quota-row {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  font-size: 0.72rem;
  color: var(--text-2);
}
.quota-row span:last-child {
  color: var(--text);
  font-variant-numeric: tabular-nums;
}
.redeem {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: 0.15rem 0.35rem 0.5rem;
  border-bottom: 1px solid var(--line);
  margin-bottom: 0.4rem;
}
.redeem input {
  font-size: 0.72rem;
  padding: 0.28rem 0.4rem;
}
.redeem button {
  font-size: 0.72rem;
  padding: 0.28rem 0.4rem;
}
.redeem-msg {
  margin: 0;
  font-size: 0.68rem;
}
.redeem-msg.ok { color: var(--ok, #6ee7a8); }
.redeem-msg.error { color: var(--red); }
.menu-logout {
  width: 100%;
  font-size: 0.72rem;
  color: var(--text-2);
  background: transparent;
  border: 0;
  padding: 0.4rem 0.35rem;
  text-align: left;
  border-radius: 6px;
}
.menu-logout:hover:not(:disabled) { background: var(--surface-2); color: var(--text); }
</style>
