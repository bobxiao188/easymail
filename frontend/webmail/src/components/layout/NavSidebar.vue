<!-- src/components/layout/NavSidebar.vue -->
<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { 
  InboxIcon, 
  PaperAirplaneIcon, 
  PencilSquareIcon, 
  TrashIcon, 
  ExclamationTriangleIcon,
  ShieldCheckIcon,
  FolderIcon,
  FolderOpenIcon,
  UsersIcon,
  Cog6ToothIcon,
} from '@heroicons/vue/24/outline'
import { useFolderStore } from '../../stores/folder'
import { FolderKind, FOLDER_ROUTE_MAP } from '../../utils/folder'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const folderStore = useFolderStore()

// Icon mapping
const FOLDER_ICONS: Record<number, any> = {
  [FolderKind.Inbox]: InboxIcon,
  [FolderKind.Sent]: PaperAirplaneIcon,
  [FolderKind.Draft]: PencilSquareIcon,
  [FolderKind.Trash]: TrashIcon,
  [FolderKind.Spam]: ExclamationTriangleIcon,
  [FolderKind.Quarantine]: ShieldCheckIcon,
}

// System folders (inbox, sent, drafts, trash, spam, quarantine) - with route info
const systemFoldersWithRoute = computed(() => {
  return folderStore.systemFolders.map(f => {
    const routeSlug = FOLDER_ROUTE_MAP[f.kind]
    return {
      ...f,
      key: routeSlug,
      icon: FOLDER_ICONS[f.kind] || FolderIcon,
      route: routeSlug,
    }
  })
})

// Custom folders - with route info
const customFoldersWithRoute = computed(() => {
  return folderStore.customFolders.map(f => {
    return {
      ...f,
      key: `folder/${f.id}`,
      icon: FolderIcon,
      route: `folder/${f.id}`,
    }
  })
})

// App items (contacts, settings)
const appItems = [
  { id: 'contacts', label: t('sidebar.contacts'), icon: UsersIcon, route: 'Contacts' },
  { id: 'settings', label: t('sidebar.settings'), icon: Cog6ToothIcon, route: 'Settings' },
]

// Check if current folder is active
function isActiveFolder(folderKey: string): boolean {
  if (folderKey.startsWith('folder/')) {
    const folderId = folderKey.split('/')[1]
    return route.params.folderId === folderId
  }
  return route.path.startsWith(`/${folderKey}`) || route.name === folderKey.charAt(0).toUpperCase() + folderKey.slice(1)
}

// Check if an app route is active
function isActiveRoute(name: string): boolean {
  return route.name === name
}

// Navigate to folder
function navigateToFolder(folderKey: string, folderId?: number) {
  if (folderKey.startsWith('folder/')) {
    router.push({ name: 'CustomFolder', params: { folderId } })
  } else {
    const routeName = folderKey.charAt(0).toUpperCase() + folderKey.slice(1)
    router.push({ name: routeName })
  }
}

// Navigate to a named route
function navigateTo(name: string) {
  router.push({ name })
}

// Compose new mail
function compose() {
  router.push('/compose')
}

onMounted(() => {
  folderStore.loadFolders()
})
</script>

