import { featureFlags } from '@/features/flags'

export interface MenuItem {
  label: string
  routeName: string
}

const baseMenu: MenuItem[] = [
  { label: '比赛看板', routeName: 'dashboard' },
  { label: '赛事管理', routeName: 'contests' },
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

export const sidebarMenu: MenuItem[] = baseMenu
