<template>
  <div class="flex-1 w-full overflow-y-auto bg-gray-50 dark:bg-dark-bg">
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

    <div class="px-6 py-8">
      <!-- Page title -->
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-dark-text">{{ t('settings.pageTitle') }}</h1>
          <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">{{ t('settings.pageDesc') }}</p>
        </div>
        <button @click="handleSaveAll" class="btn-primary-sm" :disabled="settingStore.isSaving">
          <svg v-if="settingStore.isSaving" class="animate-spin inline h-4 w-4 mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
          {{ t('settings.saveAllSettings') }}
        </button>
      </div>

      <!-- Storage & Stats -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        <!-- Storage -->
        <div class="bg-white dark:bg-dark-surface rounded-lg border border-gray-200 dark:border-dark-border p-6">
          <h2 class="text-base font-semibold text-gray-900 dark:text-dark-text mb-4">{{ t('settings.storage') }}</h2>
          <div class="flex justify-between text-sm text-gray-600 dark:text-gray-400 mb-2">
            <span>{{ t('settings.used') }}</span>
            <span>{{ settingStore.formatSize(settingStore.stats.storageUsed) }} / {{ settingStore.formatSize(settingStore.stats.storageLimit) }}</span>
          </div>
          <div class="w-full bg-gray-100 dark:bg-gray-700 rounded-full h-2.5">
            <div class="h-2.5 rounded-full bg-gray-900 dark:bg-blue-500 transition-all"
              :style="{ width: settingStore.storagePercent + '%' }">
            </div>
          </div>
          <p class="text-xs text-gray-400 dark:text-gray-500 mt-2">
            {{ settingStore.storagePercent }}% {{ t('settings.storageUsed') }}
          </p>
        </div>

        <!-- Account Stats -->
        <div class="bg-white dark:bg-dark-surface rounded-lg border border-gray-200 dark:border-dark-border p-6">
          <h2 class="text-base font-semibold text-gray-900 dark:text-dark-text mb-4">{{ t('settings.accountStats') }}</h2>
          <div class="grid grid-cols-4 gap-4">
            <div class="text-center">
              <p class="text-2xl font-semibold text-gray-900 dark:text-dark-text">{{ settingStore.stats.inboxCount }}</p>
              <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">{{ t('settings.inbox') }}</p>
            </div>
            <div class="text-center">
              <p class="text-2xl font-semibold text-red-500">{{ settingStore.stats.unreadCount }}</p>
              <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">{{ t('settings.unread') }}</p>
            </div>
            <div class="text-center">
              <p class="text-2xl font-semibold text-gray-900 dark:text-dark-text">{{ settingStore.stats.sentCount }}</p>
              <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">{{ t('settings.sent') }}</p>
            </div>
            <div class="text-center">
              <p class="text-2xl font-semibold text-gray-900 dark:text-dark-text">{{ settingStore.stats.draftCount }}</p>
              <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">{{ t('settings.drafts') }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Profile, Password, Labels -->
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-8">
        <!-- Profile -->
        <div class="bg-white dark:bg-dark-surface rounded-lg border border-gray-200 dark:border-dark-border p-6">
          <h2 class="text-base font-semibold text-gray-900 dark:text-dark-text mb-4">{{ t('settings.profile') }}</h2>
          <div class="space-y-4">
            <!-- Avatar & Name -->
            <div class="flex items-center gap-4">
              <div class="w-16 h-16 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center flex-shrink-0">
                <span class="text-2xl font-medium text-blue-600 dark:text-blue-400">{{ settingStore.settings.displayName.charAt(0) || 'U' }}</span>
              </div>
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <div v-if="!settingStore.isEditingName" class="flex items-center gap-2">
                    <span class="font-semibold text-gray-900 dark:text-dark-text cursor-pointer hover:text-gray-600 dark:hover:text-gray-300" @click="settingStore.startEditName">
                      {{ settingStore.settings.displayName || 'User' }}
                    </span>
                    <button @click="settingStore.startEditName" class="text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300">
                      <PencilIcon class="w-3.5 h-3.5" />
                    </button>
                  </div>
                  <input v-else ref="nameInputRef" v-model="settingStore.tempDisplayName" type="text"
                    class="input-field font-semibold text-base w-full"
                    @blur="settingStore.saveName" @keyup.enter="settingStore.saveName" @keyup.escape="settingStore.cancelEditName" />
                </div>
                <p class="text-sm text-gray-400 dark:text-gray-500 truncate">{{ settingStore.userEmail }}</p>
              </div>
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('settings.phone') }}</label>
              <input type="text" v-model="settingStore.settings.phone" :placeholder="t('settings.yourPhoneNumber')" class="input-field w-full" />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('settings.company') }}</label>
              <input type="text" v-model="settingStore.settings.company" :placeholder="t('settings.yourCompany')" class="input-field w-full" />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('settings.jobTitle') }}</label>
              <input type="text" v-model="settingStore.settings.jobTitle" :placeholder="t('settings.yourJobTitle')" class="input-field w-full" />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('settings.signature') }}</label>
              <textarea v-model="settingStore.settings.signature" :placeholder="t('settings.emailSignature')"
                class="textarea-field w-full h-20"></textarea>
            </div>
          </div>
        </div>

        <!-- Change Password -->
        <div class="bg-white dark:bg-dark-surface rounded-lg border border-gray-200 dark:border-dark-border p-6">
          <h2 class="text-base font-semibold text-gray-900 dark:text-dark-text mb-4">{{ t('settings.changePassword') }}</h2>
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('settings.currentPassword') }}</label>
              <input type="password" v-model="settingStore.passwordForm.oldPassword" :placeholder="t('settings.currentPasswordPlaceholder')" class="input-field w-full" />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('settings.newPassword') }}</label>
              <input type="password" v-model="settingStore.passwordForm.newPassword" :placeholder="t('settings.newPasswordPlaceholder')"
                class="input-field w-full" :class="{ 'border-red-400': settingStore.passwordErrors.newPassword }" />
              <p v-if="settingStore.passwordErrors.newPassword" class="mt-1 text-xs text-red-500">{{ settingStore.passwordErrors.newPassword }}</p>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('settings.confirmPassword') }}</label>
              <input type="password" v-model="settingStore.passwordForm.confirmPassword" :placeholder="t('settings.confirmPasswordPlaceholder')"
                class="input-field w-full" :class="{ 'border-red-400': settingStore.passwordErrors.confirmPassword }" />
              <p v-if="settingStore.passwordErrors.confirmPassword" class="mt-1 text-xs text-red-500">{{ settingStore.passwordErrors.confirmPassword }}</p>
            </div>
            <button @click="handleChangePassword" class="btn-primary-sm" :disabled="settingStore.isChangingPassword">
              <svg v-if="settingStore.isChangingPassword" class="animate-spin inline h-4 w-4 mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
              {{ t('settings.changePassword') }}
            </button>
          </div>
        </div>

        <!-- Labels -->
        <div class="bg-white dark:bg-dark-surface rounded-lg border border-gray-200 dark:border-dark-border p-6">
          <h2 class="text-base font-semibold text-gray-900 dark:text-dark-text mb-4">{{ t('settings.labels') }}</h2>
          <div class="space-y-4">
            <!-- Add Label -->
            <div class="flex items-end gap-2">
              <div class="flex-1">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('settings.name') }}</label>
                <input type="text" v-model="settingStore.newLabelName" :placeholder="t('settings.labelNamePlaceholder')" class="input-field w-full" />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('settings.color') }}</label>
                <input type="color" v-model="settingStore.newLabelColor" class="w-9 h-9 rounded-lg cursor-pointer border border-gray-200 dark:border-dark-border p-0.5 dark:bg-dark-surface" />
              </div>
              <button @click="handleAddLabel" class="btn-primary-sm" :disabled="settingStore.isAddingLabel">{{ t('settings.add') }}</button>
            </div>

            <!-- Labels List -->
            <div class="space-y-1.5 overflow-y-auto">
              <div v-for="lbl in settingStore.labels" :key="lbl.id"
                class="flex items-center justify-between px-3 py-2 bg-gray-50 dark:bg-dark-bg rounded-lg">
                <div class="flex items-center gap-2.5">
                  <span class="w-3 h-3 rounded-full flex-shrink-0" :style="{ backgroundColor: lbl.color }"></span>
                  <span class="text-sm text-gray-700 dark:text-gray-300">{{ lbl.name }}</span>
                  <span v-if="lbl.isBuiltin" class="text-[10px] text-gray-400 dark:text-gray-500 bg-gray-100 dark:bg-gray-700 px-1.5 py-0.5 rounded">{{ t('settings.builtin') }}</span>
                </div>
                <div class="flex items-center gap-1" v-if="!lbl.isBuiltin">
                  <input v-if="settingStore.editingLabelId === lbl.id" type="color" v-model="settingStore.editLabelColor"
                    class="w-7 h-7 rounded cursor-pointer border border-gray-200 dark:border-dark-border p-0.5 dark:bg-dark-surface" />
                  <button @click="settingStore.startEditLabel(lbl)" class="p-1.5 text-gray-400 dark:text-gray-500 hover:text-blue-600 dark:hover:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded transition-colors" v-if="settingStore.editingLabelId !== lbl.id">
                    <PencilIcon class="w-3 h-3" />
                  </button>
                  <button @click="handleSaveEditLabel(lbl.id)" class="p-1.5 text-green-600 dark:text-green-400 hover:text-green-700 dark:hover:text-green-300 hover:bg-green-50 dark:hover:bg-green-900/20 rounded transition-colors" v-if="settingStore.editingLabelId === lbl.id">
                    <CheckIcon class="w-3 h-3" />
                  </button>
                  <button @click="settingStore.cancelEditLabel" class="p-1.5 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-dark-bg rounded transition-colors" v-if="settingStore.editingLabelId === lbl.id">
                    <XIcon class="w-3 h-3" />
                  </button>
                  <button @click="handleRemoveLabel(lbl)" class="p-1.5 text-red-400 dark:text-red-500 hover:text-red-600 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors" :disabled="settingStore.isDeletingLabel === lbl.id">
                    <TrashIcon class="w-3 h-3" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Display Settings -->
      <div class="bg-white dark:bg-dark-surface rounded-lg border border-gray-200 dark:border-dark-border p-6 mb-8">
        <h2 class="text-base font-semibold text-gray-900 dark:text-dark-text mb-4">{{ t('settings.displaySettings') }}</h2>
        <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('settings.language') }}</label>
            <select v-model="settingStore.settings.language" class="select-field w-full">
              <option value="zh-CN">简体中文</option>
              <option value="en">English</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('settings.theme') }}</label>
            <select v-model="settingStore.settings.theme" class="select-field w-full">
              <option value="light">{{ t('settings.light') }}</option>
              <option value="dark">{{ t('settings.dark') }}</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('settings.pageSize') }}</label>
            <select v-model="settingStore.settings.pageSize" class="select-field w-full">
              <option :value="0">{{ t('settings.auto') }}</option>
              <option :value="10">10</option>
              <option :value="20">20</option>
              <option :value="50">50</option>
              <option :value="100">100</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('settings.readingPane') }}</label>
            <select v-model="settingStore.settings.readingPanePosition" class="select-field w-full">
              <option value="right">{{ t('settings.right') }}</option>
              <option value="bottom">{{ t('settings.bottom') }}</option>
            </select>
          </div>
        </div>
      </div>

      <!-- Email Behavior -->
      <div class="bg-white dark:bg-dark-surface rounded-lg border border-gray-200 dark:border-dark-border p-6 mb-8">
        <h2 class="text-base font-semibold text-gray-900 dark:text-dark-text mb-4">{{ t('settings.emailBehavior') }}</h2>
        <div class="space-y-3">
          <label class="flex items-center gap-2.5 cursor-pointer">
            <input type="checkbox" v-model="settingStore.settings.saveSent" class="checkbox-custom" />
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('settings.saveSentEmails') }}</span>
          </label>
          <label class="flex items-center gap-2.5 cursor-pointer">
            <input type="checkbox" v-model="settingStore.settings.includeOriginalOnReply" class="checkbox-custom" />
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('settings.includeOriginalOnReply') }}</span>
          </label>
          <label class="flex items-center gap-2.5 cursor-pointer">
            <input type="checkbox" v-model="settingStore.settings.forwardingEnabled" class="checkbox-custom" />
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('settings.enableEmailForwarding') }}</span>
          </label>
          <div v-if="settingStore.settings.forwardingEnabled" class="ml-6">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('settings.forwardTo') }}</label>
            <input type="email" v-model="settingStore.settings.forwardingAddress" placeholder="forward@example.com"
              class="input-field max-w-sm" :class="{ 'border-red-400': saveErrors.forwardingAddress }" />
            <p v-if="saveErrors.forwardingAddress" class="mt-1 text-xs text-red-500">{{ saveErrors.forwardingAddress }}</p>
          </div>
        </div>
      </div>

      <!-- Auto Reply -->
      <div class="bg-white dark:bg-dark-surface rounded-lg border border-gray-200 dark:border-dark-border p-6 mb-8">
        <h2 class="text-base font-semibold text-gray-900 dark:text-dark-text mb-4">{{ t('settings.autoReply') }}</h2>
        <div class="space-y-3">
          <label class="flex items-center gap-2.5 cursor-pointer">
            <input type="checkbox" v-model="settingStore.settings.autoReplyEnabled" class="checkbox-custom" />
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('settings.enableAutoReply') }}</span>
          </label>
          <div v-if="settingStore.settings.autoReplyEnabled" class="ml-6 space-y-3">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('settings.replySubject') }}</label>
              <input type="text" v-model="settingStore.settings.autoReplySubject" :placeholder="t('settings.replySubject')"
                class="input-field max-w-sm" :class="{ 'border-red-400': saveErrors.autoReplySubject }" />
              <p v-if="saveErrors.autoReplySubject" class="mt-1 text-xs text-red-500">{{ saveErrors.autoReplySubject }}</p>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('settings.replyBody') }}</label>
              <textarea v-model="settingStore.settings.autoReplyBody" :placeholder="t('settings.replyBody')"
                class="textarea-field w-full h-24 max-w-md" :class="{ 'border-red-400': saveErrors.autoReplyBody }"></textarea>
              <p v-if="saveErrors.autoReplyBody" class="mt-1 text-xs text-red-500">{{ saveErrors.autoReplyBody }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Notifications -->
      <div class="bg-white dark:bg-dark-surface rounded-lg border border-gray-200 dark:border-dark-border p-6 mb-8">
        <h2 class="text-base font-semibold text-gray-900 dark:text-dark-text mb-4">{{ t('settings.notifications') }}</h2>
        <div class="space-y-3">
          <label class="flex items-center gap-2.5 cursor-pointer">
            <input type="checkbox" v-model="settingStore.settings.notificationSound" class="checkbox-custom" />
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('settings.newEmailSound') }}</span>
          </label>
          <label class="flex items-center gap-2.5 cursor-pointer">
            <input type="checkbox" v-model="settingStore.settings.desktopNotification" class="checkbox-custom" />
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('settings.desktopNotifications') }}</span>
          </label>
          <p class="text-xs text-gray-400 dark:text-gray-500 ml-7">{{ t('settings.showDesktopNotifications') }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingStore } from '../stores/setting'
