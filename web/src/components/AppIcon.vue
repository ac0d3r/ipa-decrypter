<template>
  <svg
    class="app-icon"
    :class="[`app-icon--${name}`, { spin }]"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path v-if="name === 'plus'" d="M12 5v14M5 12h14" />
    <template v-else-if="name === 'refresh'">
      <path d="M21 12a9 9 0 1 1-2.64-6.36" />
      <path d="M21 3v6h-6" />
    </template>
    <template v-else-if="name === 'login'">
      <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4" />
      <path d="M10 17l5-5-5-5" />
      <path d="M15 12H3" />
    </template>
    <template v-else-if="name === 'key'">
      <circle cx="8" cy="16" r="4" />
      <path d="M11.5 14.5 21 5" />
      <path d="M15 5h6v6" />
    </template>
    <template v-else-if="name === 'shield'">
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
      <path d="M9 12l2 2 4-4" />
    </template>
    <template v-else-if="name === 'check'">
      <path d="M20 6 9 17l-5-5" />
    </template>
    <template v-else-if="name === 'logout'">
      <circle cx="12" cy="12" r="9" />
      <path d="m7 7 10 10" />
      <path d="M17 7 7 17" />
    </template>
    <template v-else-if="name === 'trash'">
      <path d="M3 6h18" />
      <path d="M8 6V4h8v2" />
      <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6" />
      <path d="M10 11v6" />
      <path d="M14 11v6" />
    </template>
    <template v-else-if="name === 'globe'">
      <circle cx="12" cy="12" r="9" />
      <path d="M3 12h18" />
      <path d="M12 3a15 15 0 0 1 0 18" />
      <path d="M12 3a15 15 0 0 0 0 18" />
    </template>
    <template v-else-if="name === 'pulse'">
      <path d="M22 12h-4l-3 9L9 3l-3 9H2" />
    </template>
    <template v-else-if="name === 'search'">
      <circle cx="11" cy="11" r="7" />
      <path d="m20 20-4.2-4.2" />
    </template>
  </svg>
</template>

<script setup>
defineProps({
  name: { type: String, required: true },
  spin: { type: Boolean, default: false },
})
</script>

<style scoped>
.app-icon {
  width: 1em;
  height: 1em;
  display: block;
}
.app-icon.spin {
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
