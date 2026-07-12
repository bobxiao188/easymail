<template>
  <div v-if="showConfirm" class="modal-backdrop" @click.self="handleClose">
    <div class="modal-panel max-w-sm">
      <div class="modal-panel-header">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-dark-text">{{ title }}</h3>
        <button @click="handleClose" class="p-1 hover:bg-gray-100 dark:hover:bg-dark-bg rounded-lg transition-colors">
          <svg class="w-5 h-5 text-gray-400 dark:text-gray-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
            stroke="currentColor" stroke-width="2">
            <path d="M18 6L6 18M6 6l12 12" />
          </svg>
        </button>
      </div>
      <div class="modal-panel-body">
        <p class="text-sm text-gray-600 dark:text-gray-400">{{ message }}</p>
      </div>
      <div class="modal-panel-footer">
        <button @click="handleClose"
          class="px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-dark-text hover:bg-gray-100 dark:hover:bg-dark-bg active:bg-gray-200 rounded-full transition-colors cursor-pointer">
          {{ t("common.cancel") }}
        </button>
        <button @click="handleConfirm"
          class="inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium text-white bg-red-500 hover:bg-red-600 rounded-full transition-colors"
          :disabled="loading">
          <svg v-if="loading" class="animate-spin h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none"
            viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
          {{ confirmText || 'Confirm' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  modelValue: boolean
  title: string
  message: string
  confirmText?: string
  loading?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  confirm: []
  close: []
}>()

const showConfirm = ref(props.modelValue)

watch(() => props.modelValue, (newVal) => { showConfirm.value = newVal })
watch(showConfirm, (newVal) => { emit('update:modelValue', newVal) })

function handleClose() { emit('close'); showConfirm.value = false }
function handleConfirm() { emit('confirm'); showConfirm.value = false }
</script>