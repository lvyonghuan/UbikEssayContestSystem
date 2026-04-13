import { featureFlags } from '@/features/flags'

export interface MenuItem {
  label: string
  routeName: string
}

const baseMenu: MenuItem[] = [
  { label: '比赛管理', routeName: 'dashboard' },
  { label: '赛事配置', routeName: 'contests' },
  { label: '赛道管理', routeName: 'tracks' },
  { label: '作品管理', routeName: 'works' },
  { label: '作者管理', routeName: 'authors' },
  { label: '子管理员', routeName: 'admins' },
  { label: '全局配置', routeName: 'global-config' },
]

if (featureFlags.scriptModule) {
  baseMenu.splice(4, 0,
    { label: '脚本管理', routeName: 'scripts' },
    { label: '流程管理', routeName: 'flows' },
  )
}

if (featureFlags.judgeModule) {
  const worksIndex = baseMenu.findIndex((item) => item.routeName === 'works')
  const insertAt = worksIndex >= 0 ? worksIndex + 1 : baseMenu.length
  baseMenu.splice(insertAt, 0, { label: '评审管理', routeName: 'judge-review' })
}

export const sidebarMenu: MenuItem[] = baseMenu
