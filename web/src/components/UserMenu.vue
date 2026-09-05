<template>
  <div class="user-menu" ref="root">
    <button type="button" class="admin-link menu-trigger" @click="toggle">
      {{ authState.username || t('auth.username') }}
    </button>
    <div v-if="open" class="menu">
      <button type="button" class="menu-logout" :disabled="logoutBusy" @click="logout">
        {{ t('auth.logout') }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { authApi } from '../api'
import { authState, clearAuthState } from '../auth'
import { useI18n } from '../i18n'

const { t } = useI18n()
const open = ref(false)
const logoutBusy = ref(false)
const root = ref(null)

async function toggle() {
  open.value = !open.value
}

function onDocClick(e) {
  if (!open.value) return
  if (root.value && !root.value.contains(e.target)) open.value = false
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
  min-width: 140px;
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: 10px;
  padding: 0.45rem;
  z-index: 30;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.35);
}
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
