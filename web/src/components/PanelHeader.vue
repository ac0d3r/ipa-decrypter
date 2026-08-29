<template>
  <div class="panel-header">
    <h2>{{ title }}</h2>
    <div class="panel-actions">
      <slot />
    </div>
  </div>
</template>

<script setup>
defineProps({
  title: { type: String, required: true },
})
</script>
