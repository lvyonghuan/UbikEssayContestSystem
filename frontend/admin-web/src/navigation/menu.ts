export interface MenuItem {
  label: string
  routeName: string
}

export const sidebarMenu: MenuItem[] = [
  { label: '看板', routeName: 'dashboard' },
  { label: '赛事管理', routeName: 'contests' },
  { label: '赛道管理', routeName: 'tracks' },
  { label: '全局配置', routeName: 'global-config' },
  { label: '角色权限', routeName: 'roles' },
  { label: '管理员管理', routeName: 'admins' },
]
