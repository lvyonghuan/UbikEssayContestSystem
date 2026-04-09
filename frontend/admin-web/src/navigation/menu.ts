export interface MenuItem {
  label: string
  routeName: string
}

export const sidebarMenu: MenuItem[] = [
  { label: '比赛看板', routeName: 'dashboard' },
  { label: '作者管理', routeName: 'authors' },
  { label: '子管理员', routeName: 'admins' },
  { label: '全局配置', routeName: 'global-config' },
]
