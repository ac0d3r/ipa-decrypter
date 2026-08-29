<template>
  <div class="lang-switch">
    <IconButton
      icon="globe"
      :title="locale === 'zh' ? t('lang.en') : t('lang.zh')"
      @click="toggle"
    />
  </div>
</template>

<script setup>
import IconButton from './IconButton.vue'
import { useI18n } from '../i18n'

const { locale, t, setLocale } = useI18n()

function toggle() {
  setLocale(locale.value === 'zh' ? 'en' : 'zh')
}
</script>

<style scoped>
.lang-switch {
  display: inline-flex;
}
</style>
