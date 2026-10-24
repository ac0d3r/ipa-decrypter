<template>
  <div class="public">
    <header class="top">
      <div class="brand">
        <img src="/logo.png" alt="" class="brand-logo" />
        <div>
          <h1>{{ t('brand.title') }}</h1>
          <p class="sub">{{ t('brand.subtitle') }}</p>
        </div>
      </div>
      <div class="top-actions">
        <LangSwitch />
        <RouterLink v-if="authState.isAdmin" to="/admin" class="admin-link">{{ t('nav.admin') }}</RouterLink>
        <UserMenu v-else-if="authState.authed" />
        <RouterLink v-else to="/login" class="admin-link">{{ t('nav.login') }}</RouterLink>
      </div>
    </header>

    <p v-if="notice" class="page-notice" role="alert">{{ notice }}</p>

    <div class="search-wrap">
      <div class="mode-row">
        <div class="seg">
          <button :class="{ active: mode === 'search' }" @click="mode = 'search'">{{ t('home.modeSearch') }}</button>
          <button :class="{ active: mode === 'bundle' }" @click="mode = 'bundle'">{{ t('home.modeBundle') }}</button>
        </div>
        <RegionPicker
          v-model="region"
          :regions="regions"
          :empty-label="t('home.noRegion')"
          :suffix="t('home.platform')"
          @change="onRegionChange"
        />
      </div>
      <div class="search-box">
        <input
          v-model="query"
          :placeholder="mode === 'bundle' ? t('home.bundlePlaceholder') : t('home.searchPlaceholder')"
          @keyup.enter="runSearch"
        />
        <IconButton
          icon="search"
          variant="primary"
          class="search-submit"
          :title="t('common.search')"
          :disabled="busy"
          :busy="busy"
          @click="runSearch"
        />
      </div>
    </div>

    <div v-if="queue.length" class="queue-panel">
      <div class="queue-title"><span class="dot"></span> {{ t('home.queue') }}</div>
      <div v-for="(item, i) in queue" :key="item.id" class="queue-item">
        <span class="qi-pos" :class="{ active: item.status === 'decrypting' }">{{ i + 1 }}</span>
        <span class="qi-name">{{ item.app_name }}</span>
        <span class="qi-status" :class="statusClass(item.status)">{{ taskStatus(item.status) }}</span>
      </div>
    </div>

    <div v-if="cache.length" class="feat-block">
      <div class="feat-title">{{ t('home.recent') }}</div>
      <div class="app-list">
        <article v-for="g in recentGroups" :key="g.bundleId" class="app-card done">
          <div class="app-row">
            <img v-if="g.items[0].icon_url" :src="g.items[0].icon_url" class="app-icon" alt="" />
            <div v-else class="app-icon placeholder"></div>
            <div class="app-info">
            <div class="app-name">{{ g.items[0].app_name }}</div>
            <div class="app-meta">
              <span v-if="g.items[0].display_version">v{{ g.items[0].display_version }}</span>
              <span v-if="g.items[0].size_bytes"> · {{ fmtSize(g.items[0].size_bytes) }}</span>
              <span v-if="g.items[0].region" class="meta-region"> · <CountryFlag :code="g.items[0].region" /></span>
            </div>
              <div class="app-bundle">{{ g.bundleId }}</div>
            </div>
            <div class="app-action">
              <button
                v-if="g.items.length > 1"
                type="button"
                class="btn-outline"
                :class="{ open: expandedRecent[g.bundleId] }"
                @click="expandedRecent[g.bundleId] = !expandedRecent[g.bundleId]"
              >
                {{ t('home.versions') }} <span class="chev">{{ expandedRecent[g.bundleId] ? '▴' : '▾' }}</span>
              </button>
              <a
                class="btn btn-dl"
                href="#"
                @click.prevent="downloadTask(g.items[0].id)"
              >{{ t('home.download') }}</a>
            </div>
          </div>
          <div v-if="expandedRecent[g.bundleId]" class="versions-panel">
            <div class="recent-version-list">
              <div v-for="item in g.items" :key="item.id" class="recent-version-row">
                <span class="v-label">{{ cachedVersionLabel(item) || '—' }}</span>
                <CountryFlag v-if="item.region" :code="item.region" />
                <span class="v-size">{{ fmtSize(item.size_bytes) }}</span>
                <a
                  class="btn btn-dl btn-sm"
                  href="#"
                  @click.prevent="downloadTask(item.id)"
                >{{ t('home.download') }}</a>
              </div>
            </div>
          </div>
        </article>
      </div>
    </div>

    <div
      v-if="cache.length && (results.length || loading || searched)"
      class="section-divider"
      role="separator"
    />

    <div v-if="loading" class="loading">{{ t('home.searching') }}</div>
    <div v-else-if="!results.length && searched" class="empty">{{ t('home.notFound') }}</div>

    <div class="app-list">
      <article
        v-for="app in results"
        :key="app.bundle_id"
        class="app-card"
        :class="cardClass(app.bundle_id)"
      >
        <div class="app-row">
          <img v-if="app.icon_url" :src="app.icon_url" class="app-icon" alt="" />
          <div v-else class="app-icon placeholder"></div>
          <div class="app-info">
            <div class="app-name">{{ app.name }}</div>
            <div class="app-meta">
              v{{ app.version }} · {{ fmtSize(app.size_bytes) }} · {{ app.seller }}
            </div>
            <div class="app-bundle">{{ app.bundle_id }}</div>
          </div>
            <div class="app-action">
            <template v-if="!activeJobs[app.bundle_id]">
              <button
                class="btn btn-go"
                @click="onDecrypt(app)"
                :disabled="busy || decryptDisabled(app)"
              >{{ t('home.decrypt') }}</button>
              <button
                type="button"
                class="btn-outline"
                :class="{ open: expandedVersions[app.bundle_id] }"
                @click="toggleVersions(app)"
                :disabled="busy"
              >
                {{ t('home.versions') }} <span class="chev">{{ expandedVersions[app.bundle_id] ? '▴' : '▾' }}</span>
              </button>
              <button
                v-if="cachedFor(app.bundle_id).length"
                type="button"
                class="btn-cached"
                @click="openCachedModal(app)"
              >
                {{ t('home.cached') }}
              </button>
            </template>
            <a
              v-else-if="activeJobs[app.bundle_id]?.status === 'completed'"
              class="btn btn-dl"
              href="#"
              @click.prevent="downloadTask(activeJobs[app.bundle_id].id)"
            >{{ t('home.download') }}</a>
            <button v-else class="btn btn-go" disabled>{{ t('home.processing') }}</button>
            </div>
        </div>
        <div v-if="expandedVersions[app.bundle_id]" class="versions-panel">
          <div v-if="versionState(app.bundle_id).loading" class="versions-loading">
            <span class="spinner"></span>
            <span>{{ t('home.loadingVersions') }}</span>
          </div>
          <div v-else-if="versionState(app.bundle_id).error" class="err">{{ versionState(app.bundle_id).error }}</div>
          <div v-else class="version-list">
            <button
              v-for="v in versionState(app.bundle_id).versions"
              :key="v.external_version_id"
              type="button"
              class="version-item"
              :class="{ selected: versionState(app.bundle_id).selected === v }"
              @click="selectVersion(app.bundle_id, v)"
            >
              <span class="v-label">{{ versionRowLabel(v) }}</span>
              <span v-if="v.release_date" class="v-date">{{ v.release_date.slice(0, 10) }}</span>
            </button>
            <button
              v-if="versionState(app.bundle_id).hasMore"
              type="button"
              class="version-load-more"
              :disabled="versionState(app.bundle_id).loadingMore"
              @click="loadMoreVersions(app)"
            >
              <span v-if="versionState(app.bundle_id).loadingMore" class="spinner"></span>
              {{ versionState(app.bundle_id).loadingMore ? t('common.loading') : t('home.loadMoreVersions') }}
            </button>
          </div>
        </div>
        <div v-if="activeJobs[app.bundle_id]" class="job-panel">
          <div class="job-status-row">
            <span v-if="!isDone(activeJobs[app.bundle_id])" class="spinner"></span>
            <span class="status-text" :class="statusClass(activeJobs[app.bundle_id].status)">
              {{ taskStatus(activeJobs[app.bundle_id].status) }}
            </span>
          </div>
          <div v-if="activeJobs[app.bundle_id].error" class="err">{{ activeJobs[app.bundle_id].error }}</div>
          <div class="log-box">
            <div v-for="(line, i) in activeJobs[app.bundle_id].logs || []" :key="i">{{ line }}</div>
          </div>
        </div>
      </article>
    </div>

    <div v-if="cachedModal.open" class="modal-backdrop" @click.self="closeCachedModal">
      <div class="cached-picker panel">
        <h2>{{ t('home.cached') }}</h2>
        <p class="muted">{{ cachedModal.appName }}</p>
        <p v-if="notice" class="page-notice" role="alert">{{ notice }}</p>
        <div class="cached-header">
          {{ t('home.cachedHeader', { count: cachedFor(cachedModal.bundleId).length }) }}
        </div>
        <div
          v-for="item in cachedFor(cachedModal.bundleId)"
          :key="item.id"
          class="cached-row"
        >
          <span v-if="cachedVersionLabel(item)" class="cached-ver">{{ cachedVersionLabel(item) }}</span>
          <CountryFlag :code="item.region" />
          <span class="cached-size">{{ fmtSize(item.size_bytes) }}</span>
          <a
            class="btn btn-dl btn-sm"
            href="#"
            @click.prevent="downloadTask(item.id)"
          >{{ t('home.download') }}</a>
        </div>
      </div>
    </div>

    <footer class="site-footer">
      <a href="https://github.com/ac0d3r/ipa-decrypter" target="_blank" rel="noopener noreferrer">
        {{ t('home.footerGithub') }}
      </a>
      <span class="sep">·</span>
      <a href="https://github.com/ac0d3r/ipa-decrypter/blob/main/LICENSE" target="_blank" rel="noopener noreferrer">
        {{ t('home.footerLicense') }}
      </a>
    </footer>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, reactive, ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { authApi, publicApi, statusClass } from '../api'
