<template>
  <div class="flex-1 w-full overflow-y-auto bg-gray-50 dark:bg-dark-bg">
    <div class="px-6 py-8">
      <!-- Page Title -->
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-dark-text">{{ t('folders.pageTitle') }}</h1>
          <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">{{ t('folders.pageDesc') }}</p>
        </div>
        <button @click="openCreateDialog()" class="btn-primary-sm flex items-center gap-2">
          <PlusIcon class="w-4 h-4" />
          {{ t('folders.newFolder') }}
        </button>
      </div>

      <!-- Create Folder Dialog -->
      <div v-if="showCreateDialog" class="fixed inset-0 z-50 flex items-center justify-center">
        <div class="fixed inset-0 bg-black/30" @click="showCreateDialog = false" />
        <div class="relative bg-white dark:bg-dark-surface rounded-lg shadow-xl w-full max-w-sm p-6 animate-in fade-in zoom-in duration-200">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-dark-text mb-4">{{ t('folders.newFolder') }}</h3>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('folders.folderName') }}</label>
            <input
              ref="createFolderInput"
              v-model="newFolderName"
              type="text"
              :placeholder="t('folders.enterFolderName')"
              class="input-field w-full"
              @keyup.enter="handleCreateFolder"
            />
          </div>
          <div class="flex justify-end gap-3">
            <button @click="showCreateDialog = false" class="px-4 py-2 text-sm text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-dark-text rounded hover:bg-gray-100 dark:hover:bg-dark-bg transition-colors">{{ t('common.cancel') }}</button>
            <button @click="handleCreateFolder" class="btn-primary-sm">{{ t('common.create') }}</button>
          </div>
        </div>
      </div>

      <!-- Rename Folder Dialog -->
      <div v-if="showRenameDialog" class="fixed inset-0 z-50 flex items-center justify-center">
        <div class="fixed inset-0 bg-black/30" @click="showRenameDialog = false" />
        <div class="relative bg-white dark:bg-dark-surface rounded-lg shadow-xl w-full max-w-sm p-6 animate-in fade-in zoom-in duration-200">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-dark-text mb-4">{{ t('folders.renameFolder') }}</h3>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('folders.newName') }}</label>
            <input
              ref="renameFolderInput"
              v-model="renameFolderName"
              type="text"
              :placeholder="t('folders.enterNewName')"
              class="input-field w-full"
              @keyup.enter="handleRenameFolder"
            />
          </div>
          <div class="flex justify-end gap-3">
            <button @click="showRenameDialog = false" class="px-4 py-2 text-sm text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-dark-text rounded hover:bg-gray-100 dark:hover:bg-dark-bg transition-colors">{{ t('common.cancel') }}</button>
            <button @click="handleRenameFolder" class="btn-primary-sm">{{ t('common.save') }}</button>
          </div>
        </div>
      </div>

      <!-- Delete Confirmation Dialog -->
      <div v-if="showDeleteDialog" class="fixed inset-0 z-50 flex items-center justify-center">
        <div class="fixed inset-0 bg-black/30" @click="showDeleteDialog = false" />
        <div class="relative bg-white dark:bg-dark-surface rounded-lg shadow-xl w-full max-w-sm p-6 animate-in fade-in zoom-in duration-200">
          <div class="flex items-center gap-3 mb-4">
            <div class="w-10 h-10 rounded-full bg-red-100 flex items-center justify-center flex-shrink-0">
              <TrashIcon class="w-5 h-5 text-red-600" />
            </div>
            <div>
              <h3 class="text-lg font-semibold text-gray-900 dark:text-dark-text">{{ t('folders.deleteFolder') }}</h3>
              <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('folders.actionCannotBeUndone') }}</p>
            </div>
          </div>
          <p class="text-gray-700 dark:text-gray-300 mb-6">{{ t('folders.sureDelete') }} "<span class="font-semibold">{{ deleteFolderName }}</span>"?</p>
          <div class="flex justify-end gap-3">
            <button @click="showDeleteDialog = false" class="px-4 py-2 text-sm text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-dark-text rounded hover:bg-gray-100 dark:hover:bg-dark-bg transition-colors">{{ t('common.cancel') }}</button>
            <button @click="handleDeleteFolder" class="px-4 py-2 text-sm bg-red-600 text-white rounded hover:bg-red-700 transition-colors">{{ t('common.delete') }}</button>
          </div>
        </div>
      </div>

      <!-- System Folders -->
      <div class="mb-8">
        <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wider mb-3">{{ t('folders.systemFolders') }}</h2>
        <div class="bg-white dark:bg-dark-surface rounded-lg border border-gray-200 dark:border-dark-border overflow-hidden">
          <table class="min-w-full">
            <thead class="bg-gray-50 dark:bg-dark-bg border-b border-gray-200 dark:border-dark-border">
              <tr>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('folders.folder') }}</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('folders.imapPath') }}</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('folders.unread') }}</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('folders.total') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
              <tr 
                v-for="folder in systemFolders" 
                :key="folder.id"
                class="hover:bg-gray-50 dark:hover:bg-dark-bg transition-colors"
              >
                <td class="px-6 py-4">
                  <div class="flex items-center gap-3">
                    <component :is="getFolderIcon(folder.kind)" class="w-5 h-5 text-gray-400" />
                    <span class="font-medium text-gray-900 dark:text-dark-text">{{ folder.name }}</span>
                  </div>
                </td>
                <td class="px-6 py-4 text-sm text-gray-500 dark:text-gray-400 font-mono">{{ folder.imapName }}</td>
                <td class="px-6 py-4 text-sm text-gray-900 dark:text-dark-text">{{ folder.unreadCount || 0 }}</td>
                <td class="px-6 py-4 text-sm text-gray-900 dark:text-dark-text">{{ folder.totalCount || 0 }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Custom Folders -->
      <div>
        <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wider mb-3">{{ t('folders.customFolders') }}</h2>
        
        <!-- Empty State -->
        <div v-if="customFolders.length === 0" class="bg-white dark:bg-dark-surface rounded-lg border border-gray-200 dark:border-dark-border p-8 text-center">
          <FolderIcon class="w-12 h-12 text-gray-300 mx-auto mb-3" />
          <p class="text-gray-500 dark:text-gray-400">{{ t('folders.noCustomFolders') }}</p>
          <p class="text-sm text-gray-400 mt-1">{{ t('folders.noCustomFoldersDesc') }}</p>
        </div>

        <!-- Custom Folders List -->
        <div v-else class="bg-white dark:bg-dark-surface rounded-lg border border-gray-200 dark:border-dark-border overflow-hidden">
          <table class="min-w-full">
            <thead class="bg-gray-50 dark:bg-dark-bg border-b border-gray-200 dark:border-dark-border">
              <tr>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('folders.folder') }}</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('folders.imapPath') }}</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('folders.unread') }}</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('folders.total') }}</th>
                <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
              <tr 
                v-for="folder in customFolders" 
                :key="folder.id"
                class="hover:bg-gray-50 dark:hover:bg-dark-bg transition-colors"
              >
                <td class="px-6 py-4">
                  <div class="flex items-center gap-3">
                    <FolderIcon class="w-5 h-5 text-blue-500" />
                    <span class="font-medium text-gray-900 dark:text-dark-text">{{ folder.name }}</span>
                  </div>
                </td>
                <td class="px-6 py-4 text-sm text-gray-500 dark:text-gray-400 font-mono">{{ folder.imapName }}</td>
                <td class="px-6 py-4 text-sm text-gray-900 dark:text-dark-text">{{ folder.unreadCount || 0 }}</td>
                <td class="px-6 py-4 text-sm text-gray-900 dark:text-dark-text">{{ folder.totalCount || 0 }}</td>
                <td class="px-6 py-4 text-right">
                  <div class="flex items-center justify-end gap-2">
                    <button 
                      @click="openRenameDialog(folder.id, folder.name)"
                      class="p-2 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors"
                      :title="t('common.rename')"
                    >
                      <TagIcon class="w-4 h-4" />
                    </button>
                    <button 
                      @click="openDeleteDialog(folder.id, folder.name)"
                      class="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors"
                      :title="t('common.delete')"
                    >
                      <TrashIcon class="w-4 h-4" />
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useFolderStore } from '../stores/folder'
import { FolderKind } from '../utils/folder'
import { showToast } from '../utils/toast'
import { 
  InboxIcon,
  SendIcon, 
  FileTextIcon, 
  TrashIcon, 
  AlertCircleIcon, 
  ShieldAlertIcon,
  FolderIcon,
  PlusIcon,
  TagIcon
} from 'lucide-vue-next'

