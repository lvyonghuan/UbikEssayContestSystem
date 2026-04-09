import { createRouter, createWebHistory } from 'vue-router'
import { hasTokenPair } from '@/services/auth/token'

const authorChildren = [
  {
    path: '',
    redirect: { name: 'dashboard' },
  },
  {
    path: 'dashboard',
    name: 'dashboard',
    component: () => import('@/views/dashboard/DashboardView.vue'),
    meta: { title: '比赛看板' },
  },
  {
    path: 'contests/:contestId',
    name: 'contest-detail',
    component: () => import('@/views/contest/ContestDetailView.vue'),
    props: true,
    meta: { title: '比赛详情' },
  },
  {
    path: 'contests/:contestId/tracks/:trackId',
    name: 'track-detail',
    component: () => import('@/views/track/TrackDetailView.vue'),
    props: true,
    meta: { title: '赛道详情与投稿' },
  },
  {
    path: 'submissions',
    name: 'my-submissions',
    component: () => import('@/views/submission/MySubmissionsView.vue'),
    meta: { title: '我的稿件' },
  },
  {
    path: 'submissions/:workId/edit',
    name: 'edit-submission',
    component: () => import('@/views/submission/EditSubmissionView.vue'),
    props: true,
    meta: { title: '修改稿件' },
  },
  {
    path: 'profile',
    name: 'profile',
    component: () => import('@/views/auth/ProfileView.vue'),
    meta: { title: '账号信息' },
  },
]

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
      path: '/register',
      name: 'register',
      component: () => import('@/views/auth/RegisterView.vue'),
      meta: { requiresAuth: false },
    },
    {
      path: '/',
      component: () => import('@/layouts/AdminLayout.vue'),
      meta: { requiresAuth: true },
      children: authorChildren,
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
  if ((to.name === 'login' || to.name === 'register') && authenticated) {
    return { name: 'dashboard' }
  }
  return true
})

export default router
