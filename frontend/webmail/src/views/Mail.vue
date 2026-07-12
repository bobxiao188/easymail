<template>
  <!-- Email list area -->
  <div class="flex flex-1 overflow-hidden min-w-0" :class="readingPaneLayoutClass">
    <MailList 
      class="flex-1 min-w-0"
      :emails="emails"
      :selected-emails="selectedEmails"
      :selected-email-id="selectedEmailId"
      :total="total"
      :current-page="currentPage"
      :total-pages="totalPages"
      :page-size="pageSize"
      :is-loading="isLoading"
      :folder-name="folderName"
      :labels="labels"
      :selected-label-id="selectedLabelId"
      @compose="handleCompose"
      @search="handleSearch"
      @prevPage="prevPage"
      @nextPage="nextPage"
      @goToPage="goToPage"
      @viewEmail="handleViewEmail"
      @toggleSelect="toggleSelect"
      @toggleSelectAll="toggleSelectAll"
      @toggleStar="toggleStar"
      @delete="handleDelete"
      @deleteSingle="handleDeleteEmail"
      @move="handleMove"
      @labelChange="handleLabelChange"
      @markRead="handleMarkRead"
      @markUnread="handleMarkUnread"
      @toggleStarBatch="handleToggleStarBatch"
      @clearSelection="clearSelection"
      @containerResize="handleContainerResize"
      @labelFilterChange="handleLabelFilterChange"
    />
    
    <!-- Reading pane - only show when email is selected -->
    <ReadingPane 
      v-if="selectedEmail"
      :email="selectedEmail" 
      :current-folder-id="currentFolderId"
      :class="readingPaneClass"
      @close="handleCloseReadingPane"
      @reply="handleReply"
      @replyAll="handleReplyAll"
      @forward="handleForward"
      @edit="handleEdit"
      @delete="handleDeleteEmail"
      @toggleStar="toggleStar"
      @prevEmail="handlePrevEmail"
      @nextEmail="handleNextEmail"
    />
  </div>
  
  <!-- Move Dialog -->
  <MoveDialog
    v-if="showMoveDialog"
    :folders="folderStore.allFolders"
    :current-folder-id="currentFolderId"
    @close="handleMoveClose"
    @confirm="handleMoveConfirm"
  />
  
  <!-- Delete Confirmation Dialog -->
  <ConfirmModal
    v-model="deleteConfirmModal.show"
    :title="deleteConfirmModal.title"
    :message="deleteConfirmModal.message"
    :confirm-text="deleteConfirmModal.confirmText"
    :loading="deleteConfirmModal.loading"
    @confirm="handleDeleteConfirm"
    @close="handleDeleteConfirmClose"
  />
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useFolderStore } from '../stores/folder'
import { useSettingStore } from '../stores/setting'

import { FolderKind, FOLDER_ROUTE_MAP } from '../utils/folder'
import { getEmailList, getEmail, batchOperation, toggleStar as toggleStarAPI } from '../api/email'
import { getLabels } from '../api/labels'
import type { EmailListItem, Email, Label } from '../types'
import { showToast } from '../utils/toast'
import MailList from '../components/mail/MailList.vue'
import ReadingPane from '../components/mail/ReadingPane.vue'
import MoveDialog from '../components/mail/MoveDialog.vue'
import ConfirmModal from '../components/ConfirmModal.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const folderStore = useFolderStore()
const settingStore = useSettingStore()

// State
const emails = ref<EmailListItem[]>([])
const selectedEmails = ref<number[]>([])
const selectedEmailId = ref<number | null>(null)
const selectedEmail = ref<Email | null>(null)
const total = ref(0)
const currentPage = ref(1)
const totalPages = ref(1)
const isLoading = ref(false)
const searchKeyword = ref('')

// Label filter state
const labels = ref<Label[]>([])
const selectedLabelId = ref<number | null>(null)

// Page size - can be auto-calculated or from settings
const autoPageSize = ref(20) // Default fallback

const pageSize = computed(() => {
  // If auto mode is enabled, use calculated size
  if (settingStore.settings.pageSize === 0) {
    return autoPageSize.value
  }
  return settingStore.settings.pageSize || 20
})

