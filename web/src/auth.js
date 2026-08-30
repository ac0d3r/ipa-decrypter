import { reactive } from 'vue'

export const authState = reactive({
  checked: false,
  authed: false,
  isAdmin: false,
  username: '',
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
}

export function markAuthed({ admin = false, name = '' } = {}) {
  authState.checked = true
  authState.authed = true
  authState.isAdmin = !!admin
  authState.username = name || ''
}

export function resetAuthCheck() {
  authState.checked = false
  authState.authed = false
  authState.isAdmin = false
  authState.username = ''
}