import type { Label } from '../types'
import ConfirmModal from '../components/ConfirmModal.vue'
import { TrashIcon, PencilIcon, CheckIcon, XIcon } from 'lucide-vue-next'
import { showToast } from '../utils/toast'

const { t } = useI18n()

const settingStore = useSettingStore()

const saveErrors = ref({
  forwardingAddress: '',
  autoReplySubject: '',
  autoReplyBody: ''
})

const confirmModal = ref({
  show: false,
  title: '',
  message: '',
  confirmText: '',
  loading: false,
  labelId: null as number | null
})

function handleConfirmModalConfirm() {
  if (confirmModal.value.labelId) {
    handleRemoveLabelById(confirmModal.value.labelId)
  }
}
function handleConfirmModalClose() {
  confirmModal.value.show = false
  confirmModal.value.title = ''
  confirmModal.value.message = ''
  confirmModal.value.labelId = null
}

async function handleSaveAll() {
  // Clear previous errors
  saveErrors.value = {
    forwardingAddress: '',
    autoReplySubject: '',
    autoReplyBody: ''
  }

  // Validate settings
  const { valid, errors } = settingStore.validateSettings()
  if (!valid) {
    saveErrors.value = errors
    showToast('warning', t('settings.toastPleaseFixErrors'))
    return
  }

  try {
    await settingStore.saveSettings()
    showToast('success', t('settings.toastSaved'))
  } catch (error: any) {
    showToast('error', error.response?.data?.message || t('settings.toastSaveFailed'))
  }
}

