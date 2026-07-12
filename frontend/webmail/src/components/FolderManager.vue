<template>
  <div class="folder-manager">
    <!-- Confirmation dialog -->
    <ConfirmModal
      v-model="confirmModal.show"
      :title="confirmModal.title"
      :message="confirmModal.message"
      :confirm-text="confirmModal.confirmText"
      :loading="confirmModal.loading"
      @confirm="handleConfirmModalConfirm"
      @close="handleConfirmModalClose"
    />

    <!-- Create folder dialog -->
    <div v-if="showCreateDialog" class="modal-backdrop" @click.self="cancelCreate">
      <div class="modal-panel max-w-sm">
        <div class="modal-panel-header">
          <h3 class="text-base font-semibold text-gray-900">New Folder</h3>
          <button @click="cancelCreate" class="p-1 hover:bg-gray-100 rounded">
            <X class="w-5 h-5 text-gray-400" />
          </button>
        </div>
        <div class="modal-panel-body">
          <label class="block text-sm font-medium text-gray-700 mb-2">Folder name</label>
          <input
            ref="folderNameInput"
            v-model="newFolderName"
            type="text"
            placeholder="Enter folder name"
            class="input-field w-full"
            @keyup.enter="handleCreateFolder"
          />
        </div>
        <div class="modal-panel-footer">
          <button @click="cancelCreate" class="px-4 py-2 text-sm text-gray-600 hover:text-gray-900 transition-colors">Cancel</button>
          <button @click="handleCreateFolder" class="btn-primary-sm">Create</button>
        </div>
      </div>
    </div>

    <!-- Rename folder dialog -->
    <div v-if="showRenameDialog" class="modal-backdrop" @click.self="cancelRename">
      <div class="modal-panel max-w-sm">
        <div class="modal-panel-header">
          <h3 class="text-base font-semibold text-gray-900">Rename Folder</h3>
          <button @click="cancelRename" class="p-1 hover:bg-gray-100 rounded">
            <X class="w-5 h-5 text-gray-400" />
          </button>
        </div>
        <div class="modal-panel-body">
          <label class="block text-sm font-medium text-gray-700 mb-2">New name</label>
          <input
            v-model="renameFolderName"
            type="text"
            placeholder="Enter new name"
            class="input-field w-full"
            @keyup.enter="handleRenameFolder"
          />
        </div>
        <div class="modal-panel-footer">
          <button @click="cancelRename" class="px-4 py-2 text-sm text-gray-600 hover:text-gray-900 transition-colors">Cancel</button>
          <button @click="handleRenameFolder" class="btn-primary-sm">Rename</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { useFolderStore } from '../stores/folder'
import { useI18n } from 'vue-i18n'
import ConfirmModal from './ConfirmModal.vue'
import { X } from 'lucide-vue-next'
import { showToast } from '../utils/toast'

const { t } = useI18n()
const folderStore = useFolderStore()

// State
const newFolderName = ref('')
const renameFolderName = ref('')
const renameFolderId = ref<number>(0)
const folderNameInput = ref<HTMLInputElement>()
const showCreateDialog = ref(false)
const showRenameDialog = ref(false)

// Confirmation dialog state
const confirmModal = ref({
  show: false,
  title: '',
  message: '',
  confirmText: '',
  loading: false,
  action: null as { action: string; params?: any; callback: () => Promise<void> } | null
})

// Open create folder dialog
function openCreateDialog() {
  newFolderName.value = ''
  showCreateDialog.value = true
  nextTick(() => {
    folderNameInput.value?.focus()
  })
}

// Cancel create
function cancelCreate() {
  newFolderName.value = ''
  showCreateDialog.value = false
}

// Create folder
async function handleCreateFolder() {
  const name = newFolderName.value.trim()
  if (!name) {
    showToast('warning', t('folders.toastEnterName'))
    return
  }
  try {
    await folderStore.createNewFolder(name)
    showToast('success', t('folders.toastCreated'))
    cancelCreate()
  } catch (error) {
    console.error('Failed to create folder:', error)
    showToast('error', t('folders.toastCreateFailed') + ': ' + (error as Error).message)
  }
}

// Open rename dialog
function openRenameDialog(folderId: number, folderName: string) {
  renameFolderName.value = folderName
  renameFolderId.value = folderId
  showRenameDialog.value = true
}

// Cancel rename
function cancelRename() {
  renameFolderName.value = ''
  renameFolderId.value = 0
  showRenameDialog.value = false
}

// Rename folder
async function handleRenameFolder() {
  const name = renameFolderName.value.trim()
  if (!name) {
    showToast('warning', t('folders.toastEnterNewName'))
    return
  }
  try {
    await folderStore.renameFolderAction(renameFolderId.value, name)
    showToast('success', t('folders.toastRenamed'))
    cancelRename()
  } catch (error) {
    console.error('Failed to rename folder:', error)
    showToast('error', t('folders.toastRenameFailed') + ': ' + (error as Error).message)
  }
}

// Confirm delete folder
function confirmDeleteFolder(folderId: number, folderName: string) {
  confirmModal.value = {
    show: true,
    title: t('folders.deleteFolder'),
    message: `${t('folders.sureDelete')} "${folderName}"? ${t('folders.actionCannotBeUndone')}`,
    confirmText: t('folders.deleteFolder'),
    loading: false,
    action: {
      action: 'delete_folder',
      params: { folderId },
      callback: async () => {
        try {
          confirmModal.value.loading = true
          await folderStore.deleteFolderAction(folderId)
          showToast('success', t('folders.toastDeleted'))
        } catch (error) {
          console.error('Failed to delete folder:', error)
          const errorMsg = (error as Error).message
          showToast('error', errorMsg || t('folders.toastDeleteFailed'))
        } finally {
          confirmModal.value.loading = false
        }
      }
    }
  }
}

// Handle confirmation dialog confirm
function handleConfirmModalConfirm() {
  if (confirmModal.value.action) {
    confirmModal.value.action.callback()
  }
}

// Handle confirmation dialog close
function handleConfirmModalClose() {
  confirmModal.value.show = false
  confirmModal.value.title = ''
  confirmModal.value.message = ''
  confirmModal.value.action = null
  confirmModal.value.loading = false
}

// Expose methods to parent component
defineExpose({
  openCreateDialog,
  openRenameDialog,
  confirmDeleteFolder
})
</script>

<style scoped>
</style>
