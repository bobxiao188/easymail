<!-- src/components/mail/MailList.vue -->
<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import type { EmailListItem, Label } from '../../types'
import { 
  MagnifyingGlassIcon, 
  StarIcon as StarOutline, TrashIcon, 
  FlagIcon, PaperClipIcon, EnvelopeIcon, ArrowPathIcon,
  ChevronLeftIcon, ChevronRightIcon, ChevronDownIcon
} from '@heroicons/vue/24/outline'
import { StarIcon as StarSolid } from '@heroicons/vue/24/solid'
import LabelSelector from './LabelSelector.vue'

const { t } = useI18n()
const props = withDefaults(defineProps<{
  emails?: EmailListItem[]
  selectedEmails?: number[]
  selectedEmailId?: number | null
  total?: number
  currentPage?: number
  totalPages?: number
  pageSize?: number
  isLoading?: boolean
  folderName?: string
  labels?: Label[]
  selectedLabelId?: number | null
}>(), {
  emails: () => [],
  selectedEmails: () => [],
  selectedEmailId: null,
  total: 0,
  currentPage: 1,
  totalPages: 1,
  pageSize: 20,
  isLoading: false,
  folderName: '',
  labels: () => [],
  selectedLabelId: null
})

// Emits
const emit = defineEmits<{
  compose: []
  search: [keyword: string]
  prevPage: []
  nextPage: []
  goToPage: [page: number]
  viewEmail: [id: number]
  toggleSelect: [id: number]
  toggleSelectAll: []
  toggleStar: [id: number]
  delete: []
  deleteSingle: [id: number]
  move: []
  markRead: []
  markUnread: []
  toggleStarBatch: []
  clearSelection: []
  labelChange: [data: { emailId: number | null, labelId: number | null }]
  containerResize: [height: number]
  labelFilterChange: [labelId: number | null]
}>()

// Container ref for auto page size calculation
const containerRef = ref<HTMLDivElement | null>(null)
let resizeObserver: ResizeObserver | null = null

// Notify parent of container height
function notifyContainerHeight() {
  if (containerRef.value) {
    emit('containerResize', containerRef.value.clientHeight)
  }
}

// Setup ResizeObserver
onMounted(() => {
  nextTick(() => {
    if (containerRef.value) {
      resizeObserver = new ResizeObserver(() => {
        notifyContainerHeight()
      })
      resizeObserver.observe(containerRef.value)
      
      // Initial notification
      notifyContainerHeight()
    }
  })
})

onBeforeUnmount(() => {
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
})

// Local state
const searchKeyword = ref('')

// Label filter state
const showLabelDropdown = ref(false)

// Label selector state
const showLabelSelector = ref(false)
const selectedEmailForLabel = ref<number | null>(null)

// Handle label filter change
function handleLabelFilterChange(labelId: number | null) {
  emit('labelFilterChange', labelId)
  showLabelDropdown.value = false
}

// Get label display text
function getLabelDisplayText(): string {
  if (props.selectedLabelId === null || props.selectedLabelId === 0) {
    return t('mailList.allLabels')
  }
  const label = props.labels.find(l => l.id === props.selectedLabelId)
  return label ? label.name : t('mailList.allLabels')
}

// Computed
const allSelected = computed(() => {
  return props.emails.length > 0 && 
         props.emails.every(m => props.selectedEmails.includes(m.id))
})

const visiblePages = computed(() => {
  const total = props.totalPages
  const current = props.currentPage
  const pages: number[] = []

  if (total <= 7) {
    for (let i = 1; i <= total; i++) pages.push(i)
    return pages
  }

  // Always show first page
  pages.push(1)

  // Ellipsis or gap before current range
  if (current > 3) pages.push(-1) // ellipsis marker

  // Pages around current
  const start = Math.max(2, current - 1)
  const end = Math.min(total - 1, current + 1)
  for (let i = start; i <= end; i++) pages.push(i)

  // Ellipsis or gap after current range
  if (current < total - 2) pages.push(-2)

  // Always show last page
  pages.push(total)

  return pages
})

function handleSearch() {
  emit('search', searchKeyword.value)
}

function prevPage() {
  if (props.currentPage > 1) {
    emit('prevPage')
  }
}

function nextPage() {
  if (props.currentPage < props.totalPages) {
    emit('nextPage')
  }
}

function goToPage(page: number) {
  emit('goToPage', page)
}

function handleViewEmail(id: number) {
  emit('viewEmail', id)
}

function toggleSelect(id: number) {
  emit('toggleSelect', id)
}

function toggleSelectAll() {
  emit('toggleSelectAll')
}

