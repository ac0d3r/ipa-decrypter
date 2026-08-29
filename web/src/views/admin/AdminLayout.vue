<template>
  <div class="admin-shell">
    <aside class="sidebar">
      <div class="side-head">
        <div class="side-top">
          <RouterLink to="/" class="back">{{ t('nav.back') }}</RouterLink>
          <LangSwitch />
        </div>
        <h2>{{ t('nav.adminTitle') }}</h2>
      </div>
      <nav>
        <RouterLink to="/admin/accounts">{{ t('nav.accounts') }}</RouterLink>
        <RouterLink to="/admin/devices">{{ t('nav.devices') }}</RouterLink>
        <RouterLink to="/admin/tasks">{{ t('nav.tasks') }}</RouterLink>
      </nav>
    </aside>
    <main class="admin-main">
      <RouterView />
    </main>
  </div>
</template>

<script setup>
import LangSwitch from '../../components/LangSwitch.vue'
import { useI18n } from '../../i18n'

const { t } = useI18n()
</script>

<style scoped>
.admin-shell { display: flex; min-height: 100vh; }
.sidebar {
  width: 220px;
  background: var(--surface);
  border-right: 1px solid var(--line);
  padding: 1rem;
  flex-shrink: 0;
}
.side-head { margin-bottom: 1rem; }
.side-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}
.side-head h2 { margin: 0.5rem 0 0; font-size: 1rem; }
.back { font-size: 0.8rem; color: var(--text-3); text-decoration: none; }
nav { display: flex; flex-direction: column; gap: 0.35rem; }
nav a {
  padding: 0.5rem 0.65rem;
  border-radius: 8px;
  color: var(--text-2);
  text-decoration: none;
  font-size: 0.88rem;
}
nav a.router-link-active { background: var(--surface-2); color: var(--text); }
.admin-main { flex: 1; padding: 1.25rem; max-width: 960px; }
</style>
