import { createRouter, createWebHistory } from 'vue-router'
import { hasTokenPair } from '@/services/auth/token'
import { featureFlags } from '@/features/flags'

const adminChildren = [
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
    path: 'contests/:contestId',
    name: 'contest-detail',
    component: () => import('@/views/contest/ContestDetailView.vue'),
    props: true,
  },
  {
    path: 'tracks',
    name: 'tracks',
    component: () => import('@/views/track/TrackManagementView.vue'),
  },
  {
    path: 'scripts',
    name: 'scripts',
    component: () => import('@/views/script/ScriptManagementView.vue'),
  },
  {
    path: 'flows',
    name: 'flows',
    component: () => import('@/views/flow/FlowManagementView.vue'),
  },
  {
    path: 'works',
    name: 'works',
    component: () => import('@/views/work/WorkManagementView.vue'),
  },
  {
    path: 'authors',
    name: 'authors',
    component: () => import('@/views/author/AuthorManagementView.vue'),
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
]

if (featureFlags.judgeModule) {
  adminChildren.push({
    path: 'contests/:contestId/judges',
    name: 'contest-judges',
    component: () => import('@/views/shared/JudgePlaceholderView.vue'),
  })
}

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
      children: adminChildren,
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
