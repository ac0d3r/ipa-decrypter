<template>
  <div class="admin-shell">
    <aside class="sidebar">
      <div class="side-head">
        <div class="side-top">
          <RouterLink to="/" class="back">{{ t('nav.back') }}</RouterLink>
          <LangSwitch />
        </div>
        <h2>{{ t('nav.adminTitle') }}</h2>
        <p v-if="username" class="user">{{ username }}</p>
      </div>
      <nav>
        <RouterLink to="/admin/accounts">{{ t('nav.accounts') }}</RouterLink>
        <RouterLink to="/admin/devices">{{ t('nav.devices') }}</RouterLink>
        <RouterLink to="/admin/tasks">{{ t('nav.tasks') }}</RouterLink>
        <RouterLink to="/admin/users">{{ t('nav.users') }}</RouterLink>
        <RouterLink to="/admin/vouchers">{{ t('nav.vouchers') }}</RouterLink>
      </nav>
      <button type="button" class="logout" :disabled="busy" @click="logout">
        {{ t('auth.logout') }}
      </button>
    </aside>
    <main class="admin-main">
      <RouterView />
    </main>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import LangSwitch from '../../components/LangSwitch.vue'
import { useI18n } from '../../i18n'
import { authApi } from '../../api'
import { clearAuthState, currentUsername } from '../../auth'

const { t } = useI18n()
const router = useRouter()
const username = ref(currentUsername())
const busy = ref(false)

onMounted(async () => {
  try {
    const me = await authApi.me()
    username.value = me?.username || ''
  } catch {
    username.value = ''
  }
})

async function logout() {
  busy.value = true
  try {
    await authApi.logout()
  } catch {
    /* ignore */
  } finally {
    clearAuthState()
    busy.value = false
    await router.replace('/login')
  }
}
</script>

<style scoped>
.admin-shell { display: flex; min-height: 100vh; }
.sidebar {
  width: 220px;
  background: var(--surface);
  border-right: 1px solid var(--line);
  padding: 1rem;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
}
.side-head { margin-bottom: 1rem; }
.side-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}
.side-head h2 { margin: 0.5rem 0 0; font-size: 1rem; }
.user {
  margin: 0.35rem 0 0;
  font-size: 0.75rem;
  color: var(--text-3);
}
.back { font-size: 0.8rem; color: var(--text-3); text-decoration: none; }
nav { display: flex; flex-direction: column; gap: 0.35rem; flex: 1; }
nav a {
  padding: 0.5rem 0.65rem;
  border-radius: 8px;
  color: var(--text-2);
  text-decoration: none;
  font-size: 0.88rem;
}
nav a.router-link-active { background: var(--surface-2); color: var(--text); }
.logout {
  margin-top: 1rem;
  width: 100%;
  font-size: 0.82rem;
  color: var(--text-2);
}
.admin-main { flex: 1; min-width: 0; padding: 1.25rem; max-width: 1100px; }
</style>
