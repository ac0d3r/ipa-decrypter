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

async function request(path, options = {}) {
  const res = await fetch(BASE + path, {
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
    ...options,
  })
  if (res.status === 204) return null
  const text = await res.text()
  let data = null
  try { data = text ? JSON.parse(text) : null } catch { data = { error: text } }
  if (!res.ok) {
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
  lookup: (id, region) => request(`/api/v1/lookup?id=${encodeURIComponent(id)}&region=${encodeURIComponent(region)}`),
  search: (q, region) => request(`/api/v1/search?q=${encodeURIComponent(q)}&region=${encodeURIComponent(region)}`),
  versions: (bundle_id, region, app_id = 0, offset = 0, limit = 5) => {
    const q = new URLSearchParams({ bundle_id, region })
    if (app_id) q.set('app_id', String(app_id))
    q.set('offset', String(offset))
    q.set('limit', String(limit))
    return request(`/api/v1/versions?${q}`)
  },
  request: (body) => request('/api/v1/request', { method: 'POST', body: JSON.stringify(body) }),
  job: (id) => request(`/api/v1/jobs/${id}`),
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
  downloadURL: (bundleId, region) => `/api/v1/download/${encodeURIComponent(bundleId)}?region=${encodeURIComponent(region)}`,
  taskFileURL: (id) => `/api/v1/tasks/${id}/file`,
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
}

export function statusClass(status) {
  if (['logged_in', 'online', 'succeeded', 'completed'].includes(status)) return 'ok'
  if (['failed', 'error', 'offline'].includes(status)) return 'bad'
  if (['running', 'pending', 'unknown', 'needs_2fa', 'queued', 'downloading', 'decrypting'].includes(status)) return 'warn'
  return ''
}
