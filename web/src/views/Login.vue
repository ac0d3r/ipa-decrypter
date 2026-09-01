<template>
  <div class="login-page">
    <form class="login-card panel" @submit.prevent="submit">
      <div class="login-head">
        <RouterLink to="/" class="back">{{ t('nav.back') }}</RouterLink>
        <LangSwitch />
      </div>
      <h1>{{ t('auth.title') }}</h1>
      <p class="muted">{{ t('auth.hint') }}</p>
      <label>
        <span>{{ t('auth.username') }}</span>
        <input v-model="username" autocomplete="username" required autofocus />
      </label>
      <label>
        <span>{{ t('auth.password') }}</span>
        <input v-model="password" type="password" autocomplete="current-password" required />
      </label>
      <button class="primary" type="submit" :disabled="busy">
        {{ busy ? t('common.loading') : t('common.login') }}
      </button>
      <p v-if="error" class="error">{{ error }}</p>
    </form>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import LangSwitch from '../components/LangSwitch.vue'
import { useI18n } from '../i18n'
import { authApi } from '../api'
import { markAuthed } from '../auth'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()

function safeRedirect(value) {
  if (typeof value !== 'string') return ''
  if (!value.startsWith('/') || value.startsWith('//')) return ''
  return value
}

const username = ref('')
const password = ref('')
const busy = ref(false)
const error = ref('')

async function submit() {
  busy.value = true
  error.value = ''
  try {
    const me = await authApi.login(username.value, password.value)
    markAuthed({ admin: !!me.is_admin, name: me.username || '', quota: me.quota || null })
    const redirect = safeRedirect(route.query.redirect)
    if (me.is_admin) {
      await router.replace(redirect.startsWith('/admin') ? redirect : '/admin')
      return
    }
    await router.replace(redirect && redirect !== '/login' ? redirect : '/')
  } catch (e) {
    error.value = e.message || t('auth.failed')
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
}
.login-card {
  width: 100%;
  max-width: 360px;
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
  margin: 0;
}
.login-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.login-card h1 {
  margin: 0;
  font-size: 1.2rem;
}
.back {
  font-size: 0.8rem;
  color: var(--text-3);
  text-decoration: none;
}
label {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: 0.82rem;
  color: var(--text-2);
}
button.primary {
  margin-top: 0.25rem;
  width: 100%;
  padding: 0.55rem;
}
.error {
  margin: 0;
  color: var(--red);
  font-size: 0.82rem;
}
.muted {
  margin: 0;
  color: var(--text-3);
  font-size: 0.82rem;
}
</style>
