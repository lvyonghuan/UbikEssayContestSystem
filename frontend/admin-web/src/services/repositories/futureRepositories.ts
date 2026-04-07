import type { AdminProfile, Contest, DashboardSummary, GlobalConfig, RolePermission } from '@/types/api'
import dayjs from 'dayjs'
import isBetween from 'dayjs/plugin/isBetween'
import { fetchContests } from './contestRepository'

dayjs.extend(isBetween)

export async function fetchGlobalConfig(): Promise<GlobalConfig> {
  return {
    siteName: 'Ubik Essay Contest',
    emailAddress: 'admin@example.com',
    emailSmtpServer: 'smtp.example.com',
    emailSmtpPort: '587',
  }
}

export async function fetchRolePermissions(): Promise<RolePermission[]> {
  return [
    {
      roleID: 1,
      roleName: 'superadmin',
      description: '拥有所有权限',
      permissions: ['super'],
    },
  ]
}

export async function fetchAdmins(): Promise<AdminProfile[]> {
  return [
    {
      adminID: 1,
      adminName: 'superadmin',
      adminEmail: 'superadmin@example.com',
      roleNames: ['superadmin'],
    },
  ]
}

export async function fetchDashboardSummary(): Promise<DashboardSummary> {
  return {
    totalContests: 0,
    totalTracks: 0,
    totalWorks: 0,
    totalReviewEvents: 0,
  }
}

export interface DashboardMetrics {
  trendData: Array<{ date: string; count: number }>
  trackDistribution: Array<{ name: string; value: number }>
  workStatusData: Array<{ name: string; value: number }>
}

export async function fetchDashboardMetrics(): Promise<DashboardMetrics> {
  const now = dayjs()
  const trendData = Array.from({ length: 7 }, (_, i) => ({
    date: now.subtract(6 - i, 'day').format('M/D'),
    count: Math.floor(Math.random() * 5),
  }))

  return {
    trendData,
    trackDistribution: [
      { name: '美文类', value: 45 },
      { name: '诗歌类', value: 32 },
      { name: '散文类', value: 28 },
      { name: '小说类', value: 38 },
    ],
    workStatusData: [
      { name: '草稿', value: 24 },
      { name: '待审核', value: 78 },
      { name: '已通过', value: 156 },
      { name: '已驳回', value: 12 },
    ],
  }
}

export async function fetchUpcomingContests(): Promise<Contest[]> {
  try {
    const contests = await fetchContests()
    const now = dayjs()
    const sevenDaysLater = now.add(7, 'day')

    return contests
      .filter((c: Contest) => {
        const startDate = dayjs(c.contestStartDate)
        return startDate.isAfter(now) && startDate.isBefore(sevenDaysLater)
      })
      .sort((a: Contest, b: Contest) => {
        return dayjs(a.contestStartDate).isBefore(dayjs(b.contestStartDate)) ? -1 : 1
      })
  } catch {
    return []
  }
}

export async function fetchOngoingContests(): Promise<Contest[]> {
  try {
    const contests = await fetchContests()
    const now = dayjs()

    return contests
      .filter((c: Contest) => {
        const startDate = dayjs(c.contestStartDate)
        const endDate = dayjs(c.contestEndDate)
        return now.isBetween(startDate, endDate, null, '[]')
      })
      .sort((a: Contest, b: Contest) => {
        return dayjs(a.contestEndDate).isBefore(dayjs(b.contestEndDate)) ? -1 : 1
      })
  } catch {
    return []
  }
}