import { authState, clearAuthState, markAuthed } from '../auth'
import LangSwitch from '../components/LangSwitch.vue'
import IconButton from '../components/IconButton.vue'
import RegionPicker from '../components/RegionPicker.vue'
import CountryFlag from '../components/CountryFlag.vue'
import UserMenu from '../components/UserMenu.vue'
import { useI18n } from '../i18n'

const { t } = useI18n()
const router = useRouter()
const loginTarget = computed(() => ({ path: '/login', query: { redirect: router.currentRoute.value.fullPath } }))

async function requireLogin() {
  await router.push(loginTarget.value)
}

const mode = ref('search')
const region = ref('')
const query = ref('')
const regions = ref([])
const results = ref([])
const cache = ref([])
const queue = ref([])
const loading = ref(false)
const busy = ref(false)
const searched = ref(false)
const notice = ref('')

function noticeFromError(e) {
  return e?.message || t('home.downloadFailed')
}

async function downloadTask(id) {
  if (!authState.authed) {
    await requireLogin()
    return
  }
  notice.value = ''
  try {
    const t = await publicApi.task(id)
    if (t.status !== 'completed') {
      notice.value = t('home.downloadFailed')
      return
    }
    window.location.assign(publicApi.taskFileURL(id))
  } catch (e) {
    notice.value = noticeFromError(e)
  }
}
const activeJobs = reactive({})
const cachedByBundle = reactive({})
const expandedVersions = reactive({})
const expandedRecent = reactive({})
const cachedModal = reactive({ open: false, bundleId: '', appName: '' })
const versionData = reactive({})
let pollTimer = null

