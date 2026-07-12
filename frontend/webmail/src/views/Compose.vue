<template>
  <div class="flex-1 flex flex-col min-w-0 bg-white dark:bg-dark-surface h-full">
    <!-- Toolbar -->
    <div class="flex items-center justify-between px-4 py-2 border-b border-gray-200 dark:border-dark-border bg-gray-50 dark:bg-dark-bg shrink-0">
      <!-- Left: Action Buttons -->
      <div class="flex items-center gap-1">
        <button 
          @click="handleSaveDraft"
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg hover:bg-gray-200 dark:hover:bg-dark-bg text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-dark-text transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="isSending"
        >
          <svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M19 21H5a2 2 0 01-2-2V5a2 2 0 012-2h11l5 5v11a2 2 0 01-2 2z"/>
            <polyline points="17,21 17,13 7,13 7,21"/>
            <polyline points="7,3 7,8 15,8"/>
          </svg>
          <span>{{ t('compose.saveDraft') }}</span>
        </button>
        <button 
          @click="send"
          class="flex items-center gap-1.5 px-4 py-1.5 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="isSending"
        >
          <svg v-if="isSending" class="animate-spin h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
          </svg>
          <svg v-else class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="22" y1="2" x2="11" y2="13"/>
            <polygon points="22,2 15,22 11,13 2,9"/>
          </svg>
          <span>{{ isSending ? t('compose.sending') : t('compose.send') }}</span>
        </button>
      </div>
      
      <!-- Right: Close Button -->
      <div class="flex items-center gap-1">
        <button 
          @click="goBack"
          class="p-1.5 rounded-lg hover:bg-gray-200 dark:hover:bg-dark-bg text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-dark-text transition-colors disabled:opacity-50"
          :disabled="isSending"
          :title="t('compose.cancel')"
        >
          <svg class="w-5 h-5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- Editor -->
    <div class="flex-1 overflow-y-auto">
      <div class="px-6 py-5 max-w-4xl">
        <!-- From Row -->
        <div class="flex items-center gap-3 mb-3">
          <span class="text-sm text-gray-500 dark:text-gray-400 w-20 shrink-0 text-right">{{ t('compose.from') }}:</span>
          <select 
            v-model="selectedSender" 
            class="flex-1 px-3 py-2 text-sm border border-gray-200 dark:border-dark-border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all bg-white dark:bg-dark-surface"
          >
            <option v-for="sender in senders" :key="sender.email" :value="sender">
              {{ sender.name }} &lt;{{ sender.email }}&gt;
            </option>
          </select>
        </div>

        <!-- To Row -->
        <div class="flex items-start gap-3 mb-3">
          <span class="text-sm text-gray-500 dark:text-gray-400 w-20 shrink-0 text-right py-2">{{ t('compose.to') }}:</span>
          <div class="flex-1">
            <RecipientInput v-model="recipients" :placeholder="t('compose.recipientPlaceholder')" />
          </div>
        </div>

        <!-- Cc Row -->
        <div class="flex items-start gap-3 mb-3">
          <span class="text-sm text-gray-500 dark:text-gray-400 w-20 shrink-0 text-right py-2">{{ t('compose.cc') }}:</span>
          <div class="flex-1 ">
            <RecipientInput v-model="ccRecipients" :placeholder="t('compose.recipientPlaceholder')" />
          </div>
        </div>

        <!-- Subject Row -->
        <div class="flex items-center gap-3 mb-4">
          <span class="text-sm text-gray-500 dark:text-gray-400 w-20 shrink-0 text-right">{{ t('compose.subject') }}:</span>
          <input 
            type="text" 
            v-model="subject" 
            :placeholder="t('compose.subjectPlaceholder')" 
            class="flex-1 px-3 py-2 text-sm border border-gray-200 dark:border-dark-border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all bg-white dark:bg-dark-surface" 
          />
        </div>

        <!-- Divider -->
        <div class="border-t border-gray-200 dark:border-dark-border mb-4"></div>

        <!-- Body - Rich Text Editor -->
        <div class="mb-4">
          <RichTextEditor 
            v-model="body" 
            :placeholder="t('compose.placeholder')"
          />
        </div>

        <!-- Attachments Section -->
        <div>
          <!-- Attachments Header -->
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-2">
              <PaperclipIcon class="w-4 h-4 text-gray-500 dark:text-gray-400" />
              <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('compose.attachments') }}</span>
            </div>
            <button 
              type="button" 
              @click="fileInput?.click()"
              class="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-lg transition-colors"
            >
              <svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="12" y1="5" x2="12" y2="19"/>
                <line x1="5" y1="12" x2="19" y2="12"/>
              </svg>
              {{ t('compose.addFiles') }}
            </button>
            <input type="file" ref="fileInput" multiple @change="handleFileSelect" class="hidden" />
          </div>
          
          <!-- Attachments List -->
          <div v-if="attachments.length > 0" class="grid grid-cols-1 gap-2">
            <div 
              v-for="(attachment, index) in attachments" 
              :key="index"
              class="flex items-center gap-3 p-3 border border-gray-200 dark:border-dark-border rounded-lg hover:border-blue-400 hover:bg-blue-50/50 dark:hover:bg-blue-900/20 transition-all group"
            >
              <div class="w-10 h-10 rounded-lg bg-gray-100 dark:bg-dark-bg flex items-center justify-center shrink-0 group-hover:bg-blue-100 dark:group-hover:bg-blue-900/30 transition-colors">
                <FileIcon class="w-5 h-5 text-gray-500 dark:text-gray-400 group-hover:text-blue-600" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="text-sm font-medium text-gray-700 dark:text-gray-300 truncate group-hover:text-blue-700">
                  {{ attachment.name }}
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ formatSize(attachment.size) }}
                </div>
              </div>
              <button 
                @click="removeAttachment(index)"
                class="p-1.5 rounded-lg hover:bg-red-100 text-gray-400 hover:text-red-600 transition-colors shrink-0"
              >
                <svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"/>
                  <line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </div>
          </div>
        </div>
        
        <!-- Bottom Spacer -->
        <div class="h-8"></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, useTemplateRef } from 'vue'