<template>
  <aside class="w-60 bg-sidebar dark:bg-dark-surface border-r border-border dark:border-dark-border flex flex-col overflow-hidden">
    <!-- Compose button -->
    <div class="p-3">
      <button
        @click="compose"
        class="w-full flex items-center justify-center gap-2 bg-primary hover:bg-primary/90 text-white text-sm font-medium px-3 py-2.5 rounded-md transition-colors"
      >
        <PencilSquareIcon class="w-5 h-5" />
        <span>{{ t('sidebar.compose') }}</span>
      </button>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 overflow-y-auto px-2 pb-4 space-y-5">
      <!-- Mail group -->
      <div>
        <div class="mb-1 px-3 text-xs font-semibold uppercase text-text-secondary dark:text-gray-400 tracking-wider">
          {{ t('sidebar.mail') }}
        </div>
        <div class="space-y-0.5">
          <div
            v-for="folder in systemFoldersWithRoute"
            :key="folder.key"
            @click="navigateToFolder(folder.route)"
            :class="[
              'flex items-center gap-3 px-3 py-2 rounded-md cursor-pointer text-sm transition-colors',
              isActiveFolder(folder.route)
                ? 'bg-blue-50 dark:bg-blue-900/20 text-primary border-l-2 border-primary'
                : 'hover:bg-gray-200 dark:hover:bg-dark-bg text-text-primary dark:text-dark-text border-l-2 border-transparent'
            ]"
          >
            <component :is="folder.icon" class="w-5 h-5 shrink-0" />
            <span class="flex-1 truncate">{{ folder.name }}</span>
            <span v-if="(folder.unreadCount || 0) > 0" class="bg-primary text-white text-xs font-bold px-2 py-0.5 rounded-full">{{ folder.unreadCount || 0 }}</span>
          </div>
        </div>
      </div>

      <!-- My Folders group -->
      <div>
        <div class="mb-1 px-3 text-xs font-semibold uppercase text-text-secondary dark:text-gray-400 tracking-wider">
          {{ t('sidebar.folders') }}
        </div>
        <div class="space-y-0.5">
          <div
            v-for="folder in customFoldersWithRoute"
            :key="folder.key"
            @click="navigateToFolder(folder.route, folder.id)"
            :class="[
              'flex items-center gap-3 px-3 py-2 rounded-md cursor-pointer text-sm transition-colors',
              isActiveFolder(folder.key)
                ? 'bg-blue-50 dark:bg-blue-900/20 text-primary border-l-2 border-primary'
                : 'hover:bg-gray-200 dark:hover:bg-dark-bg text-text-primary dark:text-dark-text border-l-2 border-transparent'
            ]"
          >
            <component :is="folder.icon" class="w-5 h-5 shrink-0" />
            <span class="flex-1 truncate">{{ folder.name }}</span>
            <span v-if="(folder.unreadCount || 0) > 0" class="bg-primary text-white text-xs font-bold px-2 py-0.5 rounded-full">{{ folder.unreadCount || 0 }}</span>
          </div>

          <!-- Manage folders link -->
          <div
            @click="navigateTo('Folders')"
            :class="[
              'flex items-center gap-3 px-3 py-2 rounded-md cursor-pointer text-sm transition-colors',
              isActiveRoute('Folders')
                ? 'bg-blue-50 dark:bg-blue-900/20 text-primary border-l-2 border-primary'
                : 'hover:bg-gray-200 dark:hover:bg-dark-bg text-text-secondary dark:text-gray-400 border-l-2 border-transparent'
            ]"
          >
            <FolderOpenIcon class="w-5 h-5 shrink-0" />
            <span class="flex-1 truncate">{{ t('sidebar.manageFolders') }}</span>
          </div>
        </div>
      </div>

      <!-- Apps group -->
      <div>
        <div class="mb-1 px-3 text-xs font-semibold uppercase text-text-secondary dark:text-gray-400 tracking-wider">
          {{ t('sidebar.apps') }}
        </div>
        <div class="space-y-0.5">
          <div
            v-for="item in appItems"
            :key="item.id"
            @click="navigateTo(item.route)"
            :class="[
              'flex items-center gap-3 px-3 py-2 rounded-md cursor-pointer text-sm transition-colors',
              isActiveRoute(item.route)
                ? 'bg-blue-50 dark:bg-blue-900/20 text-primary border-l-2 border-primary'
                : 'hover:bg-gray-200 dark:hover:bg-dark-bg text-text-primary dark:text-dark-text border-l-2 border-transparent'
            ]"
          >
            <component :is="item.icon" class="w-5 h-5 shrink-0" />
            <span>{{ item.label }}</span>
          </div>
        </div>
      </div>
    </nav>

    <!-- Copyright -->
    <div class="shrink-0 px-4 py-3 border-t border-border dark:border-dark-border">
      <p class="text-[11px] text-text-secondary dark:text-gray-400 text-center leading-relaxed">
        <a href="https://github.com/bobxiao188/easymail" target="_blank" rel="noopener noreferrer" class="text-text-secondary dark:text-gray-400">
          &copy; {{ new Date().getFullYear() }} EasyMail AGPLv3
        </a>
        <br/>
        <a
          href="mailto:3680010825@qq.com"
          target="_blank"
          rel="noopener noreferrer"
          class="text-text-secondary dark:text-gray-400"
        >
          <span class="opacity-80">3680010825@qq.com</span>
        </a>
      </p>
    </div>
  </aside>
</template>