function taskStatus(status) {
  return t(`tasks.status.${status}`) || status
}

function versionState(bundleId) {
  return versionData[bundleId] || {
    loading: false,
    loadingMore: false,
    error: '',
    versions: [],
    selected: null,
    loaded: false,
    hasMore: false,
    total: 0,
  }
}

function versionRowLabel(v) {
  if (v.display_version) return `v${v.display_version}`
  return t('home.buildVersion', { id: v.external_version_id })
}

function selectVersion(bundleId, v) {
  if (versionData[bundleId]) versionData[bundleId].selected = v
}

function decryptDisabled(app) {
  if (!expandedVersions[app.bundle_id]) return false
  const st = versionState(app.bundle_id)
  return st.loading || !st.selected
}

async function onDecrypt(app) {
  if (!authState.authed) {
    await requireLogin()
    return
  }
  if (expandedVersions[app.bundle_id]) {
    const sel = versionState(app.bundle_id).selected
    if (!sel) return
    expandedVersions[app.bundle_id] = false
    await startDecrypt(app, sel)
    return
  }
  await startDecrypt(app, { latest: true })
}

async function toggleVersions(app) {
  if (!authState.authed) {
    await requireLogin()
    return
  }
  const id = app.bundle_id
  expandedVersions[id] = !expandedVersions[id]
  if (expandedVersions[id]) await loadVersions(app)
}

