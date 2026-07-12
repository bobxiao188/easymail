import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/login'
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: () => import('../views/Layout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: 'profile',
        name: 'Profile',
        component: () => import('../views/Profile.vue')
      },
      {
        path: 'domains',
        name: 'Domains',
        component: () => import('../views/Domains.vue')
      },
      {
        path: 'filter/rules',
        name: 'FilterRules',
        component: () => import('../views/FilterRules.vue')
      },
      {
        path: 'classify-models',
        name: 'ClassifyModels',
        component: () => import('../views/ClassifyModels.vue')
      },
      {
        path: 'training',
        name: 'Training',
        component: () => import('../views/Training.vue')
      },
      {
        path: 'filter/logs',
        name: 'FilterLogs',
        component: () => import('../views/FilterLogs.vue')
      },
      {
        path: 'postfix/status',
        name: 'PostfixStatus',
        component: () => import('../views/PostfixStatus.vue')
      },
      {
        path: 'postfix/configs',
        name: 'PostfixConfigs',
        component: () => import('../views/PostfixConfigs.vue')
      },
      {
        path: 'postfix/agents',
        name: 'PostfixAgents',
        component: () => import('../views/PostfixAgents.vue')
      },
      {
        path: 'postfix/queue',
        name: 'PostfixQueue',
        component: () => import('../views/PostfixQueue.vue')
      },
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue')
      }
    ]
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore()
  const requiresAuth = to.matched.some(record => record.meta.requiresAuth)

  if (requiresAuth && !authStore.isLoggedIn) {
    next({ name: 'Login', query: { redirect: to.fullPath } })
  } else if (to.name === 'Login' && authStore.isLoggedIn) {
    next({ name: 'Profile' })
  } else {
    next()
  }
})

export default router
