import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { 
  listFolders, 
  createFolder, 
  renameFolder, 
  deleteFolder 
} from '../api/folder'
import type { Folder } from '../types'
import { FolderKind } from '../utils/folder'

export const useFolderStore = defineStore('folder', () => {
  // State
  const allFolders = ref<Folder[]>([])
  const isLoading = ref(false)
  const activeFolderId = ref<number | null>(null)
  
  // Computed properties
  // Primary folders (inbox, sent, drafts)
  const primaryFolders = computed(() => {
    return allFolders.value
      .filter(f => f.kind === FolderKind.Inbox || f.kind === FolderKind.Sent || f.kind === FolderKind.Draft)
  })
  
  // Other system folders (trash, spam, quarantine, etc.)
  const secondaryFolders = computed(() => {
    return allFolders.value
      .filter(f => f.kind >= 1 && f.kind < FolderKind.UserCustomMin && 
                   !(f.kind === FolderKind.Inbox || f.kind === FolderKind.Sent || f.kind === FolderKind.Draft))
  })
  
  // Custom folders
  const customFolders = computed(() => {
    return allFolders.value
      .filter(f => f.kind >= FolderKind.UserCustomMin)
  })
  
  // System folders
  const systemFolders = computed(() => {
    return allFolders.value
      .filter(f => f.kind >= 1 && f.kind < FolderKind.UserCustomMin)
  })
  
  // Total unread count
  const totalUnreadCount = computed(() => {
    return allFolders.value.reduce((sum, f) => sum + (f.unreadCount || 0), 0)
  })
  
  // Inbox unread count
  const inboxUnreadCount = computed(() => {
    const inbox = allFolders.value.find(f => f.kind === FolderKind.Inbox)
    return inbox?.unreadCount || 0
  })
  
  // Get folder by kind
  function getFolderByKind(kind: number): Folder | undefined {
    return allFolders.value.find(f => f.kind === kind)
  }
  
  // Get folder by id
  function getFolderById(id: number): Folder | undefined {
    return allFolders.value.find(f => f.id === id)
  }
  
  // Set current active folder
  function setActiveFolder(folderId: number | null) {
    activeFolderId.value = folderId
  }
  
  // Load folder list
  async function loadFolders() {
    isLoading.value = true
    try {
      const response = await listFolders()
      if (response.data) {
        allFolders.value = response.data
      }
      return response.data
    } catch (error) {
      console.error('Failed to load folders:', error)
      throw error
    } finally {
      isLoading.value = false
    }
  }
  
  // Create folder
  async function createNewFolder(name: string) {
    try {
      const response = await createFolder(name)
      if (response.data) {
        allFolders.value.push(response.data)
      }
      return response
    } catch (error) {
      console.error('Failed to create folder:', error)
      throw error
    }
  }
  
  // Rename folder
  async function renameFolderAction(id: number, name: string) {
    try {
      const response = await renameFolder(id, name)
      // Update local state
      const folder = allFolders.value.find(f => f.id === id)
      if (folder) {
        folder.name = name
      }
      return response
    } catch (error) {
      console.error('Failed to rename folder:', error)
      throw error
    }
  }
  
  // Delete folder
  async function deleteFolderAction(id: number) {
    try {
      const response = await deleteFolder(id)
      // Remove from local state
      allFolders.value = allFolders.value.filter(f => f.id !== id)
      if (activeFolderId.value === id) {
        setActiveFolder(null)
      }
      return response
    } catch (error) {
      console.error('Failed to delete folder:', error)
      throw error
    }
  }
  
  // Update folder unread count (for real-time updates)
  function updateFolderUnreadCount(id: number, unreadCount: number) {
    const folder = allFolders.value.find(f => f.id === id)
    if (folder) {
      folder.unreadCount = unreadCount
    }
  }
  
  // Clear
  function clear() {
    allFolders.value = []
    activeFolderId.value = null
  }
  
  return {
    allFolders,
    isLoading,
    activeFolderId,
    primaryFolders,
    secondaryFolders,
    customFolders,
    systemFolders,
    totalUnreadCount,
    inboxUnreadCount,
    getFolderByKind,
    getFolderById,
    setActiveFolder,
    loadFolders,
    createNewFolder,
    renameFolderAction,
    deleteFolderAction,
    updateFolderUnreadCount,
    clear
  }
})