async function loadMoreVersions(app) {
  await loadVersions(app, { append: true })
}

function openCachedModal(app) {
  cachedModal.open = true
  cachedModal.bundleId = app.bundle_id
  cachedModal.appName = app.name
}

function closeCachedModal() {
  cachedModal.open = false
  cachedModal.bundleId = ''
  cachedModal.appName = ''
}

async function loadVersions(app, { append = false } = {}) {
  const id = app.bundle_id
  const st = versionData[id]
  if (!append) {
    if (st?.loaded && !st.error) return
    versionData[id] = {
      loading: true,
      loadingMore: false,
      error: '',
      versions: [],
      selected: null,
      loaded: false,
      hasMore: false,
      total: 0,
    }
  } else {
    if (!st?.hasMore || st.loadingMore) return
    st.loadingMore = true
  }

  const offset = append ? (versionData[id].versions?.length || 0) : 0
  try {
    const d = await publicApi.versions(id, region.value, app.app_id || 0, offset, 5)
    const batch = d.versions || []
    if (!append && !batch.length) {
      versionData[id] = {
        loading: false,
        loadingMore: false,
        error: t('home.noOlderVersions'),
        versions: [],
        selected: null,
        loaded: true,
        hasMore: false,
        total: d.total || 0,
      }
      return
    }
    const versions = append ? [...(versionData[id].versions || []), ...batch] : batch
    versionData[id] = {
      loading: false,
      loadingMore: false,
      error: '',
      versions,
      selected: versionData[id]?.selected || versions[0] || null,
      loaded: true,
      hasMore: !!d.has_more,
      total: d.total || versions.length,
    }
  } catch (e) {
    if (append) {
      versionData[id].loadingMore = false
      versionData[id].error = e.message
    } else {
      versionData[id] = {
        loading: false,
        loadingMore: false,
        error: e.message,
        versions: [],
        selected: null,
        loaded: true,
        hasMore: false,
        total: 0,
      }
    }
  }
}

function fmtSize(b) {
  b = parseInt(b) || 0
  if (b < 1048576) return (b / 1024).toFixed(0) + ' KB'
  if (b < 1073741824) return (b / 1048576).toFixed(1) + ' MB'
  return (b / 1073741824).toFixed(1) + ' GB'
}

function isDone(job) {
  return job?.status === 'completed' || job?.status === 'failed'
}

function cardClass(bundleId) {
  const j = activeJobs[bundleId]
  if (!j) return ''
  if (j.status === 'completed') return 'done'
  if (j.status === 'failed') return 'error'
  return 'active'
}

function cachedFor(bundleId) {
  return cachedByBundle[bundleId] || []
}

const recentGroups = computed(() => {
  const groups = []
  const byBundle = new Map()
  for (const c of cache.value) {
    let g = byBundle.get(c.bundle_id)
    if (g) {
      g.items.push(c)
    } else {
      g = { bundleId: c.bundle_id, items: [c] }
      byBundle.set(c.bundle_id, g)
      groups.push(g)
    }
  }
  return groups
})

function cachedVersionLabel(item) {
  if (item.display_version) return `v${item.display_version}`
  if (item.external_version_id) return item.external_version_id
  return ''
}

function clearResultPanels() {
  for (const k of Object.keys(cachedByBundle)) delete cachedByBundle[k]
  for (const k of Object.keys(expandedVersions)) delete expandedVersions[k]
  for (const k of Object.keys(versionData)) delete versionData[k]
  closeCachedModal()
}

async function loadCachedForResults(apps) {
  const ids = apps.map(a => a.bundle_id).filter(Boolean)
  clearResultPanels()
  if (!ids.length) return
  try {
    const d = await publicApi.cachedByBundles(region.value, ids)
    Object.assign(cachedByBundle, d.by_bundle || {})
  } catch {}
}

async function loadRegions() {
  const d = await publicApi.regions()
  regions.value = d.regions || []
  if (!regions.value.length) {
    region.value = ''
    return
  }
  if (!regions.value.find(r => r.code === region.value)) {
    region.value = regions.value[0].code
  }
}

async function loadCache() {
  const d = await publicApi.cache()
  cache.value = d.cache || []
}

async function pollQueue() {
  try {
    const d = await publicApi.queue()
    queue.value = d.queue || []
  } catch {}
}

