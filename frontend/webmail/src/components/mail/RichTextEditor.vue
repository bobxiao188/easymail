<template>
  <div class="rich-text-editor border border-gray-200 dark:border-dark-border rounded-lg overflow-hidden">
    <!-- Toolbar -->
    <div class="flex items-center gap-0.5 px-2 py-1.5 border-b border-gray-200 dark:border-dark-border bg-gray-50 dark:bg-dark-bg flex-wrap">
      <!-- Heading Dropdown -->
      <div class="relative" v-if="showHeadingMenu">
        <button 
          @click="headingMenuOpen = !headingMenuOpen"
          class="px-2 py-1.5 rounded text-xs font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
        >
          {{ currentHeading }}
          <svg class="w-3 h-3 inline ml-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
        </button>
        <div v-if="headingMenuOpen" @click.away="headingMenuOpen = false" class="absolute top-full left-0 mt-1 w-32 bg-white dark:bg-dark-surface border border-gray-200 dark:border-dark-border rounded-lg shadow-lg z-50 py-1">
          <button @click="setHeading('paragraph')" :class="!editor?.isActive('heading') ? 'bg-gray-100 dark:bg-gray-700' : ''" class="w-full text-left px-3 py-1.5 text-sm text-gray-900 dark:text-gray-100 hover:bg-gray-100 dark:hover:bg-gray-700">Normal</button>
          <button @click="setHeading('h1')" :class="editor?.isActive('heading', { level: 1 }) ? 'bg-gray-100 dark:bg-gray-700' : ''" class="w-full text-left px-3 py-1.5 text-sm text-gray-900 dark:text-gray-100 hover:bg-gray-100 dark:hover:bg-gray-700 font-bold text-lg">H1</button>
          <button @click="setHeading('h2')" :class="editor?.isActive('heading', { level: 2 }) ? 'bg-gray-100 dark:bg-gray-700' : ''" class="w-full text-left px-3 py-1.5 text-sm text-gray-900 dark:text-gray-100 hover:bg-gray-100 dark:hover:bg-gray-700 font-bold text-base">H2</button>
          <button @click="setHeading('h3')" :class="editor?.isActive('heading', { level: 3 }) ? 'bg-gray-100 dark:bg-gray-700' : ''" class="w-full text-left px-3 py-1.5 text-sm text-gray-900 dark:text-gray-100 hover:bg-gray-100 dark:hover:bg-gray-700 font-semibold text-sm">H3</button>
        </div>
      </div>
      
      <!-- Divider -->
      <div class="w-px h-5 bg-gray-300 dark:bg-gray-600 mx-0.5"></div>
      
      <!-- Font Family Dropdown -->
      <div class="relative">
        <button @click="fontFamilyMenuOpen = !fontFamilyMenuOpen; fontSizeMenuOpen = false; colorMenuOpen = false; bgColorMenuOpen = false" class="px-2 py-1 rounded text-xs text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors border border-gray-300 dark:border-gray-600 bg-white dark:bg-dark-surface" :style="currentFontFamily ? { fontFamily: currentFontFamily } : {}">{{ currentFontFamily || 'Font' }}</button>
        <div v-if="fontFamilyMenuOpen" @click.away="fontFamilyMenuOpen = false" class="absolute top-full left-0 mt-0.5 w-40 bg-white dark:bg-dark-surface border border-gray-200 dark:border-dark-border rounded-lg shadow-lg z-50 py-1 max-h-60 overflow-y-auto">
          <button v-for="font in fontFamilies" :key="font.value" @click="setFontFamily(font.value)" :style="{ fontFamily: font.value, fontWeight: currentFontFamily === font.value ? 600 : 400 }" :class="currentFontFamily === font.value ? 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400' : 'hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100'" class="w-full text-left px-3 py-1.5 text-sm">{{ font.label }}</button>
        </div>
      </div>
      
      <!-- Font Size Dropdown -->
      <div class="relative">
        <button @click="fontSizeMenuOpen = !fontSizeMenuOpen; fontFamilyMenuOpen = false; colorMenuOpen = false; bgColorMenuOpen = false" class="px-2 py-1 rounded text-xs text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors border border-gray-300 dark:border-gray-600 bg-white dark:bg-dark-surface">{{ currentFontSize || 'Size' }}</button>
        <div v-if="fontSizeMenuOpen" @click.away="fontSizeMenuOpen = false" class="absolute top-full left-0 mt-0.5 w-24 bg-white dark:bg-dark-surface border border-gray-200 dark:border-dark-border rounded-lg shadow-lg z-50 py-1 max-h-60 overflow-y-auto">
          <button v-for="size in fontSizes" :key="size.value" @click="setFontSize(size.value)" :style="{ fontSize: size.value }" :class="currentFontSize === size.value ? 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400' : 'hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100'" class="w-full text-left px-3 py-1.5">{{ size.label }}</button>
        </div>
      </div>
      
      <!-- Text Formatting -->
      <button @click="editor?.chain().focus().toggleBold().run()" :class="editor?.isActive('bold') ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'" class="p-1.5 rounded transition-colors" title="Bold"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 4h8a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z"/><path d="M6 12h9a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z"/></svg></button>
      <button @click="editor?.chain().focus().toggleItalic().run()" :class="editor?.isActive('italic') ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'" class="p-1.5 rounded transition-colors" title="Italic"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="19" y1="4" x2="10" y2="4"/><line x1="14" y1="20" x2="5" y2="20"/><line x1="15" y1="4" x2="9" y2="20"/></svg></button>
      <button @click="editor?.chain().focus().toggleUnderline().run()" :class="editor?.isActive('underline') ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'" class="p-1.5 rounded transition-colors" title="Underline"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 4v7a6 6 0 0 0 6 6 6 6 0 0 0 6-6V4"/><line x1="4" y1="20" x2="20" y2="20"/></svg></button>
      <button @click="editor?.chain().focus().toggleStrike().run()" :class="editor?.isActive('strike') ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'" class="p-1.5 rounded transition-colors" title="Strikethrough"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="4" y1="12" x2="20" y2="12"/><path d="M17.5 7.5c0-2.5-2.5-4-5-4s-5 1.5-5 4c0 2.5 5 4 5 4s5 1.5 5 4c0 2.5-2.5 4-5 4"/></svg></button>
      
      <!-- Text Color Picker -->
      <div class="relative">
        <button @click="colorMenuOpen = !colorMenuOpen; bgColorMenuOpen = false; fontSizeMenuOpen = false; fontFamilyMenuOpen = false" class="p-1.5 rounded text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors flex items-center gap-1" title="Text Color">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-palette-icon lucide-palette"><path d="M12 22a1 1 0 0 1 0-20 10 9 0 0 1 10 9 5 5 0 0 1-5 5h-2.25a1.75 1.75 0 0 0-1.4 2.8l.3.4a1.75 1.75 0 0 1-1.4 2.8z"/><circle cx="13.5" cy="6.5" r=".5" fill="currentColor"/><circle cx="17.5" cy="10.5" r=".5" fill="currentColor"/><circle cx="6.5" cy="12.5" r=".5" fill="currentColor"/><circle cx="8.5" cy="7.5" r=".5" fill="currentColor"/></svg>          <div class="w-3 h-0.5 rounded" :style="{ backgroundColor: currentTextColor || '#000000' }"></div>
        </button>
        <div v-if="colorMenuOpen" @click.away="colorMenuOpen = false" class="absolute top-full left-0 mt-0.5 bg-white dark:bg-dark-surface border border-gray-200 dark:border-dark-border rounded-lg shadow-lg z-50 p-2">
          <ColorPicker :initialColor="currentTextColor || '#000000'" @select="c => { setTextColor(c); colorMenuOpen = false }" mode="text" />
        </div>
      </div>
      
      <!-- Background Color Picker -->
      <div class="relative">
        <button @click="bgColorMenuOpen = !bgColorMenuOpen; colorMenuOpen = false; fontSizeMenuOpen = false; fontFamilyMenuOpen = false" class="p-1.5 rounded text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors flex items-center gap-1" title="Highlight Color">
          <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m12 2 3.09 6.26L22 9.27l-5 4.87L18.18 22 12 18.27 5.82 22 7 14.14l-5-4.87 6.91-1.01z"/></svg>
        </button>
        <div v-if="bgColorMenuOpen" @click.away="bgColorMenuOpen = false" class="absolute top-full left-0 mt-0.5 bg-white dark:bg-dark-surface border border-gray-200 dark:border-dark-border rounded-lg shadow-lg z-50 p-2">
          <ColorPicker :initialColor="bgPickerColor" @select="c => { setBgColor(c); bgColorMenuOpen = false }" mode="bg" />
        </div>
      </div>
      
      <!-- Clear Formatting -->
      <button @click="clearFormatting" class="p-1.5 rounded text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors" title="Clear Formatting"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 7V4h16v3"/><path d="M9 20h6"/><path d="M12 4v16"/><line x1="2" y1="22" x2="22" y2="2" stroke-width="2" stroke="#ef4444"/></svg></button>
      
      <!-- Divider -->
      <div class="w-px h-5 bg-gray-300 dark:bg-gray-600 mx-0.5"></div>
      
      <!-- Lists -->
      <button @click="editor?.chain().focus().toggleBulletList().run()" :class="editor?.isActive('bulletList') ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'" class="p-1.5 rounded transition-colors" title="Bullet List"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><circle cx="4" cy="6" r="1" fill="currentColor"/><circle cx="4" cy="12" r="1" fill="currentColor"/><circle cx="4" cy="18" r="1" fill="currentColor"/></svg></button>
      <button @click="editor?.chain().focus().toggleOrderedList().run()" :class="editor?.isActive('orderedList') ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'" class="p-1.5 rounded transition-colors" title="Numbered List"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="10" y1="6" x2="21" y2="6"/><line x1="10" y1="12" x2="21" y2="12"/><line x1="10" y1="18" x2="21" y2="18"/><text x="2" y="8" font-size="7" fill="currentColor" stroke="none">1</text><text x="2" y="14" font-size="7" fill="currentColor" stroke="none">2</text><text x="2" y="20" font-size="7" fill="currentColor" stroke="none">3</text></svg></button>
      
      <!-- Divider -->
      <div class="w-px h-5 bg-gray-300 dark:bg-gray-600 mx-0.5"></div>
      
      <!-- Block Formatting -->
      <button @click="editor?.chain().focus().toggleBlockquote().run()" :class="editor?.isActive('blockquote') ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'" class="p-1.5 rounded transition-colors" title="Quote"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 21c3 0 7-1 7-8V5c0-1.25-.756-2.057-2-2.057A1.987 1.987 0 0 1 6 3v15"/><path d="M13 21c3 0 7-1 7-8V5c0-1.25-.757-2.057-2-2.057A1.987 1.987 0 0 1 16 3v15"/></svg></button>
      <button @click="editor?.chain().focus().toggleCode().run()" :class="editor?.isActive('code') ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'" class="p-1.5 rounded transition-colors" title="Inline Code"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg></button>
      <button @click="editor?.chain().focus().toggleCodeBlock().run()" :class="editor?.isActive('codeBlock') ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'" class="p-1.5 rounded transition-colors" title="Code Block"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="4" width="20" height="16" rx="2"/><polyline points="8 10 4 12 8 14"/><polyline points="16 10 20 12 16 14"/></svg></button>
      <button @click="editor?.chain().focus().setHorizontalRule().run()" class="p-1.5 rounded text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors" title="Horizontal Rule"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="3" y1="12" x2="21" y2="12"/></svg></button>
      
      <!-- Divider -->
      <div class="w-px h-5 bg-gray-300 dark:bg-gray-600 mx-0.5"></div>
      
      <!-- Insert -->
      <button @click="triggerImageUpload" class="p-1.5 rounded text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors" title="Insert Image"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg></button>
      <input type="file" ref="imageInput" accept="image/*" multiple @change="handleImageUpload" class="hidden" />
      <button @click="handleAddLink" :class="editor?.isActive('link') ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'" class="p-1.5 rounded transition-colors" title="Insert Link"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg></button>
      
      <!-- Divider -->
      <div class="w-px h-5 bg-gray-300 dark:bg-gray-600 mx-0.5"></div>
      
      <!-- Alignment -->
      <button @click="editor?.chain().focus().setTextAlign('left').run()" :class="editor?.isActive({ textAlign: 'left' }) ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'" class="p-1.5 rounded transition-colors" title="Align Left"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="17" y1="10" x2="3" y2="10"/><line x1="21" y1="6" x2="3" y2="6"/><line x1="21" y1="14" x2="3" y2="14"/><line x1="17" y1="18" x2="3" y2="18"/></svg></button>
      <button @click="editor?.chain().focus().setTextAlign('center').run()" :class="editor?.isActive({ textAlign: 'center' }) ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'" class="p-1.5 rounded transition-colors" title="Align Center"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="10" x2="6" y2="10"/><line x1="21" y1="6" x2="3" y2="6"/><line x1="21" y1="14" x2="3" y2="14"/><line x1="18" y1="18" x2="6" y2="18"/></svg></button>
      <button @click="editor?.chain().focus().setTextAlign('right').run()" :class="editor?.isActive({ textAlign: 'right' }) ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'" class="p-1.5 rounded transition-colors" title="Align Right"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="21" y1="10" x2="7" y2="10"/><line x1="21" y1="6" x2="3" y2="6"/><line x1="21" y1="14" x2="3" y2="14"/><line x1="21" y1="18" x2="7" y2="18"/></svg></button>
      
      <!-- Divider -->
      <div class="w-px h-5 bg-gray-300 dark:bg-gray-600 mx-0.5"></div>
      
      <!-- Undo/Redo -->
      <button @click="editor?.chain().focus().undo().run()" :disabled="!editor?.can().undo()" class="p-1.5 rounded text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed" title="Undo"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 7v6h6"/><path d="M21 17a9 9 0 0 0-9-9 9 9 0 0 0-6 2.3L3 13"/></svg></button>
      <button @click="editor?.chain().focus().redo().run()" :disabled="!editor?.can().redo()" class="p-1.5 rounded text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed" title="Redo"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 7v6h-6"/><path d="M3 17a9 9 0 0 1 9-9 9 9 0 0 1 6 2.3l3 3.7"/></svg></button>
    </div>
    
    <!-- Editor Content -->
    <div class="bg-white dark:bg-dark-surface">
      <editor-content :editor="editor" class="min-h-[400px] p-4" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { EditorContent, useEditor } from '@tiptap/vue-3'
