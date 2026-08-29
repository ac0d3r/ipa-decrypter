<template>
  <button
    type="button"
    class="icon-btn"
    :class="[variant, { busy }]"
    :title="title"
    :aria-label="title"
    :disabled="disabled || busy"
    @click="$emit('click', $event)"
  >
    <AppIcon :name="icon" :spin="busy && icon === 'refresh'" />
  </button>
</template>

<script setup>
import AppIcon from './AppIcon.vue'

defineProps({
  icon: { type: String, required: true },
  title: { type: String, default: '' },
  variant: { type: String, default: 'default' },
  disabled: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
})

defineEmits(['click'])
</script>
