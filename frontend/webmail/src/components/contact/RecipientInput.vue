<template>
  <div class="relative" ref="wrapperRef">
    <div class="flex flex-wrap items-center gap-1.5 px-3 py-2 min-h-[2.5rem] border border-gray-200 dark:border-dark-border rounded-xl
                focus-within:border-gray-400 focus-within:ring-1 focus-within:ring-gray-200 dark:focus-within:ring-dark-border transition-all cursor-text bg-white dark:bg-dark-surface"
      @click="focusInput">
      <span v-for="(r, i) in modelValue" :key="i"
        class="inline-flex items-center gap-0.5 px-2 py-0.5 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg text-xs font-medium">
        {{ r.name || r.email }}
        <button @click.stop="remove(i)" class="hover:text-red-500 ml-0.5 leading-none text-gray-400 dark:text-gray-500">&times;</button>
      </span>
      <input ref="inputRef" v-model="query"
        :placeholder="modelValue.length === 0 ? placeholder : ''"
        class="flex-1 min-w-[120px] border-none outline-none bg-transparent text-sm text-gray-900 dark:text-gray-100 placeholder:text-gray-400 dark:placeholder:text-gray-500 py-0.5"
        @input="onInput" @keydown="onKeydown" @blur="onBlur" />
    </div>

    <ul v-if="showDropdown"
      class="absolute top-full left-0 right-0 z-50 mt-1 bg-white dark:bg-dark-surface border border-gray-100 dark:border-dark-border rounded-xl shadow-lg py-1 max-h-48 overflow-y-auto">
      <li v-for="(c, i) in suggestions" :key="c.id"
        class="flex items-center gap-2 px-3 py-2 text-sm cursor-pointer"
        :class="{ 'bg-gray-50 dark:bg-dark-bg': i === activeIndex }"
        @mousedown.prevent="selectSuggestion(c)" @mouseenter="activeIndex = i">
        <div class="avatar-circle sm">
          <span>{{ (c.contactName || '?').charAt(0) }}</span>
        </div>
        <div class="min-w-0">
          <p class="font-medium text-gray-900 dark:text-gray-100 truncate">{{ c.contactName }}</p>
          <p class="text-xs text-gray-400 dark:text-gray-500 truncate">{{ c.contactEmail }}</p>
        </div>
      </li>
      <li v-if="suggestions.length === 0 && query.trim()"
        class="px-3 py-2 text-sm text-gray-400 dark:text-gray-500 pointer-events-none">
        No matching contacts found
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { getContacts } from '../../api/contact'
import type { Recipient, Contact } from '../../types'

const props = defineProps<{
  modelValue: Recipient[]
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: Recipient[]]
}>()

const query = ref('')
const suggestions = ref<Contact[]>([])
const showDropdown = ref(false)
const activeIndex = ref(0)
const inputRef = ref<HTMLInputElement>()
let debounceTimer: ReturnType<typeof setTimeout> | null = null

function focusInput() { inputRef.value?.focus() }

function remove(i: number) {
  const next = [...props.modelValue]
  next.splice(i, 1)
  emit('update:modelValue', next)
}

async function searchContacts(keyword: string) {
  if (!keyword.trim()) { suggestions.value = []; showDropdown.value = false; return }
  try {
    const res = await getContacts({ q: keyword, pageSize: 10 })
    const items = res.data.items || []
    const selectedEmails = new Set(props.modelValue.map(r => r.email))
    suggestions.value = items.filter(c => !selectedEmails.has(c.contactEmail))
    showDropdown.value = suggestions.value.length > 0 || keyword.trim().length > 0
    activeIndex.value = 0
  } catch { suggestions.value = [] }
}

function onInput() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => searchContacts(query.value), 200)
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault()
    if (showDropdown.value && suggestions.value.length > 0 && suggestions.value[activeIndex.value]) {
      selectSuggestion(suggestions.value[activeIndex.value])
    } else { addFromText() }
  } else if (e.key === 'Backspace' && !query.value) { remove(props.modelValue.length - 1) }
  else if (e.key === 'ArrowDown') { e.preventDefault(); if (suggestions.value.length > 0) activeIndex.value = (activeIndex.value + 1) % suggestions.value.length }
  else if (e.key === 'ArrowUp') { e.preventDefault(); if (suggestions.value.length > 0) activeIndex.value = (activeIndex.value - 1 + suggestions.value.length) % suggestions.value.length }
  else if (e.key === 'Escape') { showDropdown.value = false }
}

function addRecipient(r: Recipient) {
  if (props.modelValue.some(x => x.email === r.email) || !r.email) return
  emit('update:modelValue', [...props.modelValue, r])
}

function addFromText() {
  const email = query.value.trim()
  if (!email) return
  addRecipient({ name: '', email })
  query.value = ''
  suggestions.value = []
  showDropdown.value = false
}

function selectSuggestion(c: Contact) {
  addRecipient({ name: c.contactName, email: c.contactEmail })
  query.value = ''
  suggestions.value = []
  showDropdown.value = false
  focusInput()
}

function onBlur() { setTimeout(() => { showDropdown.value = false }, 200) }

watch(() => props.modelValue, () => { if (props.modelValue.length === 0) query.value = '' })
</script>