function toggleStar(id: number) {
  emit('toggleStar', id)
}

function handleDelete() {
  emit('delete')
}

function handleMove() {
  emit('move')
}

function handleMarkRead() {
  emit('markRead')
}

function handleMarkUnread() {
  emit('markUnread')
}

function handleToggleStarBatch() {
  emit('toggleStarBatch')
}

function clearSelection() {
  emit('clearSelection')
}

// Single email quick operations
async function handleDeleteSingle(id: number) {
  // Delete: move to trash
  emit('deleteSingle', id)
}

function handleToggleFlag(id: number) {
  // Open label selector
  selectedEmailForLabel.value = id
  showLabelSelector.value = true
}

function handleCloseLabelSelector() {
  showLabelSelector.value = false
  selectedEmailForLabel.value = null
}

function handleLabelChange(labelId: number | null) {
  // Label has been changed, can notify parent component via event to update local data
  emit('labelChange', { emailId: selectedEmailForLabel.value, labelId })
}

// Get current email label ID (used by LabelSelector to show selected state)
function getCurrentLabelId(emailId: number): number | null {
  const email = props.emails.find(e => e.id === emailId)
  if (email && email.labels && email.labels.length > 0) {
    return email.labels[0].id  // Return the first label's ID
  }
  return null
}

// Format email time
function formatMailTime(mailTime: string): string {
  if (!mailTime) return ''
  
  // Try to parse ISO format time
  const date = new Date(mailTime)
  if (isNaN(date.getTime())) return mailTime
  
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const diffMinutes = Math.floor(diff / 60000)
  const diffHours = Math.floor(diff / 3600000)
  const diffDays = Math.floor(diff / 86400000)
  
  // Today
  if (diffDays === 0) {
    if (diffHours === 0) {
      return t('mailList.minutesAgo', { count: diffMinutes })
    }
    return t('mailList.hoursAgo', { count: diffHours })
  }
  
  // Yesterday
  if (diffDays === 1) {
    return t('mailList.yesterday')
  }
  
  // This week
  if (diffDays < 7) {
    return t('mailList.daysAgo', { count: diffDays })
  }
  
  // Other: show date
  const month = date.getMonth() + 1
  const day = date.getDate()
  const year = date.getFullYear()
  const currentYear = now.getFullYear()
  
  if (year === currentYear) {
    return `${month}/${day}`
  }
  return `${month}/${day}/${year}`
}
</script>

