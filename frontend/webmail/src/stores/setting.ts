import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  getProfile,
  changePassword as apiChangePassword,
  getUserSettings as fetchUserSettings,
  updateUserSettings as apiUpdateUserSettings
} from '../api/auth'
import { getLabels, createLabel, updateLabel, deleteLabel as apiDeleteLabel } from '../api/labels'
import { getMailStats } from '../api/email'
import type { UserSettings, Label, MailStats } from '../types'
import { setLocale } from '../i18n'
import { applyTheme } from './../utils/theme'

const defaultSettings: UserSettings = {
  displayName: '',
  signature: '',
  language: 'en',
  theme: 'light',
  pageSize: 20,
  readingPanePosition: 'right',
  autoReplyEnabled: false,
  autoReplySubject: '',
  autoReplyBody: '',
  phone: '',
  company: '',
  jobTitle: '',
  notificationSound: true,
  desktopNotification: true,
  includeOriginalOnReply: true,
  forwardingEnabled: false,
  forwardingAddress: '',
  saveSent: true
}

export const useSettingStore = defineStore('setting', () => {
  // === User Settings ===
  const settings = ref<UserSettings>({ ...defaultSettings })
  const userEmail = ref('')

  // === Mail Statistics ===
  const stats = ref<MailStats>({
    inboxCount: 0,
    unreadCount: 0,
    sentCount: 0,
    draftCount: 0,
    trashCount: 0,
    spamCount: 0,
    storageUsed: 0,
    storageLimit: 1073741824
  })

  // === Label Management ===
  const labels = ref<Label[]>([])

  // === Loading Status ===
  const isLoading = ref(false)
  const isSaving = ref(false)
  const isChangingPassword = ref(false)
  const isAddingLabel = ref(false)
  const isDeletingLabel = ref<number | null>(null)

  // === Label Editing ===
  const editingLabelId = ref<number | null>(null)
  const editLabelColor = ref('#3b82f6')
  const newLabelName = ref('')
  const newLabelColor = ref('#3b82f6')

  // === Password Change ===
  const passwordForm = ref({ oldPassword: '', newPassword: '', confirmPassword: '' })
  const passwordErrors = ref({ newPassword: '', confirmPassword: '' })

  // === Display Name Editing ===
  const isEditingName = ref(false)
  const tempDisplayName = ref('')

  // === Computed Properties ===
  const storagePercent = computed(() => {
    if (stats.value.storageLimit <= 0) return 0
    return Math.round((stats.value.storageUsed / stats.value.storageLimit) * 100)
  })

  // === Load Data ===

  async function loadAll() {
    await Promise.all([
      loadProfile(),
      loadSettings(),
      loadStats(),
      loadLabels()
    ])
  }

  async function loadProfile() {
    try {
      const profileRes = await getProfile()
      if (profileRes.data) {
        settings.value.displayName = profileRes.data.name || ''
        settings.value.phone = profileRes.data.phone || ''
        settings.value.company = profileRes.data.company || ''
        settings.value.jobTitle = profileRes.data.jobTitle || ''
        userEmail.value = profileRes.data.email || ''
      }
    } catch (error) {
      console.error('Failed to load profile:', error)
      throw error
    }
  }

  async function loadSettings() {
    try {
      const settingsRes = await fetchUserSettings()
      if (settingsRes.data) {
        settings.value = { ...settings.value, ...settingsRes.data }
        // Sync localStorage with server settings
        localStorage.setItem('saveSent', settings.value.saveSent.toString())
        // Apply language setting
        applyLanguage(settings.value.language)
        // Apply theme setting
        applyTheme(settings.value.theme)
      }
    } catch (error) {
      console.error('Failed to load settings:', error)
      // Fallback to localStorage if API fails
      const savedSaveSent = localStorage.getItem('saveSent')
      if (savedSaveSent !== null) {
        settings.value.saveSent = savedSaveSent === 'true'
      }
      throw error
    }
  }

  function applyLanguage(lang: string) {
    // Map backend language code to frontend locale
    const localeMap: Record<string, string> = {
      'zh-CN': 'zh',
      'zh': 'zh',
      'en': 'en',
      'en-US': 'en'
    }
    const locale = localeMap[lang] || 'en'
    setLocale(locale)
  }

  async function loadStats() {
    try {
      const response = await getMailStats()
      stats.value = response.data
    } catch (error) {
      console.error('Failed to load statistics:', error)
      throw error
    }
  }

  async function loadLabels() {
    try {
      const res = await getLabels()
      labels.value = res.data
    } catch (e) {
      console.error('Failed to load labels:', e)
      throw e
    }
  }

  // === Settings Validation ===

  interface ValidationErrors {
    forwardingAddress: string
    autoReplySubject: string
    autoReplyBody: string
  }

  function validateSettings(): { valid: boolean; errors: ValidationErrors } {
    const errors: ValidationErrors = {
      forwardingAddress: '',
      autoReplySubject: '',
      autoReplyBody: ''
    }

    // Auto-forward: must have valid email address when enabled
    if (settings.value.forwardingEnabled) {
      const addr = settings.value.forwardingAddress.trim()
      if (!addr) {
        errors.forwardingAddress = 'Please enter a forwarding email address'
      } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(addr)) {
        errors.forwardingAddress = 'Please enter a valid email address'
      }
    }

    // Auto-reply: must have subject and body when enabled
    if (settings.value.autoReplyEnabled) {
      if (!settings.value.autoReplySubject.trim()) {
        errors.autoReplySubject = 'Please enter an auto-reply subject'
      }
      if (!settings.value.autoReplyBody.trim()) {
        errors.autoReplyBody = 'Please enter an auto-reply message'
      }
    }

    return {
      valid: !errors.forwardingAddress && !errors.autoReplySubject && !errors.autoReplyBody,
      errors
    }
  }

  // === Save Settings ===

  async function saveSettings() {
    isSaving.value = true
    try {
      const res = await apiUpdateUserSettings({
        displayName: settings.value.displayName,
        signature: settings.value.signature,
        language: settings.value.language,
        theme: settings.value.theme,
        pageSize: settings.value.pageSize,
        readingPanePosition: settings.value.readingPanePosition,
        autoReplyEnabled: settings.value.autoReplyEnabled,
        autoReplySubject: settings.value.autoReplySubject,
        autoReplyBody: settings.value.autoReplyBody,
        phone: settings.value.phone,
        company: settings.value.company,
        jobTitle: settings.value.jobTitle,
        notificationSound: settings.value.notificationSound,
        desktopNotification: settings.value.desktopNotification,
        includeOriginalOnReply: settings.value.includeOriginalOnReply,
        forwardingEnabled: settings.value.forwardingEnabled,
        forwardingAddress: settings.value.forwardingAddress,
        saveSent: settings.value.saveSent
      })
      if (res.data) {
        settings.value = res.data
      }
      localStorage.setItem('saveSent', settings.value.saveSent.toString())
      // Apply language setting immediately
      applyLanguage(settings.value.language)
      // Apply theme setting immediately
      applyTheme(settings.value.theme)
      return res
    } catch (error) {
      console.error('Failed to save settings:', error)
      throw error
    } finally {
      isSaving.value = false
    }
  }

  // === Change Password ===

  function validatePassword(): boolean {
    passwordErrors.value = { newPassword: '', confirmPassword: '' }
    if (!passwordForm.value.oldPassword) {
      return false // handled by caller
    }
    if (!passwordForm.value.newPassword) {
      passwordErrors.value.newPassword = 'Please enter new password'
      return false
    }
    if (passwordForm.value.newPassword.length < 8) {
      passwordErrors.value.newPassword = 'Password must be at least 8 characters'
      return false
    }
    if (passwordForm.value.newPassword !== passwordForm.value.confirmPassword) {
      passwordErrors.value.confirmPassword = 'Passwords do not match'
      return false
    }
    return true
  }

  async function changePassword() {
    isChangingPassword.value = true
    try {
      await apiChangePassword({
        oldPassword: passwordForm.value.oldPassword,
        newPassword: passwordForm.value.newPassword
      })
      passwordForm.value = { oldPassword: '', newPassword: '', confirmPassword: '' }
      passwordErrors.value = { newPassword: '', confirmPassword: '' }
    } catch (error) {
      console.error('Failed to change password:', error)
      throw error
    } finally {
      isChangingPassword.value = false
    }
  }

  // === Display Name Editing ===

  function startEditName() {
    isEditingName.value = true
    tempDisplayName.value = settings.value.displayName
  }

  function saveName() {
    isEditingName.value = false
    if (tempDisplayName.value.trim()) {
      settings.value.displayName = tempDisplayName.value.trim()
    }
  }

  function cancelEditName() {
    isEditingName.value = false
    tempDisplayName.value = settings.value.displayName
  }

  // === Label Management ===

  async function addLabel() {
    if (!newLabelName.value.trim()) return
    isAddingLabel.value = true
    try {
      await createLabel({ name: newLabelName.value.trim(), color: newLabelColor.value })
      await loadLabels()
      newLabelName.value = ''
      newLabelColor.value = '#3b82f6'
    } catch (e) {
      console.error('Failed to create label:', e)
      throw e
    } finally {
      isAddingLabel.value = false
    }
  }

  function startEditLabel(lbl: Label) {
    editingLabelId.value = lbl.id
    editLabelColor.value = lbl.color
  }

  function cancelEditLabel() {
    editingLabelId.value = null
  }

  async function saveEditLabel(id: number) {
    try {
      await updateLabel(id, { name: '', color: editLabelColor.value })
      await loadLabels()
      editingLabelId.value = null
    } catch (e) {
      console.error('Failed to update label:', e)
      throw e
    }
  }

  async function removeLabel(id: number) {
    isDeletingLabel.value = id
    try {
      await apiDeleteLabel(id)
      await loadLabels()
    } catch (e) {
      console.error('Failed to delete label:', e)
      throw e
    } finally {
      isDeletingLabel.value = null
    }
  }

  // === Utility Methods ===

  function formatSize(bytes: number): string {
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
    if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
    return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
  }

  // === Reset ===
  function clear() {
    settings.value = { ...defaultSettings }
    userEmail.value = ''
    labels.value = []
    passwordForm.value = { oldPassword: '', newPassword: '', confirmPassword: '' }
  }

  return {
    // Settings status
    settings,
    userEmail,
    isLoading,
    isSaving,
    isChangingPassword,
    // Statistics
    stats,
    storagePercent,
    // Labels
    labels,
    isAddingLabel,
    isDeletingLabel,
    editingLabelId,
    editLabelColor,
    newLabelName,
    newLabelColor,
    // Password
    passwordForm,
    passwordErrors,
    // Name editing
    isEditingName,
    tempDisplayName,
    // Loading
    loadAll,
    loadProfile,
    loadSettings,
    loadStats,
    loadLabels,
    // Save
    saveSettings,
    validateSettings,
    // Password
    validatePassword,
    changePassword,
    // Name
    startEditName,
    saveName,
    cancelEditName,
    // Labels
    addLabel,
    startEditLabel,
    cancelEditLabel,
    saveEditLabel,
    removeLabel,
    // Utilities
    formatSize,
    clear
  }
})
