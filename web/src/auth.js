let authChecked = false
let authed = false

export function isAuthChecked() {
  return authChecked
}

export function isAuthed() {
  return authed
}

export function clearAuthState() {
  authChecked = true
  authed = false
}

export function markAuthed(value = true) {
  authChecked = true
  authed = value
}

export function resetAuthCheck() {
  authChecked = false
  authed = false
}