<template>
  <div ref="containerRef" class="flex flex-col h-full bg-white dark:bg-dark-surface">
    <!-- Toolbar -->
    <div class="flex items-center justify-between px-3 py-2 border-b border-border dark:border-dark-border bg-white dark:bg-dark-surface shrink-0">
      <div class="flex items-center gap-2">
        <h2 v-if="props.folderName" class="text-lg font-semibold text-text-primary dark:text-dark-text">
          {{ props.folderName }}
        </h2>
        <span v-if="props.selectedEmails.length > 0" class="text-sm font-medium text-primary">
          {{ props.selectedEmails.length }} {{ t('mailList.selected') }}
        </span>
      </div>
      <div class="flex items-center gap-2">
        <!-- Label Filter -->
        <div v-if="props.labels.length > 0" class="relative">
          <button
            @click="showLabelDropdown = !showLabelDropdown"
            class="flex items-center gap-1 px-2.5 py-1.5 text-sm border border-border dark:border-dark-border rounded bg-white dark:bg-dark-bg text-text-primary dark:text-dark-text outline-none hover:border-primary transition-colors"
          >
            <FlagIcon class="w-3.5 h-3.5 text-gray-500 dark:text-gray-400" />
            <span class="max-w-[100px] truncate">{{ getLabelDisplayText() }}</span>
            <ChevronDownIcon class="w-3.5 h-3.5 text-gray-500 dark:text-gray-400 shrink-0" />
          </button>
          <!-- Label Dropdown Menu -->
          <div
            v-if="showLabelDropdown"
            @click.away="showLabelDropdown = false"
            class="absolute top-full right-0 mt-1 w-48 bg-white dark:bg-dark-surface border border-gray-200 dark:border-dark-border rounded-lg shadow-lg z-50 py-1 max-h-60 overflow-y-auto"
          >
            <button
              @click="handleLabelFilterChange(null)"
              class="w-full text-left px-3 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
              :class="props.selectedLabelId === null ? 'bg-primary/10 text-primary font-medium' : 'text-text-primary dark:text-gray-200'"
            >
              {{ t('mailList.allLabels') }}
            </button>
            <hr class="my-1 border-gray-200 dark:border-gray-700" />
            <button
              v-for="label in props.labels"
              :key="label.id"
              @click="handleLabelFilterChange(label.id)"
              class="w-full text-left px-3 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors flex items-center gap-2"
              :class="props.selectedLabelId === label.id ? 'bg-primary/10 text-primary font-medium' : 'text-text-primary dark:text-gray-200'"
            >
              <span
                class="w-2.5 h-2.5 rounded-full shrink-0"
                :style="{ backgroundColor: label.color }"
              ></span>
              <span class="truncate">{{ label.name }}</span>
            </button>
          </div>
        </div>
        <!-- Search -->
        <div class="relative">
          <MagnifyingGlassIcon class="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            v-model="searchKeyword"
            @keyup.enter="handleSearch"
            type="text"
            class="pl-8 pr-3 py-1.5 text-sm border border-border dark:border-dark-border rounded bg-white dark:bg-dark-bg text-text-primary dark:text-dark-text outline-none focus:border-primary w-40 sm:w-48 transition-colors"
            :placeholder="t('mailList.search')"
          />
        </div>
        <!-- Pagination control -->
        <div class="flex items-center gap-1 pl-2 border-l border-border dark:border-dark-border">
          <button @click="prevPage" :disabled="props.currentPage === 1" class="p-1 rounded hover:bg-gray-100 dark:hover:bg-dark-bg disabled:opacity-40 disabled:cursor-not-allowed transition-colors" :title="t('mailList.prevPage')">
            <ChevronLeftIcon class="w-4 h-4 text-text-secondary dark:text-dark-text" />
          </button>
          <span class="text-xs text-text-secondary dark:text-gray-400 min-w-[60px] text-center">
            {{ props.currentPage }} {{ t('mailList.of') }} {{ props.totalPages }}
          </span>
          <button @click="nextPage" :disabled="props.currentPage === props.totalPages" class="p-1 rounded hover:bg-gray-100 dark:hover:bg-dark-bg disabled:opacity-40 disabled:cursor-not-allowed transition-colors" :title="t('mailList.nextPage')">
            <ChevronRightIcon class="w-4 h-4 text-text-secondary dark:text-dark-text" />
          </button>
        </div>
      </div>
    </div>

    <!-- Batch Operations Bar -->
    <Transition name="batch-slide">
        <div v-if="props.selectedEmails.length > 0" class="flex items-center gap-2 px-3 py-2 border-b border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-900/20 shrink-0">
        <span class="text-xs font-semibold text-primary">
          {{ props.selectedEmails.length }} {{ t('mailList.selected') }}
        </span>
        <div class="flex items-center gap-1">
          <button @click="handleDelete" class="p-1.5 rounded hover:bg-blue-100 dark:hover:bg-blue-800 text-text-secondary hover:text-red-600 transition-colors" :title="t('mailList.delete')">
            <TrashIcon class="w-4 h-4" />
          </button>
          <button @click="handleMove" class="p-1.5 rounded hover:bg-blue-100 dark:hover:bg-blue-800 text-text-secondary hover:text-primary transition-colors" :title="t('mailList.move')">
            <ArrowPathIcon class="w-4 h-4" />
          </button>
          <button @click="handleMarkRead" class="p-1.5 rounded hover:bg-blue-100 dark:hover:bg-blue-800 text-text-secondary hover:text-primary transition-colors" :title="t('mailList.markRead')">
            <EnvelopeIcon class="w-4 h-4" />
          </button>
          <button @click="handleMarkUnread" class="p-1.5 rounded hover:bg-blue-100 dark:hover:bg-blue-800 text-text-secondary hover:text-gray-600 transition-colors" :title="t('mailList.markUnread')">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
            </svg>
          </button>
          <button @click="handleToggleStarBatch" class="p-1.5 rounded hover:bg-blue-100 dark:hover:bg-blue-800 text-text-secondary hover:text-yellow-500 transition-colors" :title="t('mailList.star')">
            <StarOutline class="w-4 h-4" />
          </button>
        </div>
        <button @click="clearSelection" class="ml-auto px-2 py-1 text-xs text-text-secondary hover:text-text-primary hover:bg-blue-100 dark:hover:bg-blue-800 rounded transition-colors">
          {{ t('mailList.clear') }}
        </button>
      </div>
    </Transition>

    <!-- Mail list container -->
    <div class="flex-1 overflow-y-auto">
      <!-- Loading State -->
      <div v-if="props.isLoading" class="py-2">
        <div v-for="n in 8" :key="n" class="px-3 py-3 border-b border-border dark:border-dark-border animate-pulse">
          <div class="flex items-center gap-3 mb-2">
            <div class="w-4 h-4 bg-gray-200 dark:bg-gray-700 rounded"></div>
            <div class="w-4 h-4 bg-gray-200 dark:bg-gray-700 rounded"></div>
            <div class="w-24 h-4 bg-gray-200 dark:bg-gray-700 rounded"></div>
            <div class="ml-auto w-16 h-4 bg-gray-200 dark:bg-gray-700 rounded"></div>
          </div>
          <div class="ml-9 h-3 bg-gray-200 dark:bg-gray-700 rounded w-3/4"></div>
        </div>
      </div>

      <!-- Email Items -->
      <div v-else>
        <!-- Select All Row -->
        <div v-if="props.emails.length > 0" class="sticky top-0 z-10 flex items-center gap-2 px-3 py-2 bg-gray-50 dark:bg-dark-bg border-b border-border dark:border-dark-border">
          <label class="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              class="rounded border-border dark:border-dark-border"
              :checked="allSelected"
              @change="toggleSelectAll"
            />
            <span class="text-xs text-text-secondary dark:text-gray-400">{{ t('mailList.selectAll') }}</span>
          </label>
          <span class="text-xs text-text-secondary dark:text-gray-400 ml-auto">
            {{ props.total }} {{ t('mailList.emails') }}
          </span>
        </div>

        <!-- Mail Items -->
        <div 
          v-for="(email, index) in props.emails" 
          :key="email.id"
          @click="handleViewEmail(email.id)"
          :class="[
            'flex items-center border-b border-border dark:border-dark-border cursor-pointer transition-colors group',
            props.selectedEmails.includes(email.id) ? 'bg-blue-100 dark:bg-blue-900/30' : 'hover:bg-gray-50 dark:hover:bg-dark-bg',
            props.selectedEmailId === email.id ? 'bg-blue-100 dark:bg-blue-900/30 ring-2 ring-primary' : ''
          ]"
          :style="{ animationDelay: `${index * 15}ms` }"
        >
          <!-- Left Actions -->
          <div class="flex items-center gap-1 px-2 shrink-0">
            <!-- Unread indicator -->
            <div v-if="!email.isRead" class="w-2.5 h-2.5 rounded-full bg-primary shrink-0" :title="t('mailList.unread')"></div>
            <div v-else class="w-2.5 h-2.5 shrink-0"></div>
            
            <input
              type="checkbox"
              class="rounded border-border dark:border-dark-border cursor-pointer"
              :checked="props.selectedEmails.includes(email.id)"
              @change.stop="toggleSelect(email.id)"
            />
            <button 
              @click.stop="toggleStar(email.id)" 
              class="p-0.5 rounded hover:bg-gray-200 dark:hover:bg-dark-bg transition-colors"
            >
              <StarSolid v-if="email.isStarred" class="w-4 h-4 text-yellow-500" />
              <StarOutline v-else class="w-4 h-4 text-gray-300 dark:text-gray-600" />
            </button>
          </div>

          <!-- Content: Sender, Subject, Preview, Time -->
          <div class="flex-1 min-w-0 flex flex-col justify-center py-2.5">
            <!-- Sender + Labels -->
            <div class="flex items-center gap-2 flex-wrap">
              <span :class="[
                'truncate text-sm',
                !email.isRead ? 'font-semibold text-text-primary dark:text-dark-text' : 'text-text-primary dark:text-dark-text'
              ]">
                {{ email.from.name || email.from.email }}
              </span>
              <!-- Label badges -->
              <template v-if="email.labels && email.labels.length > 0">
                <span 
                  v-for="label in email.labels"
                  :key="label.id"
                  class="shrink-0 px-1.5 py-0.5 text-[10px] font-medium rounded"
                  :style="{ 
                    backgroundColor: label.color + '20', 
                    color: label.color,
                    borderColor: label.color + '40'
                  }"
                  :title="label.name"
                >
                  {{ label.name }}
                </span>
              </template>
            </div>
            <!-- Subject + Attachment icon -->
            <div class="flex items-center gap-2">
              <span :class="[
                'truncate text-sm',
                !email.isRead ? 'font-semibold text-text-primary dark:text-dark-text' : 'text-text-secondary dark:text-gray-400'
              ]">
                {{ email.subject }}
              </span>
              <!-- Attachment badge -->
              <span v-if="email.hasAttachments" class="shrink-0 flex items-center gap-0.5 px-1.5 py-0.5 text-[10px] font-medium rounded-full bg-gray-100 dark:bg-gray-700 text-text-secondary">
                <PaperClipIcon class="w-3 h-3" />
              </span>
            </div>
            <!-- Snippet -->
            <span v-if="email.snippet" class="truncate text-xs text-text-secondary dark:text-gray-400 mt-0.5">
              {{ email.snippet }}
            </span>
          </div>

          <!-- Time -->
          <div class="shrink-0 px-3 text-xs text-text-secondary dark:text-gray-400">{{ formatMailTime(email.mailTime) }}</div>

          <!-- Hover actions -->
          <div class="shrink-0 flex items-center gap-1 px-2 opacity-0 group-hover:opacity-100 transition-opacity">
            <button @click.stop="handleDeleteSingle(email.id)" class="p-1.5 rounded hover:bg-gray-100 dark:hover:bg-dark-bg" :title="t('mailList.delete')">
              <TrashIcon class="w-4 h-4 text-text-secondary dark:text-dark-text" />
            </button>
            <button 
              @click.stop="handleToggleFlag(email.id)" 
              class="p-1.5 rounded hover:bg-gray-100 dark:hover:bg-dark-bg" 
              :title="email.labels && email.labels.length > 0 ? t('mailList.changeLabel') : t('mailList.setLabel')"
            >
              <FlagIcon :class="[
                'w-4 h-4 transition-colors',
                email.labels && email.labels.length > 0 
                  ? 'text-primary dark:text-blue-400' 
                  : 'text-text-secondary dark:text-dark-text'
              ]" />
            </button>
          </div>
        </div>

        <!-- Empty State -->
        <div v-if="props.emails.length === 0" class="flex flex-col items-center justify-center py-16 text-text-secondary dark:text-gray-400">
          <EnvelopeIcon class="w-12 h-12 mb-4 text-gray-300 dark:text-gray-600" />
          <p class="text-sm font-medium">{{ t('mailList.noEmails') }}</p>
          <p class="text-xs mt-1">{{ props.isLoading ? t('mailList.loading') : t('mailList.tryDifferentSearch') }}</p>
        </div>
      </div>
    </div>

    <!-- Pagination Footer -->
    <div v-if="props.total > props.pageSize || props.currentPage > 1" class="flex items-center justify-between px-3 py-2 border-t border-border dark:border-dark-border bg-gray-50 dark:bg-dark-bg shrink-0">
      <span class="text-xs text-text-secondary dark:text-gray-400">
        {{ (props.currentPage - 1) * props.pageSize + 1 }}-{{ Math.min(props.currentPage * props.pageSize, props.total) }} {{ t('mailList.of') }} {{ props.total }}
      </span>
      <div class="flex items-center gap-1">
        <button @click="prevPage" :disabled="props.currentPage === 1" class="p-1 rounded border border-border dark:border-dark-border hover:bg-gray-100 dark:hover:bg-dark-bg disabled:opacity-40 transition-colors">
          <ChevronLeftIcon class="w-3.5 h-3.5 text-text-secondary dark:text-dark-text" />
        </button>
        <div class="flex items-center gap-0.5">
          <template v-for="p in visiblePages" :key="p">
            <button
              v-if="p > 0"
              @click="goToPage(p)"
              :class="[
                'min-w-[28px] h-7 flex items-center justify-center text-xs font-medium rounded border transition-colors',
                p === props.currentPage 
                  ? 'bg-primary text-white border-primary' 
                  : 'border-transparent text-text-secondary hover:bg-gray-100 dark:hover:bg-dark-bg'
              ]"
            >
              {{ p }}
            </button>
            <span v-else class="px-1 text-xs text-text-secondary dark:text-gray-400 select-none">&hellip;</span>
          </template>
        </div>
        <button @click="nextPage" :disabled="props.currentPage === props.totalPages" class="p-1 rounded border border-border dark:border-dark-border hover:bg-gray-100 dark:hover:bg-dark-bg disabled:opacity-40 transition-colors">
          <ChevronRightIcon class="w-3.5 h-3.5 text-text-secondary dark:text-dark-text" />
        </button>
      </div>
    </div>

    <!-- Label Selector -->
    <LabelSelector 
      v-if="showLabelSelector && selectedEmailForLabel !== null"
      :email-id="selectedEmailForLabel"
      :current-label-id="getCurrentLabelId(selectedEmailForLabel)"
      @close="handleCloseLabelSelector"
      @change="handleLabelChange"
    />
  </div>
</template>

<style scoped>
.batch-slide-enter-active,
.batch-slide-leave-active {
  transition: all 200ms ease;
}

.batch-slide-enter-from,
.batch-slide-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>