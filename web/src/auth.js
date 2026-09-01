import { reactive } from 'vue'

export const authState = reactive({
  checked: false,
  authed: false,
  isAdmin: false,
  username: '',
  quota: null,
})

export function isAuthChecked() {
  return authState.checked
}

export function isAuthed() {
  return authState.authed
}

export function isAdminUser() {
  return authState.isAdmin
}

export function currentUsername() {
  return authState.username
}

export function clearAuthState() {
  authState.checked = true
  authState.authed = false
  authState.isAdmin = false
  authState.username = ''
  authState.quota = null
}

export function markAuthed({ admin = false, name = '', quota = null } = {}) {
  authState.checked = true
  authState.authed = true
  authState.isAdmin = !!admin
  authState.username = name || ''
  authState.quota = quota
}

export function resetAuthCheck() {
  authState.checked = false
  authState.authed = false
  authState.isAdmin = false
  authState.username = ''
  authState.quota = null
}
