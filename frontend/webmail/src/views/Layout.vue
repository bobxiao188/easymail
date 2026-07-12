<template>
  <div class="h-screen w-screen flex flex-col dark:bg-dark-bg">
    <!-- Confirm Modal -->
    <ConfirmModal
      v-model="confirmModal.show"
      :title="confirmModal.title"
      :message="confirmModal.message"
      :confirm-text="confirmModal.confirmText"
      :loading="confirmModal.loading"
      @confirm="handleConfirmModalConfirm"
      @close="handleConfirmModalClose"
    />

    <!-- App Header -->
    <AppHeader />
    
    <div class="flex flex-1 overflow-hidden dark:bg-dark-bg">
      <!-- Sidebar -->
      <NavSidebar class="w-60 shrink-0" />
      
      <!-- Main content area -->
      <div class="flex flex-1 overflow-hidden dark:bg-dark-bg">
        <router-view />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import ConfirmModal from '../components/ConfirmModal.vue'
import AppHeader from '../components/layout/AppHeader.vue'
import NavSidebar from '../components/layout/NavSidebar.vue'


const confirmModal = ref({
  show: false,
  title: '',
  message: '',
  confirmText: '',
  loading: false,
  action: null as { action: string; callback: () => void } | null
})

function handleConfirmModalConfirm() {
  if (confirmModal.value.action) {
    confirmModal.value.action.callback()
  }
}

function handleConfirmModalClose() {
  confirmModal.value.show = false
  confirmModal.value.title = ''
  confirmModal.value.message = ''
  confirmModal.value.action = null
}
</script>

<style>
body {
  margin: 0;
  font-family: 'Segoe UI', system-ui, sans-serif;
}
</style>
