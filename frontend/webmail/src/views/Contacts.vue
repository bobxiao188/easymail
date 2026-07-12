<template>
  <div class="flex-1 w-full overflow-y-auto bg-gray-50 dark:bg-dark-bg">
    <div class="px-6 py-8">
      <!-- Page Title -->
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-dark-text">{{ t('contacts.pageTitle') }}</h1>
          <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">{{ t('contacts.pageDesc') }}</p>
        </div>
        <button @click="openCreateDialog()" class="btn-primary-sm flex items-center gap-2">
          <PlusIcon class="w-4 h-4" />
          {{ t('contacts.newContact') }}
        </button>
      </div>

      <!-- Create Contact Dialog -->
      <div v-if="showCreateDialog" class="fixed inset-0 z-50 flex items-center justify-center">
        <div class="fixed inset-0 bg-black/30" @click="showCreateDialog = false" />
        <div class="relative bg-white dark:bg-dark-surface rounded-lg shadow-xl w-full max-w-md p-6 animate-in fade-in zoom-in duration-200">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-dark-text mb-4">{{ editingContact ? t('contacts.editContact') : t('contacts.newContact') }}</h3>
          <form @submit.prevent="handleSaveContact" class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('contacts.group') }} *</label>
              <select v-model="form.groupId" class="input-field w-full" required>
                <option value="" disabled>{{ t('contacts.selectGroup') }}</option>
                <option v-for="group in contactGroups" :key="group.id" :value="group.id">{{ group.groupName }}</option>
              </select>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('contacts.name') }} *</label>
              <input v-model="form.name" type="text" :placeholder="t('contacts.contactNamePlaceholder')" class="input-field w-full" required />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('contacts.email') }} *</label>
              <input v-model="form.email" type="email" :placeholder="t('contacts.emailPlaceholder')" class="input-field w-full"
                :class="{ 'border-red-400': formErrors.email }" required />
              <p v-if="formErrors.email" class="mt-1 text-xs text-red-500">{{ formErrors.email }}</p>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('contacts.phone') }}</label>
              <input v-model="form.phone" type="tel" :placeholder="t('contacts.phonePlaceholder')" class="input-field w-full" />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('contacts.address') }}</label>
              <input v-model="form.address" type="text" :placeholder="t('contacts.addressPlaceholder')" class="input-field w-full" />
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('contacts.city') }}</label>
                <input v-model="form.city" type="text" :placeholder="t('contacts.cityPlaceholder')" class="input-field w-full" />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('contacts.state') }}</label>
                <input v-model="form.state" type="text" :placeholder="t('contacts.statePlaceholder')" class="input-field w-full" />
              </div>
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('contacts.zip') }}</label>
                <input v-model="form.zip" type="text" :placeholder="t('contacts.zipPlaceholder')" class="input-field w-full" />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('contacts.country') }}</label>
                <input v-model="form.country" type="text" :placeholder="t('contacts.countryPlaceholder')" class="input-field w-full" />
              </div>
            </div>
            <div class="flex justify-end gap-3 pt-2">
              <button type="button" @click="showCreateDialog = false" class="px-4 py-2 text-sm text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-dark-text rounded hover:bg-gray-100 dark:hover:bg-dark-bg transition-colors">{{ t('common.cancel') }}</button>
              <button type="submit" class="btn-primary-sm" :disabled="isSaving">
                <svg v-if="isSaving" class="animate-spin inline h-4 w-4 mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
                {{ editingContact ? t('common.update') : t('common.create') }}
              </button>
            </div>
          </form>
        </div>
      </div>

      <!-- Delete Contact Confirmation Dialog -->
      <div v-if="showDeleteContactDialog" class="fixed inset-0 z-50 flex items-center justify-center">
        <div class="fixed inset-0 bg-black/30" @click="showDeleteContactDialog = false" />
        <div class="relative bg-white dark:bg-dark-surface rounded-lg shadow-xl w-full max-w-sm p-6 animate-in fade-in zoom-in duration-200">
          <div class="flex items-center gap-3 mb-4">
            <div class="w-10 h-10 rounded-full bg-red-100 flex items-center justify-center flex-shrink-0">
              <TrashIcon class="w-5 h-5 text-red-600" />
            </div>
            <div>
              <h3 class="text-lg font-semibold text-gray-900 dark:text-dark-text">{{ t('contacts.deleteContact') }}</h3>
              <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('contacts.actionCannotBeUndone') }}</p>
            </div>
          </div>
          <p class="text-gray-700 dark:text-gray-300 mb-6">{{ t('contacts.sureDelete') }} "<span class="font-semibold">{{ deleteContactName }}</span>"?</p>
          <div class="flex justify-end gap-3">
            <button @click="showDeleteContactDialog = false" class="px-4 py-2 text-sm text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-dark-text rounded hover:bg-gray-100 dark:hover:bg-dark-bg transition-colors">{{ t('common.cancel') }}</button>
            <button @click="handleDeleteContact" class="px-4 py-2 text-sm bg-red-600 text-white rounded hover:bg-red-700 transition-colors" :disabled="isSaving">{{ t('common.delete') }}</button>
          </div>
        </div>
      </div>

      <!-- Create Group Dialog -->
      <div v-if="showCreateGroupDialog" class="fixed inset-0 z-50 flex items-center justify-center">
        <div class="fixed inset-0 bg-black/30" @click="showCreateGroupDialog = false" />
        <div class="relative bg-white dark:bg-dark-surface rounded-lg shadow-xl w-full max-w-sm p-6 animate-in fade-in zoom-in duration-200">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-dark-text">{{ t('contacts.newGroup') }}</h3>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('contacts.groupName') }}</label>
            <input ref="createGroupInput" v-model="newGroupName" type="text" :placeholder="t('contacts.groupNamePlaceholder')" class="input-field w-full" @keyup.enter="handleCreateGroup" />
          </div>
          <div class="flex justify-end gap-3">
            <button @click="showCreateGroupDialog = false" class="px-4 py-2 text-sm text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-dark-text rounded hover:bg-gray-100 dark:hover:bg-dark-bg transition-colors">{{ t('common.cancel') }}</button>
            <button @click="handleCreateGroup" class="btn-primary-sm">{{ t('common.create') }}</button>
          </div>
        </div>
      </div>

      <!-- Rename Group Dialog -->
      <div v-if="showRenameGroupDialog" class="fixed inset-0 z-50 flex items-center justify-center">
        <div class="fixed inset-0 bg-black/30" @click="showRenameGroupDialog = false" />
        <div class="relative bg-white dark:bg-dark-surface rounded-lg shadow-xl w-full max-w-sm p-6 animate-in fade-in zoom-in duration-200">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-dark-text">{{ t('contacts.renameGroup') }}</h3>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ t('contacts.newName') }}</label>
            <input ref="renameGroupInput" v-model="renameGroupName" type="text" :placeholder="t('contacts.enterNewName')" class="input-field w-full" @keyup.enter="handleRenameGroup" />
          </div>
          <div class="flex justify-end gap-3">
            <button @click="showRenameGroupDialog = false" class="px-4 py-2 text-sm text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-dark-text rounded hover:bg-gray-100 dark:hover:bg-dark-bg transition-colors">{{ t('common.cancel') }}</button>
            <button @click="handleRenameGroup" class="btn-primary-sm">{{ t('common.rename') }}</button>
          </div>
        </div>
      </div>

      <!-- Delete Group Confirmation Dialog -->
      <div v-if="showDeleteGroupDialog" class="fixed inset-0 z-50 flex items-center justify-center">
        <div class="fixed inset-0 bg-black/30" @click="showDeleteGroupDialog = false" />
        <div class="relative bg-white dark:bg-dark-surface rounded-lg shadow-xl w-full max-w-sm p-6 animate-in fade-in zoom-in duration-200">
          <div class="flex items-center gap-3 mb-4">
            <div class="w-10 h-10 rounded-full bg-red-100 flex items-center justify-center flex-shrink-0">
              <TrashIcon class="w-5 h-5 text-red-600" />
            </div>
            <div>
              <h3 class="text-lg font-semibold text-gray-900 dark:text-dark-text">{{ t('contacts.deleteGroup') }}</h3>
              <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('contacts.actionCannotBeUndone') }}</p>
            </div>
          </div>
          <p class="text-gray-700 dark:text-gray-300 mb-6">{{ t('contacts.sureDeleteGroup') }} "<span class="font-semibold">{{ deleteGroupName }}</span>"? {{ t('contacts.contactsWillBeUngrouped') }}</p>
          <div class="flex justify-end gap-3">
            <button @click="showDeleteGroupDialog = false" class="px-4 py-2 text-sm text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-dark-text rounded hover:bg-gray-100 dark:hover:bg-dark-bg transition-colors">{{ t('common.cancel') }}</button>
            <button @click="handleDeleteGroup" class="px-4 py-2 text-sm bg-red-600 text-white rounded hover:bg-red-700 transition-colors" :disabled="isSaving">{{ t('common.delete') }}</button>
          </div>
        </div>
      </div>

      <!-- Contact Groups -->
      <div class="mb-8">
        <div class="flex items-center justify-between mb-3">
          <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wider">{{ t('contacts.pageTitle') }} - {{ t('contacts.group') }}</h2>
          <button @click="openCreateGroupDialog()" class="text-sm text-blue-600 hover:text-blue-700 font-medium transition-colors">
           {{ t('contacts.newGroup') }}
          </button>
        </div>

        <!-- Empty State -->
        <div v-if="contactGroups.length === 0" class="bg-white dark:bg-dark-surface rounded-lg border border-gray-200 dark:border-dark-border p-8 text-center">
          <UsersIcon class="w-12 h-12 text-gray-300 mx-auto mb-3" />
          <p class="text-gray-500 dark:text-gray-400">{{ t('contacts.noGroups') }}</p>
          <p class="text-sm text-gray-400 mt-1">{{ t('contacts.noGroupsDesc') }}</p>
        </div>

        <!-- Groups List -->
        <div v-else class="bg-white dark:bg-dark-surface rounded-lg border border-gray-200 dark:border-dark-border overflow-hidden">
          <table class="min-w-full">
            <thead class="bg-gray-50 dark:bg-dark-bg border-b border-gray-200 dark:border-dark-border">
              <tr>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('contacts.group') }}</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('contacts.pageTitle') }}</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('common.created') }}</th>
                <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
              <tr v-for="group in contactGroups" :key="group.id" class="hover:bg-gray-50 dark:hover:bg-dark-bg transition-colors">
                <td class="px-6 py-4">
                  <div class="flex items-center gap-3">
                    <UsersIcon class="w-5 h-5 text-gray-400" />
                    <div>
                      <span class="font-medium text-gray-900 dark:text-dark-text">{{ group.groupName }}</span>
                      <span v-if="group.isDefault" class="ml-2 text-xs text-blue-500 bg-blue-50 px-2 py-0.5 rounded-full">{{ t('contacts.defaultGroup') }}</span>
                    </div>
                  </div>
                </td>
                <td class="px-6 py-4 text-sm text-gray-900 dark:text-dark-text">{{ group.contactCount || 0 }}</td>
                <td class="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">{{ formatDate(group.createTime) }}</td>
                <td class="px-6 py-4 text-right">
                  <div class="flex items-center justify-end gap-2">
                    <button v-if="!group.isDefault" @click="openRenameGroupDialog(group.id, group.groupName)" class="p-2 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors" :title="t('common.rename')">
                      <TagIcon class="w-4 h-4" />
                    </button>
                    <button v-if="!group.isDefault" @click="openDeleteGroupDialog(group.id, group.groupName)" class="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors" :title="t('common.delete')">
                      <TrashIcon class="w-4 h-4" />
                    </button>
                    <span v-if="group.isDefault" class="text-xs text-gray-400">{{ t('common.system') }}</span>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Contacts List -->
      <div>
        <div class="flex items-center justify-between mb-3">
          <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wider">{{ t('contacts.pageTitle') }}</h2>
          <div class="flex items-center gap-3">
            <!-- Group Filter -->
            <select v-model="selectedGroupId" @change="onGroupChange" class="text-sm border border-gray-200 dark:border-dark-border rounded-lg px-3 py-2 bg-white dark:bg-dark-surface focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400">
              <option :value="null">{{ t('common.all') }}</option>
              <option v-for="group in contactGroups" :key="group.id" :value="group.id">{{ group.groupName }}</option>
            </select>
            <!-- Search -->
            <div class="relative">
              <SearchIcon class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input v-model="searchKeyword" @keyup.enter="handleSearch" :placeholder="t('common.searchContacts')"
                class="pl-9 pr-3 py-2 text-sm border border-gray-200 dark:border-dark-border rounded-lg placeholder:text-gray-400
                       focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all w-60 bg-white dark:bg-dark-surface" />
            </div>
          </div>
        </div>

        <!-- Empty State -->
        <div v-if="contactsData.items.length === 0" class="bg-white dark:bg-dark-surface rounded-lg border border-gray-200 dark:border-dark-border p-8 text-center">
          <UserIcon class="w-12 h-12 text-gray-300 mx-auto mb-3" />
          <p class="text-gray-500 dark:text-gray-400">{{ t('contacts.noContacts') }}</p>
          <p class="text-sm text-gray-400 mt-1">{{ t('contacts.noContactsDesc') }}</p>
        </div>

        <!-- Contacts Table -->
        <div v-else class="bg-white dark:bg-dark-surface rounded-lg border border-gray-200 dark:border-dark-border overflow-hidden">
          <table class="min-w-full">
            <thead class="bg-gray-50 dark:bg-dark-bg border-b border-gray-200 dark:border-dark-border">
              <tr>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('contacts.name') }}</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('contacts.email') }}</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('contacts.phone') }}</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('contacts.city') }}</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('contacts.group') }}</th>
                <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{{ t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
              <tr v-for="contact in contactsData.items" :key="contact.id" class="hover:bg-gray-50 dark:hover:bg-dark-bg transition-colors">
                <td class="px-6 py-4">
                  <div class="flex items-center gap-3">
                    <div class="w-8 h-8 rounded-full bg-blue-100 flex items-center justify-center flex-shrink-0">
                      <span class="text-sm font-medium text-blue-600">{{ getInitials(contact.contactName) }}</span>
                    </div>
                    <span class="font-medium text-gray-900 dark:text-dark-text">{{ contact.contactName }}</span>
                  </div>
                </td>
                <td class="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">{{ contact.contactEmail }}</td>
                <td class="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">{{ contact.contactPhone || '-' }}</td>
                <td class="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">{{ contact.contactCity || '-' }}</td>
                <td class="px-6 py-4">
                  <span class="text-sm text-gray-500 dark:text-gray-400">{{ getGroupName(contact.contactGroupId) }}</span>
                </td>
                <td class="px-6 py-4 text-right">
                  <div class="flex items-center justify-end gap-2">
                    <button @click="sendEmail(contact)" class="p-2 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors" :title="t('contacts.sendEmail')">
                      <MailIcon class="w-4 h-4" />
                    </button>
                    <button @click="openEditDialog(contact)" class="p-2 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors" :title="t('contacts.edit')">
                      <TagIcon class="w-4 h-4" />
                    </button>
                    <button @click="openDeleteContactDialog(contact)" class="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors" :title="t('contacts.delete')">
                      <TrashIcon class="w-4 h-4" />
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>

          <!-- Pagination -->
          <div class="px-6 py-4 border-t border-gray-200 dark:border-dark-border flex items-center justify-between">
            <span class="text-sm text-gray-500 dark:text-gray-400">{{ contactsData.total }} {{ t('contacts.contactsInGroup') }}</span>
            <div class="flex items-center gap-2">
              <button @click="changePage(pagination.currentPage - 1)" :disabled="pagination.currentPage <= 1"
                class="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:hover:bg-dark-bg rounded transition-colors disabled:opacity-40 disabled:cursor-not-allowed">
                <ChevronLeftIcon class="w-4 h-4" />
              </button>
              <span class="text-sm text-gray-500 dark:text-gray-400 font-medium px-2">{{ pagination.currentPage }} / {{ pagination.totalPages }}</span>
              <button @click="changePage(pagination.currentPage + 1)" :disabled="pagination.currentPage >= pagination.totalPages"
                class="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:hover:bg-dark-bg rounded transition-colors disabled:opacity-40 disabled:cursor-not-allowed">
                <ChevronRightIcon class="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, computed, toRef } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useContactStore } from '../stores/contact'
import { useSettingStore } from '../stores/setting'
import type { Contact, ContactInput } from '../types'
import {
  PlusIcon,
  UserIcon,
  UsersIcon,
  TrashIcon,
  TagIcon,
  MailIcon,
  SearchIcon,
  ChevronLeftIcon,
  ChevronRightIcon
} from 'lucide-vue-next'
import { showToast } from '../utils/toast'

const router = useRouter()
const { t } = useI18n()
const contactStore = useContactStore()
const settingStore = useSettingStore()

// === Dialog States ===
const showCreateDialog = ref(false)
const showDeleteContactDialog = ref(false)
const showCreateGroupDialog = ref(false)
const showRenameGroupDialog = ref(false)
const showDeleteGroupDialog = ref(false)

// === Contact Form ===
const editingContact = ref<Contact | null>(null)
const isSaving = ref(false)
const formErrors = ref({ email: '' })
const form = ref({
  groupId: '',
  name: '',
  email: '',
  phone: '',
  address: '',
  city: '',
  state: '',
  zip: '',
  country: ''
})

// === Delete Contact ===
const deleteContactId = ref<string>('')
const deleteContactName = ref('')

// === Create Group ===
const newGroupName = ref('')
const createGroupInput = ref<HTMLInputElement>()

// === Rename Group ===
const renameGroupId = ref<string>('')
const renameGroupName = ref('')
const renameGroupInput = ref<HTMLInputElement>()

// === Delete Group ===
const deleteGroupId = ref<string>('')
const deleteGroupName = ref('')

// === Get reactive references from store (for v-model) ===
const selectedGroupId = toRef(contactStore, 'selectedGroupId')
const searchKeyword = toRef(contactStore, 'searchKeyword')

// === Get computed properties from store ===
const contactGroups = computed(() => contactStore.contactGroups)
const contacts = computed(() => contactStore.contacts)
const total = computed(() => contactStore.total)
const totalPages = computed(() => contactStore.totalPages)
const currentPage = computed(() => contactStore.currentPage)
const pageSize = computed(() => contactStore.pageSize)

// Compatible with contactsData in template
const contactsData = computed(() => ({
  items: contacts.value,
  total: total.value,
  page: currentPage.value,
  pageSize: pageSize.value
}))

// Compatible with pagination in template
const pagination = computed(() => ({
  currentPage: currentPage.value,
  pageSize: pageSize.value,
  totalPages: totalPages.value
}))

// === Date Formatting ===
function formatDate(dateStr: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' })
}

// === Get Name Initials ===
function getInitials(name: string): string {
  if (!name) return '?'
  return name.charAt(0).toUpperCase()
}

// === Get Group Name ===
function getGroupName(groupId: string | null | undefined): string {
  return contactStore.getGroupName(groupId)
}

// === Load Contact Groups ===
async function loadContactGroups() {
  try {
    await contactStore.loadGroups()
  } catch (error) {
    console.error('Failed to load groups:', error)
  }
}

// === Load Contacts List ===
async function loadContacts() {
  try {
    await contactStore.loadContacts()
  } catch (error) {
    console.error('Failed to load contacts:', error)
  }
}

// === Pagination ===
function changePage(page: number) {
  contactStore.goToPage(page)
  loadContacts()
}

// === Search ===
function handleSearch() {
  contactStore.setSearch(searchKeyword.value)
  loadContacts()
}

// === Group Switch ===
function onGroupChange() {
  contactStore.selectGroup(selectedGroupId.value)
  loadContacts()
}

// === Create Contact Dialog ===
function openCreateDialog() {
  editingContact.value = null
  resetForm()
  // Default select first group
  if (contactStore.defaultGroup) {
    form.value.groupId = contactStore.defaultGroup.id
  }
  showCreateDialog.value = true
}

// === Edit Contact Dialog ===
function openEditDialog(contact: Contact) {
  editingContact.value = contact
  form.value = {
    groupId: contact.contactGroupId || '',
    name: contact.contactName,
    email: contact.contactEmail,
    phone: contact.contactPhone || '',
    address: contact.contactAddress || '',
    city: contact.contactCity || '',
    state: contact.contactState || '',
    zip: contact.contactZip || '',
    country: contact.contactCountry || ''
  }
  formErrors.value = { email: '' }
  showCreateDialog.value = true
}

// === Reset Form ===
function resetForm() {
  form.value = { groupId: '', name: '', email: '', phone: '', address: '', city: '', state: '', zip: '', country: '' }
  formErrors.value = { email: '' }
}

// === Validate Form ===
function validateForm(): boolean {
  formErrors.value = { email: '' }
  if (!form.value.name.trim()) {
    showToast('warning', t('contacts.toastEnterContactName'))
    return false
  }
  if (!form.value.email.trim()) {
    formErrors.value.email = t('contacts.toastEnterEmail')
    return false
  }
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!emailRegex.test(form.value.email)) {
    formErrors.value.email = t('common.invalidEmail')
    return false
  }
  if (!form.value.groupId) {
    showToast('warning', t('contacts.selectGroup'))
    return false
  }
  return true
}

// === Save Contact ===
async function handleSaveContact() {
  if (!validateForm()) return
  isSaving.value = true
  try {
    const data: ContactInput = {
      contactName: form.value.name,
      contactEmail: form.value.email,
      contactGroupId: form.value.groupId
    }
    if (form.value.phone) data.contactPhone = form.value.phone
    if (form.value.address) data.contactAddress = form.value.address
    if (form.value.city) data.contactCity = form.value.city
    if (form.value.state) data.contactState = form.value.state
    if (form.value.zip) data.contactZip = form.value.zip
    if (form.value.country) data.contactCountry = form.value.country

    if (editingContact.value) {
      await contactStore.editContact(editingContact.value.id, data)
      showToast('success', t('contacts.toastUpdated'))
    } else {
      await contactStore.addContact(data)
      showToast('success', t('contacts.toastCreated'))
    }
    showCreateDialog.value = false
    await loadContacts()
  } catch (error) {
    console.error('Failed to save contact:', error)
    const message = (error as any).response?.data?.message || t('common.operationFailed')
    showToast('error', message)
  } finally {
    isSaving.value = false
  }
}

// === Delete Contact Dialog ===
function openDeleteContactDialog(contact: Contact) {
  deleteContactId.value = contact.id
  deleteContactName.value = contact.contactName
  showDeleteContactDialog.value = true
}

// === Handle Delete Contact ===
async function handleDeleteContact() {
  if (!deleteContactId.value) return
  isSaving.value = true
  try {
    await contactStore.removeContact(deleteContactId.value)
    showToast('success', t('contacts.toastDeleted'))
    showDeleteContactDialog.value = false
    await loadContacts()
  } catch (error) {
    console.error('Failed to delete contact:', error)
    const message = (error as any).response?.data?.message || t('contacts.toastDeleteFailed')
    showToast('error', message)
  } finally {
    isSaving.value = false
  }
}

// === Send Email ===
function sendEmail(contact: Contact) {
  router.push({ path: '/compose', query: { recipient: contact.contactEmail, name: contact.contactName } })
}

// === Create Group Dialog ===
function openCreateGroupDialog() {
  newGroupName.value = ''
  showCreateGroupDialog.value = true
  nextTick(() => {
    createGroupInput.value?.focus()
  })
}

// === Handle Create Group ===
async function handleCreateGroup() {
  const name = newGroupName.value.trim()
  if (!name) {
    showToast('warning', t('contacts.toastEnterGroupName'))
    return
  }
  try {
    await contactStore.createGroup(name)
    showToast('success', t('contacts.toastGroupCreated'))
    showCreateGroupDialog.value = false
    await loadContacts()
  } catch (error) {
    console.error('Failed to create group:', error)
    const message = (error as any).response?.data?.message || t('contacts.toastGroupCreateFailed')
    showToast('error', message)
  }
}

// === Rename Group Dialog ===
function openRenameGroupDialog(groupId: string, groupName: string) {
  renameGroupId.value = groupId
  renameGroupName.value = groupName
  showRenameGroupDialog.value = true
  nextTick(() => {
    renameGroupInput.value?.focus()
  })
}

// === Handle Rename Group ===
async function handleRenameGroup() {
  const name = renameGroupName.value.trim()
  if (!name) {
    showToast('warning', t('contacts.toastEnterNewName'))
    return
  }
  try {
    await contactStore.renameGroup(renameGroupId.value, name)
    showToast('success', t('contacts.toastGroupRenamed'))
    showRenameGroupDialog.value = false
  } catch (error) {
    console.error('Failed to rename group:', error)
    const message = (error as any).response?.data?.message || t('contacts.toastGroupRenameFailed')
    showToast('error', message)
  }
}

// === Delete Group Dialog ===
function openDeleteGroupDialog(groupId: string, groupName: string) {
  deleteGroupId.value = groupId
  deleteGroupName.value = groupName
  showDeleteGroupDialog.value = true
}

// === Handle Delete Group ===
async function handleDeleteGroup() {
  if (!deleteGroupId.value) return
  isSaving.value = true
  try {
    await contactStore.removeGroup(deleteGroupId.value)
    showToast('success', t('contacts.toastGroupDeleted'))
    showDeleteGroupDialog.value = false
    await loadContacts()
  } catch (error) {
    console.error('Failed to delete group:', error)
    const message = (error as any).response?.data?.message || t('contacts.toastGroupDeleteFailed')
    showToast('error', message)
  } finally {
    isSaving.value = false
  }
}

// === Initialize ===
onMounted(async () => {
  // Use pageSize from settings
  contactStore.pageSize = settingStore.settings.pageSize || 20
  await loadContactGroups()
  // Default no filter, show all groups
  contactStore.selectedGroupId = null
  await loadContacts()
})
</script>
