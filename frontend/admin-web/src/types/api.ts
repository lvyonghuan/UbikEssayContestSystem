export interface ApiResponse<T = unknown> {
  code: number
  msg: T
}

export interface AdminLoginPayload {
  adminName: string
  password: string
}

export interface TokenPair {
  access_token: string
  refresh_token: string
}

export interface Contest {
  contestID?: number
  contestName: string
  contestIntroduction?: string
  contestStartDate: string
  contestEndDate: string
}

export interface Track {
  trackID?: number
  contestID?: number
  trackName: string
  trackDescription?: string
  trackSettings?: Record<string, unknown>
}

export interface GlobalConfig {
  siteName: string
  emailAddress: string
  emailSmtpServer: string
  emailSmtpPort: string
}

export interface RolePermission {
  roleID: number
  roleName: string
  description: string
  permissions: string[]
}

export interface AdminProfile {
  adminID: number
  adminName: string
  adminEmail?: string
  roleNames: string[]
}

export interface DashboardSummary {
  totalContests: number
  totalTracks: number
  totalWorks: number
  totalReviewEvents: number
}
