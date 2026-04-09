export interface MenuItem {
  label: string
  routeName: string
}

export const sidebarMenu: MenuItem[] = [
  { label: '比赛看板', routeName: 'dashboard' },
  { label: '我的稿件', routeName: 'my-submissions' },
  { label: '账号信息', routeName: 'profile' },
]
