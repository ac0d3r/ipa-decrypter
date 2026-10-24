<template>
  <div ref="root" class="region-picker" :class="{ open, empty: !hasSelection }">
    <div class="region-row">
      <button type="button" class="region-trigger" :disabled="!options.length" @click="toggle">
        <template v-if="hasSelection">
          <CountryFlag :code="modelValue" lg />
          <span class="region-code">{{ modelValue.toUpperCase() }}</span>
        </template>
        <span v-else class="region-placeholder">{{ emptyLabel }}</span>
        <span v-if="options.length" class="chevron" aria-hidden="true">▾</span>
      </button>
      <span v-if="suffix" class="region-suffix">{{ suffix }}</span>
    </div>
    <div v-if="open && options.length" class="region-menu">
      <button
        v-for="r in options"
        :key="r.code"
        type="button"
        class="region-option"
        :class="{ active: r.code === modelValue }"
        @click="pick(r.code)"
      >
        <CountryFlag :code="r.code" lg />
        <span class="region-label">{{ r.label }}</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import CountryFlag from './CountryFlag.vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  regions: { type: Array, default: () => [] },
  emptyLabel: { type: String, default: '—' },
  suffix: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue', 'change'])

const root = ref(null)
const open = ref(false)

const options = computed(() => props.regions || [])
const hasSelection = computed(() => !!props.modelValue)

function toggle() {
  if (!options.value.length) return
  open.value = !open.value
}

function pick(code) {
  if (code !== props.modelValue) {
    emit('update:modelValue', code)
    emit('change', code)
  }
  open.value = false
}

function onDocClick(e) {
  if (!root.value?.contains(e.target)) open.value = false
}

onMounted(() => document.addEventListener('click', onDocClick))
onUnmounted(() => document.removeEventListener('click', onDocClick))
</script>

<style scoped>
.region-picker {
  position: relative;
  flex-shrink: 0;
  margin-left: auto;
}
.region-row {
  display: flex;
  align-items: center;
  gap: 4px;
}
.region-trigger {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.34rem 0.55rem 0.34rem 0.45rem;
  border: 1px solid var(--line);
  border-radius: 9px;
  background: var(--surface);
  color: var(--text);
  font-size: 0.72rem;
  line-height: 1;
  cursor: pointer;
}
.region-suffix {
  flex: 0 0 auto;
  padding: 0.34rem 0.45rem;
  border: 1px solid var(--line);
  border-radius: 9px;
  background: var(--surface);
  color: var(--text-2);
  font-size: 0.72rem;
  font-weight: 600;
  line-height: 1;
  white-space: nowrap;
}
.region-trigger:hover:not(:disabled) {
  background: var(--surface-2);
}
.region-trigger:disabled {
  opacity: 0.7;
  cursor: default;
}
.region-code {
  font-weight: 600;
  letter-spacing: 0.04em;
}
.region-placeholder {
  color: var(--text-3);
  font-weight: 500;
  min-width: 1.2rem;
  text-align: center;
}
.chevron {
  color: var(--text-3);
  font-size: 0.62rem;
  margin-left: 0.1rem;
}
.region-menu {
  position: absolute;
  top: calc(100% + 4px);
  right: 0;
  min-width: 148px;
  max-height: 240px;
  overflow-y: auto;
  padding: 0.3rem;
  border: 1px solid var(--line);
  border-radius: 10px;
  background: var(--surface);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.35);
  z-index: 20;
}
.region-option {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  width: 100%;
  padding: 0.45rem 0.5rem;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--text);
  font-size: 0.78rem;
  text-align: left;
  cursor: pointer;
}
.region-option:hover {
  background: var(--surface-2);
}
.region-option.active {
  background: var(--surface-3);
}
.region-label {
  font-weight: 500;
}
</style>