const { t } = useI18n()
const folderStore = useFolderStore()

// Dialog states
const showCreateDialog = ref(false)
const showRenameDialog = ref(false)
const showDeleteDialog = ref(false)

// Create folder
const newFolderName = ref('')
const createFolderInput = ref<HTMLInputElement>()

// Rename folder
const renameFolderId = ref<number>(0)
const renameFolderName = ref('')
const renameFolderInput = ref<HTMLInputElement>()

// Delete folder
const deleteFolderId = ref<number>(0)
const deleteFolderName = ref('')

// Icon mapping
const FOLDER_ICONS: Record<number, any> = {
  [FolderKind.Inbox]: InboxIcon,
  [FolderKind.Sent]: SendIcon,
  [FolderKind.Draft]: FileTextIcon,
  [FolderKind.Trash]: TrashIcon,
  [FolderKind.Spam]: AlertCircleIcon,
  [FolderKind.Quarantine]: ShieldAlertIcon,
}

// System folders
const systemFolders = computed(() => folderStore.systemFolders)

// Custom folders
const customFolders = computed(() => folderStore.customFolders)

// Get folder icon
function getFolderIcon(kind: number): any {
  return FOLDER_ICONS[kind] || FolderIcon
}

// === Create Folder Dialog ===
function openCreateDialog() {
  newFolderName.value = ''
  showCreateDialog.value = true
  nextTick(() => {
    createFolderInput.value?.focus()
  })
}