// Handle container resize event from MailList component
function handleContainerResize(height: number) {
  // Only recalculate if auto mode is enabled
  if (settingStore.settings.pageSize !== 0) {
    return
  }
  
  // Row height derived from actual CSS classes in MailList.vue email rows:
  //   Outer div: py-3 (12px × 2 = 24px padding), border-b (1px)
  //   Content column (flex-col, gap-0.5 = 2px):
  //     Line 1 (sender): avatar h-6 (24px), text-sm (14px/20px line) → driven to 24px
  //     gap: 2px
  //     Line 2 (subject): text-sm (14px/20px line) → 20px
  //     gap: 2px
  //     Line 3 (snippet, conditional): text-xs (12px/16px line) + mt-0.5 (2px) → 18px
  //   Content total: 24 + 2 + 20 + 2 + 18 = 66px (with snippet)
  //                 24 + 2 + 20         = 46px (without)
  //   Row total: padding(24) + content(~63 avg) + border(1) ≈ 88px
  const rowHeight = 88
  
  // Overhead: fixed-height elements outside scrollable email row area
  //   Toolbar: py-2(16) + content(~28) + border(1) ≈ 45px
  //   Select-all header: py-2(16) + content(~20) + border(1) ≈ 37px
  //   Pagination footer: py-2(16) + content(~28) + border(1) ≈ 45px
  //   Base subtotal: 45 + 37 + 45 = 127px
  //   Batch bar adds ~41px, so max = 168px; using 140 as balanced midpoint
  const overheadHeight = 140
  
  // Available height for email rows
  const availableHeight = height - overheadHeight
  
  // Calculate number of rows that fit
  const calculatedSize = Math.max(5, Math.floor(availableHeight / rowHeight))
  
  // Only reload if page size actually changed
  if (calculatedSize !== autoPageSize.value) {
    autoPageSize.value = calculatedSize
    currentPage.value = 1
    loadEmails()
  }
}

// Reading pane layout based on settings
const readingPanePosition = computed(() => settingStore.settings.readingPanePosition || 'right')

// Layout class based on reading pane position
const readingPaneLayoutClass = computed(() => {
  if (readingPanePosition.value === 'bottom' && selectedEmail.value) {
    return 'flex-col'
  }
  return 'flex-row'
})

// Reading pane class based on position
const readingPaneClass = computed(() => {
  if (readingPanePosition.value === 'bottom') {
    return 'h-1/2 shrink-0 border-t border-gray-200'
  }
  return 'w-1/2 shrink-0 border-l border-gray-200'
})

// Move dialog state
const showMoveDialog = ref(false)

// Delete confirmation modal state
const deleteConfirmModal = ref({
  show: false,
  title: '',
  message: '',
  confirmText: '',
  loading: false,
  callback: null as (() => Promise<void>) | null
})

// Folder types that require permanent deletion (trash, junk/spam, quarantine)
const PERMANENT_DELETE_FOLDERS: number[] = [FolderKind.Trash, FolderKind.Spam, FolderKind.Quarantine]

// Check if current folder requires permanent deletion
const isPermanentDeleteFolder = computed(() => {
  if (!currentFolderId.value) return false
  const folder = folderStore.getFolderById(currentFolderId.value)
  if (!folder) return false
  return PERMANENT_DELETE_FOLDERS.includes(folder.kind as FolderKind)
})

// Get current folder ID from route
const currentFolderId = computed(() => {
  // Custom folders
  if (route.name === 'CustomFolder' && route.params.folderId) {
    return Number(route.params.folderId)
  }
  
  // System folders
  const folderMeta = route.meta.folder as string | undefined
  if (folderMeta) {
    // Reverse lookup kind from route slug
    const kind = Object.entries(FOLDER_ROUTE_MAP).find(([_, slug]) => slug === folderMeta)?.[0]
    if (kind) {
      const folder = folderStore.getFolderByKind(Number(kind))
      return folder?.id ?? null
    }
  }
  
  return null
})

// Folder name
const folderName = computed(() => {
  if (currentFolderId.value) {
    const folder = folderStore.getFolderById(currentFolderId.value)
    return folder?.name || ''
  }
  return route.meta.title as string || ''
})