function onRegionChange() {
  results.value = []
  searched.value = false
  clearResultPanels()
}

async function runSearch() {
  const q = query.value.trim()
  if (!q || !region.value) return
  loading.value = true
  searched.value = true
  busy.value = true
  try {
    if (mode.value === 'bundle' || q.includes('.')) {
      const d = await publicApi.search({ bundle_id: q, region: region.value })
      results.value = d.results?.length ? d.results : [{ bundle_id: q, name: q, version: '?', size_bytes: 0, seller: '' }]
    } else {
      const d = await publicApi.search({ q, region: region.value })
      results.value = d.results || []
    }
    await loadCachedForResults(results.value)
  } catch (e) {
    results.value = []
    alert(e.message)
  } finally {
    loading.value = false
    busy.value = false
  }
}

async function startDecrypt(app, ver) {
  notice.value = ''
  busy.value = true
  try {
    const body = {
      bundle_id: app.bundle_id,
      app_name: app.name,
      region: region.value,
      app_id: app.app_id || 0,
      icon_url: app.icon_url || '',
    }
    if (ver && !ver.latest && ver.external_version_id) {
      body.external_version_id = ver.external_version_id
      body.display_version = ver.display_version || ''
    } else if (app.version && app.version !== '?') {
      body.display_version = app.version
    }
    const d = await publicApi.request(body)
    const job = d.task || { id: d.task_id, status: d.status, bundle_id: app.bundle_id, logs: [] }
    if (job.status === 'completed') {
      await downloadTask(job.id)
      return
    }
    activeJobs[app.bundle_id] = job
    startPolling()
    await pollJob(job.id, app.bundle_id)
  } catch (e) {
    notice.value = noticeFromError(e)
  } finally {
    busy.value = false
  }
}

async function pollJob(id, bundleId) {
  try {
    const j = await publicApi.task(id)
    activeJobs[bundleId] = j
    if (!isDone(j)) return
    await loadCache()
    if (results.value.length) await loadCachedForResults(results.value)
  } catch {}
}

function startPolling() {
  if (pollTimer) return
  pollTimer = setInterval(async () => {
    const ids = Object.values(activeJobs).filter(j => j?.id && !isDone(j))
    if (!ids.length) {
      clearInterval(pollTimer)
      pollTimer = null
      return
    }
    for (const j of ids) {
      await pollJob(j.id, j.bundle_id)
    }
    await pollQueue()
  }, 1500)
}

onMounted(async () => {
  await refreshAuth()
  await loadRegions()
  await loadCache()
  await pollQueue()
  setInterval(pollQueue, 3000)
})
onUnmounted(() => { if (pollTimer) clearInterval(pollTimer) })

async function refreshAuth() {
  try {
    const me = await authApi.me()
    markAuthed({ admin: !!me?.is_admin, name: me?.username || '' })
  } catch {
    clearAuthState()
  }
}
</script>