import { useRouter, useRoute, onBeforeRouteLeave } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { saveDraft, sendEmail, editDraft, updateDraft, getEmail } from '../api/email'
import { getProfile } from '../api/auth'
import { useSettingStore } from '../stores/setting'
import { PaperclipIcon, FileIcon } from 'lucide-vue-next'
import type { User, Recipient, ComposeAttachment, Email } from '../types'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import zhcn from 'dayjs/locale/zh-cn'
import RecipientInput from '../components/contact/RecipientInput.vue'
import RichTextEditor from '../components/mail/RichTextEditor.vue'
import { showToast } from '../utils/toast'

const { t } = useI18n()

dayjs.extend(relativeTime)
dayjs.locale(zhcn)

function getQuoteHTML(email: Email, _language: string, mode: string = 'reply', includeOriginal: boolean = true): string {
  const isForward = mode === 'forward'
  
  const dateStr = email.mailTime ? dayjs(email.mailTime).format('YYYY-MM-DD HH:mm') : ''
  const senderName = email.from?.name || email.from?.email || t('compose.unknown')
  const originalBody = email.bodyHtml || email.body || ''
  
  // Labels
  const headerLabel = t(isForward ? 'compose.forwardedMessage' : 'compose.originalMessage')
  const subjectLabel = t('compose.subjectLabel')
  const fromLabel = t('compose.fromLabel')
  const dateLabel = t('compose.dateLabel')
  
  // Build HTML blockquote
  let quoteHTML = `
    <p></p>
    <blockquote style="border-left: 3px solid #e5e7eb; padding-left: 1em; margin: 1em 0; color: #6b7280;">
      <p style="margin: 0 0 0.5em 0; font-weight: 600;">--------${headerLabel}--------</p>
      <p style="margin: 0 0 0.25em 0;"><strong>${subjectLabel}</strong> ${email.subject || ''}</p>
      <p style="margin: 0 0 0.25em 0;"><strong>${fromLabel}</strong> ${senderName}</p>
      <p style="margin: 0 0 1em 0;"><strong>${dateLabel}</strong> ${dateStr}</p>
      <div>${originalBody}</div>
  `
  
  // Include original message content based on includeOriginal parameter
  return includeOriginal ? quoteHTML : '<p></p>'
}

