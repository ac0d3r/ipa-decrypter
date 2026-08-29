/** Convert ISO 3166-1 alpha-2 code to flag emoji (e.g. us -> 🇺🇸). */
export function countryFlag(code) {
  const normalized = String(code || '').trim().toLowerCase()
  if (!/^[a-z]{2}$/.test(normalized)) return '🌐'
  const upper = normalized.toUpperCase()
  const base = 0x1f1e6
  const a = 'A'.charCodeAt(0)
  return String.fromCodePoint(
    base + upper.charCodeAt(0) - a,
    base + upper.charCodeAt(1) - a,
  )
}
