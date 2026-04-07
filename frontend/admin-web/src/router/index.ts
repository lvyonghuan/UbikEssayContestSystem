import { createRouter, createWebHistory } from 'vue-router'
import { hasTokenPair } from '@/services/auth/token'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/auth/LoginView.vue'),
      meta: { requiresAuth: false },
    },
    {
      path: '/',
      component: () => import('@/layouts/AdminLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'dashboard',
          component: () => import('@/views/dashboard/DashboardView.vue'),
        },
        {
          path: 'contests',
          name: 'contests',
          component: () => import('@/views/contest/ContestManagementView.vue'),
        },
        {
          path: 'tracks',
          name: 'tracks',
          component: () => import('@/views/track/TrackManagementView.vue'),
        },
        {
          path: 'global-config',
          name: 'global-config',
          component: () => import('@/views/global/GlobalConfigView.vue'),
        },
        {
          path: 'roles',
          name: 'roles',
          component: () => import('@/views/rbac/RolePermissionView.vue'),
        },
        {
          path: 'admins',
          name: 'admins',
          component: () => import('@/views/admin/AdminManagementView.vue'),
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/shared/NotFoundView.vue'),
    },
  ],
})

router.beforeEach((to) => {
  const authenticated = hasTokenPair()
  if (to.meta.requiresAuth && !authenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'login' && authenticated) {
    return { name: 'dashboard' }
  }
  return true
})

export default router
