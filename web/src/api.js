const BASE = ''

export class ApiError extends Error {
  constructor(message, { status = 0, needs2FA = false, account = null, data = null } = {}) {
    super(message)
    this.status = status
    this.needs2FA = needs2FA
    this.account = account
    this.data = data
  }
}

let onUnauthorized = null

/** Register a handler invoked when an authenticated request returns 401. */
export function setUnauthorizedHandler(fn) {
  onUnauthorized = fn
}

async function request(path, options = {}) {
  const { skipAuthRedirect, ...fetchOpts } = options
  const res = await fetch(BASE + path, {
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json', ...(fetchOpts.headers || {}) },
    ...fetchOpts,
  })
  if (res.status === 204) return null
  const text = await res.text()
  let data = null
  try { data = text ? JSON.parse(text) : null } catch { data = { error: text } }
  if (!res.ok) {
    if (res.status === 401 && !skipAuthRedirect && onUnauthorized) {
      onUnauthorized()
    }
    throw new ApiError(data?.error || res.statusText, {
      status: res.status,
      needs2FA: !!data?.needs_2fa,
      account: data?.account || null,
      data,
    })
  }
  return data
}

// --- Public (前台) ---
export const publicApi = {
  regions: () => request('/api/v1/regions'),
  search: ({ q, bundle_id, region, limit } = {}) => {
    const params = new URLSearchParams({ region: region || '' })
    if (bundle_id) params.set('bundle_id', bundle_id)
    else if (q) params.set('q', q)
    if (limit) params.set('limit', String(limit))
    return request(`/api/v1/search?${params}`)
  },
  versions: (bundle_id, region, app_id = 0, offset = 0, limit = 5) => {
    const q = new URLSearchParams({ bundle_id, region })
    if (app_id) q.set('app_id', String(app_id))
    q.set('offset', String(offset))
    q.set('limit', String(limit))
    return request(`/api/v1/versions?${q}`)
  },
  request: (body) => request('/api/v1/request', { method: 'POST', body: JSON.stringify(body) }),
  task: (id) => request(`/api/v1/tasks/${id}`),
  queue: () => request('/api/v1/queue'),
  cache: () => request('/api/v1/cache'),
  cachedByBundles: (region, bundleIds) => {
    if (!bundleIds?.length) return Promise.resolve({ by_bundle: {} })
    const q = new URLSearchParams({
      region,
      bundle_ids: bundleIds.join(','),
    })
    return request(`/api/v1/cache?${q}`)
  },
  taskFileURL: (id) => `/api/v1/tasks/${id}/file`,
}

// --- Auth ---
export const authApi = {
  login: (username, password) => request('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
    skipAuthRedirect: true,
  }),
  logout: () => request('/api/v1/auth/logout', { method: 'POST', skipAuthRedirect: true }),
  me: () => request('/api/v1/auth/me', { skipAuthRedirect: true }),
  register: (username, password, code) => request('/api/v1/auth/register', {
    method: 'POST',
    body: JSON.stringify({ username, password, code }),
    skipAuthRedirect: true,
  }),
}

// --- Admin (后台) ---
export const adminApi = {
  listAccounts: () => request('/api/v1/admin/accounts'),
  createAccount: (body) => request('/api/v1/admin/accounts', { method: 'POST', body: JSON.stringify(body) }),
  loginAccount: (id, auth_code = '') => request(`/api/v1/admin/accounts/${id}/login`, { method: 'POST', body: JSON.stringify({ auth_code }) }),
  refreshAccount: (id) => request(`/api/v1/admin/accounts/${id}/refresh`, { method: 'POST' }),
  revokeAccount: (id) => request(`/api/v1/admin/accounts/${id}/revoke`, { method: 'POST' }),
  deleteAccount: (id) => request(`/api/v1/admin/accounts/${id}`, { method: 'DELETE' }),

  listDevices: () => request('/api/v1/admin/devices'),
  createDevice: (body) => request('/api/v1/admin/devices', { method: 'POST', body: JSON.stringify(body) }),
  healthDevice: (id) => request(`/api/v1/admin/devices/${id}/health`, { method: 'POST' }),
  deleteDevice: (id) => request(`/api/v1/admin/devices/${id}`, { method: 'DELETE' }),

  listTasks: () => request('/api/v1/admin/tasks'),

  listUsers: () => request('/api/v1/admin/users'),
  createUser: (body) => request('/api/v1/admin/users', { method: 'POST', body: JSON.stringify(body) }),
  updateUser: (id, body) => request(`/api/v1/admin/users/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteUser: (id) => request(`/api/v1/admin/users/${id}`, { method: 'DELETE' }),

  listVouchers: ({ status, user, offset, limit } = {}) => {
    const q = new URLSearchParams()
    if (status) q.set('status', status)
    if (user) q.set('user', user)
    if (offset != null) q.set('offset', String(offset))
    if (limit != null) q.set('limit', String(limit))
    const qs = q.toString()
    return request(`/api/v1/admin/vouchers${qs ? `?${qs}` : ''}`)
  },
  createVouchers: (body) => request('/api/v1/admin/vouchers', { method: 'POST', body: JSON.stringify(body) }),
  deleteVoucher: (id) => request(`/api/v1/admin/vouchers/${id}`, { method: 'DELETE' }),
}

export function statusClass(status) {
  if (['logged_in', 'online', 'succeeded', 'completed'].includes(status)) return 'ok'
  if (['failed', 'error', 'offline'].includes(status)) return 'bad'
  if (['running', 'pending', 'unknown', 'needs_2fa', 'queued', 'downloading', 'decrypting'].includes(status)) return 'warn'
  return ''
}