async function handleCreateFolder() {
  const name = newFolderName.value.trim()
  if (!name) {
    showToast('warning', t('folders.toastEnterName'))
    return
  }
  try {
    await folderStore.createNewFolder(name)
    showToast('success', t('folders.toastCreated'))
    showCreateDialog.value = false
  } catch (error) {
    console.error('Failed to create folder:', error)
    showToast('error', t('folders.toastCreateFailed') + ': ' + (error as Error).message)
  }
}

// === Rename Folder Dialog ===
function openRenameDialog(folderId: number, folderName: string) {
  renameFolderId.value = folderId
  renameFolderName.value = folderName
  showRenameDialog.value = true
  nextTick(() => {
    renameFolderInput.value?.focus()
  })
}

async function handleRenameFolder() {
  const name = renameFolderName.value.trim()
  if (!name) {
    showToast('warning', t('folders.toastEnterNewName'))
    return
  }
  try {
    await folderStore.renameFolderAction(renameFolderId.value, name)
    showToast('success', t('folders.toastRenamed'))
    showRenameDialog.value = false
  } catch (error) {
    console.error('Failed to rename folder:', error)
    showToast('error', t('folders.toastRenameFailed') + ': ' + (error as Error).message)
  }
}

// === Delete Folder Dialog ===
function openDeleteDialog(folderId: number, folderName: string) {
  deleteFolderId.value = folderId
  deleteFolderName.value = folderName
  showDeleteDialog.value = true
}

async function handleDeleteFolder() {
  try {
    await folderStore.deleteFolderAction(deleteFolderId.value)
    showToast('success', t('folders.toastDeleted'))
    showDeleteDialog.value = false
  } catch (error) {
    console.error('Failed to delete folder:', error)
    const errorMsg = (error as Error).message
    // Show the backend error message directly (already localized by Accept-Language header)
    showToast('error', errorMsg || t('folders.toastDeleteFailed'))
  }
}

onMounted(() => {
  folderStore.loadFolders()
})
</script>

<style scoped>
/* Page-specific styles */
</style>
