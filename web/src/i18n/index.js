import { computed, ref } from 'vue'
import { messages } from './messages'

const locale = ref(readInitialLocale())

function readInitialLocale() {
  const saved = localStorage.getItem('locale')
  if (saved === 'zh' || saved === 'en') return saved
  return navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

function resolve(obj, path) {
  return path.split('.').reduce((acc, key) => (acc == null ? undefined : acc[key]), obj)
}

function format(template, params = {}) {
  return String(template).replace(/\{(\w+)\}/g, (_, key) => params[key] ?? '')
}

export function useI18n() {
  const t = (key, params) => {
    const raw = resolve(messages[locale.value], key) ?? resolve(messages.en, key) ?? key
    return params ? format(raw, params) : raw
  }

  const setLocale = (next) => {
    if (next !== 'zh' && next !== 'en') return
    locale.value = next
    localStorage.setItem('locale', next)
    document.documentElement.lang = next === 'zh' ? 'zh-CN' : 'en'
  }

  return {
    locale: computed(() => locale.value),
    t,
    setLocale,
  }
}

export function initI18n() {
  document.documentElement.lang = locale.value === 'zh' ? 'zh-CN' : 'en'
}
