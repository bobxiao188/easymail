import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '../stores/auth'

// Dynamically import components for lazy loading
const Login = () => import('../views/Login.vue')
const Layout = () => import('../views/Layout.vue')
const Mail = () => import('../views/Mail.vue')
const Compose = () => import('../views/Compose.vue')
const Contacts = () => import('../views/Contacts.vue')
const Settings = () => import('../views/Settings.vue')
const Folder = () => import('../views/Folder.vue')
const NotFound = () => import('../views/NotFound.vue')

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: Login,
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: Layout,
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        redirect: () => ({ name: 'Inbox' })
      },
      // Email-related routes
      {
        path: 'inbox',
        name: 'Inbox',
        component: Mail,
        meta: { folder: 'inbox', title: 'Inbox' }
      },
      {
        path: 'sent',
        name: 'Sent',
        component: Mail,
        meta: { folder: 'sent', title: 'Sent' }
      },
      {
        path: 'drafts',
        name: 'Drafts',
        component: Mail,
        meta: { folder: 'drafts', title: 'Drafts' }
      },
      {
        path: 'trash',
        name: 'Trash',
        component: Mail,
        meta: { folder: 'trash', title: 'Trash' }
      },
      {
        path: 'spam',
        name: 'Spam',
        component: Mail,
        meta: { folder: 'spam', title: 'Spam' }
      },
      {
        path: 'archive',
        name: 'Archive',
        component: Mail,
        meta: { folder: 'archive', title: 'Archive' }
      },
      {
        path: 'quarantine',
        name: 'Quarantine',
        component: Mail,
        meta: { folder: 'quarantine', title: 'Quarantine' }
      },
      // Dynamic route - custom folders
      {
        path: 'folder/:folderId',
        name: 'CustomFolder',
        component: Mail,
        meta: { folder: null, title: 'Custom Folder' }
      },
      // Contacts page
      {
        path: 'contacts',
        name: 'Contacts',
        component: Contacts,
        meta: { title: 'Contacts' }
      },
      // Compose page
      {
        path: 'compose',
        name: 'Compose',
        component: Compose,
        meta: { title: 'Compose' }
      },
      // Settings page
      {
        path: 'settings',
        name: 'Settings',
        component: Settings,
        meta: { title: 'Settings' }
      },
      // Folder management page
      {
        path: 'folders',
        name: 'Folders',
        component: Folder,
        meta: { title: 'Folders' }
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'Not Found',
    component: NotFound,
    meta: { requiresAuth: false }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// Route guards
router.beforeEach(async (to: any, _from: any) => {
  const requiresAuth = to.matched.some((record: any) => record.meta?.requiresAuth)
  const token = localStorage.getItem('token')
  
  // Pages requiring authentication
  if (requiresAuth) {
    if (!token) {
      // Not logged in, redirect to login page, save redirect path
      return { name: 'Login', query: { redirect: to.fullPath } }
    }
    // On page refresh, token exists but user state is null — fetch it
    const authStore = useAuthStore()
    if (!authStore.user) {
      try {
        await authStore.fetchUser()
      } catch {
        // Token invalid, clear and redirect to login
        authStore.clear()
        return { name: 'Login', query: { redirect: to.fullPath } }
      }
    }
  }
  
  // Logged-in users accessing login page, redirect to inbox
  if (to.name === 'Login' && token) {
    return { name: 'Inbox' }
  }
  
  // Allow navigation
  return true
})

export default router