// Load email list
async function loadEmails() {
  // Wait for folderStore to load
  if (folderStore.allFolders.length === 0) {
    await folderStore.loadFolders()
  }
  // Wait for settingStore to load (needed for pageSize auto-calculation)
  if (settingStore.settings.pageSize === 20 && settingStore.userEmail === '') {
    await settingStore.loadAll()
  }
  
  if (!currentFolderId.value) {
    emails.value = []
    total.value = 0
    totalPages.value = 1
    return
  }

  isLoading.value = true
  try {
    const response = await getEmailList({
      page: currentPage.value,
      pageSize: pageSize.value,
      folderId: currentFolderId.value,
      keyword: searchKeyword.value || undefined,
      labelId: selectedLabelId.value || undefined
    })
    
    if (response.code === 0 && response.data) {
      emails.value = response.data.items || []
      total.value = response.data.total || 0
      totalPages.value = Math.ceil(total.value / pageSize.value)
      
      // If no selected email and there are emails, automatically open the first email
      if (!selectedEmailId.value && emails.value.length > 0) {
        const firstEmail = emails.value[0]
        selectedEmailId.value = firstEmail.id
        await loadEmailDetail(firstEmail.id)
      }
    }
  } catch (error) {
    console.error('Failed to load email list:', error)
    showToast('error', t('mail.failedToLoadEmails'))
  } finally {
    isLoading.value = false
  }
}

// Load single email detail
async function loadEmailDetail(id: number) {
  try {
    const response = await getEmail(id)
    if (response.code === 0 && response.data) {
      const data = response.data as any
      selectedEmail.value = {
        ...data,
        isStarred: data.flagged,
      } as Email
      
      // Sync the latest status with the email list
      const emailInList = emails.value.find(m => m.id === id)
      if (emailInList) {
        // Update read status
        if (!emailInList.isRead) {
          // Backend auto-marks as read, update local state
          emailInList.isRead = true
          
          // Update folder unread count (decrement by 1)
          const currentFolder = folderStore.getFolderById(currentFolderId.value!)
          if (currentFolder && currentFolder.unreadCount !== undefined && currentFolder.unreadCount > 0) {
            folderStore.updateFolderUnreadCount(currentFolder.id, currentFolder.unreadCount - 1)
          }
        }
        // Sync star status from detail to list
        emailInList.isStarred = selectedEmail.value.isStarred
      }
    }
  } catch (error) {
    console.error('Failed to load email detail:', error)
    showToast('error', t('mail.failedToLoadEmailDetail'))
  }
}

// Methods
function handleCompose() {
  router.push('/compose')
}

function handleSearch(keyword: string) {
  searchKeyword.value = keyword
  currentPage.value = 1
  loadEmails()
}

// Load labels
async function loadLabels() {
  try {
    const response = await getLabels()
    if (response.code === 0 && response.data) {
      labels.value = response.data as Label[]
    }
  } catch (error) {
    console.error('Failed to load labels:', error)
  }
}

// Handle label filter change
function handleLabelFilterChange(labelId: number | null) {
  selectedLabelId.value = labelId
  currentPage.value = 1
  loadEmails()
}

function prevPage() {
  if (currentPage.value > 1) {
    currentPage.value--
    loadEmails()
  }
}

function nextPage() {
  if (currentPage.value < totalPages.value) {
    currentPage.value++
    loadEmails()
  }
}

function goToPage(page: number) {
  currentPage.value = page
  loadEmails()
}

function handleViewEmail(id: number) {
  selectedEmailId.value = id
  loadEmailDetail(id)
}

function toggleSelect(id: number) {
  const index = selectedEmails.value.indexOf(id)
  if (index > -1) {
    selectedEmails.value.splice(index, 1)
  } else {
    selectedEmails.value.push(id)
  }
}

function toggleSelectAll() {
  if (selectedEmails.value.length === emails.value.length) {
    selectedEmails.value = []
  } else {
    selectedEmails.value = emails.value.map(m => m.id)
  }
}