const router = useRouter()
const route = useRoute()
const settingStore = useSettingStore()

const subject = ref('')
const body = ref('')
const recipients = ref<Recipient[]>([])
const ccRecipients = ref<Recipient[]>([])
const attachments = ref<ComposeAttachment[]>([])
const fileInput = useTemplateRef<HTMLInputElement>('fileInput')
const isEditing = ref(false)
const draftId = ref<number | null>(null)
const isUploading = ref(false)
const isSending = ref(false)
const selectedSender = ref<Recipient | null>(null)
const senders = ref<Recipient[]>([])

onMounted(async () => {
  // Ensure settings are loaded
  if (!settingStore.settings.signature && !settingStore.isLoading) {
    await settingStore.loadSettings()
  }
  try {
    const profileResponse = await getProfile()
    const user: User = profileResponse.data
    if (user.email) {
      const displayName = user.name || user.email
      senders.value = [{ name: displayName, email: user.email }]
      selectedSender.value = { name: displayName, email: user.email }
    }
  } catch (error) { console.error('Failed to load user info:', error) }

  const id = route.query.id ? Number(route.query.id) : null
  const mode = route.query.mode as string
  const recipientEmail = route.query.recipient as string
  const recipientName = route.query.name as string
  if (recipientEmail) { recipients.value = [{ email: recipientEmail, name: recipientName || '' }] }

  if (mode === 'reply' || mode === 'replyAll' || mode === 'forward') {
    const originalId = route.query.originalId ? Number(route.query.originalId) : null
    if (originalId) {
      try {
        const response = await getEmail(originalId)
        const email: Email = response.data
        const language = settingStore.settings.language
        const includeOriginal = settingStore.settings.includeOriginalOnReply
        if (mode === 'forward') {
          subject.value = `FW: ${email.subject || ''}`
          // For forward, put original body at the top, then quote
          const originalBody = email.bodyHtml || email.body || ''
          body.value = originalBody + getQuoteHTML(email, language, mode, includeOriginal)
        } else if (mode === 'replyAll') {
          subject.value = `RE: ${email.subject || ''}`
          if (email.from) { recipients.value = [email.from] }
          if (email.cc && email.cc.length > 0) { recipients.value = [...recipients.value, ...email.cc] }
          // For reply, put quote at the bottom
          body.value = getQuoteHTML(email, language, mode, includeOriginal)
        } else {
          subject.value = `RE: ${email.subject || ''}`
          if (email.from) { recipients.value = [email.from] }
          // For reply, put quote at the bottom
          body.value = getQuoteHTML(email, language, mode, includeOriginal)
        }
      } catch (error) { console.error('Failed to load original email:', error) }
    }
  } else if (mode === 'edit' && id) {
    isEditing.value = true
    draftId.value = id
    try {
      const response = await editDraft(id)
      const data = response.data
      subject.value = data.subject || ''
      body.value = data.html || data.text || ''
      recipients.value = data.to || []
      ccRecipients.value = data.cc || []
      if (data.attachments && data.attachments.length > 0) { attachments.value = data.attachments }
    } catch (error) { console.error('Failed to load draft:', error) }
  }
})

// Block navigation while email is being sent
onBeforeRouteLeave((_to, _from) => {
  if (isSending.value) {
    showToast('warning', t('compose.sendingWait'))
    return false
  }
  // Return true or undefined to allow navigation
})

