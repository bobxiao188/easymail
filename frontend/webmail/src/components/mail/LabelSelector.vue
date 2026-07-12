<!-- LabelSelector.vue - Label selector modal -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useSettingStore } from '../../stores/setting'
import { useI18n } from 'vue-i18n'
import { setEmailLabels } from '../../api/labels'
import { XMarkIcon, TagIcon } from '@heroicons/vue/24/outline'
import { showToast } from '../../utils/toast'

const { t } = useI18n()
const props = defineProps<{
  emailId: number
  currentLabelId?: number | null
}>()

const emit = defineEmits<{
  close: []
  change: [labelId: number | null]
}>()

const settingStore = useSettingStore()
const isLoading = ref(false)

// Ensure labels are loaded
onMounted(async () => {
  if (settingStore.labels.length === 0) {
    await settingStore.loadLabels()
  }
})

// Select label
async function selectLabel(labelId: number) {
  isLoading.value = true
  try {
    // Set label (replaces all existing labels)
    await setEmailLabels(props.emailId, [labelId])
    emit('change', labelId)
    emit('close')
  } catch (error) {
    console.error('Failed to set label:', error)
    showToast('error', t('mail.failedToSetLabel'))
  } finally {
    isLoading.value = false
  }
}

// Cancel label
async function removeLabel() {
  isLoading.value = true
  try {
    // Set to empty array, remove all labels
    await setEmailLabels(props.emailId, [])
    emit('change', null)
    emit('close')
  } catch (error) {
    console.error('Failed to remove label:', error)
    showToast('error', t('mail.failedToRemoveLabel'))
  } finally {
    isLoading.value = false
  }
}

// Close dialog
function handleClose() {
  emit('close')
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center" @click="handleClose">
    <!-- Semi-transparent backdrop -->
    <div class="absolute inset-0 bg-black/30 animate-in fade-in duration-200"></div>
    
    <!-- Modal content -->
    <div 
      class="relative z-10 w-full max-w-md bg-white dark:bg-dark-surface rounded-lg shadow-xl border border-gray-200 dark:border-dark-border animate-in zoom-in-95 duration-200"
      @click.stop
    >
      <!-- Header -->
      <div class="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-dark-border">
        <div class="flex items-center gap-2">
          <TagIcon class="w-5 h-5 text-primary" />
          <h3 class="text-lg font-semibold text-text-primary dark:text-dark-text">Set Label</h3>
        </div>
        <button 
          @click="handleClose" 
          class="p-1 rounded hover:bg-gray-100 dark:hover:bg-dark-bg transition-colors"
        >
          <XMarkIcon class="w-5 h-5 text-text-secondary dark:text-dark-text" />
        </button>
      </div>

      <!-- Labels list -->
      <div class="p-4 max-h-80 overflow-y-auto">
        <!-- No label option -->
        <button
          @click="removeLabel"
          :disabled="isLoading"
          class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-gray-100 dark:hover:bg-dark-bg transition-colors text-left"
          :class="{ 'bg-gray-100 dark:bg-dark-bg': props.currentLabelId === null }"
        >
          <div class="w-4 h-4 rounded border-2 border-gray-300 dark:border-gray-600"></div>
          <span class="text-sm text-text-secondary dark:text-gray-400">No label</span>
          <svg v-if="props.currentLabelId === null" class="w-4 h-4 ml-auto text-primary" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
          </svg>
        </button>

        <!-- Labels list -->
        <div class="mt-2 space-y-1">
          <div class="text-xs font-medium text-text-secondary dark:text-gray-400 px-3">Available labels</div>
          <button
            v-for="label in settingStore.labels"
            :key="label.id"
            @click="selectLabel(label.id)"
            :disabled="isLoading"
            class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-gray-100 dark:hover:bg-dark-bg transition-colors text-left"
            :class="{ 'bg-gray-100 dark:bg-dark-bg': props.currentLabelId === label.id }"
          >
            <div 
              class="w-4 h-4 rounded shrink-0" 
              :style="{ backgroundColor: label.color }"
            ></div>
            <span class="text-sm text-text-primary dark:text-dark-text">{{ label.name }}</span>
            <svg v-if="props.currentLabelId === label.id" class="w-4 h-4 ml-auto text-primary" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
            </svg>
          </button>
        </div>

        <!-- Empty state -->
        <div v-if="settingStore.labels.length === 0" class="text-center py-4 text-text-secondary dark:text-gray-400">
          <p class="text-sm">No labels available</p>
          <p class="text-xs mt-1">Create labels in Settings first</p>
        </div>
      </div>

      <!-- Footer -->
      <div class="flex justify-end px-4 py-3 border-t border-gray-200 dark:border-dark-border">
        <button
          @click="handleClose"
          :disabled="isLoading"
          class="px-4 py-2 text-sm text-text-secondary hover:text-text-primary hover:bg-gray-100 dark:hover:bg-dark-bg rounded transition-colors"
        >
          Cancel
        </button>
      </div>
    </div>
  </div>
</template>