async function toggleStar(id: number) {
  try {
    const response = await toggleStarAPI(id)
    const newStarred = response.data.starred
    // Update local state - email list
    const email = emails.value.find(m => m.id === id)
    if (email) {
      email.isStarred = newStarred
    }
    // Sync update email displayed in reading pane
    if (selectedEmail.value && selectedEmail.value.id === id) {
      selectedEmail.value.isStarred = newStarred
    }
  } catch (error) {
    console.error('Failed to toggle star:', error)
    showToast('error', t('mail.failedToToggleStar'))
  }
}

async function handleDelete() {
  if (selectedEmails.value.length === 0) {
    showToast('warning', t('mail.noEmailsSelected'))
    return
  }
  
  // If current folder is trash/junk/quarantine, need to confirm permanent deletion
  if (isPermanentDeleteFolder.value) {
    const count = selectedEmails.value.length
    deleteConfirmModal.value = {
      show: true,
      title: t('mail.deletePermanently'),
      message: t('mail.permanentDeleteConfirm', { count }),
      confirmText: t('mail.deletePermanently'),
      loading: false,
      callback: async () => {
        await executeBatchDelete()
      }
    }
    return
  }
  
  // Normal folders, directly delete (move to trash)
  await executeBatchDelete()
}

// Execute the actual logic of batch deletion
async function executeBatchDelete() {
  try {
    // If the currently viewed email is deleted, close the reading pane
    const deletedIds = new Set(selectedEmails.value)
    if (selectedEmailId.value && deletedIds.has(selectedEmailId.value)) {
      handleCloseReadingPane()
    }
    
    // Choose deletion method based on folder type: permanent delete vs soft delete
    const action = isPermanentDeleteFolder.value ? 'permanent_delete' : 'delete'
    await batchOperation(selectedEmails.value, action)
    
    // Display different success messages based on folder type
    if (isPermanentDeleteFolder.value) {
      showToast('success', t('mail.emailsPermanentlyDeleted'))
    } else {
      showToast('success', t('mail.emailsDeletedSuccessfully'))
    }
    
    selectedEmails.value = []
    // Remove from local list
    emails.value = emails.value.filter(e => !deletedIds.has(e.id))
    total.value -= deletedIds.size
    // Update totalPages
    totalPages.value = Math.ceil(total.value / pageSize.value)
    
    // Reload folder list to update unread count
    folderStore.loadFolders()
  } catch (error) {
    console.error('Failed to delete emails:', error)
    showToast('error', t('mail.failedToDeleteEmails'))
    // Reload on failure
    loadEmails()
  } finally {
    deleteConfirmModal.value.loading = false
  }
}

function handleMove() {
  if (selectedEmails.value.length === 0) {
    showToast('warning', t('mail.noEmailsSelected'))
    return
  }
  // Ensure folder list is loaded
  if (folderStore.allFolders.length === 0) {
    folderStore.loadFolders().catch(() => {
      showToast('error', t('mail.failedToLoadFolders'))
    })
  }
  showMoveDialog.value = true
}

async function handleMoveConfirm(folderId: number) {
  showMoveDialog.value = false
  
  try {
    await batchOperation(selectedEmails.value, 'move', folderId)
    showToast('success', t('mail.emailsMovedSuccessfully'))
    
    // If the currently viewed email was moved, close the reading pane
    const movedIds = new Set(selectedEmails.value)
    if (selectedEmailId.value && movedIds.has(selectedEmailId.value)) {
      handleCloseReadingPane()
    }
    
    selectedEmails.value = []
    // Remove from local list
    emails.value = emails.value.filter(e => !movedIds.has(e.id))
    total.value -= movedIds.size
    // Update totalPages
    totalPages.value = Math.ceil(total.value / pageSize.value)
    
    // Reload folder list to update unread count
    folderStore.loadFolders()
  } catch (error) {
    console.error('Failed to move emails:', error)
    showToast('error', t('mail.failedToMoveEmails'))
  }
}

function handleMoveClose() {
  showMoveDialog.value = false
}

// Delete confirmation modal handlers
async function handleDeleteConfirm() {
  if (deleteConfirmModal.value.callback) {
    deleteConfirmModal.value.loading = true
    await deleteConfirmModal.value.callback()
  }
}

function handleDeleteConfirmClose() {
  deleteConfirmModal.value.show = false
  deleteConfirmModal.value.title = ''
  deleteConfirmModal.value.message = ''
  deleteConfirmModal.value.callback = null
  deleteConfirmModal.value.loading = false
}

