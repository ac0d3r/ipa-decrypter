import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import Home from './views/Home.vue'
import AdminLayout from './views/admin/AdminLayout.vue'
import Login from './views/admin/Login.vue'
import Accounts from './views/Accounts.vue'
import Devices from './views/Devices.vue'
import Tasks from './views/admin/Tasks.vue'
import { adminApi, setUnauthorizedHandler } from './api'
import { clearAuthState, isAuthChecked, isAuthed, markAuthed } from './auth'
import { initI18n } from './i18n'
import './style.css'

initI18n()

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Home },
    { path: '/admin/login', component: Login, meta: { guest: true } },
    {
      path: '/admin',
      component: AdminLayout,
      meta: { requiresAuth: true },
      children: [
        { path: '', redirect: '/admin/accounts' },
        { path: 'accounts', component: Accounts },
        { path: 'devices', component: Devices },
        { path: 'tasks', component: Tasks },
      ],
    },
  ],
})

async function ensureAuth() {
  if (isAuthChecked()) return isAuthed()
  try {
    await adminApi.me()
    markAuthed(true)
  } catch {
    markAuthed(false)
  }
  return isAuthed()
}

setUnauthorizedHandler(() => {
  clearAuthState()
  if (router.currentRoute.value.path !== '/admin/login') {
    router.replace({ path: '/admin/login', query: { redirect: router.currentRoute.value.fullPath } })
  }
})

router.beforeEach(async (to) => {
  if (to.meta.guest) {
    if (await ensureAuth()) return '/admin'
    return true
  }
  if (to.matched.some((r) => r.meta.requiresAuth)) {
    if (!(await ensureAuth())) {
      return { path: '/admin/login', query: { redirect: to.fullPath } }
    }
  }
  return true
})

createApp(App).use(router).mount('#app')
