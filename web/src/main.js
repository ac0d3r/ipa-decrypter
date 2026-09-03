import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import Home from './views/Home.vue'
import AdminLayout from './views/admin/AdminLayout.vue'
import Login from './views/Login.vue'
import Accounts from './views/Accounts.vue'
import Devices from './views/Devices.vue'
import Tasks from './views/admin/Tasks.vue'
import Users from './views/admin/Users.vue'
import Vouchers from './views/admin/Vouchers.vue'
import { authApi, setUnauthorizedHandler } from './api'
import { clearAuthState, isAuthChecked, isAuthed, isAdminUser, markAuthed } from './auth'
import { initI18n } from './i18n'
import './style.css'

initI18n()

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Home },
    { path: '/login', component: Login, meta: { guest: true } },
    { path: '/admin/login', redirect: '/login' },
    {
      path: '/admin',
      component: AdminLayout,
      meta: { requiresAdmin: true },
      children: [
        { path: '', redirect: '/admin/accounts' },
        { path: 'accounts', component: Accounts },
        { path: 'devices', component: Devices },
        { path: 'tasks', component: Tasks },
        { path: 'users', component: Users },
        { path: 'vouchers', component: Vouchers },
      ],
    },
  ],
})

async function ensureAuth() {
  if (isAuthChecked()) return isAuthed()
  try {
    const me = await authApi.me()
    markAuthed({ admin: !!me?.is_admin, name: me?.username || '', quota: me?.quota || null })
  } catch {
    clearAuthState()
  }
  return isAuthed()
}

setUnauthorizedHandler(() => {
  clearAuthState()
  const path = router.currentRoute.value.path
  if (path !== '/login') {
    router.replace({ path: '/login', query: { redirect: router.currentRoute.value.fullPath } })
  }
})

router.beforeEach(async (to) => {
  if (to.meta.guest) {
    if (await ensureAuth()) {
      return isAdminUser() ? '/admin' : '/'
    }
    return true
  }
  if (to.matched.some((r) => r.meta.requiresAdmin)) {
    if (!(await ensureAuth())) {
      return { path: '/login', query: { redirect: to.fullPath } }
    }
    if (!isAdminUser()) {
      return '/'
    }
  }
  return true
})

createApp(App).use(router).mount('#app')