async function handleMarkRead() {
  if (selectedEmails.value.length === 0) {
    showToast('warning', t('mail.noEmailsSelected'))
    return
  }
  
  try {
    // Update local status
    const ids = [...selectedEmails.value]
    emails.value.forEach(email => {
      if (ids.includes(email.id)) {
        email.isRead = true
      }
    })
    // If selected email is in reading pane, also update
    if (selectedEmailId.value && ids.includes(selectedEmailId.value) && selectedEmail.value) {
      selectedEmail.value = { ...selectedEmail.value, isRead: true }
    }
    
    await batchOperation(selectedEmails.value, 'mark_read')
    showToast('success', t('mail.emailsMarkedAsRead'))
    selectedEmails.value = []
    
    // Reload folder list to update unread count
    folderStore.loadFolders()
  } catch (error) {
    console.error('Failed to mark as read:', error)
    showToast('error', t('mail.failedToMarkAsRead'))
    // Reload on failure
    loadEmails()
  }
}

async function handleMarkUnread() {
  if (selectedEmails.value.length === 0) {
    showToast('warning', t('mail.noEmailsSelected'))
    return
  }
  
  try {
    // Update local status
    const ids = [...selectedEmails.value]
    emails.value.forEach(email => {
      if (ids.includes(email.id)) {
        email.isRead = false
      }
    })
    // If selected email is in reading pane, also update
    if (selectedEmailId.value && ids.includes(selectedEmailId.value) && selectedEmail.value) {
      selectedEmail.value = { ...selectedEmail.value, isRead: false }
    }
    
    await batchOperation(selectedEmails.value, 'mark_unread')
    showToast('success', t('mail.emailsMarkedAsUnread'))
    selectedEmails.value = []
    
    // Reload folder list to update unread count
    folderStore.loadFolders()
  } catch (error) {
    console.error('Failed to mark as unread:', error)
    showToast('error', t('mail.failedToMarkAsUnread'))
    // Reload on failure
    loadEmails()
  }
}

async function handleToggleStarBatch() {
  if (selectedEmails.value.length === 0) {
    showToast('warning', t('mail.noEmailsSelected'))
    return
  }
  
  try {
    // Update local status
    const ids = [...selectedEmails.value]
    emails.value.forEach(email => {
      if (ids.includes(email.id)) {
        email.isStarred = !email.isStarred
      }
    })
    // If selected email is in reading pane, also update
    if (selectedEmailId.value && ids.includes(selectedEmailId.value) && selectedEmail.value) {
      selectedEmail.value.isStarred = !selectedEmail.value.isStarred
    }
    
    await batchOperation(selectedEmails.value, 'toggle_star')
    showToast('success', t('mail.starStatusToggled'))
    selectedEmails.value = []
  } catch (error) {
    console.error('Failed to batch toggle star:', error)
    showToast('error', t('mail.failedToToggleStar'))
    // Reload on failure
    loadEmails()
  }
}

function clearSelection() {
  selectedEmails.value = []
}

// Reading Pane handlers
function handleCloseReadingPane() {
  selectedEmailId.value = null
  selectedEmail.value = null
}

function handleReply(email: Email) {
  // Navigate to compose with reply mode — Compose.vue fetches original email to fill in
  router.push({
    path: '/compose',
    query: {
      mode: 'reply',
      originalId: String(email.id),
    }
  })
}

function handleReplyAll(email: Email) {
  // Navigate to compose with replyAll mode — Compose.vue fetches original email to fill in
  router.push({
    path: '/compose',
    query: {
      mode: 'replyAll',
      originalId: String(email.id),
    }
  })
}

function handleForward(email: Email) {
  // Navigate to compose with forward mode — Compose.vue fetches original email to fill in
  router.push({
    path: '/compose',
    query: {
      mode: 'forward',
      originalId: String(email.id),
    }
  })
}

function handleEdit(email: Email) {
  // Navigate to compose with edit mode
  router.push({
    path: '/compose',
    query: {
      mode: 'edit',
      id: String(email.id)
    }
  })
}