import type { EditorView } from '@tiptap/pm/view'
import { Extension } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import Image from '@tiptap/extension-image'
import Link from '@tiptap/extension-link'
import Underline from '@tiptap/extension-underline'
import Placeholder from '@tiptap/extension-placeholder'
import TextAlign from '@tiptap/extension-text-align'
import Color from '@tiptap/extension-color'
import { TextStyle } from '@tiptap/extension-text-style'
import ColorPicker from './ColorPicker.vue'
import { uploadImage } from '../../api/email'

const { t } = useI18n()

// Custom FontSize extension
const FontSize = Extension.create({
  name: 'fontSize',
  addGlobalAttributes() {
    return [{ types: ['textStyle'], attributes: { fontSize: { default: null, parseHTML: element => element.style.fontSize, renderHTML: attributes => { if (!attributes.fontSize) return {}; return { style: `font-size: ${attributes.fontSize}` } } } } }]
  },
})

// Custom FontFamily extension
const FontFamily = Extension.create({
  name: 'fontFamily',
  addGlobalAttributes() {
    return [{ types: ['textStyle'], attributes: { fontFamily: { default: null, parseHTML: element => element.style.fontFamily.replace(/"/g, ''), renderHTML: attributes => { if (!attributes.fontFamily) return {}; return { style: `font-family: ${attributes.fontFamily}` } } } } }]
  },
})

const props = defineProps<{
  modelValue?: string
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const imageInput = ref<HTMLInputElement | null>(null)
const headingMenuOpen = ref(false)
const fontSizeMenuOpen = ref(false)
const fontFamilyMenuOpen = ref(false)
const colorMenuOpen = ref(false)
const bgColorMenuOpen = ref(false)

const showHeadingMenu = ref(true)
const bgPickerColor = ref('#FFFFCC')

const fontSizes = [
  { label: '8pt', value: '8pt' },
  { label: '9pt', value: '9pt' },
  { label: '10pt', value: '10pt' },
  { label: '11pt', value: '11pt' },
  { label: '12pt', value: '12pt' },
  { label: '14pt', value: '14pt' },
  { label: '16pt', value: '16pt' },
  { label: '18pt', value: '18pt' },
  { label: '20pt', value: '20pt' },
  { label: '24pt', value: '24pt' },
  { label: '28pt', value: '28pt' },
  { label: '36pt', value: '36pt' },
]

const fontFamilies = [
  { label: 'Default', value: '' },
  { label: 'Arial', value: 'Arial' },
  { label: 'Helvetica', value: 'Helvetica' },
  { label: 'Times New Roman', value: 'Times New Roman' },
  { label: 'Georgia', value: 'Georgia' },
  { label: 'Courier New', value: 'Courier New' },
  { label: 'Verdana', value: 'Verdana' },
  { label: 'Microsoft YaHei', value: 'Microsoft YaHei' },
  { label: 'SimSun', value: 'SimSun' },
]

const currentHeading = computed(() => {
  if (!editor.value) return 'Normal'
  if (editor.value.isActive('heading', { level: 1 })) return 'H1'
  if (editor.value.isActive('heading', { level: 2 })) return 'H2'
  if (editor.value.isActive('heading', { level: 3 })) return 'H3'
  return 'Normal'
})

const currentFontSize = computed(() => {
  if (!editor.value) return ''
  return editor.value.getAttributes('textStyle').fontSize || ''
})

const currentFontFamily = computed(() => {
  if (!editor.value) return ''
  return editor.value.getAttributes('textStyle').fontFamily || ''
})

const currentTextColor = computed(() => {
  if (!editor.value) return ''
  return editor.value.getAttributes('textStyle').color || ''
})

function setHeading(type: string) {
  if (!editor.value) return
  if (type === 'paragraph') {
    editor.value.chain().focus().setParagraph().run()
  } else {
    const level = Number(type.substring(1)) as 1 | 2 | 3 | 4 | 5 | 6
    editor.value.chain().focus().toggleHeading({ level }).run()
  }
  headingMenuOpen.value = false
}

function setFontSize(size: string) {
  if (!editor.value) return
  if (!size) {
    editor.value.commands.unsetMark('textStyle')
  } else {
    const currentColor = editor.value.getAttributes('textStyle').color
    const currentFamily = editor.value.getAttributes('textStyle').fontFamily
    const attrs: Record<string, string> = { fontSize: size }
    if (currentColor) attrs.color = currentColor
    if (currentFamily) attrs.fontFamily = currentFamily
    editor.value.commands.setMark('textStyle', attrs)
  }
  fontSizeMenuOpen.value = false
}

function setFontFamily(family: string) {
  if (!editor.value) return
  if (!family) {
    editor.value.commands.unsetMark('textStyle')
  } else {
    const currentColor = editor.value.getAttributes('textStyle').color
    const currentSize = editor.value.getAttributes('textStyle').fontSize
    const attrs: Record<string, string> = { fontFamily: family }
    if (currentColor) attrs.color = currentColor
    if (currentSize) attrs.fontSize = currentSize
    editor.value.commands.setMark('textStyle', attrs)
  }
  fontFamilyMenuOpen.value = false
}

function setTextColor(color: string) {
  if (!editor.value) return
  if (color === currentTextColor.value) {
    editor.value.chain().focus().unsetColor().run()
  } else {
    editor.value.chain().focus().setColor(color).run()
  }
  colorMenuOpen.value = false
}

function setBgColor(color: string) {
  if (!editor.value) return
  if (color === '') {
    // Remove background color by resetting the mark
    const currentColor = editor.value.getAttributes('textStyle').color
    const currentSize = editor.value.getAttributes('textStyle').fontSize
    const currentFamily = editor.value.getAttributes('textStyle').fontFamily
    const attrs: Record<string, string> = {}
    if (currentColor) attrs.color = currentColor
    if (currentSize) attrs.fontSize = currentSize
    if (currentFamily) attrs.fontFamily = currentFamily
    if (Object.keys(attrs).length > 0) {
      editor.value.commands.setMark('textStyle', attrs)
    } else {
      editor.value.commands.unsetMark('textStyle')
    }
  } else {
    const currentColor = editor.value.getAttributes('textStyle').color
    const currentSize = editor.value.getAttributes('textStyle').fontSize
    const currentFamily = editor.value.getAttributes('textStyle').fontFamily
    const attrs: Record<string, string> = { backgroundColor: color }
    if (currentColor) attrs.color = currentColor
    if (currentSize) attrs.fontSize = currentSize
    if (currentFamily) attrs.fontFamily = currentFamily
    editor.value.commands.setMark('textStyle', attrs)
  }
  bgColorMenuOpen.value = false
}

function clearFormatting() {
  if (!editor.value) return
  editor.value.chain().focus().unsetBold().unsetItalic().unsetUnderline().unsetStrike().unsetColor().unsetMark('textStyle').run()
}

const editor = useEditor({
  extensions: [
    StarterKit.configure({
      dropcursor: {
        color: '#3b82f6',
        width: 2,
      },
      link: false,
      underline: false,
    }),
    Image.configure({
      inline: false,
      allowBase64: true,
      HTMLAttributes: {
        class: 'max-w-full h-auto rounded-lg my-2',
      },
    }),
    Link.configure({
      openOnClick: false,
      HTMLAttributes: {
        target: '_blank',
        rel: 'noopener noreferrer nofollow',
        class: 'text-blue-600 hover:text-blue-800 underline',
      },
    }),
    Underline,
    TextAlign.configure({
      types: ['heading', 'paragraph'],
    }),
    Color,
    TextStyle,
    FontSize,
    FontFamily,
    Placeholder.configure({
      placeholder: 'Write your message...',
    }),
  ],
  content: props.modelValue,
  editorProps: {
    attributes: {
      class: 'prose max-w-none focus:outline-none min-h-[400px]',
    },
    handlePaste: (_view: EditorView, event: ClipboardEvent) => {
      // Handle image paste from clipboard
      const items = event.clipboardData?.items
      if (items) {
        for (const item of items) {
          if (item.type.startsWith('image/')) {
            event.preventDefault()
            const file = item.getAsFile()
            if (file && editor.value) {
              handleClipboardImage(file)
            }
            return true
          }
        }
      }
      // Allow default paste behavior for formatted text
      return false
    },
    handleDrop: (_view: EditorView, event: DragEvent) => {
      // Handle image drop
      const files = event.dataTransfer?.files
      if (files) {
        for (const file of files) {
          if (file.type.startsWith('image/')) {
            event.preventDefault()
            handleClipboardImage(file)
            return true
          }
        }
      }
      return false
    },
  },
  onUpdate: () => {
    emit('update:modelValue', editor.value?.getHTML() || '')
  },
})

// Handle image upload
function triggerImageUpload() {
  imageInput.value?.click()
}

async function handleImageUpload(event: Event) {
  const target = event.target as HTMLInputElement
  const files = target.files
  
  if (files && files.length > 0 && editor.value) {
    for (let i = 0; i < files.length; i++) {
      const file = files[i]
      await insertImage(file)
    }
  }
  
  // Reset input
  if (imageInput.value) {
    imageInput.value.value = ''
  }
}

// Handle clipboard image (paste or drop)
async function handleClipboardImage(file: File) {
  if (!editor.value) return
  await insertImage(file)
}

// Insert image into editor
async function insertImage(file: File) {
  if (!editor.value) return
  
  try {
    // Try to upload image first
    const response = await uploadImage(file)
    const imageUrl = response.data?.url
    
    if (imageUrl) {
      // Insert image at current cursor position
      editor.value.chain().focus().setImage({ 
        src: imageUrl,
        alt: file.name,
      }).run()
      return
    }
  } catch (error) {
    console.error('Failed to upload image, using base64 fallback:', error)
  }
  
  // Fallback: use base64
  const reader = new FileReader()
  const base64 = await new Promise<string>((resolve, reject) => {
    reader.onload = () => resolve(reader.result as string)
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
  editor.value.chain().focus().setImage({ src: base64 }).run()
}

// Handle add link
function handleAddLink() {
  if (!editor.value) return
  
  const previousUrl = editor.value.getAttributes('link').href
  const url = window.prompt(t('mail.enterUrl'), previousUrl)
  
  // if there is a null url, the link will be removed
  if (url === null) {
    return
  }
  
  // if there is a empty url, the link will be removed
  if (url === '') {
    editor.value.chain().focus().extendMarkRange('link').unsetLink().run()
    return
  }
  
  // update link
  editor.value
    .chain()
    .focus()
    .extendMarkRange('link')
    .setLink({ href: url })
    .run()
}

// Watch for external changes to modelValue
watch(
  () => props.modelValue,
  (newValue) => {
    if (editor.value && newValue !== editor.value.getHTML()) {
      editor.value.commands.setContent(newValue || '')
    }
  }
)

onBeforeUnmount(() => {
  editor.value?.destroy()
})
</script>

<style>
/* ProseMirror Editor Content Styles - Global for TipTap */
.rich-text-editor .ProseMirror {
  outline: none;
  min-height: 400px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  font-size: 14px;
  line-height: 1.6;
  color: #374151;
  
  * {
    font-family: inherit;
    font-size: inherit;
    line-height: inherit;
  }
}

/* Dark mode for ProseMirror */
.dark .rich-text-editor .ProseMirror {
  color: #e5e7eb;
}

.rich-text-editor .ProseMirror > * + * {
  margin-top: 0.5em;
}

.rich-text-editor .ProseMirror h1 {
  font-size: 1.6em;
  font-weight: 700;
  margin-top: 1em;
  margin-bottom: 0.5em;
  color: #1f2937;
}

.dark .rich-text-editor .ProseMirror h1 {
  color: #f3f4f6;
}

.rich-text-editor .ProseMirror h2 {
  font-size: 1.4em;
  font-weight: 600;
  margin-top: 1em;
  margin-bottom: 0.5em;
  color: #1f2937;
}

.dark .rich-text-editor .ProseMirror h2 {
  color: #f3f4f6;
}

.rich-text-editor .ProseMirror h3 {
  font-size: 1.2em;
  font-weight: 600;
  margin-top: 1em;
  margin-bottom: 0.5em;
  color: #1f2937;
}

.dark .rich-text-editor .ProseMirror h3 {
  color: #f3f4f6;
}

.rich-text-editor .ProseMirror h4,
.rich-text-editor .ProseMirror h5,
.rich-text-editor .ProseMirror h6 {
  font-weight: 600;
  margin-top: 1em;
  margin-bottom: 0.5em;
  color: #1f2937;
}

.dark .rich-text-editor .ProseMirror h4,
.dark .rich-text-editor .ProseMirror h5,
.dark .rich-text-editor .ProseMirror h6 {
  color: #f3f4f6;
}

.rich-text-editor .ProseMirror p {
  margin-bottom: 0.5em;
}

.rich-text-editor .ProseMirror strong {
  font-weight: 700;
}

.rich-text-editor .ProseMirror em {
  font-style: italic;
}

.rich-text-editor .ProseMirror u {
  text-decoration: underline;
}

.rich-text-editor .ProseMirror s {
  text-decoration: line-through;
}

.rich-text-editor .ProseMirror ul {
  padding-left: 1.5em;
  margin-bottom: 0.5em;
  list-style-type: disc;
}

.rich-text-editor .ProseMirror ol {
  padding-left: 1.5em;
  margin-bottom: 0.5em;
  list-style-type: decimal;
}

.rich-text-editor .ProseMirror li {
  margin-bottom: 0.25em;
}

.rich-text-editor .ProseMirror blockquote {
  border-left: 3px solid #e5e7eb;
  padding-left: 1em;
  margin-left: 0;
  margin-bottom: 0.5em;
  color: #6b7280;
  background: #f9fafb;
  padding: 0.5em 1em;
  border-radius: 0 0.25em 0.25em 0.5em;
}

.dark .rich-text-editor .ProseMirror blockquote {
  border-left-color: #4b5563;
  color: #9ca3af;
  background: #1f2937;
}

.rich-text-editor .ProseMirror code {
  background-color: #f3f4f6;
  padding: 0.2em 0.4em;
  border-radius: 0.25em;
  font-size: 0.875em;
  font-family: 'Courier New', Consolas, monospace;
  color: #ef4444;
}

.dark .rich-text-editor .ProseMirror code {
  background-color: #374151;
  color: #f87171;
}

.rich-text-editor .ProseMirror pre {
  background: #1f2937;
  color: #f9fafb;
  padding: 0.75em 1em;
  border-radius: 0.5em;
  overflow-x: auto;
  margin-bottom: 0.5em;
  
  code {
    background: none;
    padding: 0;
    border-radius: 0;
    color: inherit;
    font-size: 0.875em;
  }
}

.dark .rich-text-editor .ProseMirror pre {
  background: #111827;
}

.rich-text-editor .ProseMirror img {
  max-width: 100%;
  height: auto;
  display: block;
  margin: 1em 0;
  border-radius: 0.5em;
}

.rich-text-editor .ProseMirror a {
  color: #2563eb;
  text-decoration: underline;
  cursor: pointer;
  
  &:hover {
    color: #1e40af;
  }
}

.dark .rich-text-editor .ProseMirror a {
  color: #60a5fa;
  
  &:hover {
    color: #93c5fd;
  }
}

.rich-text-editor .ProseMirror hr {
  border: none;
  border-top: 1px solid #e5e7eb;
  margin: 1em 0;
}

.dark .rich-text-editor .ProseMirror hr {
  border-top-color: #4b5563;
}

/* Placeholder */
.rich-text-editor .ProseMirror p.is-editor-empty:first-child::before {
  content: attr(data-placeholder);
  float: left;
  color: #9ca3af;
  pointer-events: none;
  height: 0;
}

.dark .rich-text-editor .ProseMirror p.is-editor-empty:first-child::before {
  color: #6b7280;
}

/* Remove default list padding */
.rich-text-editor .ProseMirror ul,
.rich-text-editor .ProseMirror ol {
  list-style: none;
  padding-left: 0;
}

.rich-text-editor .ProseMirror ul.nested,
.rich-text-editor .ProseMirror ol.nested {
  padding-left: 1.5em;
}
</style>
