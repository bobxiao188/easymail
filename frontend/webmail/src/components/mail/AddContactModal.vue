<!-- AddContactModal.vue -->
<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { XMarkIcon, UserPlusIcon, ExclamationTriangleIcon } from '@heroicons/vue/24/outline'
import { useContactStore } from '../../stores/contact'
import type { ContactGroup, ContactInput } from '../../types'
import { getContacts } from '../../api/contact'

// Props
const props = withDefaults(defineProps<{
  isOpen: boolean
  email: string
  name: string
}>(), {
  isOpen: false,
  email: '',
  name: ''
})

// Emits
const emit = defineEmits<{
  close: []
  success: []
}>()

// Store
const contactStore = useContactStore()

// Local state
const formData = ref({
  contactName: '',
  contactEmail: '',
  contactPhone: '',
  contactAddress: '',
  contactCity: '',
  contactState: '',
  contactZip: '',
  contactCountry: '',
  contactGroupId: ''
})
const isLoading = ref(false)
const isSubmitting = ref(false)
const errorMessage = ref('')
const emailExists = ref(false)
const groups = ref<ContactGroup[]>([])

// Computed
const hasFields = computed(() => {
  return formData.value.contactName || formData.value.contactPhone ||
         formData.value.contactAddress || formData.value.contactCity
})

const isSubmitEnabled = computed(() => {
  return !isSubmitting.value && 
         formData.value.contactEmail && 
         !emailExists.value
})

// Watch for modal open
watch(() => props.isOpen, async (newVal) => {
  if (newVal) {
    // Reset form with props data
    formData.value = {
      contactName: props.name || '',
      contactEmail: props.email || '',
      contactPhone: '',
      contactAddress: '',
      contactCity: '',
      contactState: '',
      contactZip: '',
      contactCountry: '',
      contactGroupId: ''
    }
    errorMessage.value = ''
    emailExists.value = false
    
    // Load groups and check if email exists
    await loadGroups()
    
    // Set default group
    if (contactStore.defaultGroup) {
      formData.value.contactGroupId = contactStore.defaultGroup.id
    }
    
    checkEmailExists()
  }
})

// Load contact groups
async function loadGroups() {
  isLoading.value = true
  try {
    // contactStore.contactGroups is already a ref, access it directly
    groups.value = contactStore.contactGroups
    if (groups.value.length === 0) {
      await contactStore.loadGroups()
      groups.value = contactStore.contactGroups
    }
  } catch (error) {
    console.error('Failed to load groups:', error)
  } finally {
    isLoading.value = false
  }
}

// Check if email already exists in contacts
function checkEmailExists() {
  if (!props.email) {
    emailExists.value = false
    return
  }
  
  const emailLower = props.email.toLowerCase()
  
  // Check in store contacts (contactStore.contacts is already a ref)
  for (const contact of contactStore.contacts) {
    if (contact.contactEmail?.toLowerCase() === emailLower) {
      emailExists.value = true
      return
    }
  }
  
  // Also check via API search
  searchContactByEmail(props.email)
}

// Search contact by email via API
async function searchContactByEmail(email: string) {
  try {
    const response = await getContacts({ q: email, page: 1, pageSize: 1 })
    if (response.data && response.data.items.length > 0) {
      emailExists.value = true
    } else {
      emailExists.value = false
    }
  } catch (error) {
    console.error('Failed to search contact:', error)
  }
}

// Submit form
async function handleSubmit() {
  if (!isSubmitEnabled.value || !props.email) return
  
  isSubmitting.value = true
  errorMessage.value = ''
  
  const data: ContactInput = {
    contactName: formData.value.contactName,
    contactEmail: formData.value.contactEmail,
    contactPhone: formData.value.contactPhone || undefined,
    contactAddress: formData.value.contactAddress || undefined,
    contactCity: formData.value.contactCity || undefined,
    contactState: formData.value.contactState || undefined,
    contactZip: formData.value.contactZip || undefined,
    contactCountry: formData.value.contactCountry || undefined,
    contactGroupId: formData.value.contactGroupId || contactStore.defaultGroup?.id || null
  }
  
  try {
    await contactStore.addContact(data)
    emit('success')
  } catch (error: any) {
    console.error('Failed to create contact:', error)
    errorMessage.value = error.response?.data?.message || 'Failed to add contact'
  } finally {
    isSubmitting.value = false
  }
}

// Close modal
function handleClose() {
  emit('close')
}
</script>

