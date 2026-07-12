<!-- src/components/mail/MoveDialog.vue -->
<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { 
  FolderIcon, 
  InboxIcon, 
  PaperAirplaneIcon,
  DocumentTextIcon,
  TrashIcon,
  ExclamationTriangleIcon,
  XMarkIcon,
  CheckIcon
} from '@heroicons/vue/24/outline'
import type { Folder } from '../../types'
import { FolderKind } from '../../utils/folder'

const { t } = useI18n()

const props = defineProps<{
  folders: Folder[]
  currentFolderId: number | null
}>()

const emit = defineEmits<{
  close: []
  confirm: [folderId: number]
}>()

const selectedFolderId = ref<number | null>(null)
const searchKeyword = ref('')

// Filter out current folder
const availableFolders = computed(() => {
  return props.folders.filter(f => f.id !== props.currentFolderId)
})

// Search filter
const filteredFolders = computed(() => {
  if (!searchKeyword.value) return availableFolders.value
  const keyword = searchKeyword.value.toLowerCase()
  return availableFolders.value.filter(f => 
    f.name.toLowerCase().includes(keyword) || 
    f.imapName.toLowerCase().includes(keyword)
  )
})

// Folder icon mapping
function getFolderIcon(kind: number) {
  switch (kind) {
    case FolderKind.Inbox:
      return InboxIcon
    case FolderKind.Sent:
      return PaperAirplaneIcon
    case FolderKind.Draft:
      return DocumentTextIcon
    case FolderKind.Trash:
      return TrashIcon
    case FolderKind.Spam:
      return ExclamationTriangleIcon
    default:
      return FolderIcon
  }
}

function getFolderIconClass(kind: number): string {
  switch (kind) {
    case FolderKind.Inbox:
      return 'text-blue-500'
    case FolderKind.Sent:
      return 'text-green-500'
    case FolderKind.Draft:
      return 'text-gray-500'
    case FolderKind.Trash:
      return 'text-red-500'
    case FolderKind.Spam:
      return 'text-orange-500'
    default:
      return 'text-purple-500'
  }
}

function handleSelect(folderId: number) {
  selectedFolderId.value = folderId
}

function handleConfirm() {
  if (selectedFolderId.value) {
    emit('confirm', selectedFolderId.value)
  }
}

function handleClose() {
  selectedFolderId.value = null
  searchKeyword.value = ''
  emit('close')
}

// When folder list changes, if current selection is unavailable, reset
watch(availableFolders, (folders) => {
  if (selectedFolderId.value && !folders.find(f => f.id === selectedFolderId.value)) {
    selectedFolderId.value = null
  }
})
</script>

<template>
  <!-- Backdrop -->
  <div class="fixed inset-0 z-40 flex items-center justify-center" @click="handleClose">
    <div class="fixed inset-0 bg-black/40 backdrop-blur-sm" />
    
    <!-- Dialog -->
    <div 
      class="relative z-50 w-full max-w-md mx-4 bg-white dark:bg-gray-800 rounded-xl shadow-2xl border border-gray-200 dark:border-gray-700 overflow-hidden"
      @click.stop
    >
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-4 border-b border-gray-200 dark:border-gray-700">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">{{ t('moveDialog.moveToFolder') }}</h3>
        <button 
          @click="handleClose" 
          class="p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
        >
          <XMarkIcon class="w-5 h-5 text-gray-500" />
        </button>
      </div>

      <!-- Search -->
      <div class="px-5 py-3 border-b border-gray-200 dark:border-gray-700">
        <div class="relative">
          <input
            v-model="searchKeyword"
            type="text"
            class="w-full pl-4 pr-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 outline-none focus:border-primary focus:ring-1 focus:ring-primary"
            :placeholder="t('moveDialog.searchFolders')"
          />
        </div>
      </div>

      <!-- Folder List -->
      <div class="max-h-80 overflow-y-auto py-2">
        <div v-if="filteredFolders.length === 0" class="px-5 py-8 text-center text-gray-500 dark:text-gray-400">
          <p class="text-sm">No folders available</p>
        </div>
        
        <div
          v-for="folder in filteredFolders"
          :key="folder.id"
          @click="handleSelect(folder.id)"
          :class="[
            'flex items-center gap-3 px-5 py-3 cursor-pointer transition-colors',
            selectedFolderId === folder.id 
              ? 'bg-blue-50 dark:bg-blue-900/30' 
              : 'hover:bg-gray-50 dark:hover:bg-gray-700'
          ]"
        >
          <component 
            :is="getFolderIcon(folder.kind)" 
            :class="['w-5 h-5 shrink-0', getFolderIconClass(folder.kind)]" 
          />
          <span :class="[
            'flex-1 text-sm',
            selectedFolderId === folder.id 
              ? 'font-semibold text-primary' 
              : 'text-gray-700 dark:text-gray-300'
          ]">
            {{ folder.name }}
          </span>
          <CheckIcon 
            v-if="selectedFolderId === folder.id" 
            class="w-5 h-5 text-primary" 
          />
        </div>
      </div>

      <!-- Footer -->
      <div class="flex items-center justify-end gap-2 px-5 py-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
        <button 
          @click="handleClose"
          class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors"
        >
          {{ t('common.cancel') }}
        </button>
        <button 
          @click="handleConfirm"
          :disabled="!selectedFolderId"
          class="px-4 py-2 text-sm font-medium text-white bg-primary rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {{ t('moveDialog.moveHere') }}
        </button>
      </div>
    </div>
  </div>
</template>
