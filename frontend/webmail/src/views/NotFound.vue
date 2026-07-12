<template>
  <div class="min-h-screen bg-white flex items-center justify-center">
    <div class="text-center max-w-md px-6">
      <h1 class="text-7xl font-bold text-gray-900 mb-4">404</h1>
      <h2 class="text-xl font-semibold text-gray-700 mb-3">{{ t('notFound.pageNotFound') }}</h2>
      <p class="text-sm text-gray-400 mb-8">
        {{ t('notFound.pageNotFoundDesc') }}
      </p>
      <router-link to="/inbox" class="btn-primary-sm inline-flex">
        <svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 9l9-7 9 7v11a2 2 0 01-2 2H5a2 2 0 01-2-2z"/><polyline points="9,22 9,12 15,12 15,22"/></svg>
        {{ t('notFound.backToInbox') }}
      </router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
// 404 page
</script>