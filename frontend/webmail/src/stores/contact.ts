import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  getContacts,
  createContact,
  updateContact,
  deleteContact,
  getContactGroups,
  createContactGroup,
  updateContactGroup,
  deleteContactGroup
} from '../api/contact'
import type { Contact, ContactGroup, ContactInput } from '../types'
import type { ContactListParams, ContactListResponse } from '../api/contact'

export const useContactStore = defineStore('contact', () => {
  // === State ===
  const contactGroups = ref<ContactGroup[]>([])
  const contacts = ref<Contact[]>([])
  const total = ref(0)
  const isLoading = ref(false)
  const selectedGroupId = ref<string | null>(null)
  const searchKeyword = ref('')
  const currentPage = ref(1)
  const pageSize = ref(20)

  // === Computed Properties ===
  const totalPages = computed(() => {
    if (total.value === 0) return 0
    return Math.ceil(total.value / pageSize.value)
  })

  const defaultGroup = computed(() => {
    return contactGroups.value.find(g => g.isDefault) || contactGroups.value[0] || null
  })

  // === Contact Groups ===

  async function loadGroups() {
    try {
      const response = await getContactGroups()
      contactGroups.value = response.data || []
      return response.data
    } catch (error) {
      console.error('Failed to load contact groups:', error)
      throw error
    }
  }

  async function createGroup(name: string) {
    try {
      const response = await createContactGroup(name)
      if (response.data) {
        contactGroups.value.push(response.data)
      }
      return response
    } catch (error) {
      console.error('Failed to create group:', error)
      throw error
    }
  }

  async function renameGroup(id: string, name: string) {
    try {
      const response = await updateContactGroup(id, name)
      const group = contactGroups.value.find(g => g.id === id)
      if (group) {
        group.groupName = name
      }
      return response
    } catch (error) {
      console.error('Failed to rename group:', error)
      throw error
    }
  }

  async function removeGroup(id: string) {
    try {
      await deleteContactGroup(id)
      contactGroups.value = contactGroups.value.filter(g => g.id !== id)
      if (selectedGroupId.value === id) {
        selectedGroupId.value = null
      }
    } catch (error) {
      console.error('Failed to delete group:', error)
      throw error
    }
  }

  function getGroupName(groupId: string | null | undefined): string {
    if (!groupId) return '-'
    const group = contactGroups.value.find(g => g.id === groupId)
    return group?.groupName || '-'
  }

  // === Contacts ===

  async function loadContacts() {
    isLoading.value = true
    try {
      const params: ContactListParams = {
        page: currentPage.value,
        pageSize: pageSize.value
      }
      if (searchKeyword.value) params.q = searchKeyword.value
      if (selectedGroupId.value) params.groupId = selectedGroupId.value

      const response = await getContacts(params)
      const data: ContactListResponse = response.data || { items: [], total: 0, page: 1, pageSize: 20 }
      contacts.value = data.items
      total.value = data.total
      return data
    } catch (error) {
      console.error('Failed to load contacts:', error)
      throw error
    } finally {
      isLoading.value = false
    }
  }

  async function addContact(data: ContactInput) {
    try {
      const response = await createContact(data)
      if (response.data) {
        contacts.value.unshift(response.data)
        total.value++
      }
      // Update group counts
      await loadGroups()
      return response
    } catch (error) {
      console.error('Failed to create contact:', error)
      throw error
    }
  }

  async function editContact(id: string, data: ContactInput) {
    try {
      const response = await updateContact(id, data)
      if (response.data) {
        const index = contacts.value.findIndex(c => c.id === id)
        if (index !== -1) {
          contacts.value[index] = response.data
        }
      }
      // Update group counts
      await loadGroups()
      return response
    } catch (error) {
      console.error('Failed to update contact:', error)
      throw error
    }
  }

  async function removeContact(id: string) {
    try {
      await deleteContact(id)
      contacts.value = contacts.value.filter(c => c.id !== id)
      total.value--
      // Update group counts
      await loadGroups()
    } catch (error) {
      console.error('Failed to delete contact:', error)
      throw error
    }
  }

  // === Navigation ===

  function selectGroup(groupId: string | null) {
    selectedGroupId.value = groupId
    currentPage.value = 1
  }

  function setSearch(keyword: string) {
    searchKeyword.value = keyword
    currentPage.value = 1
  }

  function goToPage(page: number) {
    if (page < 1 || page > totalPages.value) return
    currentPage.value = page
  }

  // === Clear ===
  function clear() {
    contactGroups.value = []
    contacts.value = []
    total.value = 0
    selectedGroupId.value = null
    searchKeyword.value = ''
    currentPage.value = 1
  }

  return {
    // Group state
    contactGroups,
    defaultGroup,
    selectedGroupId,
    getGroupName,
    // Contact state
    contacts,
    total,
    isLoading,
    searchKeyword,
    currentPage,
    pageSize,
    totalPages,
    // Group operations
    loadGroups,
    createGroup,
    renameGroup,
    removeGroup,
    // Contact operations
    loadContacts,
    addContact,
    editContact,
    removeContact,
    // Navigation
    selectGroup,
    setSearch,
    goToPage,
    // Utilities
    clear
  }
})