// Block browser close/refresh while sending
function onBeforeUnload(e: BeforeUnloadEvent) {
  if (isSending.value) {
    e.preventDefault()
    e.returnValue = ''
  }
}
onMounted(() => window.addEventListener('beforeunload', onBeforeUnload))
onUnmounted(() => window.removeEventListener('beforeunload', onBeforeUnload))

async function handleFileSelect(event: Event) {
  const target = event.target as HTMLInputElement
  const files = target.files
  if (files) {
    isUploading.value = true
    try {
      for (let i = 0; i < files.length; i++) {
        const file = files[i]
        const reader = new FileReader()
        const base64Content = await new Promise<string>((resolve, reject) => {
          reader.onload = () => resolve(reader.result as string)
          reader.onerror = reject
          reader.readAsDataURL(file)
        })
        attachments.value.push({ name: file.name, size: file.size, base64: base64Content })
      }
      showToast('success', t('compose.addedAttachments', { count: files.length }))
    } catch (error) { console.error('Failed to read attachments:', error); showToast('error', t('compose.failedToReadAttachments')) }
    finally { isUploading.value = false; if (fileInput.value) { fileInput.value.value = '' } }
  }
}

function removeAttachment(index: number) { attachments.value.splice(index, 1) }

function formatSize(bytes: number) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function appendSignature(html: string): string {
  const signature = settingStore.settings.signature
  if (!signature || !signature.trim()) return html
  
  // Check if signature already exists
  const signatureMarker = '<!-- signature -->'
  if (html.includes(signatureMarker)) {
    return html
  }
  
  // Append signature with marker
  const separator = '<br><br>-- \n'
  return html + separator + signature + '\n' + signatureMarker
}

const mailRefreshChannel = new BroadcastChannel('mail_refresh')

async function send() {
  if (!subject.value.trim()) { showToast('warning', t('compose.enterSubject')); return }
  if (recipients.value.length === 0) { showToast('warning', t('compose.addRecipient')); return }
  isSending.value = true
  const saveSent = settingStore.settings.saveSent
  const htmlWithSignature = appendSignature(body.value)
  if (!selectedSender.value) { showToast('error', t('compose.senderMissing')); return }

  try {
    await sendEmail({
      subject: subject.value, html: htmlWithSignature, from: selectedSender.value,
      to: recipients.value, cc: ccRecipients.value, attachments: attachments.value,
      saveSent: saveSent, draftId: isEditing.value && draftId.value ? draftId.value : undefined,
    })
    showToast('success', t('compose.emailSent'))
    mailRefreshChannel.postMessage({ action: 'refresh' })
    router.push(saveSent ? '/sent' : '/inbox')
  } catch (error: any) {
    console.error('Send email error:', error)
    const message = error.response?.data?.message || t('compose.sendFailed')
    showToast('error', message)
  } finally {
    isSending.value = false
  }
}

async function handleSaveDraft() {
  if (!subject.value.trim()) { showToast('warning', t('compose.enterSubject')); return }
  try {
    if (!selectedSender.value) { showToast('error', t('compose.senderMissing')); return }
    if (isEditing.value && draftId.value) {
      await updateDraft(draftId.value, { subject: subject.value, html: body.value, to: recipients.value, cc: ccRecipients.value, from: selectedSender.value, attachments: attachments.value })
      showToast('success', t('compose.draftSaved'))
      mailRefreshChannel.postMessage({ action: 'refresh' })
      setTimeout(() => router.push('/drafts'), 300)
    } else {
      await saveDraft({ subject: subject.value, html: body.value, to: recipients.value, cc: ccRecipients.value, from: selectedSender.value, attachments: attachments.value })
      showToast('success', t('compose.draftSaved'))
      mailRefreshChannel.postMessage({ action: 'refresh' })
      setTimeout(() => router.push('/drafts'), 300)
    }
  } catch (error: any) {
    console.error('Save draft error:', error)
    const message = error.response?.data?.message || t('compose.saveFailed')
    showToast('error', message)
  }
}

function goBack() {
  if (isSending.value) {
    showToast('warning', t('compose.sendingWait'))
    return
  }
  router.back()
}
</script>