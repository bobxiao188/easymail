<!-- src/components/mail/ReadingPane.vue -->
<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Email, Attachment } from '../../types'

const { t } = useI18n()
import { 
  ArrowUturnLeftIcon, 
  ArrowUturnRightIcon, 
  ForwardIcon, 
  XMarkIcon,
  PaperClipIcon,
  TrashIcon,
  StarIcon as StarOutline,
  ArrowDownTrayIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  PrinterIcon,
  EnvelopeIcon,
  UserPlusIcon,
  DocumentTextIcon,
  ClipboardDocumentIcon,
  CodeBracketIcon,
  TableCellsIcon
} from '@heroicons/vue/24/outline'
import { StarIcon as StarSolid } from '@heroicons/vue/24/solid'
import { downloadAttachment, downloadAllAttachments } from '../../api/email'
import { useContactStore } from '../../stores/contact'
import { useSettingStore } from '../../stores/setting'
import AddContactModal from './AddContactModal.vue'
import { showToast } from '../../utils/toast'

const settingStore = useSettingStore()

// Props
const props = withDefaults(defineProps<{
  email?: Email | null
  currentFolderId?: number | null
}>(), {
  email: null,
  currentFolderId: null
})

// Emits
const emit = defineEmits<{
  close: []
  reply: [email: Email]
  replyAll: [email: Email]
  forward: [email: Email]
  edit: [email: Email]
  delete: [id: number]
  toggleStar: [id: number]
  prevEmail: []
  nextEmail: []
}>()

// Store
const contactStore = useContactStore()

// Local state
const loadingAttachments = ref<Record<number, boolean>>({})
const emailBodyIframe = ref<HTMLIFrameElement | null>(null)
const iframeHeight = ref(0)
const showAddContactModal = ref(false)
const selectedContactEmail = ref('')
const selectedContactName = ref('')
const showPrintPreview = ref(false)

// Headers modal state
const showHeadersModal = ref(false)
const showRawSource = ref(false)
const headersCopied = ref(false)
const rawCopied = ref(false)

// Check if email already exists in contacts
function isEmailInContacts(email: string): boolean {
  if (!email) return false
  const emailLower = email.toLowerCase()
  return contactStore.contacts.some(contact => 
    contact.contactEmail?.toLowerCase() === emailLower
  )
}

// Computed properties
const hasAttachments = computed(() => {
  return props.email && props.email.attachments && props.email.attachments.length > 0
})

// Draft folder ID (from folder.ts: FolderKind.Draft = 3)
const DRAFT_FOLDER_ID = 3
const showEditButton = computed(() => {
  return props.currentFolderId === DRAFT_FOLDER_ID
})

// Generate iframe content with theme-aware styles
function generateIframeContent(isDark: boolean): string {
  if (!props.email?.bodyHtml) return ''
  return `
    <!DOCTYPE html>
    <html>
    <head>
      <meta charset="utf-8">
      <meta name="viewport" content="width=device-width, initial-scale=1.0">
      <style>
        html, body {
          margin: 0;
          padding: 0;
          overflow: hidden;
          height: auto;
        }
        body {
          font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
          font-size: 14px;
          line-height: 1.6;
          color: ${isDark ? '#e5e7eb' : '#374151'};
          background-color: ${isDark ? '#1f2937' : '#ffffff'};
        }
        a {
          color: ${isDark ? '#60a5fa' : '#2563eb'};
          text-decoration: underline;
          cursor: pointer;
        }
        a:hover {
          color: ${isDark ? '#93c5fd' : '#1e40af'};
        }
        img {
          max-width: 100%;
          height: auto;
        }
        table {
          border-collapse: collapse;
        }
        th, td {
          border: 1px solid ${isDark ? '#374151' : '#e5e7eb'};
          padding: 8px;
        }
        th {
          background-color: ${isDark ? '#374151' : '#f9fafb'};
          font-weight: 600;
        }
        h1, h2, h3, h4, h5, h6 {
          color: ${isDark ? '#f3f4f6' : '#111827'};
        }
        p {
          color: ${isDark ? '#e5e7eb' : '#374151'};
        }
        code {
          background-color: ${isDark ? '#374151' : '#f3f4f6'};
          color: ${isDark ? '#e5e7eb' : '#374151'};
          padding: 2px 4px;
          border-radius: 3px;
        }
        pre {
          background-color: ${isDark ? '#374151' : '#f3f4f6'};
          padding: 12px;
          border-radius: 4px;
          overflow-x: auto;
        }
        pre code {
          background-color: transparent;
          padding: 0;
        }
        blockquote {
          border-left: 3px solid ${isDark ? '#4b5563' : '#d1d5db'};
          padding-left: 12px;
          margin: 0;
          color: ${isDark ? '#9ca3af' : '#6b7280'};
        }
      </style>
    </head>
    <body>${props.email.bodyHtml}</body>
    </html>
  `
}