<template>
  <div v-if="isOpen" class="fixed inset-0 z-50 flex items-center justify-center">
    <!-- Backdrop -->
    <div 
      class="absolute inset-0 bg-black/30 animate-in fade-in" 
      @click="handleClose"
    ></div>
    
    <!-- Modal -->
    <div class="relative bg-white rounded-xl shadow-xl w-full max-w-lg mx-4 animate-in zoom-in-95">
      <!-- Header -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-gray-200">
        <div class="flex items-center gap-2">
          <UserPlusIcon class="w-5 h-5 text-blue-600" />
          <h2 class="text-lg font-semibold text-gray-900">Add to Contacts</h2>
        </div>
        <button 
          @click="handleClose"
          class="p-1.5 rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors"
        >
          <XMarkIcon class="w-5 h-5" />
        </button>
      </div>
      
      <!-- Body -->
      <div class="px-6 py-5 max-h-[70vh] overflow-y-auto">
        <!-- Email Already Exists Warning -->
        <div v-if="emailExists" class="mb-4 p-3 bg-amber-50 border border-amber-200 rounded-lg flex items-start gap-3">
          <ExclamationTriangleIcon class="w-5 h-5 text-amber-600 shrink-0 mt-0.5" />
          <div class="flex-1">
            <p class="text-sm font-medium text-amber-800">Contact Already Exists</p>
            <p class="text-xs text-amber-700 mt-1">
              A contact with email "{{ email }}" is already in your contacts.
            </p>
          </div>
        </div>
        
        <!-- Form -->
        <form @submit.prevent="handleSubmit" class="space-y-4">
          <!-- Email (Read-only) -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1.5">
              Email <span class="text-red-500">*</span>
            </label>
            <input 
              type="email" 
              v-model="formData.contactEmail"
              disabled
              class="w-full px-3 py-2 border border-gray-300 rounded-lg bg-gray-50 text-gray-600 cursor-not-allowed"
            />
          </div>
          
          <!-- Name -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1.5">
              Name
            </label>
            <input 
              type="text" 
              v-model="formData.contactName"
              placeholder="Contact name"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
          
          <!-- Phone -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1.5">
              Phone
            </label>
            <input 
              type="tel" 
              v-model="formData.contactPhone"
              placeholder="Phone number"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
          
          <!-- Group -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1.5">
              Group <span class="text-red-500">*</span>
            </label>
            <select 
              v-model="formData.contactGroupId"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option 
                v-for="group in groups" 
                :key="group.id" 
                :value="group.id"
              >
                {{ group.groupName }}
              </option>
            </select>
          </div>
          
          <!-- Address (Optional, collapsible) -->
          <details v-if="hasFields" class="group">
            <summary class="cursor-pointer text-sm font-medium text-gray-700 hover:text-gray-900">
              Address (Optional)
            </summary>
            <div class="mt-3 space-y-3 pl-4 border-l-2 border-gray-200">
              <div>
                <label class="block text-xs font-medium text-gray-600 mb-1">Address</label>
                <input 
                  type="text" 
                  v-model="formData.contactAddress"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="block text-xs font-medium text-gray-600 mb-1">City</label>
                  <input 
                    type="text" 
                    v-model="formData.contactCity"
                    class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
                <div>
                  <label class="block text-xs font-medium text-gray-600 mb-1">State</label>
                  <input 
                    type="text" 
                    v-model="formData.contactState"
                    class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
              </div>
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="block text-xs font-medium text-gray-600 mb-1">Zip</label>
                  <input 
                    type="text" 
                    v-model="formData.contactZip"
                    class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
                <div>
                  <label class="block text-xs font-medium text-gray-600 mb-1">Country</label>
                  <input 
                    type="text" 
                    v-model="formData.contactCountry"
                    class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
              </div>
            </div>
          </details>
          
          <!-- Error Message -->
          <div v-if="errorMessage" class="p-3 bg-red-50 border border-red-200 rounded-lg">
            <p class="text-sm text-red-700">{{ errorMessage }}</p>
          </div>
        </form>
      </div>
      
      <!-- Footer -->
      <div class="flex items-center justify-end gap-3 px-6 py-4 border-t border-gray-200 bg-gray-50 rounded-b-xl">
        <button 
          @click="handleClose"
          class="px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200 rounded-lg transition-colors"
        >
          Cancel
        </button>
        <button 
          @click="handleSubmit"
          :disabled="!isSubmitEnabled"
          class="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <span v-if="!isSubmitting">Add Contact</span>
          <span v-else class="flex items-center gap-2">
            <svg class="w-4 h-4 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            Adding...
          </span>
        </button>
      </div>
    </div>
  </div>
</template>