async function handleChangePassword() {
  if (!settingStore.passwordForm.oldPassword) {
    showToast('warning', t('settings.toastEnterCurrentPassword'))
    return
  }
  if (!settingStore.validatePassword()) {
    return
  }
  try {
    await settingStore.changePassword()
    showToast('success', t('settings.toastPasswordChanged'))
  } catch (error: any) {
    showToast('error', error.response?.data?.message || t('settings.toastChangeFailed'))
  }
}

async function handleAddLabel() {
  const name = settingStore.newLabelName.trim()
  if (!name) {
    showToast('warning', t('settings.toastEnterLabelName'))
    return
  }
  if (settingStore.labels.some(lbl => lbl.name.toLowerCase() === name.toLowerCase())) {
    showToast('warning', t('settings.toastLabelNameExists'))
    return
  }
  try {
    await settingStore.addLabel()
    showToast('success', t('settings.toastLabelAdded'))
  } catch (error) {
    console.error('Failed to create label:', error)
    showToast('error', t('settings.toastLabelAddFailed'))
  }
}

async function handleSaveEditLabel(id: number) {
  try {
    await settingStore.saveEditLabel(id)
  } catch (error) {
    console.error('Failed to update label:', error)
  }
}

function handleRemoveLabel(lbl: Label) {
  confirmModal.value = {
    show: true,
    title: t('settings.labels'),
    message: `${t('settings.sureDelete')} "${lbl.name}"?`,
    confirmText: t('settings.delete'),
    loading: false,
    labelId: lbl.id
  }
}

async function handleRemoveLabelById(id: number) {
  try {
    await settingStore.removeLabel(id)
  } catch (error) {
    console.error('Failed to delete label:', error)
  } finally {
    confirmModal.value.show = false
  }
}

onMounted(async () => {
  await settingStore.loadAll()
  // Focus name input after editing starts
  nextTick(() => {
    // Will be handled by the store's startEditName
  })
})
</script>