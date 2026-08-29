import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import Home from './views/Home.vue'
import AdminLayout from './views/admin/AdminLayout.vue'
import Accounts from './views/Accounts.vue'
import Devices from './views/Devices.vue'
import Tasks from './views/admin/Tasks.vue'
import { initI18n } from './i18n'
import './style.css'

initI18n()

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Home },
    {
      path: '/admin',
      component: AdminLayout,
      children: [
        { path: '', redirect: '/admin/accounts' },
        { path: 'accounts', component: Accounts },
        { path: 'devices', component: Devices },
        { path: 'tasks', component: Tasks },
      ],
    },
  ],
})

createApp(App).use(router).mount('#app')