<style scoped>
.public { max-width: 620px; margin: 0 auto; padding: 1rem 1rem 4rem; color: var(--text); }
.top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 1rem 0;
}
.brand { display: flex; gap: .6rem; align-items: center; min-width: 0; }
.brand-logo {
  width: 34px;
  height: 34px;
  border-radius: 9px;
  object-fit: cover;
  flex-shrink: 0;
  border: 1px solid var(--line);
}
h1 { font-size: 1.35rem; margin: 0; font-weight: 650; line-height: 1.15; }
.sub { color: var(--text-3); font-size: .74rem; margin: .2rem 0 0; }
.top-actions {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  flex-shrink: 0;
}
.admin-link {
  font: inherit;
  font-size: .68rem;
  color: var(--text-2);
  padding: 0.28rem 0.55rem;
  border: 1px solid var(--line);
  border-radius: 8px;
  text-decoration: none;
  line-height: 1.2;
  background: transparent;
  cursor: pointer;
}
.admin-link:disabled { opacity: 0.45; cursor: default; }
.page-notice {
  margin: 0 0 0.75rem;
  padding: 0.5rem 0.7rem;
  border-radius: 8px;
  border: 1px solid #4a2020;
  background: #2a1414;
  color: var(--red);
  font-size: 0.78rem;
}
.search-wrap { position: sticky; top: 0; z-index: 10; padding: .5rem 0 1rem; background: var(--bg); }
.mode-row { display: flex; gap: .5rem; align-items: center; margin-bottom: .5rem; }
.seg {
  display: inline-flex;
  padding: 2px;
  gap: 2px;
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: 9px;
}
.seg button {
  padding: .32rem .62rem;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--text-3);
  cursor: pointer;
  font-size: .72rem;
  line-height: 1.2;
}
.seg button.active { background: var(--surface-3); color: var(--text); }
.search-box { display: flex; gap: .45rem; align-items: stretch; }
.search-box input {
  flex: 1;
  min-width: 0;
  padding: .58rem .8rem;
  border-radius: 10px;
}
.search-box :deep(.search-submit.icon-btn) {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  font-size: 1.05rem;
  flex-shrink: 0;
}
.app-list { display: flex; flex-direction: column; gap: .55rem; margin-top: .5rem; }
.app-card { background: var(--surface); border: 1px solid var(--line); border-radius: 15px; overflow: hidden; }
.app-card.active { border-color: #2a3a52; }
.app-card.done { border-color: #1c4a30; }
.app-card.error { border-color: #4a2020; }
.app-row { display: flex; align-items: center; gap: .75rem; padding: .75rem .9rem; }
.app-icon { width: 48px; height: 48px; border-radius: 12px; object-fit: cover; background: var(--surface-3); flex-shrink: 0; }
.app-icon.placeholder { display: block; }
.app-info { flex: 1; min-width: 0; }
.app-name { font-weight: 590; font-size: .88rem; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.app-meta { color: var(--text-3); font-size: .68rem; margin-top: .15rem; }
.meta-region {
  display: inline-flex;
  align-items: center;
  vertical-align: middle;
}
.app-bundle { color: var(--text-4); font-size: .66rem; font-family: var(--mono); margin-top: .1rem; overflow: hidden; text-overflow: ellipsis; }
.app-action {
  flex-shrink: 0;
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: .22rem;
}
.app-action .btn,
.app-action .btn-outline,
.app-action .btn-cached {
  padding: .28rem .48rem;
  font-size: .64rem;
}
.btn-outline {
  border-radius: 8px;
  border: 1px solid var(--line-2);
  background: transparent;
  color: var(--text-2);
  font-weight: 550;
  cursor: pointer;
  line-height: 1.2;
  white-space: nowrap;
}
.btn-outline:hover:not(:disabled) { background: var(--surface-2); color: var(--text); }
.btn-outline.open { background: var(--surface-2); border-color: var(--text-3); color: var(--text); }
.btn-outline:disabled { opacity: .4; cursor: default; }
.btn-cached {
  border-radius: 8px;
  border: 1px solid #2d6b45;
  background: transparent;
  color: #6ee7a0;
  font-weight: 600;
  cursor: pointer;
  line-height: 1.2;
  white-space: nowrap;
}
.btn-cached:hover { background: #0f2419; }
.btn-cached.open { background: #0f2419; border-color: #3d8f5c; }
.chev { font-size: .58rem; opacity: .85; }
.versions-panel {
  border-top: 1px solid var(--line);
  background: var(--bg-soft);
  padding: .55rem .9rem .65rem;
}
.versions-loading {
  display: flex;
  align-items: center;
  gap: .5rem;
  font-size: .72rem;
  color: var(--text-3);
  padding: .2rem 0;
}
.version-list {
  display: flex;
  flex-direction: column;
  gap: .35rem;
  max-height: 240px;
  overflow-y: auto;
}
.version-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  text-align: left;
  padding: .5rem .65rem;
  border-radius: 9px;
  border: 1px solid var(--line);
  background: var(--surface-2);
  color: var(--text);
  cursor: pointer;
  font-size: .78rem;
}
.version-item.selected { border-color: var(--accent); background: #12203c; }
.version-load-more {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: .45rem;
  width: 100%;
  margin-top: .15rem;
  padding: .45rem .65rem;
  border-radius: 9px;
  border: 1px dashed var(--line-2);
  background: transparent;
  color: var(--text-2);
  font-size: .72rem;
  cursor: pointer;
}
.version-load-more:hover:not(:disabled) { background: var(--surface-2); color: var(--text); }
.version-load-more:disabled { opacity: .6; cursor: default; }
.v-label { font-size: .8rem; font-weight: 550; }
.v-date { font-size: .62rem; color: var(--text-3); }
.recent-version-list {
  display: flex;
  flex-direction: column;
  gap: .35rem;
  max-height: 240px;
  overflow-y: auto;
}
.recent-version-row {
  display: flex;
  align-items: center;
  gap: .45rem;
  padding: .5rem .65rem;
  border-radius: 9px;
  border: 1px solid var(--line);
  background: var(--surface-2);
  font-size: .78rem;
}
.recent-version-row .v-size { flex: 1; font-size: .62rem; color: var(--text-3); }
.cached-picker { width: min(420px, 100%); margin: 0; max-height: 80vh; overflow: auto; }
.cached-picker h2 { margin: 0 0 .35rem; font-size: 1rem; }
.cached-picker .cached-header {
  font-size: .62rem;
  font-weight: 700;
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: .04em;
  margin: .75rem 0 .45rem;
}
.cached-picker .cached-row {
  display: flex;
  align-items: center;
  gap: .45rem;
  padding: .5rem 0;
  border-top: 1px solid var(--line);
}
.cached-picker .cached-row:first-of-type { border-top: 0; }
.cached-picker .cached-ver { font-size: .76rem; font-weight: 600; min-width: 4rem; }
.cached-picker .cached-size { flex: 1; font-size: .58rem; color: var(--text-3); }
.cached-picker .btn-sm { padding: .22rem .42rem; font-size: .58rem; }
.btn-sm { padding: .28rem .5rem; font-size: .64rem; }
.btn {
  padding: .42rem .72rem;
  border-radius: 8px;
  border: 0;
  font-size: .74rem;
  font-weight: 600;
  cursor: pointer;
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  line-height: 1.2;
  white-space: nowrap;
}
.btn-go { background: var(--accent); color: #fff; }
.btn-go:disabled { opacity: .4; cursor: default; }
.btn-dl { background: var(--green); color: var(--green-ink); }
.job-panel { border-top: 1px solid var(--line); padding: .65rem .9rem; background: var(--bg-soft); }
.job-status-row { display: flex; align-items: center; gap: .5rem; font-size: .78rem; }
.spinner { width: 13px; height: 13px; border: 2px solid var(--line-2); border-top-color: var(--accent); border-radius: 50%; animation: spin .7s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.status-text.ok { color: #6ee7a0; }
.err { color: var(--red); font-size: .75rem; margin: .3rem 0; }
.log-box {
  background: #08080a;
  border: 1px solid var(--line);
  border-radius: 8px;
  padding: .55rem .7rem;
  font-family: var(--mono);
  font-size: .66rem;
  color: var(--text-3);
  max-height: 140px;
  overflow-y: auto;
  margin-top: .4rem;
}
.queue-panel { background: var(--surface); border: 1px solid var(--line); border-radius: 15px; padding: .8rem .95rem; margin-bottom: 1rem; }
.queue-title { font-size: .68rem; font-weight: 700; color: var(--text-2); text-transform: uppercase; letter-spacing: .06em; margin-bottom: .5rem; display: flex; align-items: center; gap: .4rem; }
.dot { width: 6px; height: 6px; border-radius: 50%; background: var(--green); }
.queue-item { display: flex; align-items: center; gap: .55rem; padding: .35rem 0; font-size: .76rem; border-top: 1px solid var(--line); }
.qi-pos { width: 21px; height: 21px; border-radius: 6px; background: var(--surface-3); display: flex; align-items: center; justify-content: center; font-size: .62rem; }
.qi-pos.active { background: var(--green); color: var(--green-ink); }
.qi-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.qi-status { font-size: .66rem; color: var(--text-3); }
.feat-title { font-size: .68rem; font-weight: 700; color: var(--text-3); text-transform: uppercase; margin: 1rem 0 .4rem; }
.section-divider {
  height: 0;
  border: 0;
  border-top: 1px solid var(--line);
  margin: 1.1rem 0 0.85rem;
}
.loading, .empty { text-align: center; padding: 2rem; color: var(--text-3); }
.site-footer {
  margin-top: 3rem;
  padding: 1.25rem 0 0.5rem;
  border-top: 1px solid var(--line);
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 0.35rem 0.15rem;
  font-size: 0.72rem;
  color: var(--text-3);
  line-height: 1.5;
}
.site-footer a {
  color: var(--text-2);
  text-decoration: none;
}
.site-footer a:hover { color: var(--text); text-decoration: underline; }
.site-footer .sep {
  margin: 0 0.35rem;
  color: var(--text-4);
  user-select: none;
}
</style>