const iframeContent = computed(() => {
  const isDark = settingStore.settings.theme === 'dark'
  return generateIframeContent(isDark)
})

// Key for iframe to trigger re-render when theme changes
const iframeKey = computed(() => {
  return `${props.email?.id || ''}-${settingStore.settings.theme}`
})

// Watch for email changes to reset iframe height
watch(
  () => props.email?.bodyHtml,
  () => {
    iframeHeight.value = 0
  }
)

// Watch for theme changes to re-render iframe with new theme
watch(
  () => settingStore.settings.theme,
  () => {
    // Reset iframe height to trigger re-measurement
    iframeHeight.value = 0
  }
)

function onIframeLoad() {
  if (!emailBodyIframe.value) return
  
  // Delay to ensure content is fully rendered
  setTimeout(() => {
    try {
      const iframe = emailBodyIframe.value
      if (!iframe) return
      
      const doc = iframe.contentDocument || iframe.contentWindow?.document
      if (doc) {
        // Use the larger of scrollHeight or clientHeight
        const height = Math.max(doc.body.scrollHeight, doc.body.clientHeight)
        iframeHeight.value = height
      }
    } catch (e) {
      console.warn('Unable to access iframe content:', e)
    }
  }, 100)
}

// Methods
function formatFullDateTime(mailTime: string): string {
  if (!mailTime) return ''
  
  const date = new Date(mailTime)
  if (isNaN(date.getTime())) return mailTime
  
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

function handleReply() {
  if (props.email) {
    emit('reply', props.email)
  }
}

function handleReplyAll() {
  if (props.email) {
    emit('replyAll', props.email)
  }
}

function handleForward() {
  if (props.email) {
    emit('forward', props.email)
  }
}

function handleEdit() {
  if (props.email) {
    emit('edit', props.email)
  }
}

function handleDelete() {
  if (props.email) {
    emit('delete', props.email.id)
  }
}

function handleToggleStar() {
  if (props.email) {
    emit('toggleStar', props.email.id)
  }
}

function handleClose() {
  emit('close')
}

function handlePrevEmail() {
  emit('prevEmail')
}

function handleNextEmail() {
  emit('nextEmail')
}

function handlePrint() {
  if (!props.email) return
  showPrintPreview.value = true
}

function handleClosePrintPreview() {
  showPrintPreview.value = false
}

function handleExecutePrint() {
  if (!props.email) return
  
  const email = props.email
  const fmt = (r?: Array<{name?: string; email?: string}>) => r?.map(x => x.name ? `${x.name} <${x.email}>` : x.email).join(', ') || ''
  
  const ccHtml = email.cc?.length ? '<p><b>Cc:</b> ' + fmt(email.cc) + '</p>' : ''
  const attachmentsHtml = email.attachments?.length 
    ? '<p><b>Attachments:</b> ' + email.attachments.map(a => a.name + ' (' + formatFileSize(a.size) + ')').join(', ') + '</p>' 
    : ''
  
  // Create hidden iframe for printing
  const printIframe = document.createElement('iframe')
  printIframe.style.position = 'fixed'
  printIframe.style.right = '0'
  printIframe.style.bottom = '0'
  printIframe.style.width = '0'
  printIframe.style.height = '0'
  printIframe.style.border = '0'
  document.body.appendChild(printIframe)
  
  const printDoc = printIframe.contentDocument || printIframe.contentWindow?.document
  if (printDoc) {
    printDoc.open()
    printDoc.write([
      '<html><head>',
      '<meta charset="utf-8">',
      '<title>' + (email.subject || '') + '</title>',
      '<style>',
      'body { font-family: -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif; font-size: 14px; line-height: 1.6; color: #333; padding: 40px; }',
      'h1 { font-size: 24px; margin-bottom: 20px; color: #111; }',
      '.meta { margin-bottom: 20px; }',
      '.meta p { margin: 5px 0; }',
      'hr { border: none; border-top: 1px solid #ddd; margin: 20px 0; }',
      '.content { line-height: 1.8; }',
      '.content table { border-collapse: collapse; }',
      '.content th, .content td { border: 1px solid #ddd; padding: 8px; }',
      '.content img { max-width: 100%; }',
      '</style>',
      '</head><body>',
      '<h1>' + (email.subject || '(No Subject)') + '</h1>',
      '<div class="meta">',
      '<p><b>From:</b> ' + fmt([email.from]) + '</p>',
      '<p><b>To:</b> ' + fmt(email.to) + '</p>',
      ccHtml,
      '<p><b>Date:</b> ' + formatFullDateTime(email.mailTime || email.createdAt || '') + '</p>',
      '</div>',
      '<hr>',
      '<div class="content">',
      email.bodyHtml || email.body || 'No content',
      '</div>',
      attachmentsHtml,
      '</body></' + 'html>'
    ].join(''))
    printDoc.close()
    
    setTimeout(() => {
      printIframe.contentWindow?.print()
      showPrintPreview.value = false
      // Remove iframe after printing
      setTimeout(() => {
        document.body.removeChild(printIframe)
      }, 1000)
    }, 200)
  }
}

function openAddContactModal(email: string, name: string) {
  selectedContactEmail.value = email
  selectedContactName.value = name
  showAddContactModal.value = true
}

function handleAddContactSuccess() {
  showAddContactModal.value = false
}

function handleCloseAddContactModal() {
  showAddContactModal.value = false
}

// Headers modal methods
function openHeadersModal() {
  showRawSource.value = false
  showHeadersModal.value = true
}

function closeHeadersModal() {
  showHeadersModal.value = false
  headersCopied.value = false
  rawCopied.value = false
}

async function copyHeaders() {
  if (!props.email?.headers) return
  const text = Object.entries(props.email.headers)
    .map(([key, value]) => `${key}: ${value}`)
    .join('\n')
  await navigator.clipboard.writeText(text)
  headersCopied.value = true
  setTimeout(() => { headersCopied.value = false }, 2000)
  showToast('success', t('readingPane.copied'))
}

async function copyRawSource() {
  if (!props.email?.headers) return
  const text = Object.entries(props.email.headers)
    .map(([key, value]) => `${key}: ${value}`)
    .join('\r\n') + '\r\n\r\n' + (props.email.body || '')
  await navigator.clipboard.writeText(text)
  rawCopied.value = true
  setTimeout(() => { rawCopied.value = false }, 2000)
  showToast('success', t('readingPane.copied'))
}

// Commonly displayed headers in order
const COMMON_HEADERS = [
  'from', 'to', 'cc', 'date', 'subject', 'message-id',
  'received', 'mime-version', 'content-type', 'content-transfer-encoding',
  'x-mailer', 'x-priority', 'return-path', 'reply-to', 'in-reply-to', 'references'
]

const sortedHeaders = computed(() => {
  if (!props.email?.headers) return []
  const entries = Object.entries(props.email.headers)
  // Sort: common headers first (in defined order), then the rest alphabetically
  const commonMap = new Map(COMMON_HEADERS.map((h, i) => [h, i]))
  return entries.sort(([a], [b]) => {
    const aCommon = commonMap.has(a)
    const bCommon = commonMap.has(b)
    if (aCommon && bCommon) return commonMap.get(a)! - commonMap.get(b)!
    if (aCommon) return -1
    if (bCommon) return 1
    return a.localeCompare(b)
  })
})

function formatHeaderName(key: string): string {
  return key.split('-').map(part => part.charAt(0).toUpperCase() + part.slice(1)).join('-')
}

async function handleDownloadAttachment(attachment: Attachment) {
  if (!props.email) return
  
  loadingAttachments.value[attachment.index] = true
  try {
    const blob = await downloadAttachment(props.email.id, attachment.index)
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = attachment.name
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  } catch (error) {
    console.error('Failed to download attachment:', error)
    showToast('error', t('mail.failedToDownload'))
  } finally {
    loadingAttachments.value[attachment.index] = false
  }
}

async function handleDownloadAll() {
  if (!props.email) return
  
  try {
    const blob = await downloadAllAttachments(props.email.id)
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `attachments_${props.email.id}.zip`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  } catch (error) {
    console.error('Failed to download all attachments:', error)
    showToast('error', t('mail.failedToDownloadAll'))
  }
}
</script>

<template>
  <div class="h-full flex flex-col bg-white dark:bg-dark-surface">
    <div v-if="props.email" class="flex flex-col h-full">
      <!-- Top Toolbar: Navigation & Actions -->
      <div class="flex items-center justify-between px-4 py-2 border-b border-gray-200 dark:border-dark-border bg-gray-50 dark:bg-dark-bg shrink-0">
        <!-- Left: Navigation -->
        <div class="flex items-center gap-2">
          <button 
            @click="handlePrevEmail"
            class="p-1.5 rounded-lg hover:bg-gray-200 dark:hover:bg-dark-bg text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-dark-text transition-colors"
            :title="t('mail.prevEmail')"
          >
            <ChevronLeftIcon class="w-4 h-4" />
          </button>
          <button 
            @click="handleNextEmail"
            class="p-1.5 rounded-lg hover:bg-gray-200 dark:hover:bg-dark-bg text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-dark-text transition-colors"
            :title="t('mail.nextEmail')"
          >
            <ChevronRightIcon class="w-4 h-4" />
          </button>
        </div>
        
        <!-- Center: Action Buttons -->
        <div class="flex items-center gap-1">
          <button 
            @click="handleReply"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg hover:bg-gray-200 dark:hover:bg-dark-bg text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-dark-text transition-colors"
          >
            <ArrowUturnLeftIcon class="w-4 h-4" />
            <span>{{ t('mail.reply') }}</span>
          </button>
          <button 
            @click="handleReplyAll"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg hover:bg-gray-200 dark:hover:bg-dark-bg text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-dark-text transition-colors"
          >
            <ArrowUturnRightIcon class="w-4 h-4" />
            <span>{{ t('mail.replyAll') }}</span>
          </button>
          <button 
            @click="handleForward"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg hover:bg-gray-200 dark:hover:bg-dark-bg text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-dark-text transition-colors"
          >
            <ForwardIcon class="w-4 h-4" />
            <span>{{ t('mail.forward') }}</span>
          </button>
          <button 
            v-if="showEditButton"
            @click="handleEdit"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg hover:bg-gray-200 dark:hover:bg-dark-bg text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-dark-text transition-colors"
          >
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/></svg>
            <span>{{ t('mail.edit') }}</span>
          </button>
        </div>
        
        <!-- Right: More Actions -->
        <div class="flex items-center gap-1">
          <button 
            @click="handleToggleStar"
            class="p-1.5 rounded-lg hover:bg-gray-200 dark:hover:bg-dark-bg transition-colors"
            :title="props.email.isStarred ? t('mail.removeStar') : t('mail.addStar')"
          >
            <StarSolid v-if="props.email.isStarred" class="w-5 h-5 text-yellow-500" />
            <StarOutline v-else class="w-5 h-5 text-gray-400 dark:text-gray-500" />
          </button>
          <button 
            @click="openHeadersModal"
            class="p-1.5 rounded-lg hover:bg-gray-200 dark:hover:bg-dark-bg text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-dark-text transition-colors"
            :title="t('readingPane.viewHeaders')"
            v-if="false"
          >
            <DocumentTextIcon class="w-5 h-5" />
          </button>
          <button 
            @click="handlePrint"
            class="p-1.5 rounded-lg hover:bg-gray-200 dark:hover:bg-dark-bg text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-dark-text transition-colors"
            :title="t('mail.print')"
          >
            <PrinterIcon class="w-5 h-5" />
          </button>
          <button 
            @click="handleDelete"
            class="p-1.5 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 text-gray-600 dark:text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors"
            :title="t('mail.delete')"
          >
            <TrashIcon class="w-5 h-5" />
          </button>
          <div class="w-px h-5 bg-gray-300 mx-1"></div>
          <button 
            @click="handleClose"
            class="p-1.5 rounded-lg hover:bg-gray-200 dark:hover:bg-dark-bg text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-dark-text transition-colors"
            :title="t('mail.close')"
          >
            <XMarkIcon class="w-5 h-5" />
          </button>
        </div>
      </div>

      <!-- Email Content Area -->
      <div class="flex-1 overflow-y-auto">
        <!-- Email Header with Subject & Sender -->
        <div class="px-6 py-5">
          <!-- Subject Line -->
          <div class="flex items-start justify-between gap-4 mb-4">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-dark-text leading-tight">
              {{ props.email.subject }}
            </h2>
            <div class="flex items-center gap-2 shrink-0">
              <span v-if="props.email.isRead" class="px-2 py-1 text-xs font-medium text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 rounded">
                {{ t('mail.read') }}
              </span>
              <span v-if="props.email.labels && props.email.labels.length > 0" 
                    v-for="label in props.email.labels" 
                    :key="label.id"
                    class="px-2 py-1 text-xs font-medium rounded"
                    :style="{ 
                      backgroundColor: label.color + '20', 
                      color: label.color 
                    }">
                {{ label.name }}
              </span>
            </div>
          </div>
          
          <!-- Sender Info Row -->
          <div class="flex items-start gap-3">
            <!-- Avatar -->
            <div class="w-10 h-10 rounded-full bg-gradient-to-br from-blue-500 to-blue-600 text-white flex items-center justify-center text-base font-semibold shrink-0 shadow-sm">
              {{ props.email.from.name?.charAt(0)?.toUpperCase() || props.email.from.email?.charAt(0)?.toUpperCase() || '?' }}
            </div>
            
            <!-- Sender Details -->
            <div class="min-w-0 flex-1">
              <div class="flex items-baseline justify-between gap-2">
                <div class="min-w-0 flex items-center gap-1">
                  <span class="text-base font-semibold text-gray-900 dark:text-dark-text">
                    {{ props.email.from.name || props.email.from.email }}
                  </span>
                  <span v-if="props.email.from.name && props.email.from.name !== props.email.from.email" class="text-sm text-gray-500 dark:text-gray-400 ml-1">
                    <{{ props.email.from.email }}>
                  </span>
                  <button 
                    v-if="!isEmailInContacts(props.email.from.email)"
                    @click="openAddContactModal(props.email.from.email, props.email.from.name || '')"
                    class="ml-1.5 p-1 rounded hover:bg-blue-100 dark:hover:bg-blue-900/30 text-gray-400 dark:text-gray-500 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
                    :title="t('mail.addToContacts')"
                  >
                    <UserPlusIcon class="w-4 h-4" />
                  </button>
                </div>
                <span class="text-sm text-gray-500 dark:text-gray-400 shrink-0">
                  {{ formatFullDateTime(props.email.mailTime || props.email.createdAt || '') }}
                </span>
              </div>
            </div>
          </div>
          
          <!-- Recipients Info -->
          <div class="mt-3 space-y-1 text-sm">
            <div v-if="props.email.to && props.email.to.length > 0" class="flex items-start gap-2">
              <span class="text-gray-500 dark:text-gray-400 w-12 shrink-0">{{ t('mail.to') }}:</span>
              <div class="flex flex-wrap items-center gap-1.5 flex-1">
                <template v-for="(recipient, idx) in props.email.to" :key="idx">
                  <span class="text-gray-700 dark:text-gray-300">
                    {{ recipient.name ? `${recipient.name} <${recipient.email}>` : recipient.email }}
                  </span>
                  <button 
                    v-if="!isEmailInContacts(recipient.email)"
                    @click="openAddContactModal(recipient.email, recipient.name || '')"
                    class="p-0.5 rounded hover:bg-blue-100 dark:hover:bg-blue-900/30 text-gray-400 dark:text-gray-500 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
                    :title="t('mail.addToContacts')"
                  >
                    <UserPlusIcon class="w-3.5 h-3.5" />
                  </button>
                  <span v-if="idx < props.email.to!.length - 1" class="text-gray-500 dark:text-gray-400">, </span>
                </template>
              </div>
            </div>
            <div v-if="props.email.cc && props.email.cc.length > 0" class="flex items-start gap-2">
              <span class="text-gray-500 dark:text-gray-400 w-12 shrink-0">{{ t('mail.cc') }}:</span>
              <div class="flex flex-wrap items-center gap-1.5 flex-1">
                <template v-for="(recipient, idx) in props.email.cc" :key="idx">
                  <span class="text-gray-700 dark:text-gray-300">
                    {{ recipient.name ? `${recipient.name} <${recipient.email}>` : recipient.email }}
                  </span>
                  <button 
                    v-if="!isEmailInContacts(recipient.email)"
                    @click="openAddContactModal(recipient.email, recipient.name || '')"
                    class="p-0.5 rounded hover:bg-blue-100 dark:hover:bg-blue-900/30 text-gray-400 dark:text-gray-500 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
                    :title="t('mail.addToContacts')"
                  >
                    <UserPlusIcon class="w-3.5 h-3.5" />
                  </button>
                  <span v-if="idx < props.email.cc!.length - 1" class="text-gray-500 dark:text-gray-400">, </span>
                </template>
              </div>
            </div>
            <div v-if="props.email.bcc && props.email.bcc.length > 0" class="flex items-start gap-2">
              <span class="text-gray-500 dark:text-gray-400 w-12 shrink-0">{{ t('mail.bcc') }}:</span>
              <div class="flex flex-wrap items-center gap-1.5 flex-1">
                <template v-for="(recipient, idx) in props.email.bcc" :key="idx">
                  <span class="text-gray-700 dark:text-gray-300">
                    {{ recipient.name ? `${recipient.name} <${recipient.email}>` : recipient.email }}
                  </span>
                  <button 
                    v-if="!isEmailInContacts(recipient.email)"
                    @click="openAddContactModal(recipient.email, recipient.name || '')"
                    class="p-0.5 rounded hover:bg-blue-100 dark:hover:bg-blue-900/30 text-gray-400 dark:text-gray-500 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
                    :title="t('mail.addToContacts')"
                  >
                    <UserPlusIcon class="w-3.5 h-3.5" />
                  </button>
                  <span v-if="idx < props.email.bcc!.length - 1" class="text-gray-500 dark:text-gray-400">, </span>
                </template>
              </div>
            </div>
          </div>
        </div>
        
        <!-- Divider -->
        <div class="border-t border-gray-200 dark:border-dark-border"></div>

        <!-- Email Body -->
        <div class="px-6 py-4">
          <div v-if="props.email.bodyHtml" class="email-body-wrapper dark:text-dark-text">
            <iframe
              :key="iframeKey"
              ref="emailBodyIframe"
              class="w-full border-none overflow-hidden"
              :style="{ height: iframeHeight + 'px', minHeight: '100px' }"
              :srcdoc="iframeContent"
              sandbox="allow-same-origin allow-scripts"
              @load="onIframeLoad"
            ></iframe>
          </div>
          <div 
            v-else-if="props.email.body" 
            class="text-sm text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap"
          >
            {{ props.email.body }}
          </div>
          <div v-else class="text-sm text-gray-500 dark:text-gray-400 italic py-4">
            {{ t('mail.noContent') }}
          </div>
        </div>

        <!-- Attachments Section -->
        <div v-if="hasAttachments" class="px-6 pb-6">
          <div class="border-t border-gray-200 dark:border-dark-border pt-4">
            <!-- Attachments Header -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-2">
                <PaperClipIcon class="w-4 h-4 text-gray-500 dark:text-gray-400" />
                <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('mail.attachmentCount', { count: props.email.attachments.length }) }}
                </span>
              </div>
              <button 
                @click="handleDownloadAll"
                class="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-lg transition-colors"
              >
                <ArrowDownTrayIcon class="w-4 h-4" />
                {{ t('mail.downloadAll') }}
              </button>
            </div>
            
            <!-- Attachments Grid -->
            <div class="grid grid-cols-2 gap-2">
              <div 
                v-for="attachment in props.email.attachments" 
                :key="attachment.index"
                @click="!loadingAttachments[attachment.index] && handleDownloadAttachment(attachment)"
                class="flex items-center gap-3 p-3 border border-gray-200 dark:border-dark-border rounded-lg hover:border-blue-400 hover:bg-blue-50/50 dark:hover:bg-blue-900/20 cursor-pointer transition-all group"
              >
                <div class="w-10 h-10 rounded-lg bg-gray-100 dark:bg-gray-700 flex items-center justify-center shrink-0 group-hover:bg-blue-100 dark:group-hover:bg-blue-900/30 transition-colors">
                  <PaperClipIcon class="w-5 h-5 text-gray-500 dark:text-gray-400 group-hover:text-blue-600 dark:group-hover:text-blue-400" />
                </div>
                <div class="min-w-0 flex-1">
                  <div class="text-sm font-medium text-gray-700 dark:text-gray-300 truncate group-hover:text-blue-700 dark:group-hover:text-blue-400">
                    {{ attachment.name }}
                  </div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">
                    {{ formatFileSize(attachment.size) }}
                  </div>
                </div>
                <button 
                  @click.stop="handleDownloadAttachment(attachment)"
                  :disabled="loadingAttachments[attachment.index]"
                  class="p-1.5 rounded-lg hover:bg-blue-100 dark:hover:bg-blue-900/30 text-gray-400 dark:text-gray-500 hover:text-blue-600 dark:hover:text-blue-400 transition-colors disabled:opacity-50 shrink-0"
                >
                  <ArrowDownTrayIcon v-if="!loadingAttachments[attachment.index]" class="w-4 h-4" />
                  <svg v-else class="w-4 h-4 animate-spin text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
        
        <!-- Bottom Spacer -->
        <div class="h-8"></div>
      </div>
    </div>
    
    <!-- Empty State -->
    <div v-else class="flex flex-col h-full items-center justify-center bg-gray-50 dark:bg-dark-bg">
      <div class="w-24 h-24 rounded-2xl bg-gradient-to-br from-blue-100 to-blue-200 flex items-center justify-center mb-5 shadow-sm">
        <EnvelopeIcon class="w-12 h-12 text-blue-500" />
      </div>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-dark-text mb-2">{{ t('mail.noEmailSelected') }}</h3>
      <p class="text-sm text-gray-500 dark:text-gray-400 max-w-xs text-center">
        {{ t('mail.noEmailSelectedDesc') }}
      </p>
    </div>

    <!-- Headers Modal -->
    <div v-if="showHeadersModal && props.email" class="fixed inset-0 z-50" @click.self="closeHeadersModal">
      <div class="fixed inset-0 bg-black/40 animate-in fade-in duration-200"></div>
      <div class="fixed inset-4 md:inset-8 lg:inset-12 xl:inset-16 bg-white dark:bg-dark-surface rounded-xl shadow-2xl flex flex-col overflow-hidden animate-in zoom-in-95 duration-200">
        <!-- Modal Header -->
        <div class="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-dark-border bg-gray-50 dark:bg-dark-bg shrink-0">
          <div class="flex items-center gap-3">
            <DocumentTextIcon class="w-5 h-5 text-blue-600 dark:text-blue-400" />
            <h2 class="text-lg font-semibold text-gray-900 dark:text-dark-text">{{ t('readingPane.headers') }}</h2>
            <span class="px-2 py-0.5 text-xs font-medium bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400 rounded-full">
              {{ sortedHeaders.length }}
            </span>
          </div>
          <div class="flex items-center gap-2">
            <!-- Toggle: Table / Raw -->
            <div class="flex items-center bg-gray-200 dark:bg-dark-bg rounded-lg p-0.5">
              <button 
                @click="showRawSource = false"
                class="flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-md transition-all"
                :class="showRawSource 
                  ? 'text-gray-600 dark:text-gray-400 hover:text-gray-900' 
                  : 'bg-white dark:bg-dark-surface text-gray-900 dark:text-dark-text shadow-sm'"
              >
                <TableCellsIcon class="w-3.5 h-3.5" />
                {{ t('readingPane.headers') }}
              </button>
              <button 
                @click="showRawSource = true"
                class="flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-md transition-all"
                :class="!showRawSource 
                  ? 'text-gray-600 dark:text-gray-400 hover:text-gray-900' 
                  : 'bg-white dark:bg-dark-surface text-gray-900 dark:text-dark-text shadow-sm'"
              >
                <CodeBracketIcon class="w-3.5 h-3.5" />
                {{ t('readingPane.rawSource') }}
              </button>
            </div>
            <!-- Copy Button -->
            <button 
              @click="showRawSource ? copyRawSource() : copyHeaders()"
              class="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-lg transition-colors"
            >
              <ClipboardDocumentIcon class="w-4 h-4" />
              {{ (showRawSource ? rawCopied : headersCopied) ? t('readingPane.copied') : t('readingPane.copyHeaders') }}
            </button>
            <!-- Close Button -->
            <button 
              @click="closeHeadersModal"
              class="p-2 rounded-lg hover:bg-gray-200 dark:hover:bg-dark-bg text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-dark-text transition-colors"
            >
              <XMarkIcon class="w-5 h-5" />
            </button>
          </div>
        </div>
        
        <!-- Modal Body -->
        <div class="flex-1 overflow-y-auto">
          <!-- Table View -->
          <div v-if="!showRawSource" class="p-6">
            <div v-if="sortedHeaders.length > 0" class="border border-gray-200 dark:border-dark-border rounded-lg overflow-hidden">
              <div v-for="[key, value] in sortedHeaders" :key="key"
                class="flex border-b border-gray-100 dark:border-dark-border last:border-b-0 hover:bg-gray-50 dark:hover:bg-dark-bg transition-colors">
                <div class="w-56 shrink-0 px-4 py-3 text-sm font-mono font-semibold text-gray-700 dark:text-gray-300 border-r border-gray-100 dark:border-dark-border bg-gray-50/50 dark:bg-dark-bg/50">
                  {{ formatHeaderName(key) }}
                </div>
                <div class="flex-1 px-4 py-3 text-sm text-gray-600 dark:text-gray-400 break-all font-mono leading-relaxed">
                  {{ value }}
                </div>
              </div>
            </div>
            <div v-else class="flex flex-col items-center justify-center py-12">
              <DocumentTextIcon class="w-12 h-12 text-gray-300 dark:text-gray-600 mb-3" />
              <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('readingPane.noHeaders') }}</p>
            </div>
          </div>
          
          <!-- Raw Source View -->
          <div v-else class="p-6">
            <div class="bg-gray-900 dark:bg-black rounded-lg overflow-hidden">
              <div class="flex items-center justify-between px-4 py-2 bg-gray-800 dark:bg-gray-900 border-b border-gray-700">
                <div class="flex items-center gap-1.5">
                  <div class="w-3 h-3 rounded-full bg-red-500"></div>
                  <div class="w-3 h-3 rounded-full bg-yellow-500"></div>
                  <div class="w-3 h-3 rounded-full bg-green-500"></div>
                </div>
                <span class="text-xs text-gray-400 font-mono">raw source</span>
                <div class="w-16"></div>
              </div>
              <pre class="p-4 text-sm text-green-400 font-mono overflow-x-auto whitespace-pre-wrap break-all"><code>{{ sortedHeaders.map(([k, v]) => `${k}: ${v}`).join('\r\n') + '\r\n\r\n' + (props.email.body || '') }}</code></pre>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Add Contact Modal -->
    <AddContactModal 
      :is-open="showAddContactModal"
      :email="selectedContactEmail"
      :name="selectedContactName"
      @close="handleCloseAddContactModal"
      @success="handleAddContactSuccess"
    />

    <!-- Print Preview Modal -->
    <div v-if="showPrintPreview && props.email" class="fixed inset-0 z-50">
      <div class="fixed inset-0 bg-black/50" @click="handleClosePrintPreview"></div>
      <div class="fixed inset-4 bg-white dark:bg-dark-surface rounded-lg shadow-2xl flex flex-col overflow-hidden">
        <!-- Toolbar -->
        <div class="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-dark-border bg-gray-50 dark:bg-dark-bg shrink-0">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-dark-text">{{ t('mail.printPreview') }}</h2>
          <div class="flex items-center gap-3">
            <button 
              @click="handleExecutePrint"
              class="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              <PrinterIcon class="w-5 h-5" />
              {{ t('mail.print') }}
            </button>
            <button 
              @click="handleClosePrintPreview"
              class="p-2 rounded-lg hover:bg-gray-200 dark:hover:bg-dark-bg text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-dark-text transition-colors"
            >
              <XMarkIcon class="w-5 h-5" />
            </button>
          </div>
        </div>
        
        <!-- Preview Content -->
        <div class="flex-1 overflow-y-auto p-8 print-area">
          <div class="max-w-3xl mx-auto">
            <!-- Header -->
            <div class="border-b-2 border-gray-200 dark:border-dark-border pb-4 mb-4">
              <h1 class="text-2xl font-bold text-gray-900 dark:text-dark-text mb-4">
                {{ props.email.subject || t('mail.noSubject') }}
              </h1>
              <div class="space-y-2 text-sm">
                <div class="flex">
                  <span class="font-semibold text-gray-600 dark:text-gray-400 w-20">{{ t('mail.from') }}:</span>
                  <span class="text-gray-900 dark:text-dark-text">{{ props.email.from.name || props.email.from.email }}{{ props.email.from.name ? ' <' + props.email.from.email + '>' : '' }}</span>
                </div>
                <div class="flex">
                  <span class="font-semibold text-gray-600 dark:text-gray-400 w-20">{{ t('mail.to') }}:</span>
                  <span class="text-gray-900 dark:text-dark-text">{{ props.email.to?.map(r => r.name ? `${r.name} <${r.email}>` : r.email).join(', ') }}</span>
                </div>
                <div v-if="props.email.cc && props.email.cc.length > 0" class="flex">
                  <span class="font-semibold text-gray-600 dark:text-gray-400 w-20">{{ t('mail.cc') }}:</span>
                  <span class="text-gray-900 dark:text-dark-text">{{ props.email.cc.map(r => r.name ? `${r.name} <${r.email}>` : r.email).join(', ') }}</span>
                </div>
                <div class="flex">
                  <span class="font-semibold text-gray-600 dark:text-gray-400 w-20">{{ t('mail.date') }}:</span>
                  <span class="text-gray-900 dark:text-dark-text">{{ formatFullDateTime(props.email.mailTime || props.email.createdAt || '') }}</span>
                </div>
              </div>
            </div>
            
            <!-- Body -->
            <div class="mb-4">
              <div v-if="props.email.bodyHtml" v-html="props.email.bodyHtml" class="prose max-w-none"></div>
              <div v-else-if="props.email.body" class="whitespace-pre-wrap">{{ props.email.body }}</div>
              <div v-else class="text-gray-500 dark:text-gray-400 italic">{{ t('mail.noContent') }}</div>
            </div>
            
            <!-- Attachments -->
            <div v-if="props.email.attachments && props.email.attachments.length > 0" class="border-t border-gray-200 dark:border-dark-border pt-4">
              <h3 class="font-semibold text-gray-700 dark:text-gray-300 mb-2">{{ t('mail.attachments') }} ({{ props.email.attachments.length }})</h3>
              <div class="flex flex-wrap gap-2">
                <div v-for="a in props.email.attachments" :key="a.index" class="text-sm text-gray-600 dark:text-gray-400">
                  {{ a.name }} ({{ formatFileSize(a.size) }})
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Email body wrapper */
.email-body-wrapper {
  width: 100%;
}

.email-body-wrapper :deep(iframe) {
  width: 100%;
}

/* Print styles */
@media print {
  .print-area {
    overflow: visible !important;
  }
  
  .print-area ::v-deep * {
    all: revert;
  }
}
</style>
