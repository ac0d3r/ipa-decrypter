<template>
  <span class="country-flag" :class="{ lg }" :title="code?.toUpperCase()">{{ flag }}</span>
</template>

<script setup>
import { computed } from 'vue'
import { countryFlag } from '../utils/flags'

const props = defineProps({
  code: { type: String, default: '' },
  lg: { type: Boolean, default: false },
})

const flag = computed(() => countryFlag(props.code))
</script>

<style scoped>
.country-flag {
  display: inline-block;
  font-size: 1em;
  line-height: 1;
  font-family: 'Apple Color Emoji', 'Segoe UI Emoji', 'Noto Color Emoji', sans-serif;
}
.country-flag.lg {
  font-size: 1.25em;
}
</style>