async function handleDeleteEmail(id: number) {
  // If current folder is trash/junk/quarantine, need to confirm physical deletion
  if (isPermanentDeleteFolder.value) {
    deleteConfirmModal.value = {
      show: true,
      title: t('mail.deletePermanently'),
      message: t('mail.permanentDeleteSingle'),
      confirmText: t('mail.deletePermanently'),
      loading: false,
      callback: async () => {
        await executeSingleDelete(id)
      }
    }
    return
  }
  
  // Normal folders, directly delete (move to trash)
  await executeSingleDelete(id)
}

// Execute the actual logic of single email deletion
async function executeSingleDelete(id: number) {
  try {
    // Choose deletion method based on folder type: permanent delete vs soft delete
    const action = isPermanentDeleteFolder.value ? 'permanent_delete' : 'delete'
    await batchOperation([id], action)
    
    // Display different success messages based on folder type
    if (isPermanentDeleteFolder.value) {
      showToast('success', t('mail.emailPermanentlyDeleted'))
    } else {
      showToast('success', t('mail.emailDeletedSuccessfully'))
    }
    
    // Remove from local list
    const index = emails.value.findIndex(m => m.id === id)
    if (index > -1) {
      emails.value.splice(index, 1)
      total.value--
    }
    // If the deleted email was selected, close reading pane
    if (selectedEmailId.value === id) {
      handleCloseReadingPane()
    }
    // Reload folder list to update unread count
    folderStore.loadFolders()
  } catch (error) {
    console.error('Failed to delete email:', error)
    showToast('error', t('mail.failedToDeleteEmail'))
  } finally {
    deleteConfirmModal.value.loading = false
  }
}

// Handle label changes
function handleLabelChange(data: { emailId: number | null, labelId: number | null }) {
  if (!data.emailId) return
  
  // Get label info from settingStore
  const label = settingStore.labels.find(l => l.id === data.labelId)
  
  // Update labels in local email list
  const email = emails.value.find(m => m.id === data.emailId)
  if (email) {
    if (data.labelId === null) {
      // Remove label
      email.labels = []
      showToast('success', t('mail.labelRemoved'))
    } else if (label) {
      // Set label
      email.labels = [{ id: label.id, name: label.name, color: label.color }]
      showToast('success', t('mail.labelSet', { labelName: label.name }))
    }
  }
  
  // If selected email label is changed, also update reading pane
  if (selectedEmailId.value === data.emailId && selectedEmail.value) {
    // Create new Email object to ensure reactive update
    const updatedEmail = { ...selectedEmail.value }
    if (data.labelId === null) {
      updatedEmail.labels = []
    } else if (label) {
      updatedEmail.labels = [{ id: label.id, name: label.name, color: label.color }]
    }
    selectedEmail.value = updatedEmail
  }
}

function handlePrevEmail() {
  if (!selectedEmailId.value) return
  
  const currentIndex = emails.value.findIndex(m => m.id === selectedEmailId.value)
  if (currentIndex > 0) {
    // There is a previous email
    const prevEmail = emails.value[currentIndex - 1]
    selectedEmailId.value = prevEmail.id
    loadEmailDetail(prevEmail.id)
  }
}

function handleNextEmail() {
  if (!selectedEmailId.value) return
  
  const currentIndex = emails.value.findIndex(m => m.id === selectedEmailId.value)
  if (currentIndex !== -1 && currentIndex < emails.value.length - 1) {
    // There is a next email
    const nextEmail = emails.value[currentIndex + 1]
    selectedEmailId.value = nextEmail.id
    loadEmailDetail(nextEmail.id)
  }
}

// Watch route changes, reload emails
watch(
  () => route.fullPath,
  () => {
    currentPage.value = 1
    searchKeyword.value = ''
    selectedLabelId.value = null
    selectedEmailId.value = null
    selectedEmail.value = null
    selectedEmails.value = []
    loadEmails()
  }
)

onMounted(async () => {
  // Load settings first (for pageSize auto-calculation)
  if (settingStore.settings.pageSize === 20 && settingStore.userEmail === '') {
    await settingStore.loadAll()
  }
  // Load labels in parallel with emails
  loadLabels()
  loadEmails()
})
</script>
